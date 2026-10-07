package middleware

import (
	"log"
	"net/http"
	"note-vault/app/utils"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Recovery 捕获请求处理链中的 Panic，避免单个请求导致服务进程退出。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf(
					"request_id=%s panic=%v stack=%s",
					c.GetString(utils.ContextKeyRequestID),
					recovered,
					debug.Stack(),
				)
				if !c.Writer.Written() {
					utils.Error(c, http.StatusInternalServerError, utils.CodeInternalError, "系统内部错误")
				}
				c.Abort()
			}
		}()

		c.Next()
	}
}
