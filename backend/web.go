package main

import "embed"
import "fmt"
import "io/fs"
import "mime"
import "net/http"
import "path"
import "strings"

import "github.com/gin-gonic/gin"

// 前端构建产物由 build.bat / Dockerfile 输出到 web/dist,再随 go build 嵌进二进制。
// 这里嵌入整个 web 目录而不是 web/dist,并且用 all: 前缀(否则点开头的文件会被排除):
// web/.gitkeep 是入库的占位文件,保证没构建过前端时 go build / go run 也能通过,
// 否则 embed 匹配不到任何文件会直接编译失败。
// 带 go:embed 的变量不能写初始值,是 var 规范里唯一的例外。
//
//go:embed all:web
var webFiles embed.FS

const webDistDir string = "web/dist"

// embeddedDist 返回内嵌的前端目录,根就是 index.html 所在位置。
func embeddedDist() (fs.FS, error) {
	var distFS, err = fs.Sub(webFiles, webDistDir)
	if err != nil {
		return nil, fmt.Errorf("读取内嵌前端: %w", err)
	}
	return distFS, nil
}

// frontendBuilt 内嵌的前端是否真的构建过。
func frontendBuilt() bool {
	var distFS, err = embeddedDist()
	if err != nil {
		return false
	}
	var _, statErr = fs.Stat(distFS, "index.html")
	return statErr == nil
}

// registerMimeTypes 写死常用扩展名的类型。Go 在 Windows 上会读注册表,
// 注册表里 .js 被别的软件改成 text/plain 时,浏览器会拒绝执行脚本,页面直接白屏。
func registerMimeTypes() error {
	type mimeEntry struct {
		extension string
		mimeType  string
	}
	var entries []mimeEntry = []mimeEntry{
		{extension: ".js", mimeType: "text/javascript; charset=utf-8"},
		{extension: ".css", mimeType: "text/css; charset=utf-8"},
		{extension: ".html", mimeType: "text/html; charset=utf-8"},
		{extension: ".json", mimeType: "application/json; charset=utf-8"},
		{extension: ".svg", mimeType: "image/svg+xml"},
		{extension: ".ico", mimeType: "image/x-icon"},
		{extension: ".png", mimeType: "image/png"},
	}
	var index int = 0
	for index = 0; index < len(entries); index++ {
		var err error = mime.AddExtensionType(entries[index].extension, entries[index].mimeType)
		if err != nil {
			return fmt.Errorf("注册 MIME 类型 %s: %w", entries[index].extension, err)
		}
	}
	return nil
}

func cacheControlFor(name string) string {
	// CRA 构建出来的 static/ 下文件名带内容哈希,内容变了名字就变,可以永久缓存
	if strings.HasPrefix(name, "static/") {
		return "public, max-age=31536000, immutable"
	}
	// 入口页和自注销脚本必须每次都问服务器,否则发布新版本后用户一直看到旧页面
	if name == "index.html" || name == "service-worker.js" {
		return "no-cache"
	}
	return "public, max-age=86400"
}

// noStore 接口响应里有明文密码,不允许浏览器或中间代理缓存。
func noStore(ginContext *gin.Context) {
	ginContext.Header("Cache-Control", "no-store")
	ginContext.Next()
}

// staticHandler 提供前端静态文件。挂在 NoRoute 上,所以 /api 下已注册的接口优先。
// 只服务具体文件:目录不列出,没有的路径直接 404(前端没有客户端路由,不需要回落到 index.html)。
func staticHandler(distFS fs.FS) gin.HandlerFunc {
	return func(ginContext *gin.Context) {
		var request *http.Request = ginContext.Request
		var urlPath string = request.URL.Path
		if urlPath == "/api" || strings.HasPrefix(urlPath, "/api/") {
			ginContext.JSON(http.StatusNotFound, gin.H{"error": "接口不存在"})
			return
		}
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			ginContext.String(http.StatusNotFound, "404 page not found")
			return
		}

		var name string = strings.TrimPrefix(path.Clean("/"+urlPath), "/")
		if name == "" {
			name = "index.html"
		}
		var info, fileErr = fs.Stat(distFS, name)
		if fileErr != nil || info.IsDir() {
			// 根路径找不到 index.html,说明编译时没带上前端
			if name == "index.html" {
				ginContext.String(http.StatusServiceUnavailable, "前端尚未内嵌：请先构建前端再编译后端（运行 build.bat），详见 README。")
				return
			}
			ginContext.String(http.StatusNotFound, "404 page not found")
			return
		}
		ginContext.Header("Cache-Control", cacheControlFor(name))
		http.ServeFileFS(ginContext.Writer, request, distFS, name)
	}
}
