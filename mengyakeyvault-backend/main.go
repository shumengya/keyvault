package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const (
	DefaultPassword = "shumengya520"
	DataFile        = "data/data.json"
)

func init() {
	// 确保数据目录存在
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Printf("创建数据目录失败: %v", err)
	}
	loadData()
}

type PasswordEntry struct {
	ID           int    `json:"id"`
	Account      string `json:"account"`      // 账号
	Password     string `json:"password"`     // 密码
	Username     string `json:"username"`     // 用户名
	Phone        string `json:"phone"`        // 手机号
	Email        string `json:"email"`        // 邮箱
	Website      string `json:"website"`      // 网站地址
	OfficialName string `json:"officialName"` // 官方名称（必填）
	Tags         string `json:"tags"`         // 标签
	Logo         string `json:"logo"`         // Logo图标URL
}

type PasswordStore struct {
	Entries []PasswordEntry `json:"entries"`
	mu      sync.RWMutex
}

var store = &PasswordStore{
	Entries: make([]PasswordEntry, 0),
}

func loadData() {
	store.mu.Lock()
	defer store.mu.Unlock()

	if _, err := os.Stat(DataFile); os.IsNotExist(err) {
		// 文件不存在，创建空数据
		store.Entries = make([]PasswordEntry, 0)
		return
	}

	data, err := ioutil.ReadFile(DataFile)
	if err != nil {
		log.Printf("读取数据文件失败: %v", err)
		store.Entries = make([]PasswordEntry, 0)
		return
	}

	if len(data) == 0 {
		store.Entries = make([]PasswordEntry, 0)
		return
	}

	err = json.Unmarshal(data, store)
	if err != nil {
		log.Printf("解析数据文件失败: %v", err)
		store.Entries = make([]PasswordEntry, 0)
	}
}

func saveData() error {
	store.mu.RLock()
	defer store.mu.RUnlock()

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(DataFile, data, 0644)
}

func verifyPassword(c *gin.Context) {
	var req struct {
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求"})
		return
	}

	if req.Password == DefaultPassword {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "密码验证成功"})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
	}
}

func getEntries(c *gin.Context) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusOK, gin.H{"entries": store.Entries})
		return
	}

	// 关键词搜索
	keyword = strings.ToLower(keyword)
	var results []PasswordEntry
	for _, entry := range store.Entries {
		if strings.Contains(strings.ToLower(entry.Account), keyword) ||
			strings.Contains(strings.ToLower(entry.Username), keyword) ||
			strings.Contains(strings.ToLower(entry.Email), keyword) ||
			strings.Contains(strings.ToLower(entry.Website), keyword) ||
			strings.Contains(strings.ToLower(entry.OfficialName), keyword) ||
			strings.Contains(strings.ToLower(entry.Tags), keyword) {
			results = append(results, entry)
		}
	}

	c.JSON(http.StatusOK, gin.H{"entries": results})
}

func addEntry(c *gin.Context) {
	var entry PasswordEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	// 验证必填字段
	if entry.OfficialName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "官方名称不能为空"})
		return
	}

	store.mu.Lock()
	// 生成新ID
	maxID := 0
	for _, e := range store.Entries {
		if e.ID > maxID {
			maxID = e.ID
		}
	}
	entry.ID = maxID + 1
	store.Entries = append(store.Entries, entry)
	store.mu.Unlock()

	if err := saveData(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "entry": entry})
}

func updateEntry(c *gin.Context) {
	var entry PasswordEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	// 验证必填字段
	if entry.OfficialName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "官方名称不能为空"})
		return
	}

	store.mu.Lock()
	found := false
	for i, e := range store.Entries {
		if e.ID == entry.ID {
			store.Entries[i] = entry
			found = true
			break
		}
	}
	store.mu.Unlock()

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "条目不存在"})
		return
	}

	if err := saveData(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "entry": entry})
}

func deleteEntry(c *gin.Context) {
	id := c.Param("id")
	var entryID int
	fmt.Sscanf(id, "%d", &entryID)

	store.mu.Lock()
	found := false
	for i, e := range store.Entries {
		if e.ID == entryID {
			store.Entries = append(store.Entries[:i], store.Entries[i+1:]...)
			found = true
			break
		}
	}
	store.mu.Unlock()

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "条目不存在"})
		return
	}

	if err := saveData(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func main() {
	r := gin.Default()

	// 配置CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	r.Use(cors.New(config))

	// API路由
	api := r.Group("/api")
	{
		api.POST("/verify", verifyPassword)
		api.GET("/entries", getEntries)
		api.POST("/entries", addEntry)
		api.PUT("/entries", updateEntry)
		api.DELETE("/entries/:id", deleteEntry)
	}

	port := ":8080"
	log.Printf("服务器启动在端口 %s", port)
	if err := r.Run(port); err != nil {
		log.Fatal(err)
	}
}
