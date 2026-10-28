package main

import "errors"
import "os"
import "path/filepath"
import "testing"

func openTestStore(tester *testing.T) *Store {
	tester.Helper()
	var store, err = openStore(filepath.Join(tester.TempDir(), "test.db"))
	if err != nil {
		tester.Fatalf("打开测试库: %v", err)
	}
	// Cleanup 后进先出,所以这里会先于 TempDir 的清理执行,Windows 下才删得掉文件
	tester.Cleanup(func() {
		var closeErr error = store.Close()
		if closeErr != nil {
			tester.Errorf("关闭测试库: %v", closeErr)
		}
	})
	return store
}

func writeLegacyFile(tester *testing.T, content string) string {
	tester.Helper()
	var path string = filepath.Join(tester.TempDir(), "data.json")
	var err error = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		tester.Fatalf("写旧数据文件: %v", err)
	}
	return path
}

func entryIDs(entries []PasswordEntry) []int {
	var ids []int = []int{}
	var index int = 0
	for index = 0; index < len(entries); index++ {
		ids = append(ids, entries[index].ID)
	}
	return ids
}

const legacySample string = `{"entries":[
	{"id":3,"account":"alice","password":"p1","officialName":"站点甲","tags":"社交","website":"https://a.example"},
	{"id":7,"account":"bob","password":"p2","officialName":"站点乙","phone":"123"}
]}`

func TestImportLegacyJSONKeepsIDs(tester *testing.T) {
	var store *Store = openTestStore(tester)
	var path string = writeLegacyFile(tester, legacySample)

	var count, err = store.ImportLegacyJSON(path)
	if err != nil {
		tester.Fatalf("导入: %v", err)
	}
	if count != 2 {
		tester.Fatalf("导入条数 = %d, 期望 2", count)
	}

	var entries []PasswordEntry
	entries, err = store.ListEntries("")
	if err != nil {
		tester.Fatalf("列表: %v", err)
	}
	var ids []int = entryIDs(entries)
	if len(ids) != 2 || ids[0] != 3 || ids[1] != 7 {
		tester.Fatalf("id = %v, 期望 [3 7]", ids)
	}
	if entries[0].Account != "alice" || entries[0].Website != "https://a.example" || entries[1].Phone != "123" {
		tester.Fatalf("字段没有完整导入: %+v", entries)
	}

	// 自增 id 必须接在导入的最大 id 之后,不能撞上旧 id
	var created PasswordEntry
	created, err = store.CreateEntry(PasswordEntry{OfficialName: "新条目"})
	if err != nil {
		tester.Fatalf("新增: %v", err)
	}
	if created.ID != 8 {
		tester.Fatalf("新增条目 id = %d, 期望 8", created.ID)
	}
}

func TestImportLegacyJSONOnlyOnce(tester *testing.T) {
	var store *Store = openTestStore(tester)
	var path string = writeLegacyFile(tester, legacySample)

	var _, err = store.ImportLegacyJSON(path)
	if err != nil {
		tester.Fatalf("首次导入: %v", err)
	}
	// 用户把数据删光后重启,不能把旧数据又导回来
	err = store.DeleteEntry(3)
	if err != nil {
		tester.Fatalf("删除 3: %v", err)
	}
	err = store.DeleteEntry(7)
	if err != nil {
		tester.Fatalf("删除 7: %v", err)
	}

	var count int
	count, err = store.ImportLegacyJSON(path)
	if err != nil {
		tester.Fatalf("再次导入: %v", err)
	}
	if count != 0 {
		tester.Fatalf("再次导入条数 = %d, 期望 0", count)
	}
	var entries []PasswordEntry
	entries, err = store.ListEntries("")
	if err != nil {
		tester.Fatalf("列表: %v", err)
	}
	if len(entries) != 0 {
		tester.Fatalf("删光后又出现了 %d 条记录", len(entries))
	}
}

func TestImportLegacyJSONSkipsNonEmptyTable(tester *testing.T) {
	var store *Store = openTestStore(tester)
	var _, err = store.CreateEntry(PasswordEntry{OfficialName: "已有数据"})
	if err != nil {
		tester.Fatalf("新增: %v", err)
	}

	var count int
	count, err = store.ImportLegacyJSON(writeLegacyFile(tester, legacySample))
	if err != nil {
		tester.Fatalf("导入: %v", err)
	}
	if count != 0 {
		tester.Fatalf("非空表导入条数 = %d, 期望 0", count)
	}
}

func TestImportLegacyJSONMissingAndEmptyFile(tester *testing.T) {
	var store *Store = openTestStore(tester)

	var count, err = store.ImportLegacyJSON(filepath.Join(tester.TempDir(), "不存在.json"))
	if err != nil || count != 0 {
		tester.Fatalf("文件不存在应当静默跳过, count=%d err=%v", count, err)
	}

	count, err = store.ImportLegacyJSON(writeLegacyFile(tester, "  \n"))
	if err != nil || count != 0 {
		tester.Fatalf("空文件应当静默跳过, count=%d err=%v", count, err)
	}
}

func TestImportLegacyJSONRejectsMalformed(tester *testing.T) {
	var store *Store = openTestStore(tester)
	var badPath string = writeLegacyFile(tester, `{"entries":[{"id":1,`)

	var _, err = store.ImportLegacyJSON(badPath)
	if err == nil {
		tester.Fatal("损坏的 JSON 应当返回错误")
	}

	// 重复 id 会在事务中途失败,必须整体回滚,不能留下半截数据
	var duplicatePath string = writeLegacyFile(tester, `{"entries":[{"id":1,"officialName":"a"},{"id":1,"officialName":"b"}]}`)
	_, err = store.ImportLegacyJSON(duplicatePath)
	if err == nil {
		tester.Fatal("重复 id 应当返回错误")
	}
	var entries []PasswordEntry
	entries, err = store.ListEntries("")
	if err != nil {
		tester.Fatalf("列表: %v", err)
	}
	if len(entries) != 0 {
		tester.Fatalf("导入失败后残留了 %d 条记录", len(entries))
	}

	// 失败时不能打上"已导入"标记,修好文件后下次启动还能导入
	var count int
	count, err = store.ImportLegacyJSON(writeLegacyFile(tester, legacySample))
	if err != nil || count != 2 {
		tester.Fatalf("修复后应能导入, count=%d err=%v", count, err)
	}
}

func TestCrudAndSearch(tester *testing.T) {
	var store *Store = openTestStore(tester)

	var first, err = store.CreateEntry(PasswordEntry{OfficialName: "GitHub", Account: "octocat", Tags: "开发 代码"})
	if err != nil {
		tester.Fatalf("新增 first: %v", err)
	}
	var second PasswordEntry
	second, err = store.CreateEntry(PasswordEntry{OfficialName: "折扣100%站", Email: "me@example.com", Password: "secret_pw"})
	if err != nil {
		tester.Fatalf("新增 second: %v", err)
	}
	if first.ID != 1 || second.ID != 2 {
		tester.Fatalf("id = %d,%d, 期望 1,2", first.ID, second.ID)
	}

	type searchCase struct {
		keyword string
		wantIDs []int
	}
	var cases []searchCase = []searchCase{
		{keyword: "", wantIDs: []int{1, 2}},
		{keyword: "github", wantIDs: []int{1}},
		{keyword: "代码", wantIDs: []int{1}},
		{keyword: "EXAMPLE.com", wantIDs: []int{2}},
		{keyword: "100%", wantIDs: []int{2}},
		{keyword: "%", wantIDs: []int{2}},
		{keyword: "_", wantIDs: []int{}},
		{keyword: "secret", wantIDs: []int{}},
		{keyword: "不存在", wantIDs: []int{}},
	}
	var caseIndex int = 0
	for caseIndex = 0; caseIndex < len(cases); caseIndex++ {
		var current searchCase = cases[caseIndex]
		var found []PasswordEntry
		found, err = store.ListEntries(current.keyword)
		if err != nil {
			tester.Fatalf("搜索 %q: %v", current.keyword, err)
		}
		var gotIDs []int = entryIDs(found)
		if len(gotIDs) != len(current.wantIDs) {
			tester.Fatalf("搜索 %q 得到 %v, 期望 %v", current.keyword, gotIDs, current.wantIDs)
		}
		var idIndex int = 0
		for idIndex = 0; idIndex < len(gotIDs); idIndex++ {
			if gotIDs[idIndex] != current.wantIDs[idIndex] {
				tester.Fatalf("搜索 %q 得到 %v, 期望 %v", current.keyword, gotIDs, current.wantIDs)
			}
		}
	}

	first.Account = "changed"
	err = store.UpdateEntry(first)
	if err != nil {
		tester.Fatalf("更新: %v", err)
	}
	// 更新为相同内容,SQLite 也应算作命中,不能误报不存在
	err = store.UpdateEntry(first)
	if err != nil {
		tester.Fatalf("重复更新: %v", err)
	}

	err = store.DeleteEntry(second.ID)
	if err != nil {
		tester.Fatalf("删除: %v", err)
	}
	var left []PasswordEntry
	left, err = store.ListEntries("")
	if err != nil {
		tester.Fatalf("列表: %v", err)
	}
	if len(left) != 1 || left[0].Account != "changed" {
		tester.Fatalf("剩余条目不对: %+v", left)
	}

	// 删除后的 id 不应被复用
	var third PasswordEntry
	third, err = store.CreateEntry(PasswordEntry{OfficialName: "第三个"})
	if err != nil {
		tester.Fatalf("新增 third: %v", err)
	}
	if third.ID != 3 {
		tester.Fatalf("id = %d, 期望 3(不复用已删除的 id)", third.ID)
	}
}

func TestUpdateDeleteMissing(tester *testing.T) {
	var store *Store = openTestStore(tester)

	var err error = store.UpdateEntry(PasswordEntry{ID: 99, OfficialName: "x"})
	if !errors.Is(err, ErrEntryNotFound) {
		tester.Fatalf("更新不存在的条目 err = %v, 期望 ErrEntryNotFound", err)
	}
	err = store.DeleteEntry(99)
	if !errors.Is(err, ErrEntryNotFound) {
		tester.Fatalf("删除不存在的条目 err = %v, 期望 ErrEntryNotFound", err)
	}
}
