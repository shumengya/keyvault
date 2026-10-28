package main

import "testing"

// README 里写的"忘记密码"恢复办法:删掉库里两条密码哈希再重启。这里确认它真的管用。
func TestRecoverByDeletingPasswordHashes(tester *testing.T) {
	var env *testEnv = newTestEnv(tester, nil)
	var oldCookie = env.login(tester)

	var _, err = env.server.store.database.Exec(
		"DELETE FROM settings WHERE key IN (?, ?)",
		settingPasswordHashFirst,
		settingPasswordHashSecond,
	)
	if err != nil {
		tester.Fatalf("删除密码哈希: %v", err)
	}

	tester.Setenv("MENGYA_PASSWORD_1", "recovered-one")
	tester.Setenv("MENGYA_PASSWORD_2", "recovered-two")
	err = ensurePasswords(env.server.store)
	if err != nil {
		tester.Fatalf("重启时初始化密码: %v", err)
	}

	var matched bool
	matched, err = env.server.passwordsMatch("recovered-two", "recovered-one")
	if err != nil || !matched {
		tester.Fatalf("恢复后新密码应当可用, matched=%v err=%v", matched, err)
	}
	matched, err = env.server.passwordsMatch(testFirst, testSecond)
	if err != nil || matched {
		tester.Fatalf("旧密码应当失效, matched=%v err=%v", matched, err)
	}
	// 重设密码会作废所有旧会话,拿着旧 Cookie 的人也要重新登录
	var _, valid, sessionErr = env.server.store.SessionExpiry(hashToken(oldCookie.Value))
	if sessionErr != nil || valid {
		tester.Fatalf("旧会话应当被作废, found=%v err=%v", valid, sessionErr)
	}
}
