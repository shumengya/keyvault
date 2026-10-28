package main

import "cmp"
import "errors"
import "fmt"
import "log"
import "net/http"
import "os"
import "path/filepath"
import "strconv"

import "github.com/gin-contrib/cors"
import "github.com/gin-gonic/gin"

const DefaultPassword string = "shumengya520"

// Server 持有各接口共用的存储层。
type Server struct {
	store *Store
}

func (server *Server) verifyPassword(ginContext *gin.Context) {
	var request struct {
		Password string `json:"password"`
	}
	var err error = ginContext.ShouldBindJSON(&request)
	if err != nil {
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求"})
		return
	}
	if request.Password != DefaultPassword {
		ginContext.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}
	ginContext.JSON(http.StatusOK, gin.H{"success": true, "message": "密码验证成功"})
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

func (server *Server) newRouter() *gin.Engine {
	var router *gin.Engine = gin.Default()

	var corsConfig cors.Config = cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	router.Use(cors.New(corsConfig))

	var api *gin.RouterGroup = router.Group("/api")
	api.POST("/verify", server.verifyPassword)
	api.GET("/entries", server.getEntries)
	api.POST("/entries", server.addEntry)
	api.PUT("/entries", server.updateEntry)
	api.DELETE("/entries/:id", server.deleteEntry)
	return router
}

func run() error {
	// 数据目录默认是运行目录下的 data;旧版 data.json 放在同一目录,首次启动会自动导入
	var dataDir string = cmp.Or(os.Getenv("MENGYA_DATA_DIR"), "data")
	var store, err = openStore(filepath.Join(dataDir, "mengyakeyvault.db"))
	if err != nil {
		return fmt.Errorf("打开存储: %w", err)
	}

	var imported int = 0
	imported, err = store.ImportLegacyJSON(filepath.Join(dataDir, "data.json"))
	if err != nil {
		return errors.Join(fmt.Errorf("导入旧数据: %w", err), store.Close())
	}
	if imported > 0 {
		log.Printf("已从旧版 data.json 导入 %d 条记录(原文件保持不变)", imported)
	}

	var server *Server = &Server{store: store}
	var address string = cmp.Or(os.Getenv("MENGYA_ADDR"), ":8080")
	log.Printf("服务器启动在 %s", address)
	var runErr error = server.newRouter().Run(address)
	var closeErr error = store.Close()
	if runErr != nil {
		return errors.Join(fmt.Errorf("运行服务: %w", runErr), closeErr)
	}
	return closeErr
}

func main() {
	var err error = run()
	if err != nil {
		log.Fatal(err)
	}
}
