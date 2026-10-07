package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"note-vault/app/config"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// minimumTokenBytes 是会话 Token 允许使用的最小随机字节数。
	// 32 字节等于 256 位随机熵，可以降低 Token 被猜测或暴力枚举的风险。
	minimumTokenBytes = 32

	// AllAPIPermissions 表示允许访问全部受保护接口的通配权限。
	// 当前仅用于超级管理员会话，普通用户应保存明确的接口权限列表。
	AllAPIPermissions = "*"
)

// SessionData 表示保存在 Redis 中的服务端登录会话。
// 会话不保存原始 Token 和过期时间：原始 Token 只返回客户端，过期由 Redis TTL 控制。
type SessionData struct {
	// UserID 是当前登录用户的数据库主键，用于业务数据归属判断。
	UserID uint `json:"userId"`

	// Account 是登录账号快照，用于恢复当前操作用户信息。
	Account string `json:"account"`

	// APIPermissions 是用户可访问的接口权限键列表。
	// 普通权限格式为“HTTP方法:路由模板”，例如 GET:/api/v1/users/:id；
	// 超级管理员可以使用 AllAPIPermissions 表示拥有全部接口权限。
	APIPermissions []string `json:"apiPermissions"`

	// LoginIP 是用户登录时的客户端 IP，开启 IP 绑定后用于校验请求来源。
	LoginIP string `json:"loginIp"`

	// 到期时间
	ExpirationDate time.Time `json:"expirationDate"`
}

// GenerateSessionToken 生成密码学安全的随机会话 Token。
// tokenBytes 表示随机字节数；传入 0 时使用默认的 32 字节，小于 32 时返回错误。
// 返回值使用无填充 Base64URL 编码，可以安全地放入 HTTP Authorization 请求头。
func GenerateSessionToken(account string, tokenBytes int) (string, error) {
	if tokenBytes == 0 {
		tokenBytes = minimumTokenBytes
	}
	if tokenBytes < minimumTokenBytes {
		return "", errors.New("session token must contain at least 32 random bytes")
	}

	randomBytes := make([]byte, tokenBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	// 特别处理，将账号放在最后
	return base64.RawURLEncoding.EncodeToString(randomBytes) + account, nil
}

// SessionTokenLength 返回随机部分经过无填充 Base64URL 编码后的字符数，不包含末尾拼接的账号。
// tokenBytes 为 0 时按默认的 32 字节计算；小于最小随机字节数时返回 0。
func SessionTokenLength(tokenBytes int) int {
	if tokenBytes == 0 {
		tokenBytes = minimumTokenBytes
	}
	if tokenBytes < minimumTokenBytes {
		return 0
	}
	return base64.RawURLEncoding.EncodedLen(tokenBytes)
}

// HashSessionToken 计算原始会话 Token 的 SHA-256 哈希值。
// Redis Key 只保存该哈希值，避免原始 Token 直接出现在服务端缓存键和排查信息中。
func HashSessionToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

// SessionKey 根据原始 Token 生成 Redis 会话 Key。
// Key 格式为 auth:session:{tokenHash}，其中 tokenHash 是 Token 的 SHA-256 十六进制结果。
func SessionKey(token string) string {
	return config.RedisSessionKeyPrefix + HashSessionToken(token)
}

// APIPermissionKey 根据 HTTP 方法和 Gin 路由模板生成接口权限键。
// HTTP 方法会被转换为大写，方法和路径两侧的空白会被移除。
// routePath 应传入 Gin 路由模板而不是实际请求路径，例如 GET:/api/v1/users/:id。
func APIPermissionKey(method, routePath string) string {
	return strings.ToUpper(strings.TrimSpace(method)) + ":" + strings.TrimSpace(routePath)
}

// 从Context中获取当前用户信息
func CurrentSession(c *gin.Context) (SessionData, bool) {
	value, exists := c.Get("user")
	if !exists {
		return SessionData{}, false
	}

	session, ok := value.(SessionData)
	return session, ok
}
