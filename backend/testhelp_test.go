package main

import "fmt"
import "net/http"
import "net/http/httptest"
import "os"
import "strings"
import "testing"
import "time"

import "github.com/gin-gonic/gin"
import "golang.org/x/crypto/bcrypt"

const testFirst string = "alpha-secret"
const testSecond string = "bravo-secret"
const loginPath string = "/api/login"

func TestMain(runner *testing.M) {
	// 最低成本的 bcrypt,否则每次登录算 4 次哈希,用例会慢得受不了
	bcryptCost = bcrypt.MinCost
	gin.SetMode(gin.TestMode)
	os.Exit(runner.Run())
}

// fakeClock 让测试自己推进时间,不用真等 24 小时。
type fakeClock struct {
	current time.Time
}

func (clock *fakeClock) now() time.Time {
	return clock.current
}

func (clock *fakeClock) advance(duration time.Duration) {
	clock.current = clock.current.Add(duration)
}

type testEnv struct {
	server *Server
	router *gin.Engine
	clock  *fakeClock
}

// newEnvFromStore 不预置密码,用来测"尚未初始化"这类场景。
func newEnvFromStore(tester *testing.T, store *Store, trustedProxies []string) *testEnv {
	tester.Helper()
	var server *Server = newServer(store, trustedProxies)
	var clock *fakeClock = &fakeClock{current: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	server.clock = clock.now
	var router, err = server.newRouter()
	if err != nil {
		tester.Fatalf("创建路由: %v", err)
	}
	return &testEnv{server: server, router: router, clock: clock}
}

// newTestEnv 预置 testFirst / testSecond 两个访问密码。
func newTestEnv(tester *testing.T, trustedProxies []string) *testEnv {
	tester.Helper()
	var store *Store = openTestStore(tester)
	var hashFirst, err = hashPassword(testFirst)
	if err != nil {
		tester.Fatalf("哈希密码 1: %v", err)
	}
	var hashSecond string
	hashSecond, err = hashPassword(testSecond)
	if err != nil {
		tester.Fatalf("哈希密码 2: %v", err)
	}
	err = store.SetPasswordHashes(hashFirst, hashSecond, "")
	if err != nil {
		tester.Fatalf("写入密码: %v", err)
	}
	return newEnvFromStore(tester, store, trustedProxies)
}

type testRequest struct {
	method         string
	path           string
	body           string
	cookie         *http.Cookie
	forwardedFor   string
	forwardedProto string
}

func (env *testEnv) send(input testRequest) *httptest.ResponseRecorder {
	var request *http.Request = httptest.NewRequest(input.method, input.path, strings.NewReader(input.body))
	request.Header.Set("Content-Type", "application/json")
	if input.cookie != nil {
		request.AddCookie(input.cookie)
	}
	if input.forwardedFor != "" {
		request.Header.Set("X-Forwarded-For", input.forwardedFor)
	}
	if input.forwardedProto != "" {
		request.Header.Set("X-Forwarded-Proto", input.forwardedProto)
	}
	var recorder *httptest.ResponseRecorder = httptest.NewRecorder()
	env.router.ServeHTTP(recorder, request)
	return recorder
}

func (env *testEnv) call(method string, path string, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	return env.send(testRequest{method: method, path: path, body: body, cookie: cookie})
}

func loginBody(first string, second string) string {
	return fmt.Sprintf(`{"first":%q,"second":%q}`, first, second)
}

func expectStatus(tester *testing.T, label string, recorder *httptest.ResponseRecorder, want int) {
	tester.Helper()
	if recorder.Code != want {
		tester.Fatalf("%s: 状态码 %d, 期望 %d, 响应 %s", label, recorder.Code, want, recorder.Body.String())
	}
}

func sessionCookieFrom(tester *testing.T, recorder *httptest.ResponseRecorder) *http.Cookie {
	tester.Helper()
	var cookies []*http.Cookie = recorder.Result().Cookies()
	var index int = 0
	for index = 0; index < len(cookies); index++ {
		if cookies[index].Name == sessionCookieName {
			return cookies[index]
		}
	}
	tester.Fatalf("响应里没有会话 Cookie")
	return nil
}

func (env *testEnv) login(tester *testing.T) *http.Cookie {
	tester.Helper()
	var recorder *httptest.ResponseRecorder = env.call(http.MethodPost, loginPath, loginBody(testFirst, testSecond), nil)
	expectStatus(tester, "登录", recorder, http.StatusOK)
	return sessionCookieFrom(tester, recorder)
}
