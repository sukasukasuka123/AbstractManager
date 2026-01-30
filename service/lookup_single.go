package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// LookupSingleOptions 单个缓存查询配置选项
type LookupSingleOptions struct {
	CacheExpire  time.Duration
	FallbackToDB bool
	Refresh      bool
}

// LookupSingle 从缓存中查询单个数据
func (sm *ServiceManager[T]) LookupSingle(
	ctx context.Context,
	key string,
	opts *LookupSingleOptions,
) (*T, error) {
	rdb := GetRedis() // 🛠️ 保持使用 rdb 避免遮蔽包名

	// 1. 检查是否需要从缓存读取
	if opts == nil || !opts.Refresh {
		var result T
		// 🛠️ 优化：直接使用 Scan 自动处理 JSON 解码
		err := rdb.Get(ctx, key).Scan(&result)
		if err == nil {
			return &result, nil
		}

		// 如果是真正的错误（非 key 不存在），则返回
		if err != redis.Nil {
			return nil, fmt.Errorf("redis lookup failed: %w", err)
		}
	}

	// 2. 缓存未命中且允许回源
	if opts != nil && opts.FallbackToDB {
		// 注意：这里的 queryFunc 在通用 lookup 中较难确定，建议配合 ID 使用
		return nil, fmt.Errorf("fallback requested but no query logic provided for key: %s", key)
	}

	return nil, redis.Nil // 显式返回未命中
}

// LookupSingleWithFallback 核心方法：带自动回填的查询
func (sm *ServiceManager[T]) LookupSingleWithFallback(
	ctx context.Context,
	key string,
	queryFunc func(*gorm.DB) *gorm.DB,
	expiration time.Duration,
) (*T, error) {
	rdb := GetRedis()

	// 1. 尝试缓存
	var result T
	err := rdb.Get(ctx, key).Scan(&result)
	if err == nil {
		return &result, nil
	}
	if err != redis.Nil {
		return nil, fmt.Errorf("cache error: %w", err)
	}

	// 2. 缓存未命中，回源数据库
	data, err := sm.GetSingle(ctx, queryFunc, nil)
	if err != nil {
		return nil, err // GetSingle 内部已处理 ErrRecordNotFound
	}

	// 3. 异步回填缓存（Y-like 风格：不让主流程等待非核心写入）
	sm.WritedownSingleAsync(ctx, key, data, expiration)

	return data, nil
}

// InvalidateSingleCache 使单个缓存失效
func (sm *ServiceManager[T]) InvalidateSingleCache(ctx context.Context, key string) error {
	rdb := GetRedis()
	// 🛠️ 修复：.Err() 获取错误，修复 %w 类型报错
	if err := rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to invalidate cache: %w", err)
	}
	return nil
}

// ExistsInCache 检查缓存中是否存在
func (sm *ServiceManager[T]) ExistsInCache(ctx context.Context, key string) (bool, error) {
	rdb := GetRedis()
	n, err := rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("exists check failed: %w", err)
	}
	return n > 0, nil
}

// ExtendCacheTTL 延长缓存的过期时间
func (sm *ServiceManager[T]) ExtendCacheTTL(ctx context.Context, key string, expiration time.Duration) error {
	rdb := GetRedis()
	// 🛠️ 修复：使用 .Err() 确保传给 %w的是 error 类型
	if err := rdb.Expire(ctx, key, expiration).Err(); err != nil {
		return fmt.Errorf("failed to extend TTL: %w", err)
	}
	return nil
}

// --- 便捷封装 ---

func (sm *ServiceManager[T]) LookupSingleByID(ctx context.Context, id interface{}, expiration time.Duration) (*T, error) {
	key := sm.buildCacheKey(id)
	return sm.LookupSingleWithFallback(ctx, key, func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", id)
	}, expiration)
}

// InvalidateSingleCacheByID 根据 ID 使单个缓存失效
func (sm *ServiceManager[T]) InvalidateSingleCacheByID(ctx context.Context, id interface{}) error {
	key := sm.buildCacheKey(id)
	return sm.InvalidateSingleCache(ctx, key)
}

// GetCacheTTL 获取缓存的剩余过期时间
func (sm *ServiceManager[T]) GetCacheTTL(ctx context.Context, key string) (time.Duration, error) {
	redisManager := GetRedis()
	return redisManager.TTL(ctx, key).Result()
}

// buildCacheKey 构建缓存键
func (sm *ServiceManager[T]) buildCacheKey(id interface{}) string {
	if sm.CacheKeyType == "none" {
		return fmt.Sprintf("%s:%v", sm.CacheKeyName, id)
	}
	return fmt.Sprintf("%s:%s:%v", sm.CacheKeyType, sm.CacheKeyName, id)
}
