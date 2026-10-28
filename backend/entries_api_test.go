package main

import "net/http"
import "net/http/httptest"
import "strings"
import "testing"

func TestEntriesAPI(tester *testing.T) {
	var env *testEnv = newTestEnv(tester, nil)
	var cookie *http.Cookie = env.login(tester)

	var recorder *httptest.ResponseRecorder = env.call(http.MethodGet, "/api/entries", "", cookie)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"entries":[]`) {
		tester.Fatalf("空库列表应返回 entries:[], 实际 %d %s", recorder.Code, recorder.Body.String())
	}

	recorder = env.call(http.MethodPost, "/api/entries", `{"officialName":""}`, cookie)
	expectStatus(tester, "官方名称为空", recorder, http.StatusBadRequest)

	recorder = env.call(http.MethodPost, "/api/entries", `{"officialName":"站点","account":"u1"}`, cookie)
	expectStatus(tester, "新增", recorder, http.StatusOK)
	if !strings.Contains(recorder.Body.String(), `"id":1`) {
		tester.Fatalf("新增应返回 id:1, 实际 %s", recorder.Body.String())
	}

	recorder = env.call(http.MethodPut, "/api/entries", `{"id":1,"officialName":"站点","account":"u2"}`, cookie)
	expectStatus(tester, "更新", recorder, http.StatusOK)
	recorder = env.call(http.MethodPut, "/api/entries", `{"id":42,"officialName":"站点"}`, cookie)
	expectStatus(tester, "更新不存在的条目", recorder, http.StatusNotFound)

	recorder = env.call(http.MethodGet, "/api/entries?keyword=u2", "", cookie)
	expectStatus(tester, "搜索", recorder, http.StatusOK)
	if !strings.Contains(recorder.Body.String(), `"account":"u2"`) {
		tester.Fatalf("搜索结果不对 %s", recorder.Body.String())
	}

	expectStatus(tester, "非法 id", env.call(http.MethodDelete, "/api/entries/abc", "", cookie), http.StatusBadRequest)
	expectStatus(tester, "删除", env.call(http.MethodDelete, "/api/entries/1", "", cookie), http.StatusOK)
	expectStatus(tester, "重复删除", env.call(http.MethodDelete, "/api/entries/1", "", cookie), http.StatusNotFound)
}
