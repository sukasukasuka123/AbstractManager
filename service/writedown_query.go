package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// WritedownQueryOptions 批量写入缓存配置选项
type WritedownQueryOptions struct {
	Expiration time.Duration
	BatchSize  int
	Overwrite  bool
}

// WritedownQuery 批量将数据写入缓存
func (sm *ServiceManager[T]) WritedownQuery(
	ctx context.Context,
	data []T,
	buildKeyFunc func(*T) string,
	opts *WritedownQueryOptions,
) error {
	if len(data) == 0 {
		return nil
	}

	if opts == nil {
		opts = &WritedownQueryOptions{
			Expiration: 1 * time.Hour,
			BatchSize:  100,
			Overwrite:  true,
		}
	}

	redis := GetRedis() // 假设返回的是 *redis.Client
	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	for i := 0; i < len(data); i += batchSize {
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}

		batch := data[i:end]
		cacheItems := make(map[string]interface{})

		for j := range batch {
			item := &batch[j]
			key := buildKeyFunc(item)

			if !opts.Overwrite {
				// 🛠️ 修复 1: Exists 返回的是 *IntCmd，需要调用 .Val() 获取结果
				if redis.Exists(ctx, key).Val() > 0 {
					continue
				}
			}
			valueBytes, err := json.Marshal(item)
			if err != nil {
				return fmt.Errorf("failed to marshal item for key %s: %w", key, err)
			}

			cacheItems[key] = valueBytes // 存 []byte
		}

		if len(cacheItems) > 0 {
			// 🛠️ 修复 2: go-redis 标准方法是 MSet，而不是 SetMultiple
			if err := redis.MSet(ctx, cacheItems).Err(); err != nil {
				return fmt.Errorf("failed to write batch to cache: %w", err)
			}
			// 💡 注意：MSet 不支持在同一条命令设置过期时间，需要后续配合 Expire 处理或改用 Pipeline
			for key := range cacheItems {
				redis.Expire(ctx, key, opts.Expiration)
			}
		}
	}

	return nil
}

// WritedownWithPipeline 修复了 Pipeline 的调用错误
func (sm *ServiceManager[T]) WritedownWithPipeline(
	ctx context.Context,
	data []T,
	buildKeyFunc func(*T) string,
	opts *WritedownQueryOptions,
) error {
	if len(data) == 0 {
		return nil
	}

	if opts == nil {
		opts = &WritedownQueryOptions{Expiration: 1 * time.Hour, BatchSize: 1000, Overwrite: true}
	}

	rdb := GetRedis()

	for i := 0; i < len(data); i += opts.BatchSize {
		end := i + opts.BatchSize
		if end > len(data) {
			end = len(data)
		}

		pipe := rdb.Pipeline()

		for j := i; j < end; j++ {
			item := &data[j]
			key := buildKeyFunc(item)

			// ★★★ 核心修复：先 marshal
			valueBytes, err := json.Marshal(item)
			if err != nil {
				return fmt.Errorf("failed to marshal item for key %s: %w", key, err)
			}

			pipe.Set(ctx, key, valueBytes, opts.Expiration)
		}

		if _, err := pipe.Exec(ctx); err != nil {
			return fmt.Errorf("failed to execute pipeline: %w", err)
		}
	}

	return nil
}

// WritedownIncremental 修复了 Get 和 Set 的返回值错误
func (sm *ServiceManager[T]) WritedownIncremental(
	ctx context.Context,
	data []T,
	buildKeyFunc func(*T) string,
	compareFunc func(*T, *T) bool,
	opts *WritedownQueryOptions,
) error {
	if len(data) == 0 {
		return nil
	}

	redis := GetRedis()

	for i := range data {
		item := &data[i]
		key := buildKeyFunc(item)

		var cachedItem T
		// 修复 4: Get 只有两个参数，结果需要通过 .Scan() 注入结构体
		err := redis.Get(ctx, key).Scan(&cachedItem)

		if err == nil && compareFunc != nil && !compareFunc(item, &cachedItem) {
			continue
		}

		// 修复 5: Set 返回的是 *StatusCmd，需要调用 .Err() 转换为 error 接口
		if err := redis.Set(ctx, key, item, opts.Expiration).Err(); err != nil {
			return fmt.Errorf("failed to write cache for key %s: %w", key, err)
		}
	}
	return nil
}

// --- 辅助方法保持不变 ---
func (sm *ServiceManager[T]) WritedownQueryFromDB(ctx context.Context, queryFunc func(*gorm.DB) *gorm.DB, buildKeyFunc func(*T) string, opts *WritedownQueryOptions) error {
	result, err := sm.GetQueryWithoutTransaction(ctx, queryFunc, nil)
	if err != nil || len(result.Data) == 0 {
		return err
	}
	return sm.WritedownQuery(ctx, result.Data, buildKeyFunc, opts)
}

func (sm *ServiceManager[T]) WritedownQueryByIDs(ctx context.Context, ids []interface{}, buildKeyFunc func(*T) string, opts *WritedownQueryOptions) error {
	return sm.WritedownQueryFromDB(ctx, func(db *gorm.DB) *gorm.DB { return db.Where("id IN ?", ids) }, buildKeyFunc, opts)
}

func (sm *ServiceManager[T]) WritedownAllToCache(ctx context.Context, buildKeyFunc func(*T) string, opts *WritedownQueryOptions) error {
	return sm.WritedownQueryFromDB(ctx, nil, buildKeyFunc, opts)
}

func (sm *ServiceManager[T]) WarmupCache(ctx context.Context, queryFunc func(*gorm.DB) *gorm.DB, buildKeyFunc func(*T) string, expiration time.Duration) error {
	result, err := sm.GetQueryWithoutTransaction(ctx, queryFunc, &QueryOptions{
		OrderBy: "id", Order: "DESC", Page: 1, PageSize: 1000,
	})
	if err != nil || len(result.Data) == 0 {
		return err
	}
	return sm.WritedownQuery(ctx, result.Data, buildKeyFunc, &WritedownQueryOptions{Expiration: expiration, BatchSize: 100, Overwrite: true})
}
