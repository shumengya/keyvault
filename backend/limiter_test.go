package main

import "fmt"
import "net/http"
import "net/http/httptest"
import "strings"
import "testing"
import "time"

func TestLoginRateLimit(tester *testing.T) {
	var env *testEnv = newTestEnv(tester, nil)
	var wrong string = loginBody(testFirst, "wrong")
	var attempt int = 0
	for attempt = 0; attempt < maxAttempts; attempt++ {
		expectStatus(tester, "错误尝试", env.call(http.MethodPost, loginPath, wrong, nil), http.StatusUnauthorized)
	}

	var recorder *httptest.ResponseRecorder = env.call(http.MethodPost, loginPath, wrong, nil)
	expectStatus(tester, "超过次数", recorder, http.StatusTooManyRequests)
	if recorder.Header().Get("Retry-After") != "900" {
		tester.Errorf("Retry-After = %q, 期望 900", recorder.Header().Get("Retry-After"))
	}
	if !strings.Contains(recorder.Body.String(), "15 分钟") {
		tester.Errorf("提示里应写明 15 分钟, 实际 %s", recorder.Body.String())
	}
	// 锁定期间密码对了也不放行
	expectStatus(tester, "锁定期间正确密码", env.call(http.MethodPost, loginPath, loginBody(testFirst, testSecond), nil), http.StatusTooManyRequests)

	env.clock.advance(5 * time.Minute)
	recorder = env.call(http.MethodPost, loginPath, wrong, nil)
	expectStatus(tester, "5 分钟后仍锁定", recorder, http.StatusTooManyRequests)
	if recorder.Header().Get("Retry-After") != "600" {
		tester.Errorf("Retry-After = %q, 期望 600", recorder.Header().Get("Retry-After"))
	}

	env.clock.advance(10 * time.Minute)
	expectStatus(tester, "窗口过后恢复", env.call(http.MethodPost, loginPath, loginBody(testFirst, testSecond), nil), http.StatusOK)

	// 登录成功要清零计数:每轮 4 次失败 + 1 次成功,如果没清零第二轮就会被锁
	var round int = 0
	for round = 0; round < 3; round++ {
		for attempt = 0; attempt < maxAttempts-1; attempt++ {
			expectStatus(tester, "清零后的失败尝试", env.call(http.MethodPost, loginPath, wrong, nil), http.StatusUnauthorized)
		}
		expectStatus(tester, "清零后登录", env.call(http.MethodPost, loginPath, loginBody(testFirst, testSecond), nil), http.StatusOK)
	}
}

func TestRateLimitIgnoresSpoofedForwardedFor(tester *testing.T) {
	var env *testEnv = newTestEnv(tester, nil)
	var wrong string = loginBody(testFirst, "wrong")
	var attempt int = 0
	for attempt = 0; attempt < maxAttempts; attempt++ {
		var recorder *httptest.ResponseRecorder = env.send(testRequest{
			method:       http.MethodPost,
			path:         loginPath,
			body:         wrong,
			forwardedFor: fmt.Sprintf("10.0.0.%d", attempt+1),
		})
		expectStatus(tester, "换着 X-Forwarded-For 试", recorder, http.StatusUnauthorized)
	}
	var last *httptest.ResponseRecorder = env.send(testRequest{method: http.MethodPost, path: loginPath, body: wrong, forwardedFor: "10.0.0.99"})
	expectStatus(tester, "伪造 X-Forwarded-For 不能绕过限流", last, http.StatusTooManyRequests)
}

func TestTrustedProxyUsesForwardedFor(tester *testing.T) {
	// httptest 的默认来源地址是 192.0.2.1,把它配成可信代理
	var env *testEnv = newTestEnv(tester, []string{"192.0.2.1"})
	var wrong string = loginBody(testFirst, "wrong")
	var attempt int = 0
	for attempt = 0; attempt < maxAttempts*2; attempt++ {
		var recorder *httptest.ResponseRecorder = env.send(testRequest{
			method:       http.MethodPost,
			path:         loginPath,
			body:         wrong,
			forwardedFor: fmt.Sprintf("10.0.1.%d", attempt+1),
		})
		expectStatus(tester, "不同访客各算各的", recorder, http.StatusUnauthorized)
	}
	for attempt = 0; attempt < maxAttempts; attempt++ {
		expectStatus(tester, "同一访客", env.send(testRequest{method: http.MethodPost, path: loginPath, body: wrong, forwardedFor: "10.0.2.1"}), http.StatusUnauthorized)
	}
	expectStatus(tester, "同一访客超限", env.send(testRequest{method: http.MethodPost, path: loginPath, body: wrong, forwardedFor: "10.0.2.1"}), http.StatusTooManyRequests)
}

func TestInvalidTrustedProxy(tester *testing.T) {
	var server *Server = newServer(openTestStore(tester), []string{"not-an-ip"})
	var _, err = server.newRouter()
	if err == nil {
		tester.Fatal("非法的可信代理地址应当报错")
	}
}

func TestAttemptLimiter(tester *testing.T) {
	var limiter *attemptLimiter = newAttemptLimiter()
	var start time.Time = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var attempt int = 0
	for attempt = 0; attempt < maxAttempts; attempt++ {
		var allowed, _ = limiter.allow("src", start)
		if !allowed {
			tester.Fatalf("第 %d 次不该被拒绝", attempt+1)
		}
	}
	var allowed, retryAfter = limiter.allow("src", start)
	if allowed || retryAfter != attemptWindow {
		tester.Fatalf("第 %d 次应被拒绝并等满整个窗口, allowed=%v retryAfter=%v", maxAttempts+1, allowed, retryAfter)
	}
	var otherAllowed, _ = limiter.allow("other", start)
	if !otherAllowed {
		tester.Fatal("别的来源不该被连坐")
	}
	var recovered, _ = limiter.allow("src", start.Add(attemptWindow))
	if !recovered {
		tester.Fatal("窗口过后应当恢复")
	}
}

func TestAttemptLimiterSweepsStaleRecords(tester *testing.T) {
	var limiter *attemptLimiter = newAttemptLimiter()
	var start time.Time = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var index int = 0
	for index = 0; index < attemptSweepSize; index++ {
		limiter.allow(fmt.Sprintf("src-%d", index), start)
	}
	limiter.allow("fresh", start.Add(attemptWindow))
	if len(limiter.records) != 1 {
		tester.Fatalf("过期记录应被清掉, 剩余 %d 条", len(limiter.records))
	}
}

func TestLoginBadRequests(tester *testing.T) {
	var env *testEnv = newTestEnv(tester, nil)
	expectStatus(tester, "坏 JSON", env.call(http.MethodPost, loginPath, `{"first":`, nil), http.StatusBadRequest)
	expectStatus(tester, "缺字段", env.call(http.MethodPost, loginPath, `{}`, nil), http.StatusUnauthorized)
	var huge string = `{"first":"` + strings.Repeat("a", int(maxBodyBytes)+1) + `","second":"x"}`
	expectStatus(tester, "超大请求体", env.call(http.MethodPost, loginPath, huge, nil), http.StatusBadRequest)
}

func TestLoginFailsClosedWhenPasswordsMissing(tester *testing.T) {
	var env *testEnv = newEnvFromStore(tester, openTestStore(tester), nil)
	expectStatus(tester, "密码未初始化", env.call(http.MethodPost, loginPath, loginBody("a", "b"), nil), http.StatusInternalServerError)
}
