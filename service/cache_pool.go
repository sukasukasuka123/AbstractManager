package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisManager Redis 管理器
type RedisManager struct {
	Client *redis.Client
}

var globalRedisManager *RedisManager

// InitRedis 初始化 Redis 连接
func InitRedis() (*RedisManager, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
		Password:     os.Getenv("REDIS_PASSWORD"),
		DB:           0, // 使用默认 DB
		PoolSize:     50,
		MinIdleConns: 10,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect redis: %w", err)
	}

	globalRedisManager = &RedisManager{Client: client}
	return globalRedisManager, nil
}

// GetRedis 获取全局 Redis 实例
func GetRedis() *redis.Client {
	if globalRedisManager == nil {
		panic("redis not initialized, call InitRedis first")
	}
	return globalRedisManager.Client
}

// Close 关闭 Redis 连接
func (rm *RedisManager) Close() error {
	return rm.Client.Close()
}

// Set 设置缓存（带过期时间）
func (rm *RedisManager) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return rm.Client.Set(ctx, key, data, expiration).Err()
}

// Get 获取缓存
func (rm *RedisManager) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := rm.Client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// Delete 删除缓存
func (rm *RedisManager) Delete(ctx context.Context, keys ...string) error {
	return rm.Client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func (rm *RedisManager) Exists(ctx context.Context, key string) (bool, error) {
	n, err := rm.Client.Exists(ctx, key).Result()
	return n > 0, err
}

// SetMultiple 批量设置缓存
func (rm *RedisManager) SetMultiple(ctx context.Context, items map[string]interface{}, expiration time.Duration) error {
	pipe := rm.Client.Pipeline()
	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value for key %s: %w", key, err)
		}
		pipe.Set(ctx, key, data, expiration)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// GetMultiple 批量获取缓存
func (rm *RedisManager) GetMultiple(ctx context.Context, keys []string) (map[string][]byte, error) {
	pipe := rm.Client.Pipeline()
	cmds := make([]*redis.StringCmd, len(keys))

	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	result := make(map[string][]byte)
	for i, cmd := range cmds {
		if data, err := cmd.Bytes(); err == nil {
			result[keys[i]] = data
		}
	}

	return result, nil
}
