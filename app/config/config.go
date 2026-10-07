package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 是应用的完整配置。
type Config struct {
	Server   ServerConfig
	Web      WebConfig
	CORS     CORSConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Session  SessionConfig
	Auth     AuthConfig
	Storage  StorageConfig
	Trash    TrashConfig
	Log      LogConfig
}

// WebConfig 定义前端静态资源配置。
type WebConfig struct {
	DistDir string
}

// ServerConfig 定义 HTTP 服务配置。
type ServerConfig struct {
	Host               string
	Port               int
	ReadTimeout        string
	WriteTimeout       string
	ShutdownTimeout    string
	HealthCheckTimeout string
	MaxBodyBytes       int64
}

// CORSConfig 定义跨域请求策略。
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           string
}

// DatabaseConfig 定义 PostgreSQL 和连接池配置。
type DatabaseConfig struct {
	DSN  string
	Pool PoolConfig
}

// PoolConfig 定义数据库连接池参数。
type PoolConfig struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime string
	ConnMaxIdleTime string
}

// RedisConfig 定义 Redis 客户端配置。
type RedisConfig struct {
	Addr             string
	Password         string
	DB               int
	PoolSize         int
	MinIdleConns     int
	MaxRetries       int
	OperationTimeout string
}

// SessionConfig 定义服务端登录会话配置。
type SessionConfig struct {
	ExpireHours int
	TokenBytes  int
	BindIP      bool
}

// AuthConfig 定义公开接口和鉴权相关配置。
type AuthConfig struct {
	TrustedProxies []string
	MaxSessions    int
}

// StorageConfig 定义文件存储配置。
type StorageConfig struct {
	Type              string
	BasePath          string
	MaxFileSize       int64
	AllowedMimeTypes  []string
	AllowedExtensions []string
}

// AllowsMIMEType 判断文件 MIME 类型是否在允许列表中。
func (c StorageConfig) AllowsMIMEType(mimeType string) bool {
	target := strings.ToLower(strings.TrimSpace(mimeType))
	for _, allowed := range c.AllowedMimeTypes {
		if strings.ToLower(strings.TrimSpace(allowed)) == target {
			return true
		}
	}
	return false
}

// AllowsExtension 判断文件扩展名是否在允许列表中。
func (c StorageConfig) AllowsExtension(extension string) bool {
	target := strings.ToLower(strings.TrimSpace(extension))
	for _, allowed := range c.AllowedExtensions {
		if strings.ToLower(strings.TrimSpace(allowed)) == target {
			return true
		}
	}
	return false
}

// TrashConfig 定义回收站保留和清理配置。
type TrashConfig struct {
	RetentionDays   int
	CleanupInterval string
}

// CleanupIntervalDuration 返回通过配置校验后的回收站清理周期。
func (c TrashConfig) CleanupIntervalDuration() time.Duration {
	duration, _ := time.ParseDuration(c.CleanupInterval)
	return duration
}

// LogConfig 定义应用日志配置。
type LogConfig struct {
	Level  string
	Format string
	Output string
}

// AppConfig 保存校验通过后的全局配置。
var AppConfig Config

// LoadConfig 读取 config.yaml，应用默认值并校验必要配置。
func LoadConfig() error {
	viper.SetConfigFile("config.yaml")
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	var loaded Config
	if err := viper.Unmarshal(&loaded); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}
	if err := loaded.Validate(); err != nil {
		return err
	}

	AppConfig = loaded
	return nil
}

// Validate 校验启动所需配置，避免服务带着无效配置继续运行。
func (c Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("配置 server.port 必须在 1 到 65535 之间")
	}
	if err := validatePositiveDuration("server.readTimeout", c.Server.ReadTimeout); err != nil {
		return err
	}
	if err := validatePositiveDuration("server.writeTimeout", c.Server.WriteTimeout); err != nil {
		return err
	}
	if err := validatePositiveDuration("server.shutdownTimeout", c.Server.ShutdownTimeout); err != nil {
		return err
	}
	if err := validatePositiveDuration("server.healthCheckTimeout", c.Server.HealthCheckTimeout); err != nil {
		return err
	}
	if c.Server.MaxBodyBytes <= 0 {
		return fmt.Errorf("配置 server.maxBodyBytes 必须大于 0")
	}
	if strings.TrimSpace(c.Web.DistDir) == "" {
		return fmt.Errorf("配置 web.distDir 不能为空")
	}

	if len(c.CORS.AllowedOrigins) == 0 {
		return fmt.Errorf("配置 cors.allowedOrigins 不能为空")
	}
	if len(c.CORS.AllowedMethods) == 0 {
		return fmt.Errorf("配置 cors.allowedMethods 不能为空")
	}
	if len(c.CORS.AllowedHeaders) == 0 {
		return fmt.Errorf("配置 cors.allowedHeaders 不能为空")
	}
	if c.CORS.AllowCredentials && contains(c.CORS.AllowedOrigins, "*") {
		return fmt.Errorf("cors.allowCredentials=true 时 allowedOrigins 不能包含通配符 *")
	}
	if err := validatePositiveDuration("cors.maxAge", c.CORS.MaxAge); err != nil {
		return err
	}

	if strings.TrimSpace(c.Database.DSN) == "" {
		return fmt.Errorf("配置 database.dsn 不能为空")
	}
	if c.Database.Pool.MaxIdleConns < 0 || c.Database.Pool.MaxOpenConns <= 0 {
		return fmt.Errorf("数据库连接池配置必须满足 maxIdleConns >= 0 且 maxOpenConns > 0")
	}
	if c.Database.Pool.MaxIdleConns > c.Database.Pool.MaxOpenConns {
		return fmt.Errorf("配置 database.pool.maxIdleConns 不能大于 maxOpenConns")
	}
	if err := validatePositiveDuration("database.pool.connMaxLifetime", c.Database.Pool.ConnMaxLifetime); err != nil {
		return err
	}
	if err := validatePositiveDuration("database.pool.connMaxIdleTime", c.Database.Pool.ConnMaxIdleTime); err != nil {
		return err
	}

	if strings.TrimSpace(c.Redis.Addr) == "" {
		return fmt.Errorf("配置 redis.addr 不能为空")
	}
	if _, _, err := net.SplitHostPort(c.Redis.Addr); err != nil {
		return fmt.Errorf("配置 redis.addr 格式无效: %w", err)
	}
	if c.Redis.DB < 0 || c.Redis.PoolSize <= 0 || c.Redis.MinIdleConns < 0 {
		return fmt.Errorf("Redis 配置必须满足 db >= 0、poolSize > 0、minIdleConns >= 0")
	}
	if c.Redis.MinIdleConns > c.Redis.PoolSize {
		return fmt.Errorf("配置 redis.minIdleConns 不能大于 poolSize")
	}
	if err := validatePositiveDuration("redis.operationTimeout", c.Redis.OperationTimeout); err != nil {
		return err
	}

	if c.Session.ExpireHours <= 0 {
		return fmt.Errorf("配置 session.expireHours 必须大于 0")
	}
	if c.Session.TokenBytes < 32 {
		return fmt.Errorf("配置 session.tokenBytes 不能小于 32")
	}
	if c.Auth.MaxSessions <= 0 {
		return fmt.Errorf("配置 auth.maxSessions 必须大于 0")
	}

	storageType := strings.ToLower(strings.TrimSpace(c.Storage.Type))
	if storageType != "local" {
		return fmt.Errorf("配置 storage.type 当前仅支持 local")
	}
	if strings.TrimSpace(c.Storage.BasePath) == "" {
		return fmt.Errorf("本地存储模式下 storage.basePath 不能为空")
	}
	if c.Storage.MaxFileSize <= 0 {
		return fmt.Errorf("配置 storage.maxFileSize 必须大于 0")
	}
	if len(c.Storage.AllowedMimeTypes) == 0 {
		return fmt.Errorf("配置 storage.allowedMimeTypes 不能为空")
	}
	for _, mimeType := range c.Storage.AllowedMimeTypes {
		mimeType = strings.ToLower(strings.TrimSpace(mimeType))
		if mimeType == "" || !strings.Contains(mimeType, "/") {
			return fmt.Errorf("配置 storage.allowedMimeTypes 包含无效 MIME 类型")
		}
	}
	if len(c.Storage.AllowedExtensions) == 0 {
		return fmt.Errorf("配置 storage.allowedExtensions 不能为空")
	}
	for _, extension := range c.Storage.AllowedExtensions {
		extension = strings.ToLower(strings.TrimSpace(extension))
		if len(extension) < 2 || !strings.HasPrefix(extension, ".") {
			return fmt.Errorf("配置 storage.allowedExtensions 必须使用以 . 开头的扩展名")
		}
	}

	if c.Trash.RetentionDays <= 0 {
		return fmt.Errorf("配置 trash.retentionDays 必须大于 0")
	}
	if err := validatePositiveDuration("trash.cleanupInterval", c.Trash.CleanupInterval); err != nil {
		return err
	}

	if !contains([]string{"debug", "info", "warn", "error"}, c.Log.Level) {
		return fmt.Errorf("配置 log.level 仅支持 debug、info、warn、error")
	}
	if !contains([]string{"json", "console"}, c.Log.Format) {
		return fmt.Errorf("配置 log.format 仅支持 json、console")
	}
	if strings.TrimSpace(c.Log.Output) == "" {
		return fmt.Errorf("配置 log.output 不能为空")
	}

	return nil
}

// ServerAddress 返回 HTTP Server 使用的监听地址。
func (c Config) ServerAddress() string {
	return net.JoinHostPort(c.Server.Host, strconv.Itoa(c.Server.Port))
}

// HealthCheckTimeoutDuration 返回通过校验后的健康检查超时时间。
func (c ServerConfig) HealthCheckTimeoutDuration() time.Duration {
	duration, _ := time.ParseDuration(c.HealthCheckTimeout)
	return duration
}

// MaxAgeDuration 返回通过校验后的 CORS 预检缓存时间。
func (c CORSConfig) MaxAgeDuration() time.Duration {
	duration, _ := time.ParseDuration(c.MaxAge)
	return duration
}

// OperationTimeoutDuration 返回通过校验后的 Redis 单次操作超时时间。
func (c RedisConfig) OperationTimeoutDuration() time.Duration {
	duration, _ := time.ParseDuration(c.OperationTimeout)
	return duration
}

func setDefaults() {
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.readTimeout", "15s")
	viper.SetDefault("server.writeTimeout", "15s")
	viper.SetDefault("server.shutdownTimeout", "10s")
	viper.SetDefault("server.healthCheckTimeout", "3s")
	viper.SetDefault("server.maxBodyBytes", 16*1024*1024)
	viper.SetDefault("web.distDir", "dist")
	viper.SetDefault("cors.allowedOrigins", []string{"*"})
	viper.SetDefault("cors.allowedMethods", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"})
	viper.SetDefault("cors.allowedHeaders", []string{"Authorization", "Content-Type", "X-Request-ID"})
	viper.SetDefault("cors.exposedHeaders", []string{"X-Request-ID"})
	viper.SetDefault("cors.allowCredentials", false)
	viper.SetDefault("cors.maxAge", "12h")
	viper.SetDefault("redis.operationTimeout", "3s")
	viper.SetDefault("session.tokenBytes", 32)
	viper.SetDefault("auth.maxSessions", 5)
	viper.SetDefault("storage.type", "local")
	viper.SetDefault("storage.basePath", "./data")
	viper.SetDefault("storage.maxFileSize", 10*1024*1024)
	viper.SetDefault("storage.allowedMimeTypes", []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
		"application/pdf",
		"text/plain",
		"text/markdown",
	})
	viper.SetDefault("storage.allowedExtensions", []string{
		".jpg",
		".jpeg",
		".png",
		".gif",
		".webp",
		".pdf",
		".txt",
		".md",
	})
	viper.SetDefault("trash.retentionDays", 30)
	viper.SetDefault("trash.cleanupInterval", "1h")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "console")
	viper.SetDefault("log.output", "stdout")
}

func validatePositiveDuration(name, value string) error {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("配置 %s 格式无效: %w", name, err)
	}
	if duration <= 0 {
		return fmt.Errorf("配置 %s 必须大于 0", name)
	}
	return nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
