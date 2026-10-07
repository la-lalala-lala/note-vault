package middleware

import (
	"net/http"
	"note-vault/app/utils"
	"strconv"
	"strings"

	"note-vault/app/config"

	"github.com/gin-gonic/gin"
)

// CORS 根据配置处理跨域请求和 OPTIONS 预检请求。
func CORS(corsConfig config.CORSConfig) gin.HandlerFunc {
	allowedMethods := strings.Join(corsConfig.AllowedMethods, ", ")
	allowedHeaders := strings.Join(corsConfig.AllowedHeaders, ", ")
	exposedHeaders := strings.Join(corsConfig.ExposedHeaders, ", ")
	maxAgeSeconds := strconv.FormatInt(int64(corsConfig.MaxAgeDuration().Seconds()), 10)

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		allowedOrigin, allowed := matchOrigin(origin, corsConfig.AllowedOrigins)
		if !allowed {
			utils.Abort(c, http.StatusForbidden, utils.CodePermissionDenied, "跨域来源不被允许")
			return
		}

		c.Header("Access-Control-Allow-Origin", allowedOrigin)
		c.Header("Access-Control-Allow-Methods", allowedMethods)
		c.Header("Access-Control-Allow-Headers", allowedHeaders)
		c.Header("Access-Control-Expose-Headers", exposedHeaders)
		c.Header("Access-Control-Max-Age", maxAgeSeconds)
		c.Header("Vary", "Origin")
		if corsConfig.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// SecurityHeaders 为 API 和前端静态资源响应增加安全响应头。
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// API 和前端页面共用同一个服务，因此 CSP 需要根据请求路径动态选择。
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Content-Security-Policy", contentSecurityPolicy(c.Request.URL.Path))
		c.Header("Cache-Control", "no-store")
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}

func contentSecurityPolicy(requestPath string) string {
	// API 和健康检查只返回 JSON，不需要加载脚本、样式或其他外部资源，使用最严格的策略。
	if requestPath == "/api" || strings.HasPrefix(requestPath, "/api/") ||
		requestPath == "/health" || strings.HasPrefix(requestPath, "/health/") {
		return "default-src 'none'; frame-ancestors 'none'"
	}
	// 前端页面需要加载同源 JS、CSS、字体、图片，并向同源 API 发起请求。
	// unsafe-inline 用于兼容 UI 框架生成的内联样式，data/blob 用于图片和字体资源。
	return "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob: https:; font-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'"
}

// BodyLimit 限制请求体大小，防止超大请求持续占用服务资源。
func BodyLimit(maxBodyBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBodyBytes {
			utils.Abort(c, http.StatusRequestEntityTooLarge, utils.CodePayloadTooLarge, "请求体大小超过限制")
			return
		}
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
		}
		c.Next()
	}
}

func matchOrigin(origin string, allowedOrigins []string) (string, bool) {
	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin == "*" {
			return "*", true
		}
		if origin == allowedOrigin {
			return origin, true
		}
	}
	return "", false
}
