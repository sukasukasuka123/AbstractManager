package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9" // 确保导入了正确的 redis 包
	"gorm.io/gorm"
)

// WritedownSingleOptions 单个写入缓存配置选项
type WritedownSingleOptions struct {
	Expiration time.Duration
	Overwrite  bool
	NX         bool
	XX         bool
}

// WritedownSingle 将单个数据写入缓存
func (sm *ServiceManager[T]) WritedownSingle(
	ctx context.Context,
	key string,
	data *T,
	opts *WritedownSingleOptions,
) error {
	if opts == nil {
		opts = &WritedownSingleOptions{Expiration: 1 * time.Hour, Overwrite: true}
	}

	rdb := GetRedis() // 🛠️ 修复：变量名改为 rdb，避免遮蔽 redis 包名

	if !opts.Overwrite && !opts.NX {
		if rdb.Exists(ctx, key).Val() > 0 {
			return fmt.Errorf("key already exists: %s", key)
		}
	}

	var err error
	if opts.NX {
		err = rdb.SetNX(ctx, key, data, opts.Expiration).Err()
	} else if opts.XX {
		err = rdb.SetXX(ctx, key, data, opts.Expiration).Err()
	} else {
		err = rdb.Set(ctx, key, data, opts.Expiration).Err()
	}

	if err != nil {
		return fmt.Errorf("failed to write cache: %w", err)
	}
	return nil
}

// WritedownSingleWithLock 修正了 redis.Nil 的访问方式
func (sm *ServiceManager[T]) WritedownSingleWithLock(
	ctx context.Context,
	key string,
	queryFunc func(*gorm.DB) *gorm.DB,
	expiration time.Duration,
	lockTimeout time.Duration,
) (*T, error) {
	rdb := GetRedis()

	var result T
	// 🛠️ 修复：使用 redis.Nil (来自包) 而不是 rdb.Nil
	err := rdb.Get(ctx, key).Scan(&result)
	if err == nil {
		return &result, nil
	}

	lockKey := fmt.Sprintf("lock:%s", key)
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())

	locked, _ := rdb.SetNX(ctx, lockKey, lockValue, lockTimeout).Result()
	if !locked {
		time.Sleep(50 * time.Millisecond)
		if err := rdb.Get(ctx, key).Scan(&result); err == nil {
			return &result, nil
		}
		return nil, fmt.Errorf("failed to acquire lock and cache miss for %s", key)
	}

	defer rdb.Del(ctx, lockKey)

	data, err := sm.GetSingle(ctx, queryFunc, nil)
	if err != nil {
		return nil, err
	}

	if err := sm.WritedownSingle(ctx, key, data, &WritedownSingleOptions{Expiration: expiration, Overwrite: true}); err != nil {
		return nil, err
	}

	return data, nil
}

// WritedownSingleWithVersion 修正了 redis.Tx 和 redis.Pipeliner 类型报错
func (sm *ServiceManager[T]) WritedownSingleWithVersion(
	ctx context.Context,
	key string,
	data *T,
	version int64,
	expiration time.Duration,
) error {
	rdb := GetRedis()
	versionKey := key + ":version"

	// 🛠️ 修复：Watch 的回调函数中，redis.Tx 此时指向的是包里的类型
	err := rdb.Watch(ctx, func(tx *redis.Tx) error {
		currentVersion, err := tx.Get(ctx, versionKey).Int64()
		// 🛠️ 修复：使用 redis.Nil
		if err != nil && err != redis.Nil {
			return err
		}

		if err != redis.Nil && currentVersion >= version {
			return fmt.Errorf("version outdated: current %d, provided %d", currentVersion, version)
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, key, data, expiration)
			pipe.Set(ctx, versionKey, version, expiration)
			return nil
		})
		return err
	}, key, versionKey)

	return err
}

// WritedownSingleAsync 异步写入缓存（不阻塞主流程）
func (sm *ServiceManager[T]) WritedownSingleAsync(
	ctx context.Context,
	key string,
	data *T,
	expiration time.Duration,
) {
	// 💡 注意：这里必须脱离原 ctx，避免因主请求结束导致异步操作被 Cancel
	go func() {
		asyncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := sm.WritedownSingle(asyncCtx, key, data, &WritedownSingleOptions{Expiration: expiration}); err != nil {
			// 这种错误应记录日志，但不应抛给用户
			fmt.Printf("[AsyncCache] Failed for key %s: %v\n", key, err)
		}
	}()
}

// --- 便捷封装方法 ---

func (sm *ServiceManager[T]) WritedownSingleByID(ctx context.Context, id interface{}, opts *WritedownSingleOptions) error {
	key := sm.buildCacheKey(id)
	data, err := sm.GetSingle(ctx, func(db *gorm.DB) *gorm.DB { return db.Where("id = ?", id) }, nil)
	if err != nil {
		return err
	}
	return sm.WritedownSingle(ctx, key, data, opts)
}

func (sm *ServiceManager[T]) RefreshSingleCacheFromDB(ctx context.Context, key string, queryFunc func(*gorm.DB) *gorm.DB, expiration time.Duration) error {
	data, err := sm.GetSingle(ctx, queryFunc, nil)
	if err != nil {
		return err
	}
	return sm.WritedownSingle(ctx, key, data, &WritedownSingleOptions{Expiration: expiration, Overwrite: true})
}
