package server

import (
	"ALLinSSL/backend/middleware"
	"ALLinSSL/backend/public"
	"ALLinSSL/backend/route"
	"crypto/rand"
	"encoding/gob"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/memstore"
	"github.com/gin-gonic/gin"
	"time"
)

// NewRouter 保留原业务路由，解锁会话只存在于当前进程。
func NewRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.CustomRecovery(func(c *gin.Context, recovered any) {
		public.Error(fmt.Sprintf("业务处理失败: %v", recovered))
		public.FailMsg(c, "业务处理失败，请查看软件日志")
	}))

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		panic("生成解锁会话密钥失败: " + err.Error())
	}
	store := memstore.NewStore(secret) // 只在内存中，不持久化
	r.Use(sessions.Sessions(public.SessionKey, store))
	r.Use(middleware.LoggerMiddleware())
	gob.Register(time.Time{})
	r.Use(middleware.SessionAuthMiddleware())
	// r.Use(middleware.OpLoggerMiddleware())
	route.Register(r)
	return r
}
