# service 模块 使用说明（之后要跟router模块集成到abstractmanager里面去）

一个功能强大的 Go 服务管理器，提供数据库操作、缓存管理、事务处理等完整功能。

## 功能特性

### 核心模块

- create是用于创建数据表的操作
- get_query是用于处理条件检索、查询非单个状态数据或者有分页需求的查询操作
- get_single是用于处理单个状态数据的查询操作
- lookup_query是处理从缓存中查询系列数据的操作
- lookup_single是处理从缓存中查询单个数据的操作
- set_query是数据库处理批量设置数据的操作,批量设置包括新增和修改
- set_single是数据库处理单个数据的设置操作,单个数据的设置包括新增和修改
- writedown_query是用于处理批量状态数据的设置到缓存的操作，设置到缓存包括新增和修改（但修改很少所以命名writedown）
- writedown_single是用于处理单个状态数据的设置到缓存的操作，设置到缓存包括新增和修改（但修改很少所以命名writedown）
- service_model是设置service管理器的操作

### 核心特性

- **泛型支持** - 使用 Go 1.18+ 泛型，类型安全
- **GORM 集成** - 强大的 ORM 功能，支持复杂查询
- **Redis 缓存** - 完整的缓存管理，支持分布式锁
- **事务管理** - 自动事务处理，支持不同隔离级别
- **连接池** - 数据库和 Redis 连接池管理
- **Lambda 查询** - 灵活的查询条件构建
- **批量操作** - 高效的批量插入、更新、删除
- **缓存策略** - 多种缓存策略（回源、预热、失效等）
- **分布式锁** - 防止缓存击穿
- **软删除** - 支持软删除操作

## 环境

```bash
go get -u gorm.io/gorm
go get -u gorm.io/driver/mysql
go get -u github.com/redis/go-redis/v9
go get -u github.com/joho/godotenv
```

## 配置

创建 `.env` 文件：

```env
# MySQL 配置
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=your_database

# Redis 配置
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_PASSWORD=
```

## 快速开始

### 1. 初始化

```go
import (
    "context"
    "your-module/service"
)

// 初始化数据库和 Redis
dbManager, _ := service.InitDB()
defer dbManager.Close()

redisManager, _ := service.InitRedis()
defer redisManager.Close()
```

### 2. 定义模型

```go
type User struct {
    ID        uint      `gorm:"primaryKey"`
    Username  string    `gorm:"uniqueIndex;size:100"`
    Email     string    `gorm:"uniqueIndex;size:255"`
    Age       int       `gorm:"index"`
    Status    string    `gorm:"size:20"`
    CreatedAt time.Time `gorm:"autoCreateTime"`
    UpdatedAt time.Time `gorm:"autoUpdateTime"`
}
```

### 3. 创建 ServiceManager

```go
userService := service.NewServiceManager[User](User{})
ctx := context.Background()
```

### 4. 创建表

```go
err := userService.Create(ctx, &service.CreateOptions{
    IfNotExists: true,
})
```

### 5. 基本操作

#### 插入数据

```go
// 单条插入
user := &User{Username: "alice", Email: "alice@example.com", Age: 25}
userService.Insert(ctx, user)

// 批量插入
users := []User{
    {Username: "bob", Email: "bob@example.com", Age: 30},
    {Username: "charlie", Email: "charlie@example.com", Age: 28},
}
userService.SetQuery(ctx, users, &service.SetQueryOptions{
    BatchSize:        100,
    OnConflictUpdate: true,
})
```

#### 查询数据

```go
// 查询单条
user, err := userService.GetSingle(ctx, func(db *gorm.DB) *gorm.DB {
    return db.Where("username = ?", "alice")
}, nil)

// 条件查询（分页）
result, err := userService.GetQuery(ctx, func(db *gorm.DB) *gorm.DB {
    return db.Where("age > ?", 25).Where("status = ?", "active")
}, &service.QueryOptions{
    Page:     1,
    PageSize: 10,
    OrderBy:  "created_at",
    Order:    "DESC",
})
```

#### 更新数据

```go
// 根据 ID 更新
userService.UpdateByID(ctx, 1, map[string]interface{}{
    "age":    26,
    "status": "premium",
})

// 批量更新
affected, err := userService.BatchUpdate(ctx, map[string]interface{}{
    "status": "verified",
}, func(db *gorm.DB) *gorm.DB {
    return db.Where("age > ?", 30)
})
```

#### 删除数据

```go
// 软删除
userService.SoftDeleteByID(ctx, 1)

// 硬删除
userService.DeleteByID(ctx, 1)

// 批量删除
affected, err := userService.BatchDelete(ctx, func(db *gorm.DB) *gorm.DB {
    return db.Where("status = ?", "inactive")
})
```

### 6. 缓存操作

#### 写入缓存

```go
// 单条写入
userService.WritedownSingleByID(ctx, 1, &service.WritedownSingleOptions{
    Expiration: 10 * time.Minute,
    Overwrite:  true,
})

// 批量写入
users, _ := userService.GetQuery(ctx, func(db *gorm.DB) *gorm.DB {
    return db.Where("status = ?", "active")
}, nil)

userService.WritedownQuery(ctx, users.Data, func(u *User) string {
    return fmt.Sprintf("user:%d", u.ID)
}, &service.WritedownQueryOptions{
    Expiration: 10 * time.Minute,
    BatchSize:  100,
})
```

#### 读取缓存

```go
// 从缓存读取（带回源）
user, err := userService.LookupSingleByID(ctx, 1, &service.LookupSingleOptions{
    CacheExpire:  10 * time.Minute,
    FallbackToDB: true,
})

// 使用分布式锁读取（防止缓存击穿）
key := "user:1"
user, err := userService.WritedownSingleWithLock(ctx, key, func(db *gorm.DB) *gorm.DB {
    return db.Where("id = ?", 1)
}, 10*time.Minute, 5*time.Second)
```

#### 缓存管理

```go
// 缓存预热
userService.WarmupCache(ctx, func(db *gorm.DB) *gorm.DB {
    return db.Where("status = ?", "active").Order("created_at DESC")
}, func(u *User) string {
    return fmt.Sprintf("user:%d", u.ID)
}, 30*time.Minute)

// 使缓存失效
userService.InvalidateSingleCacheByID(ctx, 1)

// 按模式使缓存失效
userService.InvalidateCacheByPattern(ctx, "user:*")

// 刷新缓存
userService.RefreshSingleCacheByID(ctx, 1, 10*time.Minute)
```

## 高级功能

### 事务隔离级别

- **读操作**: 使用较低的事务隔离级别（READ COMMITTED）
- **写操作**: 使用较高的事务隔离级别（REPEATABLE READ）

```go
// 带事务的查询（READ COMMITTED）
result, err := userService.GetQuery(ctx, queryFunc, opts)

// 不带事务的查询（高并发场景）
result, err := userService.GetQueryWithoutTransaction(ctx, queryFunc, opts)
```

### Lambda 查询构建

```go
// 复杂查询条件
result, err := userService.GetQuery(ctx, func(db *gorm.DB) *gorm.DB {
    return db.Where("age BETWEEN ? AND ?", 25, 35).
        Where("status IN ?", []string{"active", "premium"}).
        Or("vip = ?", true).
        Joins("LEFT JOIN orders ON orders.user_id = users.id").
        Group("users.id").
        Having("COUNT(orders.id) > ?", 5)
}, &service.QueryOptions{
    Page:     1,
    PageSize: 20,
    OrderBy:  "created_at",
    Order:    "DESC",
    Preload:  []string{"Orders", "Profile"},
})
```

### 批量操作优化

```go
// Upsert（插入或更新）
userService.BatchUpsert(ctx, users, 
    []string{"email"}, // 冲突判断列
    []string{"username", "age", "updated_at"}, // 更新列
    100, // 批次大小
)

// 增量更新
affected, err := userService.BatchIncrement(ctx, "login_count", 1, func(db *gorm.DB) *gorm.DB {
    return db.Where("last_login > ?", time.Now().Add(-24*time.Hour))
})
```

### 缓存策略

```go
// 1. 增量缓存写入（只写入变化的数据）
userService.WritedownIncremental(ctx, users, buildKeyFunc, compareFunc, opts)

// 2. 带过期时间差异化的缓存
userService.WritedownWithExpiry(ctx, users, buildKeyFunc, func(u *User) time.Duration {
    if u.Status == "premium" {
        return 1 * time.Hour // VIP 用户缓存时间长
    }
    return 10 * time.Minute // 普通用户缓存时间短
})

// 3. 异步写入缓存（不阻塞主流程）
userService.WritedownSingleAsync(ctx, key, user, 10*time.Minute)

// 4. 带版本号的缓存（乐观锁）
userService.WritedownSingleWithVersion(ctx, key, user, version, expiration)
```

### 分布式锁

```go
// 防止缓存击穿
user, err := userService.WritedownSingleWithLock(
    ctx, 
    cacheKey, 
    queryFunc, 
    10*time.Minute,  // 缓存过期时间
    5*time.Second,   // 锁超时时间
)
```

## 性能优化建议

### 1. 数据库连接池配置

根据你的 2C4G 服务器配置，推荐设置：

```go
sqlDB.SetMaxOpenConns(100)  // 最大打开连接数
sqlDB.SetMaxIdleConns(10)   // 最大空闲连接数
sqlDB.SetConnMaxLifetime(time.Hour)
```

### 2. Redis 连接池配置

```go
redis.NewClient(&redis.Options{
    PoolSize:     50,  // 连接池大小
    MinIdleConns: 10,  // 最小空闲连接
    MaxRetries:   3,   // 最大重试次数
})
```

### 3. 批量操作优化

```go
// 使用合适的批次大小
SetQueryOptions{
    BatchSize: 100,  // 推荐 100-500
}

// 高并发场景使用无事务查询
GetQueryWithoutTransaction(ctx, queryFunc, opts)
```

### 4. 缓存策略

- 热点数据使用缓存预热
- 高并发读使用分布式锁防止击穿
- 使用 Pipeline 批量写入缓存
- 设置合理的过期时间

## 注意事项

1. **事务管理**: 写操作会自动使用事务，确保数据一致性
2. **缓存键设计**: 合理设计缓存键结构，便于批量操作
3. **连接池**: 根据服务器配置调整连接池参数
4. **错误处理**: 始终检查返回的错误
5. **Context**: 使用 context 进行超时和取消控制

## service 模块 — 对外公有方法汇总

以下按文件分组列出 `service/` 目录中对外（exported）的公有方法：

- **文件**: [service/service_model.go](service/service_model.go) : 方法: `NewServiceManager`
- **文件**: [service/get_single.go](service/get_single.go) : 方法: `GetSingle`, `GetSingleByID`, `GetSingleOrCreate`, `GetSingleWithLock`, `GetFirst`, `GetLast`
- **文件**: [service/get_query.go](service/get_query.go) : 方法: `GetQuery`, `GetQueryWithoutTransaction`, `CountQuery`, `ExistsQuery`
- **文件**: [service/set_single.go](service/set_single.go) : 方法: `SetSingle`, `Update`, `Save`, `Upsert`, `Delete`, `Increment`, `Decrement`, `Insert`, `UpdateByID`, `DeleteByID`, `SoftDelete`, `SoftDeleteByID`, `IncrementByID`, `DecrementByID`
- **文件**: [service/set_query.go](service/set_query.go) : 方法: `SetQuery`, `BatchUpdate`, `BatchUpsert`, `BatchDelete`, `BatchInsert`, `BatchSoftDelete`, `BatchIncrement`, `BatchDecrement`
- **文件**: [service/lookup_single.go](service/lookup_single.go) : 方法: `LookupSingle`, `LookupSingleWithFallback`, `InvalidateSingleCache`, `ExistsInCache`, `ExtendCacheTTL`, `LookupSingleByID`, `InvalidateSingleCacheByID`, `GetCacheTTL`
- **文件**: [service/lookup_query.go](service/lookup_query.go) : 方法: `LookupQuery`, `LookupQueryByPattern`, `LookupQueryWithRefresh`, `RefreshCache`, `InvalidateCache`, `InvalidateCacheByPattern`
- **文件**: [service/create.go](service/create.go) : 方法: `Create`, `CreateWithIndexes`, `DropTable`, `HasTable`
- **文件**: [service/writedown_single.go](service/writedown_single.go) : 方法: `WritedownSingle`, `WritedownSingleWithLock`, `WritedownSingleWithVersion`, `WritedownSingleAsync`, `WritedownSingleByID`, `RefreshSingleCacheFromDB`
- **文件**: [service/writedown_query.go](service/writedown_query.go) : 方法: `WritedownQuery`, `WritedownWithPipeline`, `WritedownIncremental`, `WritedownQueryFromDB`, `WritedownQueryByIDs`, `WritedownAllToCache`, `WarmupCache`
- **文件**: [service/sql_pool.go](service/sql_pool.go) : 方法: `InitDB`, `GetDB`, `(DBManager).Close`
- **文件**: [service/cache_pool.go](service/cache_pool.go) : 方法: `InitRedis`, `GetRedis`, `(RedisManager).Close`, `Set`, `Get`, `Delete`, `Exists`, `SetMultiple`, `GetMultiple`

---

说明：

- 仅列出以大写字母开头的导出方法（即对外公有方法）。
