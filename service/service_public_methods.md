# service 模块 — 对外公有方法汇总

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

- 若需我把这些内容直接合并回 `service/service_readme.md`，或导出为 CSV/表格/文档，请说明所需格式。
