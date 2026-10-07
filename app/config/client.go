package config

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	// DB 是应用共享的 PostgreSQL GORM 客户端。
	DB *gorm.DB

	// Redis 是应用共享的 Redis 客户端。
	Redis *redis.Client
)

// InitDB 初始化 PostgreSQL 客户端、连接池并执行连通性检查。
func InitDB(ctx context.Context) error {
	database, err := gorm.Open(postgres.Open(AppConfig.Database.DSN), &gorm.Config{
		Logger:         logger.Default.LogMode(gormLogLevel(AppConfig.Log.Level)),
		TranslateError: true,
	})
	if err != nil {
		return fmt.Errorf("连接 PostgreSQL 失败: %w", err)
	}

	sqlDB, err := database.DB()
	if err != nil {
		return fmt.Errorf("获取 PostgreSQL 连接池失败: %w", err)
	}

	connMaxLifetime, err := time.ParseDuration(AppConfig.Database.Pool.ConnMaxLifetime)
	if err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("解析数据库连接最大生命周期失败: %w", err)
	}
	connMaxIdleTime, err := time.ParseDuration(AppConfig.Database.Pool.ConnMaxIdleTime)
	if err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("解析数据库连接最大空闲时间失败: %w", err)
	}

	sqlDB.SetMaxIdleConns(AppConfig.Database.Pool.MaxIdleConns)
	sqlDB.SetMaxOpenConns(AppConfig.Database.Pool.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("PostgreSQL 连通性检查失败: %w", err)
	}

	DB = database
	return nil
}

// InitRedis 独立初始化 Redis 客户端并执行连通性检查。
func InitRedis(ctx context.Context) error {
	operationTimeout := AppConfig.Redis.OperationTimeoutDuration()
	client := redis.NewClient(&redis.Options{
		Addr:         AppConfig.Redis.Addr,
		Password:     AppConfig.Redis.Password,
		DB:           AppConfig.Redis.DB,
		PoolSize:     AppConfig.Redis.PoolSize,
		MinIdleConns: AppConfig.Redis.MinIdleConns,
		MaxRetries:   AppConfig.Redis.MaxRetries,
		DialTimeout:  operationTimeout,
		ReadTimeout:  operationTimeout,
		WriteTimeout: operationTimeout,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return fmt.Errorf("Redis 连通性检查失败: %w", err)
	}

	Redis = client
	return nil
}

// CloseDB 关闭 PostgreSQL 连接池。
func CloseDB() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取 PostgreSQL 连接池失败: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("关闭 PostgreSQL 连接池失败: %w", err)
	}

	DB = nil
	return nil
}

// CloseRedis 关闭 Redis 客户端。
func CloseRedis() error {
	if Redis == nil {
		return nil
	}

	if err := Redis.Close(); err != nil {
		return fmt.Errorf("关闭 Redis 客户端失败: %w", err)
	}

	Redis = nil
	return nil
}

// CloseClients 关闭应用持有的全部外部客户端。
func CloseClients() error {
	return errors.Join(CloseRedis(), CloseDB())
}

func gormLogLevel(level string) logger.LogLevel {
	switch level {
	case "debug", "info":
		return logger.Info
	case "warn":
		return logger.Warn
	default:
		return logger.Error
	}
}
