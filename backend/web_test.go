package main

import "net/http"
import "net/http/httptest"
import "strings"
import "testing"
import "testing/fstest"

func newWebTestEnv(tester *testing.T) *testEnv {
	tester.Helper()
	var err error = registerMimeTypes()
	if err != nil {
		tester.Fatalf("注册 MIME: %v", err)
	}
	var env *testEnv = newTestEnv(tester, nil)
	env.server.webFS = fstest.MapFS{
		"index.html":              {Data: []byte("<html>首页</html>")},
		"logo.png":                {Data: []byte("png-bytes")},
		"static/js/main.abc.js":   {Data: []byte("console.log(1)")},
		"static/css/main.abc.css": {Data: []byte("body{}")},
	}
	var router, routerErr = env.server.newRouter()
	if routerErr != nil {
		tester.Fatalf("创建路由: %v", routerErr)
	}
	env.router = router
	return env
}

func TestStaticFiles(tester *testing.T) {
	var env *testEnv = newWebTestEnv(tester)
	type staticCase struct {
		path         string
		wantStatus   int
		wantType     string
		wantCache    string
		wantContains string
	}
	var cases []staticCase = []staticCase{
		{path: "/", wantStatus: 200, wantType: "text/html", wantCache: "no-cache", wantContains: "首页"},
		{path: "/index.html", wantStatus: 301},
		{path: "/static/js/main.abc.js", wantStatus: 200, wantType: "text/javascript", wantCache: "public, max-age=31536000, immutable", wantContains: "console.log"},
		{path: "/static/css/main.abc.css", wantStatus: 200, wantType: "text/css", wantCache: "public, max-age=31536000, immutable"},
		{path: "/logo.png", wantStatus: 200, wantType: "image/png", wantCache: "public, max-age=86400"},
		{path: "/static/", wantStatus: 404},
		{path: "/static", wantStatus: 404},
		{path: "/不存在.js", wantStatus: 404},
		{path: "/../../etc/passwd", wantStatus: 404},
		// URL 路径里带 .. 的请求,标准库的 ServeFileFS 会直接拒绝(400),不会去读文件
		{path: "/static/../index.html", wantStatus: 400},
	}
	var index int = 0
	for index = 0; index < len(cases); index++ {
		var current staticCase = cases[index]
		var recorder *httptest.ResponseRecorder = env.call(http.MethodGet, current.path, "", nil)
		if recorder.Code != current.wantStatus {
			tester.Errorf("%s: 状态码 %d, 期望 %d", current.path, recorder.Code, current.wantStatus)
			continue
		}
		if current.wantType != "" && !strings.HasPrefix(recorder.Header().Get("Content-Type"), current.wantType) {
			tester.Errorf("%s: Content-Type %q, 期望以 %q 开头", current.path, recorder.Header().Get("Content-Type"), current.wantType)
		}
		if current.wantCache != "" && recorder.Header().Get("Cache-Control") != current.wantCache {
			tester.Errorf("%s: Cache-Control %q, 期望 %q", current.path, recorder.Header().Get("Cache-Control"), current.wantCache)
		}
		if current.wantContains != "" && !strings.Contains(recorder.Body.String(), current.wantContains) {
			tester.Errorf("%s: 响应内容不对 %q", current.path, recorder.Body.String())
		}
	}
}

func TestUnknownAPIPathReturnsJSON(tester *testing.T) {
	var env *testEnv = newWebTestEnv(tester)
	var recorder *httptest.ResponseRecorder = env.call(http.MethodGet, "/api/nope", "", nil)
	expectStatus(tester, "未知接口", recorder, http.StatusNotFound)
	if !strings.Contains(recorder.Header().Get("Content-Type"), "application/json") {
		tester.Errorf("未知接口应返回 JSON, 实际 %q", recorder.Header().Get("Content-Type"))
	}
	// POST 到未注册的接口也要返回 JSON,不能落到静态文件的纯文本 404
	recorder = env.call(http.MethodPost, "/api/nope", "{}", nil)
	if !strings.Contains(recorder.Header().Get("Content-Type"), "application/json") {
		tester.Errorf("POST 未知接口应返回 JSON, 实际 %q", recorder.Header().Get("Content-Type"))
	}
	expectStatus(tester, "静态文件不接受 POST", env.call(http.MethodPost, "/logo.png", "{}", nil), http.StatusNotFound)
}

func TestAPIResponsesAreNotCached(tester *testing.T) {
	var env *testEnv = newWebTestEnv(tester)
	var cookie *http.Cookie = env.login(tester)
	var recorder *httptest.ResponseRecorder = env.call(http.MethodGet, "/api/entries", "", cookie)
	if recorder.Header().Get("Cache-Control") != "no-store" {
		tester.Errorf("接口响应 Cache-Control = %q, 期望 no-store", recorder.Header().Get("Cache-Control"))
	}
	recorder = env.call(http.MethodGet, "/api/entries", "", nil)
	if recorder.Header().Get("Cache-Control") != "no-store" {
		tester.Errorf("401 响应 Cache-Control = %q, 期望 no-store", recorder.Header().Get("Cache-Control"))
	}
}

func TestMissingFrontendShowsHint(tester *testing.T) {
	var env *testEnv = newTestEnv(tester, nil)
	env.server.webFS = fstest.MapFS{}
	var router, err = env.server.newRouter()
	if err != nil {
		tester.Fatalf("创建路由: %v", err)
	}
	env.router = router
	var recorder *httptest.ResponseRecorder = env.call(http.MethodGet, "/", "", nil)
	expectStatus(tester, "没有前端", recorder, http.StatusServiceUnavailable)
	if !strings.Contains(recorder.Body.String(), "build.bat") {
		tester.Errorf("应提示怎么构建前端, 实际 %q", recorder.Body.String())
	}
}
