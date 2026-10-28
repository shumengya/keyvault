package main

import "crypto/rand"
import "errors"
import "fmt"
import "io"
import "os"
import "path/filepath"

// resetPasswords 生成两个新的随机访问密码写进库里,并打印出来。
// 库里只存 bcrypt 哈希,旧密码无从显示,忘记了就只能重置。
// 只动密码和会话:所有已登录的设备会被踢下线,密码记录一条不动。
// 先写库再打印:打印失败也不会丢,重新运行一次即可。
func resetPasswords(store *Store, output io.Writer, dataDirDisplay string) error {
	var first string = rand.Text()
	var second string = rand.Text()
	var hashFirst, err = hashPassword(first)
	if err != nil {
		return err
	}
	var hashSecond string
	hashSecond, err = hashPassword(second)
	if err != nil {
		return err
	}
	err = store.SetPasswordHashes(hashFirst, hashSecond, "")
	if err != nil {
		return err
	}

	// 把条目数一起打出来,用来核对没有连错库(连到空库时这里会是 0)
	var count int
	count, err = store.CountEntries()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(
		output,
		"数据目录: %s\n现有记录: %d 条(不会被改动)\n"+
			"已重置两个访问密码，所有已登录的设备需要重新登录。登录时两个都要输入，顺序不限。\n"+
			"  密码 1: %s\n  密码 2: %s\n"+
			"请立即记下，登录后可以在设置里改成自己好记的。\n",
		dataDirDisplay,
		count,
		first,
		second,
	)
	if err != nil {
		return fmt.Errorf("输出新密码: %w", err)
	}
	return nil
}

// runReset 是 -reset-passwords 的入口。
// 数据库不存在时拒绝执行:否则连错目录会悄悄新建一个空库再"重置"一遍,什么都没救回来。
func runReset(output io.Writer) error {
	var dataDir, dataDirDisplay = dataDirectory()
	var databasePath string = filepath.Join(dataDir, databaseFileName)
	var _, statErr = os.Stat(databasePath)
	if errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf(
			"在 %s 下找不到数据库 %s，没有可重置的密码。请确认数据目录是否正确（环境变量 MENGYA_DATA_DIR）",
			dataDirDisplay,
			databaseFileName,
		)
	}
	if statErr != nil {
		return fmt.Errorf("检查数据库文件: %w", statErr)
	}

	var store, err = openStore(databasePath)
	if err != nil {
		return fmt.Errorf("打开存储: %w", err)
	}
	return errors.Join(resetPasswords(store, output, dataDirDisplay), store.Close())
}
