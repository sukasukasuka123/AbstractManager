# router 模块 使用说明（与 service 模块集成）

基于 Gin 框架构建的高性能 HTTP 路由管理器。该模块采用分组设计策略，将 HTTP 请求无缝映射到 `service` 模块的具体方法，支持自动化的 CRUD、缓存操作及系统监控。

## 功能特性

### 核心设计

* **Gin 框架集成** - 基于 Go 社区最流行的 Web 框架，高性能且生态丰富。
* **分组路由架构** - 将路由分为查询、写入、缓存、表管理、监控、管理六大类，职责边界清晰。
* **安全性隔离** - 支持将高风险接口（如表结构修改、全量缓存清洗）绑定到独立端口或特定中间件，防止敏感接口暴露。
* **显式注册** - 遵循 Go 语言 "显式优于隐式" 的哲学，不依赖注解，通过代码显式注册所需的路由组。
* **Service 深度绑定** - 路由层与 `service` 模块方法一一对应，极大减少胶水代码。
* **RESTful 规范** - 接口设计遵循 RESTful 风格，语义清晰。

## 环境准备

```bash
go get -u github.com/gin-gonic/gin

```

## 快速开始

### 1. 初始化与注册

通常在 `main.go` 或 `bootstrap` 流程中进行初始化。

```go
import (
    "github.com/gin-gonic/gin"
    "your-module/service"
    "your-module/router"
)

func main() {
    // 1. 初始化 Service (如 User Service)
    userService := service.NewServiceManager[User](User{})
    
    // 2. 初始化 Gin 引擎
    r := gin.Default()
    
    // 3. 创建 Router 管理器
    // router.NewManager 需要传入具体的 service 实例
    userRouter := router.NewManager[User](userService)

    // 4. 注册路由组 (按需注册)
    v1 := r.Group("/api/v1")
    {
        // 注册用户资源的查询和写入接口
        userRouter.RegisterQueryRoutes(v1, "users") 
        userRouter.RegisterWriteRoutes(v1, "users")
        
        // 注册缓存操作接口
        userRouter.RegisterCacheRoutes(v1, "cache/users")
    }

    // 注册监控接口 (通常在根路径或独立端口)
    router.RegisterMonitorRoutes(r)

    // 5. 启动服务
    r.Run(":8080")
}

```

## 路由组详解

### 1. 查询路由组 (Query Routes)

主要用于数据的读取操作，支持单条查询、分页检索及统计。

**基础路径**: `/api/v1/{resource}`

| HTTP 方法 | 路由路径 | 对应 Service 方法 | 功能说明 |
| --- | --- | --- | --- |
| `GET` | `/:id` | `GetSingleByID` | 根据主键 ID 查询单条详情 |
| `GET` | `/single` | `GetSingle` | 根据 QueryString 条件查询单条 |
| `GET` | `/list` | `GetQuery` | 通用条件查询（支持分页、排序、过滤） |
| `GET` | `/list/notx` | `GetQueryWithoutTransaction` | 无事务查询（适用于极高并发读取场景） |
| `GET` | `/count` | `CountQuery` | 根据条件统计记录数 |
| `GET` | `/exists` | `ExistsQuery` | 检查符合条件的记录是否存在 |
| `GET` | `/first` | `GetFirst` | 获取满足条件的第一条记录 |
| `GET` | `/last` | `GetLast` | 获取满足条件的最后一条记录 |

**请求示例**:

```http
GET /api/v1/users/list?page=1&pageSize=10&age_gt=25&status=active&orderBy=created_at&order=DESC
GET /api/v1/users/count?status=active
GET /api/v1/users/exists?username=alice

```

### 2. 写入路由组 (Write Routes)

用于数据的增删改操作，包含单条处理和批量处理。

**基础路径**: `/api/v1/{resource}`

| HTTP 方法 | 路由路径 | 对应 Service 方法 | 功能说明 |
| --- | --- | --- | --- |
| **单条操作** |  |  |  |
| `POST` | `/` | `Insert` | 插入单条记录 |
| `POST` | `/create` | `SetSingle` | 设置单条（智能判断新增或更新） |
| `PUT` | `/:id` | `UpdateByID` | 根据 ID 全量/部分更新 |
| `PATCH` | `/update` | `Update` | 根据条件更新 |
| `POST` | `/save` | `Save` | 保存记录 (GORM Save 语义) |
| `POST` | `/upsert` | `Upsert` | 冲突时更新 (On Duplicate Key Update) |
| `DELETE` | `/:id` | `DeleteByID` | 根据 ID **硬删除** |
| `DELETE` | `/delete` | `Delete` | 根据条件 **硬删除** |
| `DELETE` | `/:id/soft` | `SoftDeleteByID` | 根据 ID **软删除** |
| `POST` | `/soft-delete` | `SoftDelete` | 根据条件 **软删除** |
| `PATCH` | `/:id/increment/:field` | `IncrementByID` | 指定字段数值增加 |
| `PATCH` | `/:id/decrement/:field` | `DecrementByID` | 指定字段数值减少 |
| **批量操作** |  |  |  |
| `POST` | `/batch` | `SetQuery` | 批量设置（新增/更新） |
| `PATCH` | `/batch/update` | `BatchUpdate` | 批量更新 |
| `POST` | `/batch/upsert` | `BatchUpsert` | 批量插入或更新 |
| `DELETE` | `/batch/delete` | `BatchDelete` | 批量 **硬删除** |
| `POST` | `/batch/insert` | `BatchInsert` | 批量插入（纯新增） |
| `DELETE` | `/batch/soft-delete` | `BatchSoftDelete` | 批量 **软删除** |
| `PATCH` | `/batch/increment/:field` | `BatchIncrement` | 批量字段增量更新 |
| `PATCH` | `/batch/decrement/:field` | `BatchDecrement` | 批量字段减量更新 |

**请求示例**:

```json
// POST /api/v1/users/batch
[
  {"username": "bob", "email": "bob@example.com"},
  {"username": "charlie", "email": "charlie@example.com"}
]

```

### 3. 缓存管理路由组 (Cache Routes)

直接对 Redis 缓存进行精细化控制，支持回源策略和分布式锁。

**基础路径**: `/api/v1/cache/{resource}`

| HTTP 方法 | 路由路径 | 对应 Service 方法 | 功能说明 |
| --- | --- | --- | --- |
| **读取** |  |  |  |
| `GET` | `/:id` | `LookupSingleByID` | 根据 ID 查缓存（支持自动回源 DB） |
| `GET` | `/single` | `LookupSingle` | 根据 Key 查单条缓存 |
| `GET` | `/single/fallback` | `LookupSingleWithFallback` | 查缓存带自定义回源逻辑 |
| `GET` | `/list` | `LookupQuery` | 批量查缓存 |
| `GET` | `/list/pattern` | `LookupQueryByPattern` | 按 Key 模式匹配查询 |
| `GET` | `/list/refresh` | `LookupQueryWithRefresh` | 查询并强制刷新缓存 |
| **写入** |  |  |  |
| `POST` | `/:id/write` | `WritedownSingleByID` | 将指定 ID 数据写入缓存 |
| `POST` | `/single/write` | `WritedownSingle` | 写入单条缓存 |
| `POST` | `/single/write-lock` | `WritedownSingleWithLock` | **分布式锁**写入（防击穿） |
| `POST` | `/single/write-version` | `WritedownSingleWithVersion` | 带版本号写入（乐观锁） |
| `POST` | `/single/write-async` | `WritedownSingleAsync` | 异步写入（不阻塞响应） |
| `POST` | `/list/write` | `WritedownQuery` | 批量写入缓存 |
| `POST` | `/list/write-pipeline` | `WritedownWithPipeline` | 使用 Pipeline 高效批量写入 |
| `POST` | `/list/write-incremental` | `WritedownIncremental` | 增量写入（只写变更数据） |
| `POST` | `/list/write-from-db` | `WritedownQueryFromDB` | 从 DB 拉取并批量写入缓存 |
| **管理** |  |  |  |
| `POST` | `/:id/refresh` | `RefreshSingleCacheFromDB` | 强制从 DB 刷新某 ID 缓存 |
| `POST` | `/refresh` | `RefreshCache` | 刷新通用缓存 |
| `DELETE` | `/:id` | `InvalidateSingleCacheByID` | 清除指定 ID 缓存 |
| `DELETE` | `/single` | `InvalidateSingleCache` | 清除单条缓存 |
| `DELETE` | `/invalidate` | `InvalidateCache` | 批量清除缓存 |
| `DELETE` | `/invalidate/pattern` | `InvalidateCacheByPattern` | 按模式批量清除（如 `user:*`） |
| `GET` | `/:id/exists` | `ExistsInCache` | 检查 Key 是否存在 |
| `PATCH` | `/:id/extend-ttl` | `ExtendCacheTTL` | 延长缓存过期时间 |

**请求示例**:

```http
DELETE /api/v1/cache/users/invalidate/pattern?pattern=user:*
POST /api/v1/cache/users/single/write-lock (Body: key, data, ttl)

```

### 4. 表管理路由组 (Table Routes)

用于数据库表结构的动态管理。**警告：生产环境慎用，建议通过权限中间件保护。**

**基础路径**: `/api/v1/table/{resource}`

| HTTP 方法 | 路由路径 | 对应 Service 方法 | 功能说明 |
| --- | --- | --- | --- |
| `POST` | `/create` | `Create` | 创建数据表 |
| `POST` | `/create-with-indexes` | `CreateWithIndexes` | 创建表并同时创建索引 |
| `DELETE` | `/drop` | `DropTable` | **删除数据表** (高危操作) |
| `GET` | `/exists` | `HasTable` | 检查表是否存在 |

### 5. 监控路由组 (Monitor Routes)

提供系统健康状态检查和指标暴露，通常对接 Kubernetes 探针或 Prometheus。

**基础路径**: `/monitor`

| HTTP 方法 | 路由路径 | 对应 Service 方法 | 功能说明 |
| --- | --- | --- | --- |
| `GET` | `/health` | - | 综合健康检查（DB + Redis） |
| `GET` | `/health/db` | `GetDB` | 仅检查数据库连接状态 |
| `GET` | `/health/redis` | `GetRedis` | 仅检查 Redis 连接状态 |
| `GET` | `/cache/exists/:key` | `ExistsInCache` | 探针：检查特定缓存键 |
| `GET` | `/metrics` | - | Prometheus 指标导出（预留） |

**响应示例**:

```json
{
  "status": "healthy",
  "components": {
      "database": "up",
      "redis": "up"
  },
  "timestamp": 1738150288
}

```

### 6. 管理路由组 (Admin Routes)

执行耗时较长、影响范围广的运维操作。建议绑定到独立的内网端口。

**基础路径**: `/admin/{resource}`

| HTTP 方法 | 路由路径 | 对应 Service 方法 | 功能说明 |
| --- | --- | --- | --- |
| `POST` | `/cache/warmup` | `WarmupCache` | **缓存预热**：按条件加载数据到缓存 |
| `POST` | `/cache/write-all` | `WritedownAllToCache` | 全量数据刷入缓存 |
| `POST` | `/batch/upsert` | `BatchUpsert` | 大批量数据导入/更新 |
| `DELETE` | `/batch/soft-delete` | `BatchSoftDelete` | 大批量数据归档/软删除 |
| `POST` | `/batch/import` | `SetQuery` | 数据批量导入接口 |
| `DELETE` | `/cache/clear` | `InvalidateCacheByPattern` | 紧急清理缓存 |

**请求示例**:

```json
// POST /admin/users/cache/warmup
{
  "conditions": { "status": "active" },
  "expiration": "30m",
  "batch_size": 1000
}

```

# Get Router Group 使用文档

## 概述

`get_router_group` 是一个通用的查询路由封装模块，基于 Gin 框架和泛型 ServiceManager 实现，提供了完整的 RESTful 查询 API。

## 快速开始

### 1. 基本使用示例

```go
package main

import (
    "github.com/gin-gonic/gin"
    "your_project/http_router"
    "your_project/service"
    "your_project/models"
)

func main() {
    r := gin.Default()
    
    // 创建 ServiceManager
    userService := service.NewServiceManager[models.User](
        service.WithTableName("users"),
        service.WithSchema("public"),
    )
    
    // 创建查询路由组
    apiGroup := r.Group("/api/v1")
    userQueryRouter := http_router.NewQueryRouterGroup(
        apiGroup,
        userService,
        "/users",
    )
    
    // 注册标准路由
    userQueryRouter.RegisterRoutes()
    
    r.Run(":8080")
}
```

注册后，将自动创建以下路由：
- `GET /api/v1/users` - 分页查询
- `GET /api/v1/users/:id` - 根据ID查询
- `GET /api/v1/users/count` - 计数
- `GET /api/v1/users/exists` - 存在性检查
- `GET /api/v1/users/first` - 获取第一条记录
- `GET /api/v1/users/last` - 获取最后一条记录

### 2. 自定义查询路由

```go
// 注册自定义查询路由
userQueryRouter.RegisterCustomRoute(
    "GET",
    "/users/active",
    func(db *gorm.DB, c *gin.Context) *gorm.DB {
        // 自定义过滤逻辑
        return db.Where("status = ?", "active").Where("deleted_at IS NULL")
    },
)

// 更复杂的示例：根据角色和时间范围查询用户
userQueryRouter.RegisterCustomRoute(
    "GET",
    "/users/by-role",
    func(db *gorm.DB, c *gin.Context) *gorm.DB {
        role := c.Query("role")
        startDate := c.Query("start_date")
        endDate := c.Query("end_date")
        
        if role != "" {
            db = db.Where("role = ?", role)
        }
        if startDate != "" && endDate != "" {
            db = db.Where("created_at BETWEEN ? AND ?", startDate, endDate)
        }
        
        return db
    },
)
```

## API 接口文档

### 1. 分页查询 (GET /api/v1/{resource})

#### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|--------|------|------|------|------|
| page | int | 否 | 页码（从1开始） | 1 |
| page_size | int | 否 | 每页数量 | 10 |
| order_by | string | 否 | 排序字段 | created_at |
| order | string | 否 | 排序方向（ASC/DESC） | DESC |
| preload | []string | 否 | 预加载关联 | ["Profile", "Orders"] |
| select | []string | 否 | 选择字段 | ["id", "name", "email"] |
| distinct | bool | 否 | 是否去重 | true |
| group | string | 否 | 分组字段 | status |
| filters | string | 否 | JSON格式过滤条件 | 见下方说明 |

#### 请求示例

```bash
# 基本分页查询
GET /api/v1/users?page=1&page_size=10

# 带排序
GET /api/v1/users?page=1&page_size=10&order_by=created_at&order=DESC

# 带预加载
GET /api/v1/users?page=1&page_size=10&preload=Profile&preload=Orders

# 带过滤条件
GET /api/v1/users?page=1&page_size=10&filters=[{"field":"status","operator":"eq","value":"active"}]
```

#### 响应示例

```json
{
    "code": 0,
    "message": "success",
    "data": [
        {
            "id": 1,
            "name": "John Doe",
            "email": "john@example.com",
            "status": "active",
            "created_at": "2024-01-01T00:00:00Z"
        }
    ],
    "total": 100,
    "page": 1,
    "page_size": 10,
    "total_pages": 10
}
```

### 2. 根据ID查询 (GET /api/v1/{resource}/:id)

#### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|--------|------|------|------|------|
| id | string | 是 | 资源ID（路径参数） | 123 |
| preload | []string | 否 | 预加载关联 | ["Profile"] |
| select | []string | 否 | 选择字段 | ["id", "name"] |

#### 请求示例

```bash
GET /api/v1/users/123
GET /api/v1/users/123?preload=Profile&preload=Orders
```

#### 响应示例

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "id": 123,
        "name": "John Doe",
        "email": "john@example.com"
    }
}
```

### 3. 计数查询 (GET /api/v1/{resource}/count)

#### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|--------|------|------|------|------|
| filters | string | 否 | JSON格式过滤条件 | 见过滤器说明 |

#### 请求示例

```bash
GET /api/v1/users/count
GET /api/v1/users/count?filters=[{"field":"status","operator":"eq","value":"active"}]
```

#### 响应示例

```json
{
    "code": 0,
    "message": "success",
    "count": 42
}
```

### 4. 存在性检查 (GET /api/v1/{resource}/exists)

#### 请求参数

同计数查询

#### 响应示例

```json
{
    "code": 0,
    "message": "success",
    "exists": true
}
```

### 5. 获取第一条/最后一条记录

#### 请求示例

```bash
GET /api/v1/users/first
GET /api/v1/users/last
GET /api/v1/users/first?filters=[{"field":"status","operator":"eq","value":"active"}]
```

## 过滤器 (Filters) 详细说明

### 过滤器格式

过滤器使用 JSON 数组格式，每个元素包含：
- `field`: 字段名
- `operator`: 操作符
- `value`: 值

```json
[
    {
        "field": "status",
        "operator": "eq",
        "value": "active"
    },
    {
        "field": "age",
        "operator": "gte",
        "value": 18
    }
]
```

### 支持的操作符对照表

| 操作符 | 别名 | 说明 | 示例 | SQL 等价 |
|--------|------|------|------|----------|
| eq | = | 等于 | `{"field":"status","operator":"eq","value":"active"}` | `status = 'active'` |
| ne | != | 不等于 | `{"field":"status","operator":"ne","value":"deleted"}` | `status != 'deleted'` |
| gt | > | 大于 | `{"field":"age","operator":"gt","value":18}` | `age > 18` |
| gte | >= | 大于等于 | `{"field":"age","operator":"gte","value":18}` | `age >= 18` |
| lt | < | 小于 | `{"field":"age","operator":"lt","value":65}` | `age < 65` |
| lte | <= | 小于等于 | `{"field":"age","operator":"lte","value":65}` | `age <= 65` |
| like | - | 模糊匹配 | `{"field":"name","operator":"like","value":"John"}` | `name LIKE '%John%'` |
| in | - | 在列表中 | `{"field":"status","operator":"in","value":["active","pending"]}` | `status IN ('active','pending')` |
| not_in | - | 不在列表中 | `{"field":"status","operator":"not_in","value":["deleted"]}` | `status NOT IN ('deleted')` |
| is_null | - | 为空 | `{"field":"deleted_at","operator":"is_null"}` | `deleted_at IS NULL` |
| is_not_null | - | 不为空 | `{"field":"email","operator":"is_not_null"}` | `email IS NOT NULL` |
| between | - | 在范围内 | `{"field":"age","operator":"between","value":[18,65]}` | `age BETWEEN 18 AND 65` |

### 过滤器使用示例

#### 1. 单条件过滤

```bash
# 查询状态为 active 的用户
GET /api/v1/users?filters=[{"field":"status","operator":"eq","value":"active"}]
```

#### 2. 多条件过滤（AND 关系）

```bash
# 查询状态为 active 且年龄大于等于 18 的用户
GET /api/v1/users?filters=[
    {"field":"status","operator":"eq","value":"active"},
    {"field":"age","operator":"gte","value":18}
]
```

#### 3. 模糊搜索

```bash
# 搜索名字包含 "John" 的用户
GET /api/v1/users?filters=[{"field":"name","operator":"like","value":"John"}]
```

#### 4. IN 查询

```bash
# 查询状态为 active 或 pending 的用户
GET /api/v1/users?filters=[{"field":"status","operator":"in","value":["active","pending"]}]
```

#### 5. 范围查询

```bash
# 查询年龄在 18-65 之间的用户
GET /api/v1/users?filters=[{"field":"age","operator":"between","value":[18,65]}]
```

#### 6. 空值检查

```bash
# 查询未删除的用户
GET /api/v1/users?filters=[{"field":"deleted_at","operator":"is_null"}]
```

## 前后端参数映射表

### 查询参数映射

| 前端参数 | 后端类型 | 说明 | 前端示例 | 后端处理 |
|----------|----------|------|----------|----------|
| page | int | 页码 | `page: 1` | `opts.Page = 1` |
| page_size | int | 每页数量 | `pageSize: 10` | `opts.PageSize = 10` |
| order_by | string | 排序字段 | `orderBy: 'created_at'` | `opts.OrderBy = "created_at"` |
| order | string | 排序方向 | `order: 'DESC'` | `opts.Order = "DESC"` |
| preload | []string | 预加载 | `preload: ['Profile']` | `opts.Preload = ["Profile"]` |
| select | []string | 选择字段 | `select: ['id','name']` | `opts.Select = ["id","name"]` |
| filters | JSON | 过滤条件 | 见下方 | 转换为 GORM 查询 |

### 前端 JavaScript 示例

#### 使用 Axios

```javascript
import axios from 'axios';

// 1. 基本分页查询
async function getUsers(page = 1, pageSize = 10) {
    const response = await axios.get('/api/v1/users', {
        params: {
            page,
            page_size: pageSize,
            order_by: 'created_at',
            order: 'DESC'
        }
    });
    return response.data;
}

// 2. 带过滤条件的查询
async function getActiveUsers(page = 1) {
    const filters = [
        { field: 'status', operator: 'eq', value: 'active' },
        { field: 'age', operator: 'gte', value: 18 }
    ];
    
    const response = await axios.get('/api/v1/users', {
        params: {
            page,
            page_size: 10,
            filters: JSON.stringify(filters)
        }
    });
    return response.data;
}

// 3. 搜索功能
async function searchUsers(keyword) {
    const filters = [
        { field: 'name', operator: 'like', value: keyword }
    ];
    
    const response = await axios.get('/api/v1/users', {
        params: {
            filters: JSON.stringify(filters)
        }
    });
    return response.data;
}

// 4. 根据ID获取用户
async function getUserById(id) {
    const response = await axios.get(`/api/v1/users/${id}`, {
        params: {
            preload: ['Profile', 'Orders']
        }
    });
    return response.data;
}

// 5. 计数查询
async function countActiveUsers() {
    const filters = [
        { field: 'status', operator: 'eq', value: 'active' }
    ];
    
    const response = await axios.get('/api/v1/users/count', {
        params: {
            filters: JSON.stringify(filters)
        }
    });
    return response.data.count;
}
```

#### 使用 Fetch API

```javascript
// 通用查询函数
async function queryUsers(options = {}) {
    const {
        page = 1,
        pageSize = 10,
        orderBy = 'created_at',
        order = 'DESC',
        filters = [],
        preload = [],
        select = []
    } = options;
    
    const params = new URLSearchParams();
    params.append('page', page);
    params.append('page_size', pageSize);
    params.append('order_by', orderBy);
    params.append('order', order);
    
    if (filters.length > 0) {
        params.append('filters', JSON.stringify(filters));
    }
    
    preload.forEach(p => params.append('preload', p));
    select.forEach(s => params.append('select', s));
    
    const response = await fetch(`/api/v1/users?${params}`);
    return await response.json();
}

// 使用示例
const result = await queryUsers({
    page: 1,
    pageSize: 20,
    filters: [
        { field: 'status', operator: 'eq', value: 'active' },
        { field: 'role', operator: 'in', value: ['admin', 'user'] }
    ],
    preload: ['Profile']
});
```

### 前端 Vue 3 完整示例

```vue
<template>
    <div class="user-list">
        <div class="search-bar">
            <input 
                v-model="searchKeyword" 
                @input="handleSearch"
                placeholder="搜索用户名..."
            />
            <select v-model="statusFilter" @change="handleFilterChange">
                <option value="">所有状态</option>
                <option value="active">激活</option>
                <option value="inactive">未激活</option>
            </select>
        </div>
        
        <table>
            <thead>
                <tr>
                    <th @click="handleSort('id')">ID</th>
                    <th @click="handleSort('name')">姓名</th>
                    <th @click="handleSort('email')">邮箱</th>
                    <th @click="handleSort('created_at')">创建时间</th>
                </tr>
            </thead>
            <tbody>
                <tr v-for="user in users" :key="user.id">
                    <td>{{ user.id }}</td>
                    <td>{{ user.name }}</td>
                    <td>{{ user.email }}</td>
                    <td>{{ formatDate(user.created_at) }}</td>
                </tr>
            </tbody>
        </table>
        
        <div class="pagination">
            <button @click="prevPage" :disabled="page === 1">上一页</button>
            <span>第 {{ page }} / {{ totalPages }} 页</span>
            <button @click="nextPage" :disabled="page === totalPages">下一页</button>
        </div>
    </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue';
import axios from 'axios';

const users = ref([]);
const page = ref(1);
const pageSize = ref(10);
const total = ref(0);
const totalPages = ref(0);
const searchKeyword = ref('');
const statusFilter = ref('');
const orderBy = ref('created_at');
const order = ref('DESC');

// 构建过滤条件
const buildFilters = () => {
    const filters = [];
    
    if (searchKeyword.value) {
        filters.push({
            field: 'name',
            operator: 'like',
            value: searchKeyword.value
        });
    }
    
    if (statusFilter.value) {
        filters.push({
            field: 'status',
            operator: 'eq',
            value: statusFilter.value
        });
    }
    
    return filters;
};

// 加载用户数据
const loadUsers = async () => {
    try {
        const filters = buildFilters();
        const response = await axios.get('/api/v1/users', {
            params: {
                page: page.value,
                page_size: pageSize.value,
                order_by: orderBy.value,
                order: order.value,
                filters: filters.length > 0 ? JSON.stringify(filters) : undefined,
                preload: ['Profile']
            }
        });
        
        users.value = response.data.data;
        total.value = response.data.total;
        totalPages.value = response.data.total_pages;
    } catch (error) {
        console.error('加载用户失败:', error);
    }
};

// 处理搜索
const handleSearch = () => {
    page.value = 1; // 重置到第一页
    loadUsers();
};

// 处理过滤器变化
const handleFilterChange = () => {
    page.value = 1;
    loadUsers();
};

// 处理排序
const handleSort = (field) => {
    if (orderBy.value === field) {
        order.value = order.value === 'ASC' ? 'DESC' : 'ASC';
    } else {
        orderBy.value = field;
        order.value = 'DESC';
    }
    loadUsers();
};

// 翻页
const prevPage = () => {
    if (page.value > 1) {
        page.value--;
        loadUsers();
    }
};

const nextPage = () => {
    if (page.value < totalPages.value) {
        page.value++;
        loadUsers();
    }
};

// 格式化日期
const formatDate = (date) => {
    return new Date(date).toLocaleDateString('zh-CN');
};

onMounted(() => {
    loadUsers();
});
</script>
```

## 高级用法

### 1. 组合多个查询条件

```javascript
// 复杂查询：查询年龄在18-65之间的活跃用户，且邮箱已验证
const filters = [
    { field: 'age', operator: 'between', value: [18, 65] },
    { field: 'status', operator: 'eq', value: 'active' },
    { field: 'email_verified', operator: 'eq', value: true },
    { field: 'deleted_at', operator: 'is_null' }
];

const response = await axios.get('/api/v1/users', {
    params: {
        page: 1,
        page_size: 20,
        filters: JSON.stringify(filters),
        order_by: 'created_at',
        order: 'DESC'
    }
});
```

### 2. 使用自定义路由

```go
// 后端注册自定义路由
userQueryRouter.RegisterCustomRoute(
    "GET",
    "/users/vip",
    func(db *gorm.DB, c *gin.Context) *gorm.DB {
        minSpending := c.Query("min_spending")
        if minSpending != "" {
            db = db.Where("total_spending >= ?", minSpending)
        }
        return db.Where("membership_level = ?", "VIP")
    },
)
```

```javascript
// 前端调用
const response = await axios.get('/api/v1/users/vip', {
    params: {
        min_spending: 1000,
        page: 1,
        page_size: 10
    }
});
```

## 错误处理

### 错误响应格式

```json
{
    "error": "invalid parameters",
    "message": "详细错误信息"
}
```

### HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

### 前端错误处理示例

```javascript
try {
    const response = await axios.get('/api/v1/users/123');
    return response.data;
} catch (error) {
    if (error.response) {
        switch (error.response.status) {
            case 400:
                console.error('请求参数错误:', error.response.data.message);
                break;
            case 404:
                console.error('用户不存在');
                break;
            case 500:
                console.error('服务器错误:', error.response.data.message);
                break;
        }
    }
}
```

## 性能优化建议

1. **使用 select 参数**: 只查询需要的字段
2. **合理设置 page_size**: 避免一次查询过多数据
3. **使用索引**: 在经常用于过滤的字段上建立数据库索引
4. **避免过度预加载**: 只在需要时使用 preload
5. **使用缓存**: 对于不常变化的数据可以在前端缓存

## 注意事项

1. filters 参数需要进行 URL 编码
2. 所有查询参数都是可选的
3. 默认按 created_at DESC 排序
4. 默认分页大小为 10
5. 最大分页大小建议不超过 100
