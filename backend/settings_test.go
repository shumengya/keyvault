package main

import "fmt"
import "net/http"
import "strings"
import "testing"
import "time"

const changePath string = "/api/settings/passwords"

func changeBody(currentFirst string, currentSecond string, newFirst string, newSecond string) string {
	return fmt.Sprintf(
		`{"currentFirst":%q,"currentSecond":%q,"newFirst":%q,"newSecond":%q}`,
		currentFirst,
		currentSecond,
		newFirst,
		newSecond,
	)
}

func TestSessionTTLSetting(tester *testing.T) {
	var env *testEnv = newTestEnv(tester, nil)
	var longCookie *http.Cookie = env.login(tester)

	var recorder = env.call(http.MethodGet, "/api/settings", "", longCookie)
	expectStatus(tester, "读设置", recorder, http.StatusOK)
	if !strings.Contains(recorder.Body.String(), `"sessionTTLHours":24`) {
		tester.Fatalf("默认有效期应为 24 小时, 实际 %s", recorder.Body.String())
	}

	type ttlCase struct {
		body string
		want int
	}
	var cases []ttlCase = []ttlCase{
		{body: `{"hours":0}`, want: http.StatusBadRequest},
		{body: `{"hours":-5}`, want: http.StatusBadRequest},
		{body: `{"hours":8761}`, want: http.StatusBadRequest},
		{body: `{"hours":"abc"}`, want: http.StatusBadRequest},
		{body: `{}`, want: http.StatusBadRequest},
		{body: `{"hours":1}`, want: http.StatusOK},
		{body: `{"hours":8760}`, want: http.StatusOK},
	}
	var index int = 0
	for index = 0; index < len(cases); index++ {
		var current ttlCase = cases[index]
		expectStatus(tester, current.body, env.call(http.MethodPut, "/api/settings/session", current.body, longCookie), current.want)
	}

	expectStatus(tester, "设为 2 小时", env.call(http.MethodPut, "/api/settings/session", `{"hours":2}`, longCookie), http.StatusOK)
	var shortCookie *http.Cookie = env.login(tester)
	if shortCookie.MaxAge != 2*3600 {
		tester.Errorf("新登录的 MaxAge = %d, 期望 7200", shortCookie.MaxAge)
	}

	env.clock.advance(3 * time.Hour)
	expectStatus(tester, "2 小时的会话到期", env.call(http.MethodGet, "/api/entries", "", shortCookie), http.StatusUnauthorized)
	expectStatus(tester, "改有效期不影响已签发的会话", env.call(http.MethodGet, "/api/entries", "", longCookie), http.StatusOK)
}

func TestChangePasswords(tester *testing.T) {
	var env *testEnv = newTestEnv(tester, nil)
	var cookie *http.Cookie = env.login(tester)
	var otherCookie *http.Cookie = env.login(tester)

	// 当前密码错返回 403 而不是 401,前端才不会把它当成登录过期踢回登录页
	expectStatus(tester, "当前密码错", env.call(http.MethodPut, changePath, changeBody(testFirst, "wrong", "new-one", "new-two"), cookie), http.StatusForbidden)
	expectStatus(tester, "新密码相同", env.call(http.MethodPut, changePath, changeBody(testFirst, testSecond, "same", "same"), cookie), http.StatusBadRequest)
	expectStatus(tester, "新密码为空", env.call(http.MethodPut, changePath, changeBody(testFirst, testSecond, "", "new-two"), cookie), http.StatusBadRequest)
	var tooLong string = strings.Repeat("a", maxPasswordBytes+1)
	expectStatus(tester, "新密码超长", env.call(http.MethodPut, changePath, changeBody(testFirst, testSecond, tooLong, "new-two"), cookie), http.StatusBadRequest)

	// 当前密码颠倒顺序给也行
	expectStatus(tester, "改密码", env.call(http.MethodPut, changePath, changeBody(testSecond, testFirst, "new-one", "new-two"), cookie), http.StatusOK)

	expectStatus(tester, "当前设备保持登录", env.call(http.MethodGet, "/api/entries", "", cookie), http.StatusOK)
	expectStatus(tester, "其他设备被踢下线", env.call(http.MethodGet, "/api/entries", "", otherCookie), http.StatusUnauthorized)
	expectStatus(tester, "旧密码失效", env.call(http.MethodPost, loginPath, loginBody(testFirst, testSecond), nil), http.StatusUnauthorized)
	expectStatus(tester, "新密码(颠倒顺序)可登录", env.call(http.MethodPost, loginPath, loginBody("new-two", "new-one"), nil), http.StatusOK)
}

func TestChangePasswordsRateLimitIsSeparate(tester *testing.T) {
	var env *testEnv = newTestEnv(tester, nil)
	var cookie *http.Cookie = env.login(tester)

	var attempt int = 0
	for attempt = 0; attempt < maxAttempts; attempt++ {
		expectStatus(tester, "输错当前密码", env.call(http.MethodPut, changePath, changeBody("x", "y", "new-one", "new-two"), cookie), http.StatusForbidden)
	}
	expectStatus(tester, "超过次数", env.call(http.MethodPut, changePath, changeBody("x", "y", "new-one", "new-two"), cookie), http.StatusTooManyRequests)
	expectStatus(tester, "改密码锁定不影响登录", env.call(http.MethodPost, loginPath, loginBody(testFirst, testSecond), nil), http.StatusOK)
}
