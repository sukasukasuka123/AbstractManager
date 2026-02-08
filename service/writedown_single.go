package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// WritedownSingleOptions 单个写入缓存配置选项
type WritedownSingleOptions struct {
	Expiration time.Duration
	Overwrite  bool
	NX         bool
	XX         bool
}

// ----------------- 核心写缓存方法 -----------------

// marshalForRedis 统一处理序列化
func marshalForRedis[T any](data *T) ([]byte, error) {
	if data == nil {
		return nil, fmt.Errorf("cannot marshal nil data")
	}
	// 使用 JSON 序列化，避免 BinaryMarshaler 错误
	return json.Marshal(data)
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

	rdb := GetRedis()

	valueBytes, err := marshalForRedis(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data for key %s: %w", key, err)
	}

	var cmdErr error
	if opts.NX {
		cmdErr = rdb.SetNX(ctx, key, valueBytes, opts.Expiration).Err()
	} else if opts.XX {
		cmdErr = rdb.SetXX(ctx, key, valueBytes, opts.Expiration).Err()
	} else {
		cmdErr = rdb.Set(ctx, key, valueBytes, opts.Expiration).Err()
	}

	if cmdErr != nil {
		return fmt.Errorf("failed to write cache for key %s: %w", key, cmdErr)
	}
	return nil
}

// ----------------- 带锁写缓存 -----------------

func (sm *ServiceManager[T]) WritedownSingleWithLock(
	ctx context.Context,
	key string,
	queryFunc func(*gorm.DB) *gorm.DB,
	expiration time.Duration,
	lockTimeout time.Duration,
) (*T, error) {
	rdb := GetRedis()
	var result T

	// 尝试直接读取缓存
	val, err := rdb.Get(ctx, key).Bytes()
	if err == nil {
		if err := json.Unmarshal(val, &result); err == nil {
			return &result, nil
		}
	}

	lockKey := fmt.Sprintf("lock:%s", key)
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())
	locked, _ := rdb.SetNX(ctx, lockKey, lockValue, lockTimeout).Result()
	if !locked {
		time.Sleep(50 * time.Millisecond)
		val, err := rdb.Get(ctx, key).Bytes()
		if err == nil {
			if err := json.Unmarshal(val, &result); err == nil {
				return &result, nil
			}
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

// ----------------- 带版本控制写缓存 -----------------

func (sm *ServiceManager[T]) WritedownSingleWithVersion(
	ctx context.Context,
	key string,
	data *T,
	version int64,
	expiration time.Duration,
) error {
	rdb := GetRedis()
	versionKey := key + ":version"

	valueBytes, err := marshalForRedis(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data for key %s: %w", key, err)
	}

	// 使用 Watch 保证原子性
	return rdb.Watch(ctx, func(tx *redis.Tx) error {
		currentVersion, err := tx.Get(ctx, versionKey).Int64()
		if err != nil && err != redis.Nil {
			return err
		}
		if err != redis.Nil && currentVersion >= version {
			return fmt.Errorf("version outdated: current %d, provided %d", currentVersion, version)
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, key, valueBytes, expiration)
			pipe.Set(ctx, versionKey, version, expiration)
			return nil
		})
		return err
	}, key, versionKey)
}

// ----------------- 异步写缓存 -----------------

func (sm *ServiceManager[T]) WritedownSingleAsync(
	ctx context.Context,
	key string,
	data *T,
	expiration time.Duration,
) {
	go func() {
		asyncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := sm.WritedownSingle(asyncCtx, key, data, &WritedownSingleOptions{Expiration: expiration}); err != nil {
			fmt.Printf("[AsyncCache] Failed for key %s: %v\n", key, err)
		}
	}()
}

// ----------------- 便捷方法 -----------------

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
