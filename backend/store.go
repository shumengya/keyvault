package main

import "database/sql"
import "encoding/json"
import "errors"
import "fmt"
import "os"
import "path/filepath"
import "strings"

import _ "modernc.org/sqlite"

// PasswordEntry 一条密码记录,JSON 字段名与前端、旧版 data.json 保持一致。
type PasswordEntry struct {
	ID           int    `json:"id"`
	Account      string `json:"account"`
	Password     string `json:"password"`
	Username     string `json:"username"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	Website      string `json:"website"`
	OfficialName string `json:"officialName"`
	Tags         string `json:"tags"`
	Logo         string `json:"logo"`
}

// ErrEntryNotFound 按 id 找不到条目。
var ErrEntryNotFound error = errors.New("条目不存在")

// 旧版 data.json 导入完成的标记,防止用户清空数据后重启又把旧数据导回来。
const settingLegacyImported string = "legacyImported"

const schemaSQL string = `
CREATE TABLE IF NOT EXISTS entries (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	account      TEXT NOT NULL DEFAULT '',
	password     TEXT NOT NULL DEFAULT '',
	username     TEXT NOT NULL DEFAULT '',
	phone        TEXT NOT NULL DEFAULT '',
	email        TEXT NOT NULL DEFAULT '',
	website      TEXT NOT NULL DEFAULT '',
	officialName TEXT NOT NULL DEFAULT '',
	tags         TEXT NOT NULL DEFAULT '',
	logo         TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS settings (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
	tokenHash TEXT PRIMARY KEY,
	expiresAt INTEGER NOT NULL
);
`

const upsertSettingSQL string = "INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value"

const selectEntries string = "SELECT id, account, password, username, phone, email, website, officialName, tags, logo FROM entries"

// 搜索范围与旧版一致(不含手机号和密码)。LIKE 对 ASCII 不区分大小写,中文无大小写问题。
const searchClause string = ` WHERE account LIKE ?1 ESCAPE '\'
	OR username LIKE ?1 ESCAPE '\'
	OR email LIKE ?1 ESCAPE '\'
	OR website LIKE ?1 ESCAPE '\'
	OR officialName LIKE ?1 ESCAPE '\'
	OR tags LIKE ?1 ESCAPE '\'`

// Store 基于 SQLite 的存储层。
type Store struct {
	database *sql.DB
}

// openStore 打开(必要时创建)数据库文件并建表。
func openStore(databasePath string) (*Store, error) {
	var err error = os.MkdirAll(filepath.Dir(databasePath), 0755)
	if err != nil {
		return nil, fmt.Errorf("创建数据目录: %w", err)
	}

	// busy_timeout 让并发写入排队等锁而不是立刻报 SQLITE_BUSY
	var dsn string = databasePath + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	var database *sql.DB
	database, err = sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库: %w", err)
	}
	// 单连接串行化读写,个人密码库的并发量远用不着连接池;
	// 副作用是持有结果集或事务期间不能再用 database 发起新查询,否则会死锁
	database.SetMaxOpenConns(1)

	_, err = database.Exec(schemaSQL)
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("初始化表结构: %w", err),
			closeDatabase(database),
		)
	}
	return &Store{database: database}, nil
}

func closeDatabase(database *sql.DB) error {
	var err error = database.Close()
	if err != nil {
		return fmt.Errorf("关闭数据库: %w", err)
	}
	return nil
}

// Close 关闭数据库。
func (store *Store) Close() error {
	return closeDatabase(store.database)
}

func (store *Store) getSetting(key string) (string, error) {
	var value string = ""
	var err error = store.database.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("读取设置 %s: %w", key, err)
	}
	return value, nil
}

func closeRows(rows *sql.Rows, previous error) error {
	var closeErr error = rows.Close()
	if closeErr == nil {
		return previous
	}
	return errors.Join(previous, fmt.Errorf("关闭结果集: %w", closeErr))
}

func rollbackWith(transaction *sql.Tx, cause error) error {
	var rollbackErr error = transaction.Rollback()
	if rollbackErr == nil {
		return cause
	}
	return errors.Join(cause, fmt.Errorf("回滚事务: %w", rollbackErr))
}

func scanEntries(rows *sql.Rows) ([]PasswordEntry, error) {
	var entries []PasswordEntry = []PasswordEntry{}
	for rows.Next() {
		var entry PasswordEntry = PasswordEntry{}
		var err error = rows.Scan(
			&entry.ID,
			&entry.Account,
			&entry.Password,
			&entry.Username,
			&entry.Phone,
			&entry.Email,
			&entry.Website,
			&entry.OfficialName,
			&entry.Tags,
			&entry.Logo,
		)
		if err != nil {
			return nil, closeRows(rows, fmt.Errorf("读取条目: %w", err))
		}
		entries = append(entries, entry)
	}
	var iterateErr error = rows.Err()
	if iterateErr != nil {
		return nil, closeRows(rows, fmt.Errorf("遍历条目: %w", iterateErr))
	}
	var closeErr error = closeRows(rows, nil)
	if closeErr != nil {
		return nil, closeErr
	}
	return entries, nil
}

// likePattern 把关键词转成 LIKE 模式,转义 % _ \ 让它们按字面匹配。
func likePattern(keyword string) string {
	var escaper *strings.Replacer = strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_")
	return "%" + escaper.Replace(keyword) + "%"
}

// ListEntries 按 id 升序返回条目;keyword 非空时在账号、用户名、邮箱、网站、官方名称、标签里搜索。
func (store *Store) ListEntries(keyword string) ([]PasswordEntry, error) {
	if keyword == "" {
		var rows, err = store.database.Query(selectEntries + " ORDER BY id")
		if err != nil {
			return nil, fmt.Errorf("查询条目: %w", err)
		}
		return scanEntries(rows)
	}

	var rows, err = store.database.Query(selectEntries+searchClause+" ORDER BY id", likePattern(keyword))
	if err != nil {
		return nil, fmt.Errorf("搜索条目: %w", err)
	}
	return scanEntries(rows)
}

// CreateEntry 新增条目并返回带有新 id 的记录,传入的 ID 会被忽略。
func (store *Store) CreateEntry(entry PasswordEntry) (PasswordEntry, error) {
	var result, err = store.database.Exec(
		`INSERT INTO entries (account, password, username, phone, email, website, officialName, tags, logo)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.Account,
		entry.Password,
		entry.Username,
		entry.Phone,
		entry.Email,
		entry.Website,
		entry.OfficialName,
		entry.Tags,
		entry.Logo,
	)
	if err != nil {
		return PasswordEntry{}, fmt.Errorf("插入条目: %w", err)
	}
	var newID int64 = 0
	newID, err = result.LastInsertId()
	if err != nil {
		return PasswordEntry{}, fmt.Errorf("获取新条目 id: %w", err)
	}
	entry.ID = int(newID)
	return entry, nil
}

// UpdateEntry 按 entry.ID 整条覆盖,id 不存在返回 ErrEntryNotFound。
func (store *Store) UpdateEntry(entry PasswordEntry) error {
	var result, err = store.database.Exec(
		`UPDATE entries SET account = ?, password = ?, username = ?, phone = ?, email = ?,
		website = ?, officialName = ?, tags = ?, logo = ? WHERE id = ?`,
		entry.Account,
		entry.Password,
		entry.Username,
		entry.Phone,
		entry.Email,
		entry.Website,
		entry.OfficialName,
		entry.Tags,
		entry.Logo,
		entry.ID,
	)
	if err != nil {
		return fmt.Errorf("更新条目: %w", err)
	}
	return requireAffected(result)
}

// DeleteEntry 按 id 删除,id 不存在返回 ErrEntryNotFound。
func (store *Store) DeleteEntry(id int) error {
	var result, err = store.database.Exec("DELETE FROM entries WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除条目: %w", err)
	}
	return requireAffected(result)
}

func requireAffected(result sql.Result) error {
	var affected, err = result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数: %w", err)
	}
	if affected == 0 {
		return ErrEntryNotFound
	}
	return nil
}

// ImportLegacyJSON 把旧版 data.json 一次性导入空表,保留原 id,返回导入条数。
// 旧文件不会被修改或删除。文件格式损坏时返回错误而不是当成空库,避免悄悄丢数据。
func (store *Store) ImportLegacyJSON(jsonPath string) (int, error) {
	var imported string
	var err error
	imported, err = store.getSetting(settingLegacyImported)
	if err != nil {
		return 0, err
	}
	if imported == "1" {
		return 0, nil
	}

	var existing int = 0
	err = store.database.QueryRow("SELECT COUNT(*) FROM entries").Scan(&existing)
	if err != nil {
		return 0, fmt.Errorf("统计条目: %w", err)
	}
	// 表里已有数据说明不是首次启动,不再导入以免 id 冲突
	if existing > 0 {
		return 0, store.markLegacyImported()
	}

	var content []byte
	content, err = os.ReadFile(jsonPath)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("读取旧数据文件: %w", err)
	}
	if len(strings.TrimSpace(string(content))) == 0 {
		return 0, store.markLegacyImported()
	}

	var legacy struct {
		Entries []PasswordEntry `json:"entries"`
	}
	err = json.Unmarshal(content, &legacy)
	if err != nil {
		return 0, fmt.Errorf("解析旧数据文件 %s: %w", jsonPath, err)
	}

	var transaction *sql.Tx
	transaction, err = store.database.Begin()
	if err != nil {
		return 0, fmt.Errorf("开启事务: %w", err)
	}
	var index int = 0
	for index = 0; index < len(legacy.Entries); index++ {
		var entry PasswordEntry = legacy.Entries[index]
		_, err = transaction.Exec(
			`INSERT INTO entries (id, account, password, username, phone, email, website, officialName, tags, logo)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			entry.ID,
			entry.Account,
			entry.Password,
			entry.Username,
			entry.Phone,
			entry.Email,
			entry.Website,
			entry.OfficialName,
			entry.Tags,
			entry.Logo,
		)
		if err != nil {
			return 0, rollbackWith(transaction, fmt.Errorf("导入条目 id=%d: %w", entry.ID, err))
		}
	}
	_, err = transaction.Exec(upsertSettingSQL, settingLegacyImported, "1")
	if err != nil {
		return 0, rollbackWith(transaction, fmt.Errorf("写入导入标记: %w", err))
	}
	err = transaction.Commit()
	if err != nil {
		return 0, fmt.Errorf("提交事务: %w", err)
	}
	return len(legacy.Entries), nil
}

func (store *Store) markLegacyImported() error {
	return store.setSetting(settingLegacyImported, "1")
}
