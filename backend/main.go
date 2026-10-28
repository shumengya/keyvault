package main

import "cmp"
import "errors"
import "flag"
import "fmt"
import "io/fs"
import "log"
import "net/http"
import "os"
import "path/filepath"
import "strconv"
import "strings"
import "time"

import "github.com/gin-gonic/gin"

// 单个请求体上限。条目字段都是短文本,1MiB 绰绰有余,主要是防未登录接口被灌大请求。
const maxBodyBytes int64 = 1 << 20

// Server 持有各接口共用的存储层和登录限流器。
type Server struct {
	store          *Store
	limiter        *attemptLimiter
	clock          func() time.Time
	trustedProxies []string
	// webFS 前端静态文件;为 nil 时用编译进二进制的那份,测试里可以换成假的
	webFS fs.FS
}

func newServer(store *Store, trustedProxies []string) *Server {
	return &Server{
		store:          store,
		limiter:        newAttemptLimiter(),
		clock:          time.Now,
		trustedProxies: trustedProxies,
	}
}

func limitBody(ginContext *gin.Context) {
	ginContext.Request.Body = http.MaxBytesReader(ginContext.Writer, ginContext.Request.Body, maxBodyBytes)
	ginContext.Next()
}

func (server *Server) getEntries(ginContext *gin.Context) {
	var entries, err = server.store.ListEntries(ginContext.Query("keyword"))
	if err != nil {
		log.Printf("查询条目失败: %v", err)
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	ginContext.JSON(http.StatusOK, gin.H{"entries": entries})
}

func (server *Server) addEntry(ginContext *gin.Context) {
	var entry PasswordEntry
	var err error = ginContext.ShouldBindJSON(&entry)
	if err != nil {
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}
	if entry.OfficialName == "" {
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": "官方名称不能为空"})
		return
	}

	var created PasswordEntry
	created, err = server.store.CreateEntry(entry)
	if err != nil {
		log.Printf("新增条目失败: %v", err)
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	ginContext.JSON(http.StatusOK, gin.H{"success": true, "entry": created})
}

func (server *Server) updateEntry(ginContext *gin.Context) {
	var entry PasswordEntry
	var err error = ginContext.ShouldBindJSON(&entry)
	if err != nil {
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}
	if entry.OfficialName == "" {
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": "官方名称不能为空"})
		return
	}

	err = server.store.UpdateEntry(entry)
	if errors.Is(err, ErrEntryNotFound) {
		ginContext.JSON(http.StatusNotFound, gin.H{"error": "条目不存在"})
		return
	}
	if err != nil {
		log.Printf("更新条目失败: %v", err)
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	ginContext.JSON(http.StatusOK, gin.H{"success": true, "entry": entry})
}

func (server *Server) deleteEntry(ginContext *gin.Context) {
	var id, err = strconv.Atoi(ginContext.Param("id"))
	if err != nil {
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": "无效的条目 id"})
		return
	}

	err = server.store.DeleteEntry(id)
	if errors.Is(err, ErrEntryNotFound) {
		ginContext.JSON(http.StatusNotFound, gin.H{"error": "条目不存在"})
		return
	}
	if err != nil {
		log.Printf("删除条目失败: %v", err)
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	ginContext.JSON(http.StatusOK, gin.H{"success": true})
}

func (server *Server) newRouter() (*gin.Engine, error) {
	var router *gin.Engine = gin.Default()

	// 不配可信代理时传 nil:ClientIP 只认直连地址,不会被伪造的 X-Forwarded-For 骗过,限流才靠谱
	var err error = router.SetTrustedProxies(server.trustedProxies)
	if err != nil {
		return nil, fmt.Errorf("设置可信代理: %w", err)
	}

	// 前端和接口同源,不需要 CORS;Cookie 会话也不该对其他源开放
	var api *gin.RouterGroup = router.Group("/api")
	api.Use(noStore)
	api.Use(limitBody)
	api.POST("/login", server.login)
	api.POST("/logout", server.logout)
	api.GET("/session", server.sessionStatus)

	var protected *gin.RouterGroup = api.Group("")
	protected.Use(server.requireSession)
	protected.GET("/entries", server.getEntries)
	protected.POST("/entries", server.addEntry)
	protected.PUT("/entries", server.updateEntry)
	protected.DELETE("/entries/:id", server.deleteEntry)
	protected.GET("/settings", server.getSettings)
	protected.PUT("/settings/session", server.updateSessionTTL)
	protected.PUT("/settings/passwords", server.changePasswords)

	var distFS fs.FS = server.webFS
	if distFS == nil {
		distFS, err = embeddedDist()
		if err != nil {
			return nil, err
		}
	}
	router.NoRoute(staticHandler(distFS))
	return router, nil
}

// splitList 把逗号分隔的环境变量拆成列表,忽略空项。
func splitList(text string) []string {
	var items []string = nil
	var parts []string = strings.Split(text, ",")
	var index int = 0
	for index = 0; index < len(parts); index++ {
		var item string = strings.TrimSpace(parts[index])
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

// browseURL 把监听地址转成能直接在浏览器打开的网址。
func browseURL(address string) string {
	if strings.HasPrefix(address, ":") {
		return "http://localhost" + address
	}
	return "http://" + address
}

const databaseFileName string = "mengyakeyvault.db"
const legacyFileName string = "data.json"

// defaultDataDir 没设置 MENGYA_DATA_DIR 时的数据目录:二进制所在目录下的 data。
// 不跟"运行时所在目录"走,这样双击、命令行、在别的目录里启动,用的都是同一个库,
// 不会因为换了个目录启动就悄悄新建一个空库。
// 例外:go run 的临时二进制放在系统临时目录里,退出就被删,数据不能放在它旁边,
// 这种情况退回运行目录下的 data(start_backend.cmd 就是在 backend 目录里 go run)。
func defaultDataDir(executablePath string, tempDir string) string {
	if executablePath == "" {
		return "data"
	}
	var executableDir string = filepath.Dir(executablePath)
	var relative, err = filepath.Rel(tempDir, executableDir)
	var insideTemp bool = err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
	if insideTemp {
		return "data"
	}
	return filepath.Join(executableDir, "data")
}

// dataDirectory 返回数据目录和用于展示的绝对路径。
// 启动和重置密码时都会把实际用的位置打出来,用错库一眼能看出来。
func dataDirectory() (string, string) {
	var executablePath, executableErr = os.Executable()
	if executableErr != nil {
		executablePath = ""
	}
	var dataDir string = cmp.Or(os.Getenv("MENGYA_DATA_DIR"), defaultDataDir(executablePath, os.TempDir()))
	var display string = dataDir
	var absolute, err = filepath.Abs(dataDir)
	if err == nil {
		display = absolute
	}
	return dataDir, display
}

func run() error {
	var err error = registerMimeTypes()
	if err != nil {
		return err
	}
	// 发布的二进制默认用 release 模式,不刷调试日志;要调试时设 GIN_MODE=debug
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 旧版 data.json 放在数据目录里,首次启动会自动导入
	var dataDir, dataDirDisplay = dataDirectory()
	log.Printf("数据目录: %s", dataDirDisplay)
	var store *Store
	store, err = openStore(filepath.Join(dataDir, databaseFileName))
	if err != nil {
		return fmt.Errorf("打开存储: %w", err)
	}

	var imported int = 0
	imported, err = store.ImportLegacyJSON(filepath.Join(dataDir, legacyFileName))
	if err != nil {
		return errors.Join(fmt.Errorf("导入旧数据: %w", err), store.Close())
	}
	if imported > 0 {
		log.Printf("已从旧版 data.json 导入 %d 条记录(原文件保持不变)", imported)
	}
	err = ensurePasswords(store)
	if err != nil {
		return errors.Join(fmt.Errorf("初始化访问密码: %w", err), store.Close())
	}

	// 放在反向代理(nginx 等)后面时,用 MENGYA_TRUSTED_PROXIES 填代理的 IP,多个用逗号分隔
	var server *Server = newServer(store, splitList(os.Getenv("MENGYA_TRUSTED_PROXIES")))
	var router *gin.Engine
	router, err = server.newRouter()
	if err != nil {
		return errors.Join(err, store.Close())
	}
	var address string = cmp.Or(os.Getenv("MENGYA_ADDR"), ":8080")
	if !frontendBuilt() {
		log.Printf("警告:这个二进制编译时没有带上前端页面,只有接口可用。请先运行 build.cmd 再编译。")
	}
	log.Printf("服务器已启动,用浏览器打开 %s", browseURL(address))
	var runErr error = router.Run(address)
	var closeErr error = store.Close()
	if runErr != nil {
		return errors.Join(fmt.Errorf("运行服务: %w", runErr), closeErr)
	}
	return closeErr
}

func main() {
	var resetRequested *bool = flag.Bool(
		"reset-passwords",
		false,
		"重置两个访问密码:生成两个新的随机密码并打印,然后退出(不启动服务,不改动已有记录)",
	)
	flag.Parse()

	var runner func() error = run
	if *resetRequested {
		runner = func() error {
			return runReset(os.Stdout)
		}
	}
	var err error = runner()
	if err != nil {
		log.Fatal(err)
	}
}
