package main

import "net/http"
import "net/http/httptest"
import "strings"
import "testing"
import "time"

func TestSecureCookieBehindHTTPS(tester *testing.T) {
	var env *testEnv = newTestEnv(tester, nil)
	var recorder *httptest.ResponseRecorder = env.send(testRequest{
		method:         http.MethodPost,
		path:           loginPath,
		body:           loginBody(testFirst, testSecond),
		forwardedProto: "https",
	})
	expectStatus(tester, "经 HTTPS 反代登录", recorder, http.StatusOK)
	if !sessionCookieFrom(tester, recorder).Secure {
		tester.Error("HTTPS 下 Cookie 应带 Secure")
	}
}

func TestSessionExpiry(tester *testing.T) {
	var env *testEnv = newTestEnv(tester, nil)
	var cookie *http.Cookie = env.login(tester)

	env.clock.advance(24*time.Hour - time.Second)
	expectStatus(tester, "到期前一秒", env.call(http.MethodGet, "/api/entries", "", cookie), http.StatusOK)

	env.clock.advance(time.Second)
	expectStatus(tester, "刚好到期", env.call(http.MethodGet, "/api/entries", "", cookie), http.StatusUnauthorized)
	var recorder *httptest.ResponseRecorder = env.call(http.MethodGet, "/api/session", "", cookie)
	if !strings.Contains(recorder.Body.String(), `"authenticated":false`) {
		tester.Fatalf("过期后应返回 authenticated:false, 实际 %s", recorder.Body.String())
	}

	// 过期会话被访问时顺手清掉
	var _, found, err = env.server.store.SessionExpiry(hashToken(cookie.Value))
	if err != nil || found {
		tester.Fatalf("过期会话应已删除, found=%v err=%v", found, err)
	}
}

func TestProtectedRoutesRequireLogin(tester *testing.T) {
	var env *testEnv = newTestEnv(tester, nil)
	type routeCase struct {
		method string
		path   string
		body   string
	}
	var cases []routeCase = []routeCase{
		{method: http.MethodGet, path: "/api/entries"},
		{method: http.MethodPost, path: "/api/entries", body: `{"officialName":"x"}`},
		{method: http.MethodPut, path: "/api/entries", body: `{"id":1,"officialName":"x"}`},
		{method: http.MethodDelete, path: "/api/entries/1"},
		{method: http.MethodGet, path: "/api/settings"},
		{method: http.MethodPut, path: "/api/settings/session", body: `{"hours":2}`},
		{method: http.MethodPut, path: changePath, body: changeBody("a", "b", "c", "d")},
	}
	var forged *http.Cookie = &http.Cookie{Name: sessionCookieName, Value: "not-a-real-token"}
	var index int = 0
	for index = 0; index < len(cases); index++ {
		var current routeCase = cases[index]
		var label string = current.method + " " + current.path
		expectStatus(tester, label+" 无 Cookie", env.call(current.method, current.path, current.body, nil), http.StatusUnauthorized)
		expectStatus(tester, label+" 伪造 Cookie", env.call(current.method, current.path, current.body, forged), http.StatusUnauthorized)
	}

	// 旧的单密码接口已删除,不能再有绕过两个密码的入口
	expectStatus(tester, "旧 /api/verify", env.call(http.MethodPost, "/api/verify", `{"password":"x"}`, nil), http.StatusNotFound)
}
