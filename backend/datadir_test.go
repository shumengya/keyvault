package main

import "path/filepath"
import "testing"

func TestDefaultDataDir(tester *testing.T) {
	var tempDir string = tester.TempDir()
	var outside string = filepath.Join(filepath.Dir(tempDir), "installed-app")
	type dirCase struct {
		name       string
		executable string
		want       string
	}
	var cases []dirCase = []dirCase{
		{name: "普通位置:用二进制旁边的 data", executable: filepath.Join(outside, "bin", "app.exe"), want: filepath.Join(outside, "bin", "data")},
		{name: "go run 的临时二进制:退回运行目录", executable: filepath.Join(tempDir, "go-build123", "b001", "exe", "app.exe"), want: "data"},
		{name: "二进制直接在临时目录里", executable: filepath.Join(tempDir, "app.exe"), want: "data"},
		{name: "目录名只是前缀相同,不算在临时目录里", executable: tempDir + "-other" + string(filepath.Separator) + "app.exe", want: filepath.Join(tempDir+"-other", "data")},
		{name: "拿不到二进制路径:退回运行目录", executable: "", want: "data"},
	}
	var index int = 0
	for index = 0; index < len(cases); index++ {
		var current dirCase = cases[index]
		var got string = defaultDataDir(current.executable, tempDir)
		if got != current.want {
			tester.Errorf("%s: 得到 %q, 期望 %q", current.name, got, current.want)
		}
	}
}

func TestDataDirectoryEnvironmentOverride(tester *testing.T) {
	var custom string = filepath.Join(tester.TempDir(), "自定义")
	tester.Setenv("MENGYA_DATA_DIR", custom)
	var dataDir, display = dataDirectory()
	if dataDir != custom || display != custom {
		tester.Errorf("环境变量应优先, dataDir=%q display=%q", dataDir, display)
	}
}

func TestDataDirectoryDefaultIsAbsoluteForDisplay(tester *testing.T) {
	tester.Setenv("MENGYA_DATA_DIR", "")
	var _, display = dataDirectory()
	if !filepath.IsAbs(display) {
		tester.Errorf("展示用的路径应是绝对路径, 实际 %q", display)
	}
}
