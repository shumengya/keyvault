package main

import "bytes"
import "errors"
import "net/http"
import "os"
import "path/filepath"
import "strings"
import "testing"

func TestResetPasswords(tester *testing.T) {
	var env *testEnv = newTestEnv(tester, nil)
	var oldCookie *http.Cookie = env.login(tester)
	var _, err = env.server.store.CreateEntry(PasswordEntry{OfficialName: "保留我", Password: "entry-secret"})
	if err != nil {
		tester.Fatalf("新增条目: %v", err)
	}

	var output bytes.Buffer
	err = resetPasswords(env.server.store, &output, "测试目录")
	if err != nil {
		tester.Fatalf("重置: %v", err)
	}
	var first string = extractPassword(tester, output.String(), "密码 1: ")
	var second string = extractPassword(tester, output.String(), "密码 2: ")
	if first == second || len(first) < 20 || len(second) < 20 {
		tester.Fatalf("新密码应当不同且足够长: %q %q", first, second)
	}
	if !strings.Contains(output.String(), "现有记录: 1 条") || !strings.Contains(output.String(), "数据目录: 测试目录") {
		tester.Errorf("输出里应有数据目录和记录数, 实际 %q", output.String())
	}

	var matched bool
	matched, err = env.server.passwordsMatch(second, first)
	if err != nil || !matched {
		tester.Fatalf("新密码(颠倒顺序)应当可用, matched=%v err=%v", matched, err)
	}
	matched, err = env.server.passwordsMatch(testFirst, testSecond)
	if err != nil || matched {
		tester.Fatalf("旧密码应当失效, matched=%v err=%v", matched, err)
	}
	expectStatus(tester, "旧会话被踢下线", env.call(http.MethodGet, "/api/entries", "", oldCookie), http.StatusUnauthorized)
	expectStatus(tester, "新密码登录", env.call(http.MethodPost, loginPath, loginBody(first, second), nil), http.StatusOK)

	var count int
	count, err = env.server.store.CountEntries()
	if err != nil || count != 1 {
		tester.Fatalf("重置不应改动条目, count=%d err=%v", count, err)
	}
}

func TestResetPasswordsOnStoreWithoutPasswords(tester *testing.T) {
	var store *Store = openTestStore(tester)
	var output bytes.Buffer
	var err error = resetPasswords(store, &output, "空库")
	if err != nil {
		tester.Fatalf("重置: %v", err)
	}
	var first string = extractPassword(tester, output.String(), "密码 1: ")
	var second string = extractPassword(tester, output.String(), "密码 2: ")
	var matched, matchErr = newServer(store, nil).passwordsMatch(first, second)
	if matchErr != nil || !matched {
		tester.Fatalf("没有密码的库也应能重置, matched=%v err=%v", matched, matchErr)
	}
}

func TestRunResetRefusesMissingDatabase(tester *testing.T) {
	var dataDir string = filepath.Join(tester.TempDir(), "不存在的目录")
	tester.Setenv("MENGYA_DATA_DIR", dataDir)

	var output bytes.Buffer
	var err error = runReset(&output)
	if err == nil {
		tester.Fatal("找不到数据库时应当报错")
	}
	if !strings.Contains(err.Error(), "MENGYA_DATA_DIR") {
		tester.Errorf("报错里应提示检查数据目录, 实际 %v", err)
	}
	if output.Len() != 0 {
		tester.Errorf("失败时不该输出任何密码, 实际 %q", output.String())
	}
	var _, statErr = os.Stat(dataDir)
	if !errors.Is(statErr, os.ErrNotExist) {
		tester.Errorf("连错目录时不该顺手新建数据目录, statErr=%v", statErr)
	}
}

func TestRunResetUsesConfiguredDataDir(tester *testing.T) {
	var dataDir string = tester.TempDir()
	tester.Setenv("MENGYA_DATA_DIR", dataDir)

	// 先造一个有旧密码和一条记录的库,再关掉,模拟"服务停着时在命令行重置"
	var store, err = openStore(filepath.Join(dataDir, databaseFileName))
	if err != nil {
		tester.Fatalf("打开库: %v", err)
	}
	var hashFirst string
	hashFirst, err = hashPassword(testFirst)
	if err != nil {
		tester.Fatalf("哈希: %v", err)
	}
	var hashSecond string
	hashSecond, err = hashPassword(testSecond)
	if err != nil {
		tester.Fatalf("哈希: %v", err)
	}
	err = store.SetPasswordHashes(hashFirst, hashSecond, "")
	if err != nil {
		tester.Fatalf("写入旧密码: %v", err)
	}
	_, err = store.CreateEntry(PasswordEntry{OfficialName: "一条记录"})
	if err != nil {
		tester.Fatalf("新增条目: %v", err)
	}
	err = store.Close()
	if err != nil {
		tester.Fatalf("关库: %v", err)
	}

	var output bytes.Buffer
	err = runReset(&output)
	if err != nil {
		tester.Fatalf("重置: %v", err)
	}
	if !strings.Contains(output.String(), "现有记录: 1 条") {
		tester.Errorf("应显示原有 1 条记录, 实际 %q", output.String())
	}
	var first string = extractPassword(tester, output.String(), "密码 1: ")
	var second string = extractPassword(tester, output.String(), "密码 2: ")

	store, err = openStore(filepath.Join(dataDir, databaseFileName))
	if err != nil {
		tester.Fatalf("重新打开库: %v", err)
	}
	tester.Cleanup(func() {
		var closeErr error = store.Close()
		if closeErr != nil {
			tester.Errorf("关库: %v", closeErr)
		}
	})
	var matched bool
	matched, err = newServer(store, nil).passwordsMatch(second, first)
	if err != nil || !matched {
		tester.Fatalf("新密码应当写进了库里, matched=%v err=%v", matched, err)
	}
	matched, err = newServer(store, nil).passwordsMatch(testFirst, testSecond)
	if err != nil || matched {
		tester.Fatalf("旧密码应当失效, matched=%v err=%v", matched, err)
	}
}
