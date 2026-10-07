package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"note-vault/app/utils"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

const maxRequestIDLength = 128

var requestIDFallbackSequence uint64

// RequestID 为每个请求分配请求 ID，并将访问结果写入系统日志。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(utils.HeaderRequestID)
		if !validRequestID(requestID) {
			requestID = newRequestID()
		}

		c.Set(utils.ContextKeyRequestID, requestID)
		c.Header(utils.HeaderRequestID, requestID)

		startedAt := time.Now()
		c.Next()

		routePath := c.FullPath()
		if routePath == "" {
			routePath = c.Request.URL.Path
		}
		log.Printf(
			"request_id=%s method=%s path=%s status=%d latency=%s client_ip=%s response_bytes=%d",
			requestID,
			c.Request.Method,
			routePath,
			c.Writer.Status(),
			time.Since(startedAt),
			c.ClientIP(),
			c.Writer.Size(),
		)
	}
}

func validRequestID(requestID string) bool {
	if requestID == "" || len(requestID) > maxRequestIDLength {
		return false
	}
	for _, character := range requestID {
		if character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' ||
			character == '-' || character == '_' || character == '.' {
			continue
		}
		return false
	}
	return true
}

func newRequestID() string {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err == nil {
		return hex.EncodeToString(randomBytes)
	}

	sequence := atomic.AddUint64(&requestIDFallbackSequence, 1)
	return fmt.Sprintf("%x-%x", time.Now().UnixNano(), sequence)
}
