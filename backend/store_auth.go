package main

import "database/sql"
import "errors"
import "fmt"
import "strconv"
import "time"

// 两个访问密码只存 bcrypt 哈希,设置页面也读不回明文。
const settingPasswordHashFirst string = "passwordHashFirst"
const settingPasswordHashSecond string = "passwordHashSecond"
const settingSessionTTLHours string = "sessionTTLHours"

const defaultSessionTTLHours int = 24
const minSessionTTLHours int = 1
const maxSessionTTLHours int = 8760

func (store *Store) setSetting(key string, value string) error {
	var _, err = store.database.Exec(upsertSettingSQL, key, value)
	if err != nil {
		return fmt.Errorf("写入设置 %s: %w", key, err)
	}
	return nil
}

// PasswordHashes 返回两个访问密码的哈希,尚未设置时为空串。
func (store *Store) PasswordHashes() (string, string, error) {
	var first, err = store.getSetting(settingPasswordHashFirst)
	if err != nil {
		return "", "", err
	}
	var second string
	second, err = store.getSetting(settingPasswordHashSecond)
	if err != nil {
		return "", "", err
	}
	return first, second, nil
}

// SetPasswordHashes 同时更新两个密码哈希,并作废除 keepTokenHash 以外的所有会话,
// 改密码后其他设备必须重新登录。keepTokenHash 传空串则作废全部会话。
func (store *Store) SetPasswordHashes(hashFirst string, hashSecond string, keepTokenHash string) error {
	var transaction, err = store.database.Begin()
	if err != nil {
		return fmt.Errorf("开启事务: %w", err)
	}
	_, err = transaction.Exec(upsertSettingSQL, settingPasswordHashFirst, hashFirst)
	if err != nil {
		return rollbackWith(transaction, fmt.Errorf("写入密码 1: %w", err))
	}
	_, err = transaction.Exec(upsertSettingSQL, settingPasswordHashSecond, hashSecond)
	if err != nil {
		return rollbackWith(transaction, fmt.Errorf("写入密码 2: %w", err))
	}
	_, err = transaction.Exec("DELETE FROM sessions WHERE tokenHash <> ?", keepTokenHash)
	if err != nil {
		return rollbackWith(transaction, fmt.Errorf("作废其他会话: %w", err))
	}
	err = transaction.Commit()
	if err != nil {
		return fmt.Errorf("提交事务: %w", err)
	}
	return nil
}

// SessionTTLHours 登录有效期(小时),未设置或值损坏时回落到默认的 24 小时。
func (store *Store) SessionTTLHours() (int, error) {
	var value, err = store.getSetting(settingSessionTTLHours)
	if err != nil {
		return 0, err
	}
	var hours int
	hours, err = strconv.Atoi(value)
	if err != nil || hours < minSessionTTLHours || hours > maxSessionTTLHours {
		return defaultSessionTTLHours, nil
	}
	return hours, nil
}

// SetSessionTTLHours 只影响之后的新登录,已签发的会话保持原到期时间。
func (store *Store) SetSessionTTLHours(hours int) error {
	return store.setSetting(settingSessionTTLHours, strconv.Itoa(hours))
}

// CreateSession 保存会话(只存令牌的哈希),顺手清掉已过期的会话免得表无限增长。
func (store *Store) CreateSession(tokenHash string, now time.Time, expiresAt time.Time) error {
	var _, err = store.database.Exec("DELETE FROM sessions WHERE expiresAt <= ?", now.Unix())
	if err != nil {
		return fmt.Errorf("清理过期会话: %w", err)
	}
	_, err = store.database.Exec(
		"INSERT INTO sessions (tokenHash, expiresAt) VALUES (?, ?)",
		tokenHash,
		expiresAt.Unix(),
	)
	if err != nil {
		return fmt.Errorf("保存会话: %w", err)
	}
	return nil
}

// SessionExpiry 查会话到期时间,found 为 false 表示没有这个会话。
func (store *Store) SessionExpiry(tokenHash string) (time.Time, bool, error) {
	var expiresAt int64 = 0
	var err error = store.database.QueryRow("SELECT expiresAt FROM sessions WHERE tokenHash = ?", tokenHash).Scan(&expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("查询会话: %w", err)
	}
	return time.Unix(expiresAt, 0), true, nil
}

// DeleteSession 删除会话,不存在也不报错(退出登录要幂等)。
func (store *Store) DeleteSession(tokenHash string) error {
	var _, err = store.database.Exec("DELETE FROM sessions WHERE tokenHash = ?", tokenHash)
	if err != nil {
		return fmt.Errorf("删除会话: %w", err)
	}
	return nil
}
