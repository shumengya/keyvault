package main

import "crypto/rand"
import "crypto/sha256"
import "encoding/hex"
import "errors"
import "fmt"
import "log"
import "maps"
import "net/http"
import "os"
import "strconv"
import "sync"
import "time"

import "github.com/gin-gonic/gin"
import "golang.org/x/crypto/bcrypt"

const sessionCookieName string = "mengyakeyvault_session"

// 中间件把当前会话的令牌哈希放进 gin 上下文,改密码时用它保留当前会话。
const sessionHashKey string = "sessionTokenHash"

// bcrypt 只取前 72 字节,超出的部分会被悄悄忽略,所以直接拒绝。
const maxPasswordBytes int = 72

const maxAttempts int = 5
const attemptWindow time.Duration = 15 * time.Minute
const attemptSweepSize int = 1024

// 测试里调成 bcrypt.MinCost,否则每次登录要算 4 次哈希,用例会很慢。
var bcryptCost int = bcrypt.DefaultCost

type attemptRecord struct {
	count       int
	windowStart time.Time
}

// attemptLimiter 按来源限制尝试次数(固定窗口),防止在线暴力猜密码。
type attemptLimiter struct {
	mutex   sync.Mutex
	records map[string]attemptRecord
}

func newAttemptLimiter() *attemptLimiter {
	return &attemptLimiter{records: map[string]attemptRecord{}}
}

// allow 先登记一次尝试再返回是否放行。必须在校验密码之前调用:
// 如果等校验失败后再计数,并发请求会在计数生效前全部通过检查。
func (limiter *attemptLimiter) allow(source string, now time.Time) (bool, time.Duration) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	// 来源字符串可能被伪造(没配可信代理时不会,配错了会),所以表大了就顺手清过期项
	if len(limiter.records) >= attemptSweepSize {
		maps.DeleteFunc(limiter.records, func(recordKey string, record attemptRecord) bool {
			return now.Sub(record.windowStart) >= attemptWindow
		})
	}

	var record attemptRecord = limiter.records[source]
	if now.Sub(record.windowStart) >= attemptWindow {
		record = attemptRecord{count: 0, windowStart: now}
	}
	record.count++
	limiter.records[source] = record
	if record.count > maxAttempts {
		return false, attemptWindow - now.Sub(record.windowStart)
	}
	return true, 0
}

func (limiter *attemptLimiter) reset(source string) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()
	delete(limiter.records, source)
}

func validatePassword(label string, password string) error {
	if password == "" {
		return fmt.Errorf("%s不能为空", label)
	}
	if len(password) > maxPasswordBytes {
		return fmt.Errorf("%s不能超过 %d 个字节(一个汉字占 3 个字节)", label, maxPasswordBytes)
	}
	return nil
}

// validatePasswordPair 校验一组新密码。两个密码相同的话等于只有一个密码,没有意义,直接拒绝。
func validatePasswordPair(first string, second string) error {
	var err error = validatePassword("密码 1", first)
	if err != nil {
		return err
	}
	err = validatePassword("密码 2", second)
	if err != nil {
		return err
	}
	if first == second {
		return errors.New("两个密码不能相同")
	}
	return nil
}

func hashPassword(password string) (string, error) {
	var hashed, err = bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("生成密码哈希: %w", err)
	}
	return string(hashed), nil
}

func matchPassword(hash string, password string) bool {
	var err error = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == nil {
		return true
	}
	// 不匹配是正常情况;哈希本身损坏才值得记一笔
	if !errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		log.Printf("校验密码哈希出错: %v", err)
	}
	return false
}

// passwordsMatch 判断输入的两个密码是否正好是设定的那一对,顺序不限,缺一个或错一个都不行。
// 四次比较全部执行、不提前返回,耗时不会泄露是哪一个密码错了。
func (server *Server) passwordsMatch(first string, second string) (bool, error) {
	var hashFirst, hashSecond, err = server.store.PasswordHashes()
	if err != nil {
		return false, err
	}
	if hashFirst == "" || hashSecond == "" {
		return false, errors.New("访问密码尚未初始化")
	}
	if len(first) > maxPasswordBytes || len(second) > maxPasswordBytes {
		return false, nil
	}

	var firstIsFirst bool = matchPassword(hashFirst, first)
	var secondIsSecond bool = matchPassword(hashSecond, second)
	var firstIsSecond bool = matchPassword(hashSecond, first)
	var secondIsFirst bool = matchPassword(hashFirst, second)
	return (firstIsFirst && secondIsSecond) || (firstIsSecond && secondIsFirst), nil
}

// ensurePasswords 首次启动时写入两个访问密码。
// 优先用环境变量 MENGYA_PASSWORD_1 / MENGYA_PASSWORD_2(必须同时给);都没给就随机生成并打印一次。
// 只在库里还没有密码时生效:之后在设置页改了密码,重启不会被环境变量改回去。
func ensurePasswords(store *Store) error {
	var hashFirst, hashSecond, err = store.PasswordHashes()
	if err != nil {
		return err
	}
	if hashFirst != "" && hashSecond != "" {
		return nil
	}

	var first string = os.Getenv("MENGYA_PASSWORD_1")
	var second string = os.Getenv("MENGYA_PASSWORD_2")
	var generated bool = false
	if first == "" && second == "" {
		first = rand.Text()
		second = rand.Text()
		generated = true
	}
	err = validatePasswordPair(first, second)
	if err != nil {
		return fmt.Errorf("初始访问密码无效(MENGYA_PASSWORD_1 和 MENGYA_PASSWORD_2 要同时设置): %w", err)
	}

	hashFirst, err = hashPassword(first)
	if err != nil {
		return err
	}
	hashSecond, err = hashPassword(second)
	if err != nil {
		return err
	}
	err = store.SetPasswordHashes(hashFirst, hashSecond, "")
	if err != nil {
		return err
	}
	if generated {
		log.Printf("首次启动,已随机生成两个访问密码(登录时两个都要输入,顺序不限)。\n  密码 1: %s\n  密码 2: %s\n只显示这一次,请立即记下,登录后可在设置里修改。", first, second)
	}
	return nil
}

func hashToken(token string) string {
	var sum [sha256.Size]byte = sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// setSessionCookie 写入会话 Cookie。HttpOnly 让脚本读不到,SameSite=Strict 挡跨站请求。
// Secure 只在 HTTPS(直连或经反代的 X-Forwarded-Proto)下打开,否则本地 http 访问时浏览器不收。
func setSessionCookie(ginContext *gin.Context, token string, maxAgeSeconds int) {
	var secure bool = ginContext.Request.TLS != nil || ginContext.GetHeader("X-Forwarded-Proto") == "https"
	http.SetCookie(ginContext.Writer, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAgeSeconds,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// currentSession 返回当前请求会话的令牌哈希;未登录或已过期时 valid 为 false。
func (server *Server) currentSession(ginContext *gin.Context) (string, bool, error) {
	var token, cookieErr = ginContext.Cookie(sessionCookieName)
	if cookieErr != nil || token == "" {
		return "", false, nil
	}
	var tokenHash string = hashToken(token)
	var expiresAt, found, err = server.store.SessionExpiry(tokenHash)
	if err != nil {
		return "", false, err
	}
	if !found {
		return "", false, nil
	}
	if !server.clock().Before(expiresAt) {
		err = server.store.DeleteSession(tokenHash)
		if err != nil {
			log.Printf("清理过期会话失败: %v", err)
		}
		return "", false, nil
	}
	return tokenHash, true, nil
}

func (server *Server) requireSession(ginContext *gin.Context) {
	var tokenHash, valid, err = server.currentSession(ginContext)
	if err != nil {
		log.Printf("校验会话失败: %v", err)
		ginContext.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}
	if !valid {
		ginContext.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录或登录已过期"})
		return
	}
	ginContext.Set(sessionHashKey, tokenHash)
}

func rejectTooManyAttempts(ginContext *gin.Context, retryAfter time.Duration) {
	// 向上取整,刚好 15 分钟就显示 15 而不是 16
	var minutes int = int((retryAfter + time.Minute - 1) / time.Minute)
	ginContext.Header("Retry-After", strconv.Itoa(int((retryAfter+time.Second-1)/time.Second)))
	ginContext.JSON(http.StatusTooManyRequests, gin.H{"error": fmt.Sprintf("尝试次数过多,请 %d 分钟后再试", minutes)})
}

func (server *Server) login(ginContext *gin.Context) {
	var source string = ginContext.ClientIP()
	var allowed, retryAfter = server.limiter.allow("login:"+source, server.clock())
	if !allowed {
		rejectTooManyAttempts(ginContext, retryAfter)
		return
	}

	var request struct {
		First  string `json:"first"`
		Second string `json:"second"`
	}
	var err error = ginContext.ShouldBindJSON(&request)
	if err != nil {
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求"})
		return
	}
	var matched bool
	matched, err = server.passwordsMatch(request.First, request.Second)
	if err != nil {
		log.Printf("校验访问密码失败: %v", err)
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}
	// 不区分是哪个密码错了,避免被拿来逐个猜
	if !matched {
		ginContext.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}
	server.limiter.reset("login:" + source)

	var hours int
	hours, err = server.store.SessionTTLHours()
	if err != nil {
		log.Printf("读取登录有效期失败: %v", err)
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}
	var token string = rand.Text()
	var now time.Time = server.clock()
	var lifetime time.Duration = time.Duration(hours) * time.Hour
	err = server.store.CreateSession(hashToken(token), now, now.Add(lifetime))
	if err != nil {
		log.Printf("创建会话失败: %v", err)
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}
	setSessionCookie(ginContext, token, int(lifetime/time.Second))
	ginContext.JSON(http.StatusOK, gin.H{"success": true})
}

func (server *Server) logout(ginContext *gin.Context) {
	var token, cookieErr = ginContext.Cookie(sessionCookieName)
	if cookieErr == nil && token != "" {
		var err error = server.store.DeleteSession(hashToken(token))
		if err != nil {
			log.Printf("退出登录时删除会话失败: %v", err)
			ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
			return
		}
	}
	setSessionCookie(ginContext, "", -1)
	ginContext.JSON(http.StatusOK, gin.H{"success": true})
}

// sessionStatus 前端启动时用来判断要不要显示登录页。始终返回 200,避免未登录时控制台刷红。
func (server *Server) sessionStatus(ginContext *gin.Context) {
	var _, valid, err = server.currentSession(ginContext)
	if err != nil {
		log.Printf("校验会话失败: %v", err)
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}
	ginContext.JSON(http.StatusOK, gin.H{"authenticated": valid})
}
