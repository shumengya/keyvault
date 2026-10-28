package main

import "net/http"
import "net/http/httptest"
import "strings"
import "testing"
import "time"

func TestPasswordsMatch(tester *testing.T) {
	var env *testEnv = newTestEnv(tester, nil)
	type matchCase struct {
		name   string
		first  string
		second string
		want   bool
	}
	var cases []matchCase = []matchCase{
		{name: "正序", first: testFirst, second: testSecond, want: true},
		{name: "颠倒顺序", first: testSecond, second: testFirst, want: true},
		{name: "只对第一个", first: testFirst, second: "wrong", want: false},
		{name: "只对第二个", first: "wrong", second: testSecond, want: false},
		{name: "密码1输两遍", first: testFirst, second: testFirst, want: false},
		{name: "密码2输两遍", first: testSecond, second: testSecond, want: false},
		{name: "全错", first: "x", second: "y", want: false},
		{name: "全空", first: "", second: "", want: false},
		{name: "超长", first: strings.Repeat("a", maxPasswordBytes+1), second: testSecond, want: false},
	}
	var index int = 0
	for index = 0; index < len(cases); index++ {
		var current matchCase = cases[index]
		var got, err = env.server.passwordsMatch(current.first, current.second)
		if err != nil {
			tester.Fatalf("%s: %v", current.name, err)
		}
		if got != current.want {
			tester.Errorf("%s: 结果 %v, 期望 %v", current.name, got, current.want)
		}
	}
}

func TestLoginFlow(tester *testing.T) {
	var env *testEnv = newTestEnv(tester, nil)

	expectStatus(tester, "未登录读条目", env.call(http.MethodGet, "/api/entries", "", nil), http.StatusUnauthorized)
	var recorder *httptest.ResponseRecorder = env.call(http.MethodGet, "/api/session", "", nil)
	expectStatus(tester, "未登录查会话", recorder, http.StatusOK)
	if !strings.Contains(recorder.Body.String(), `"authenticated":false`) {
		tester.Fatalf("未登录应返回 authenticated:false, 实际 %s", recorder.Body.String())
	}

	recorder = env.call(http.MethodPost, loginPath, loginBody(testFirst, "wrong"), nil)
	expectStatus(tester, "密码错一个", recorder, http.StatusUnauthorized)
	if len(recorder.Result().Cookies()) != 0 {
		tester.Fatal("登录失败不该下发 Cookie")
	}

	// 颠倒顺序也能登录
	recorder = env.call(http.MethodPost, loginPath, loginBody(testSecond, testFirst), nil)
	expectStatus(tester, "颠倒顺序登录", recorder, http.StatusOK)
	var cookie *http.Cookie = sessionCookieFrom(tester, recorder)
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/" || cookie.Secure {
		tester.Errorf("Cookie 属性不对: %+v", cookie)
	}
	if cookie.MaxAge != 24*3600 {
		tester.Errorf("默认有效期 MaxAge = %d, 期望 86400", cookie.MaxAge)
	}

	// 库里只存令牌哈希,拿明文令牌查不到
	var expiresAt, found, err = env.server.store.SessionExpiry(hashToken(cookie.Value))
	if err != nil || !found {
		tester.Fatalf("按哈希应能查到会话, found=%v err=%v", found, err)
	}
	if !expiresAt.Equal(env.clock.current.Add(24 * time.Hour)) {
		tester.Errorf("到期时间 %v, 期望 %v", expiresAt, env.clock.current.Add(24*time.Hour))
	}
	_, found, err = env.server.store.SessionExpiry(cookie.Value)
	if err != nil || found {
		tester.Fatalf("明文令牌不该出现在库里, found=%v err=%v", found, err)
	}

	expectStatus(tester, "登录后读条目", env.call(http.MethodGet, "/api/entries", "", cookie), http.StatusOK)
	recorder = env.call(http.MethodGet, "/api/session", "", cookie)
	if !strings.Contains(recorder.Body.String(), `"authenticated":true`) {
		tester.Fatalf("登录后应返回 authenticated:true, 实际 %s", recorder.Body.String())
	}

	recorder = env.call(http.MethodPost, "/api/logout", "", cookie)
	expectStatus(tester, "退出登录", recorder, http.StatusOK)
	if sessionCookieFrom(tester, recorder).MaxAge >= 0 {
		tester.Error("退出登录应让 Cookie 立即过期")
	}
	expectStatus(tester, "退出后再用旧 Cookie", env.call(http.MethodGet, "/api/entries", "", cookie), http.StatusUnauthorized)
	expectStatus(tester, "没有 Cookie 也能退出", env.call(http.MethodPost, "/api/logout", "", nil), http.StatusOK)
}
