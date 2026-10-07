package middleware

import (
	"errors"
	"log"
	"note-vault/app/utils"

	"github.com/gin-gonic/gin"
)

// ErrorLogger 统一记录已经转换为 HTTP 错误响应的业务错误。
// 错误响应仍然只向客户端返回安全消息，底层原因保留在服务端日志中，便于结合 Request ID 排查。
func ErrorLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		for _, recordedError := range c.Errors {
			var httpError *utils.HTTPError
			if errors.As(recordedError.Err, &httpError) {
				log.Printf(
					"request_error request_id=%s method=%s path=%s status=%d code=%d client_ip=%s error=%q",
					c.GetString(utils.ContextKeyRequestID),
					c.Request.Method,
					requestPath(c),
					httpError.HTTPStatus,
					httpError.Code,
					c.ClientIP(),
					httpError.Error(),
				)
				continue
			}

			log.Printf(
				"request_error request_id=%s method=%s path=%s status=%d client_ip=%s error=%q",
				c.GetString(utils.ContextKeyRequestID),
				c.Request.Method,
				requestPath(c),
				c.Writer.Status(),
				c.ClientIP(),
				recordedError.Error(),
			)
		}
	}
}

func requestPath(c *gin.Context) string {
	path := c.FullPath()
	if path == "" {
		return c.Request.URL.Path
	}
	return path
}
