package middleware

import (
	"ALLinSSL/backend/public"
	"crypto/md5"
	"encoding/hex"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var Html404 = []byte(`<html>
<head><title>404 Not Found</title></head>
<body bgcolor="white">
<center><h1>404 Not Found</h1></center>
<hr><center>AllinSSL</center>
</body>
</html>`)

func SessionAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		session := sessions.Default(c)
		last, ok := session.Get("lastRequestTime").(time.Time)
		public.AuthMu.RLock()
		unlocked := !public.DesktopPasswordEnabled || (session.Get("login") == true && session.Get("__login_key") == public.LoginKey && ok && time.Since(last) < time.Duration(public.TimeOut)*time.Second)
		public.AuthMu.RUnlock()
		c.Set("desktop_unlocked", unlocked)
		if !unlocked && session.Get("login") != nil {
			session.Delete("login")
			_ = session.Save()
		}
		if path == "/login" || path == "/v1/login/sign" || path == "/v1/login/skip_setup" || path == "/v1/login/sign-out" || path == "/v1/login/get_code" || path == "/v1/login/state" || path == "/v1/login/touch" || strings.HasPrefix(path, "/static/") || path == "/favicon.ico" {
			c.Next()
			return
		}
		if unlocked || checkApiKey(c) {
			c.Next()
			return
		}
		if c.IsAborted() {
			return
		}
		if strings.HasPrefix(path, "/v1/") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": false, "message": "解锁已过期，请输入软件密码"})
			return
		}
		c.Redirect(http.StatusFound, "/login")
		c.Abort()
	}
}

func checkApiKey(c *gin.Context) bool {
	var form struct {
		ApiToken  string `form:"api_token"`
		Timestamp string `form:"timestamp"`
	}
	err := c.Bind(&form)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		c.Abort()
		return false
	}
	if form.ApiToken == "" || form.Timestamp == "" {
		return false
	}
	apiKey := public.GetSettingIgnoreError("api_key")
	if apiKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未开启api"})
		c.Abort()
		return false
	}
	// timestamp := time.Now().Unix()
	ApiToken := generateSignature(form.Timestamp, apiKey)
	if form.ApiToken != ApiToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		c.Abort()
		return false
	}
	// 这里可以添加其他的验证逻辑，比如检查时间戳是否过期等
	timestamp, err := strconv.ParseInt(form.Timestamp, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid timestamp"})
		return false
	}
	if time.Now().Unix()-timestamp > 60*5 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "timestamp expired"})
		return false
	}
	return true
}

func generateSignature(timestamp, apiKey string) string {
	keyMd5 := md5.Sum([]byte(apiKey))
	keyMd5Hex := strings.ToLower(hex.EncodeToString(keyMd5[:]))

	signMd5 := md5.Sum([]byte(timestamp + keyMd5Hex))
	signMd5Hex := strings.ToLower(hex.EncodeToString(signMd5[:]))
	return signMd5Hex
}
