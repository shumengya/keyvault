package main

import "fmt"
import "log"
import "net/http"

import "github.com/gin-gonic/gin"

func (server *Server) getSettings(ginContext *gin.Context) {
	var hours, err = server.store.SessionTTLHours()
	if err != nil {
		log.Printf("读取设置失败: %v", err)
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}
	ginContext.JSON(http.StatusOK, gin.H{
		"sessionTTLHours":    hours,
		"minSessionTTLHours": minSessionTTLHours,
		"maxSessionTTLHours": maxSessionTTLHours,
	})
}

func (server *Server) updateSessionTTL(ginContext *gin.Context) {
	var request struct {
		Hours int `json:"hours"`
	}
	var err error = ginContext.ShouldBindJSON(&request)
	if err != nil {
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求"})
		return
	}
	if request.Hours < minSessionTTLHours || request.Hours > maxSessionTTLHours {
		ginContext.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("有效期需要在 %d 到 %d 小时之间", minSessionTTLHours, maxSessionTTLHours),
		})
		return
	}
	err = server.store.SetSessionTTLHours(request.Hours)
	if err != nil {
		log.Printf("保存登录有效期失败: %v", err)
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	ginContext.JSON(http.StatusOK, gin.H{"success": true, "sessionTTLHours": request.Hours})
}

// changePasswords 修改两个访问密码。必须再输入一遍当前的两个密码:
// 只拿到登录会话(比如借用了没锁屏的电脑)不足以把主人锁在外面。
func (server *Server) changePasswords(ginContext *gin.Context) {
	var request struct {
		CurrentFirst  string `json:"currentFirst"`
		CurrentSecond string `json:"currentSecond"`
		NewFirst      string `json:"newFirst"`
		NewSecond     string `json:"newSecond"`
	}
	var err error = ginContext.ShouldBindJSON(&request)
	if err != nil {
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求"})
		return
	}
	err = validatePasswordPair(request.NewFirst, request.NewSecond)
	if err != nil {
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 和登录分开计数:设置页输错当前密码不该把登录也锁住
	var source string = "change:" + ginContext.ClientIP()
	var allowed, retryAfter = server.limiter.allow(source, server.clock())
	if !allowed {
		rejectTooManyAttempts(ginContext, retryAfter)
		return
	}
	var matched bool
	matched, err = server.passwordsMatch(request.CurrentFirst, request.CurrentSecond)
	if err != nil {
		log.Printf("校验当前密码失败: %v", err)
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}
	// 用 403 而不是 401:前端收到 401 会当成登录过期直接踢回登录页
	if !matched {
		ginContext.JSON(http.StatusForbidden, gin.H{"error": "当前密码错误"})
		return
	}
	server.limiter.reset(source)

	var hashFirst string
	hashFirst, err = hashPassword(request.NewFirst)
	if err != nil {
		log.Printf("%v", err)
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}
	var hashSecond string
	hashSecond, err = hashPassword(request.NewSecond)
	if err != nil {
		log.Printf("%v", err)
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}
	err = server.store.SetPasswordHashes(hashFirst, hashSecond, ginContext.GetString(sessionHashKey))
	if err != nil {
		log.Printf("保存新密码失败: %v", err)
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	ginContext.JSON(http.StatusOK, gin.H{"success": true})
}
