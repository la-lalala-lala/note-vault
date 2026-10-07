package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	RedisSessionKeyPrefix = "auth:session:"
	defaultScanCount      = int64(100)
)

var (
	// ErrRedisUnavailable 表示 Redis 客户端未初始化，当前操作不能继续执行。
	ErrRedisUnavailable = errors.New("redis client is unavailable")

	// ErrInvalidRedisJSON 表示 Redis 中的数据不是目标结构所需的有效 JSON。
	ErrInvalidRedisJSON = errors.New("invalid redis json data")
)

// RedisTemplate 封装项目统一使用的 Redis 操作。
// 每次操作都会基于调用方 Context 追加超时，避免 Redis 故障时请求无限等待。
type RedisTemplate struct {
	client           *redis.Client
	operationTimeout time.Duration
}

// NewRedisTemplate 创建 Redis 工具实例。
// operationTimeout 必须大于零，正常情况下应使用已通过配置校验的 redis.operationTimeout。
func NewRedisTemplate(client *redis.Client, operationTimeout time.Duration) *RedisTemplate {
	return &RedisTemplate{
		client:           client,
		operationTimeout: operationTimeout,
	}
}

// SetJSON 将 value 序列化为 JSON 后写入 Redis，并设置过期时间。
func (r *RedisTemplate) SetJSON(ctx context.Context, key string, value any, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化 Redis JSON 数据失败: %w", err)
	}

	operationContext, cancel, err := r.withTimeout(ctx)
	if err != nil {
		return err
	}
	defer cancel()

	if err := r.client.Set(operationContext, key, data, expiration).Err(); err != nil {
		return fmt.Errorf("写入 Redis JSON 数据失败: %w", err)
	}
	return nil
}

// GetJSON 读取 Redis 中的 JSON 数据并反序列化到 destination。
// Redis Key 不存在时保留并返回 redis.Nil，调用方可据此区分缓存未命中和服务故障。
func (r *RedisTemplate) GetJSON(ctx context.Context, key string, destination any) error {
	operationContext, cancel, err := r.withTimeout(ctx)
	if err != nil {
		return err
	}
	defer cancel()

	data, err := r.client.Get(operationContext, key).Bytes()
	if err != nil {
		return fmt.Errorf("读取 Redis JSON 数据失败: %w", err)
	}
	if err := json.Unmarshal(data, destination); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidRedisJSON, err)
	}
	return nil
}

// HSet 向 Redis Hash 写入一个字段和值，并设置 Hash Key 的过期时间。
// 参数依次为 Hash Key、字段名、字段值和过期时间，例如 HSet(ctx, key, "name", "note-vault", time.Hour)。
func (r *RedisTemplate) HSet(ctx context.Context, key, field string, value any, expiration time.Duration) error {
	if expiration <= 0 {
		return errors.New("Redis Hash 过期时间必须大于零")
	}

	operationContext, cancel, err := r.withTimeout(ctx)
	if err != nil {
		return err
	}
	defer cancel()

	if _, err := r.client.TxPipelined(operationContext, func(pipe redis.Pipeliner) error {
		pipe.HSet(operationContext, key, field, value)
		pipe.Expire(operationContext, key, expiration)
		return nil
	}); err != nil {
		return fmt.Errorf("写入 Redis Hash 数据并设置过期时间失败: %w", err)
	}
	return nil
}

// HGet 从 Redis Hash 读取指定字段的值。
// 字段不存在时保留并返回 redis.Nil，调用方可据此区分未找到和 Redis 故障。
func (r *RedisTemplate) HGet(ctx context.Context, key, field string) (string, error) {
	operationContext, cancel, err := r.withTimeout(ctx)
	if err != nil {
		return "", err
	}
	defer cancel()

	value, err := r.client.HGet(operationContext, key, field).Result()
	if err != nil {
		return "", fmt.Errorf("读取 Redis Hash 字段失败: %w", err)
	}
	return value, nil
}

// HGetAll 读取 Redis Hash 的全部字段和值。
// Redis Hash 不存在时返回空 Map；Redis 故障时返回具体错误。
func (r *RedisTemplate) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	operationContext, cancel, err := r.withTimeout(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()

	values, err := r.client.HGetAll(operationContext, key).Result()
	if err != nil {
		return nil, fmt.Errorf("读取 Redis Hash 全部字段失败: %w", err)
	}
	return values, nil
}

// HExists 判断 Redis Hash 中是否存在指定字段。
func (r *RedisTemplate) HExists(ctx context.Context, key, field string) (bool, error) {
	operationContext, cancel, err := r.withTimeout(ctx)
	if err != nil {
		return false, err
	}
	defer cancel()

	exists, err := r.client.HExists(operationContext, key, field).Result()
	if err != nil {
		return false, fmt.Errorf("检查 Redis Hash 字段失败: %w", err)
	}
	return exists, nil
}

// HDel 删除 Redis Hash 中指定的一个或多个字段。
// 返回实际删除的字段数量。
func (r *RedisTemplate) HDel(ctx context.Context, key string, fields ...string) (int64, error) {
	operationContext, cancel, err := r.withTimeout(ctx)
	if err != nil {
		return 0, err
	}
	defer cancel()

	deleted, err := r.client.HDel(operationContext, key, fields...).Result()
	if err != nil {
		return 0, fmt.Errorf("删除 Redis Hash 字段失败: %w", err)
	}
	return deleted, nil
}

// TTL 返回 Redis Key 的剩余有效期。
func (r *RedisTemplate) TTL(ctx context.Context, key string) (time.Duration, error) {
	operationContext, cancel, err := r.withTimeout(ctx)
	if err != nil {
		return 0, err
	}
	defer cancel()

	ttl, err := r.client.TTL(operationContext, key).Result()
	if err != nil {
		return 0, fmt.Errorf("读取 Redis TTL 失败: %w", err)
	}
	return ttl, nil
}

// Delete 删除一个或多个明确指定的 Redis Key。
func (r *RedisTemplate) Delete(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	operationContext, cancel, err := r.withTimeout(ctx)
	if err != nil {
		return 0, err
	}
	defer cancel()

	deleted, err := r.client.Del(operationContext, keys...).Result()
	if err != nil {
		return 0, fmt.Errorf("删除 Redis Key 失败: %w", err)
	}
	return deleted, nil
}

// DeleteByPattern 使用 SCAN 分批查找并删除匹配 pattern 的 Redis Key。
// 该方法不使用会阻塞 Redis 的 KEYS 命令，适用于权限变更、用户禁用等批量失效场景。
func (r *RedisTemplate) DeleteByPattern(ctx context.Context, pattern string) (int64, error) {
	if strings.TrimSpace(pattern) == "" {
		return 0, errors.New("Redis Key 匹配模式不能为空")
	}

	operationContext, cancel, err := r.withTimeout(ctx)
	if err != nil {
		return 0, err
	}
	defer cancel()

	var cursor uint64
	var deleted int64
	for {
		keys, nextCursor, scanErr := r.client.Scan(operationContext, cursor, pattern, defaultScanCount).Result()
		if scanErr != nil {
			return deleted, fmt.Errorf("扫描 Redis Key 失败: %w", scanErr)
		}
		if len(keys) > 0 {
			count, deleteErr := r.client.Del(operationContext, keys...).Result()
			if deleteErr != nil {
				return deleted, fmt.Errorf("批量删除 Redis Key 失败: %w", deleteErr)
			}
			deleted += count
		}
		if nextCursor == 0 {
			return deleted, nil
		}
		cursor = nextCursor
	}
}

func (r *RedisTemplate) withTimeout(parent context.Context) (context.Context, context.CancelFunc, error) {
	if r == nil || r.client == nil {
		return nil, nil, ErrRedisUnavailable
	}
	if r.operationTimeout <= 0 {
		return nil, nil, errors.New("Redis 操作超时时间必须大于零")
	}
	if parent == nil {
		return nil, nil, errors.New("Redis 操作 Context 不能为空")
	}

	operationContext, cancel := context.WithTimeout(parent, r.operationTimeout)
	return operationContext, cancel, nil
}
