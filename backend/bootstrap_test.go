package main

import "bytes"
import "log"
import "os"
import "strings"
import "testing"

func captureLog(tester *testing.T) *bytes.Buffer {
	tester.Helper()
	var output bytes.Buffer
	log.SetOutput(&output)
	tester.Cleanup(func() {
		log.SetOutput(os.Stderr)
	})
	return &output
}

func TestEnsurePasswordsFromEnvironment(tester *testing.T) {
	var store *Store = openTestStore(tester)
	var output *bytes.Buffer = captureLog(tester)
	tester.Setenv("MENGYA_PASSWORD_1", "env-one")
	tester.Setenv("MENGYA_PASSWORD_2", "env-two")

	var err error = ensurePasswords(store)
	if err != nil {
		tester.Fatalf("初始化密码: %v", err)
	}
	var matched, matchErr = newServer(store, nil).passwordsMatch("env-two", "env-one")
	if matchErr != nil || !matched {
		tester.Fatalf("环境变量里的密码应当生效, matched=%v err=%v", matched, matchErr)
	}
	if strings.Contains(output.String(), "env-one") || strings.Contains(output.String(), "env-two") {
		tester.Fatal("用户自己设定的密码不该写进日志")
	}

	// 已有密码后环境变量不再覆盖,否则设置页改的密码一重启就被改回去了
	var hashBefore, _, beforeErr = store.PasswordHashes()
	if beforeErr != nil {
		tester.Fatalf("读哈希: %v", beforeErr)
	}
	tester.Setenv("MENGYA_PASSWORD_1", "changed-one")
	err = ensurePasswords(store)
	if err != nil {
		tester.Fatalf("再次初始化: %v", err)
	}
	var hashAfter, _, afterErr = store.PasswordHashes()
	if afterErr != nil || hashBefore != hashAfter {
		tester.Fatalf("已有密码不应被覆盖, err=%v", afterErr)
	}
}

func TestEnsurePasswordsRejectsBadEnvironment(tester *testing.T) {
	type badCase struct {
		name   string
		first  string
		second string
	}
	var cases []badCase = []badCase{
		{name: "只设了密码1", first: "only-one", second: ""},
		{name: "只设了密码2", first: "", second: "only-two"},
		{name: "两个相同", first: "same", second: "same"},
		{name: "超长", first: strings.Repeat("a", maxPasswordBytes+1), second: "ok"},
	}
	var index int = 0
	for index = 0; index < len(cases); index++ {
		var current badCase = cases[index]
		var store *Store = openTestStore(tester)
		tester.Setenv("MENGYA_PASSWORD_1", current.first)
		tester.Setenv("MENGYA_PASSWORD_2", current.second)
		var err error = ensurePasswords(store)
		if err == nil {
			tester.Errorf("%s: 应当报错", current.name)
		}
		var hashFirst, hashSecond, hashErr = store.PasswordHashes()
		if hashErr != nil || hashFirst != "" || hashSecond != "" {
			tester.Errorf("%s: 失败后不该写入任何密码, err=%v", current.name, hashErr)
		}
	}
}

func extractPassword(tester *testing.T, logText string, label string) string {
	tester.Helper()
	var start int = strings.Index(logText, label)
	if start < 0 {
		tester.Fatalf("日志里没有 %q: %s", label, logText)
	}
	var rest string = logText[start+len(label):]
	var end int = strings.IndexAny(rest, "\r\n")
	if end < 0 {
		end = len(rest)
	}
	return strings.TrimSpace(rest[:end])
}

func TestEnsurePasswordsGeneratesRandom(tester *testing.T) {
	var store *Store = openTestStore(tester)
	var output *bytes.Buffer = captureLog(tester)
	tester.Setenv("MENGYA_PASSWORD_1", "")
	tester.Setenv("MENGYA_PASSWORD_2", "")

	var err error = ensurePasswords(store)
	if err != nil {
		tester.Fatalf("初始化密码: %v", err)
	}
	var first string = extractPassword(tester, output.String(), "密码 1: ")
	var second string = extractPassword(tester, output.String(), "密码 2: ")
	if first == second || len(first) < 20 || len(second) < 20 {
		tester.Fatalf("随机密码应当不同且足够长: %q %q", first, second)
	}
	var matched, matchErr = newServer(store, nil).passwordsMatch(second, first)
	if matchErr != nil || !matched {
		tester.Fatalf("日志里打印的密码应当能登录, matched=%v err=%v", matched, matchErr)
	}
}
