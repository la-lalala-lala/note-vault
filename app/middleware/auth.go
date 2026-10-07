package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"note-vault/app/config"
	"note-vault/app/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

const ContextKeySessionTokenHash = "sessionTokenHash"

// SessionAuthMiddleware 创建基于不透明随机 Token 和 Redis 服务端会话的认证中间件。
// 客户端直接将 Token 写入 Authorization 请求头，服务端不会解析
// Token 中的用户信息，而是对 Token 计算 SHA-256 后查询对应的 Redis 会话。
func SessionAuthMiddleware() gin.HandlerFunc {
	redisTemplate := config.NewRedisTemplate(config.Redis, config.AppConfig.Redis.OperationTimeoutDuration())

	return func(c *gin.Context) {
		// Authorization 请求头的完整内容就是 Token，不使用 Bearer 前缀。
		token := c.GetHeader("Authorization")
		if len(token) < config.AppConfig.Session.TokenBytes {
			utils.Abort(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "token无效")
			return
		}
		// 从token中 提取第 utils.SessionTokenLength(config.AppConfig.Session.TokenBytes) 到最后的字符串
		account := token[utils.SessionTokenLength(config.AppConfig.Session.TokenBytes):]
		if account == "" {
			utils.Abort(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "token无效")
			return
		}
		sessionId := utils.HashSessionToken(token)
		// Redis 是服务端会话的唯一认证来源，不可用时不能降级为绕过认证。
		if redisTemplate == nil {
			utils.Abort(c, http.StatusServiceUnavailable, utils.CodeServiceUnavailable, "认证服务暂不可用")
			return
		}

		cache, err := redisTemplate.HGet(c.Request.Context(), account, sessionId)
		if err != nil {
			// Redis 中不存在会话，说明 Token 已过期、已退出或已被主动撤销。
			if errors.Is(err, redis.Nil) || errors.Is(err, config.ErrInvalidRedisJSON) {
				utils.Abort(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "未登录或会话无效")
				return
			}
			// Redis 查询异常属于认证服务不可用，不能按未登录错误处理。
			utils.Abort(c, http.StatusServiceUnavailable, utils.CodeServiceUnavailable, "认证服务暂不可用")
			return
		}
		var session utils.SessionData
		if err := json.Unmarshal([]byte(cache), &session); err != nil {
			utils.Abort(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "未登录或会话无效")
			return
		}

		// 会话是否过期由 Redis TTL 决定，过期后查询会直接返回 redis.Nil。
		if session.UserID == 0 || session.Account == "" {
			utils.Abort(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "未登录或会话无效")
			return
		}

		// 判断 session 中的 ExpirationDate，与当前时间进行比较，判断是否过期
		if session.ExpirationDate.IsZero() || !session.ExpirationDate.After(time.Now()) {
			utils.Abort(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "会话已过期")
			return
		}

		// 开启 IP 绑定后，请求 IP 必须与登录时记录的 IP 一致。
		if config.AppConfig.Session.BindIP && session.LoginIP != c.ClientIP() {
			utils.Abort(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "登录IP与当前IP不一致")
			return
		}

		// 使用请求方法和 Gin 路由模板直接匹配用户可访问的接口。
		if !hasAPIPermission(session.APIPermissions, c.Request.Method, c.FullPath()) {
			utils.Abort(c, http.StatusForbidden, utils.CodePermissionDenied, "无接口访问权限")
			return
		}
		// 会话续期
		expiration := time.Duration(config.AppConfig.Session.ExpireHours) * time.Hour
		session.ExpirationDate = time.Now().Add(expiration)
		serializedSession, err := json.Marshal(session)
		if err != nil {
			utils.Abort(c, http.StatusForbidden, utils.CodePermissionDenied, "序列化登录会话失败")
			return
		}
		if err := redisTemplate.HSet(c, account, sessionId, serializedSession, expiration); err != nil {
			utils.Abort(c, http.StatusForbidden, utils.CodePermissionDenied, "会话续期异常")
			return
		}
		// 将可信身份信息写入请求上下文，供后续业务处理使用。
		c.Set("user", session)
		c.Set(ContextKeySessionTokenHash, sessionId)
		c.Request = c.Request.WithContext(utils.WithAuditActor(c.Request.Context(), session.Account, c.ClientIP()))
		c.Next()
	}
}

// hasAPIPermission 判断会话是否允许访问当前接口。
// routePath 使用 Gin 路由模板，例如 /api/v1/users/:id，不能使用带实际 ID 的请求路径。
func hasAPIPermission(apiPermissions []string, method, routePath string) bool {
	requestPermission := utils.APIPermissionKey(method, routePath)
	for _, apiPermission := range apiPermissions {
		if apiPermission == utils.AllAPIPermissions || apiPermission == requestPermission {
			return true
		}
	}
	return false
}
