首先，因为本人记忆力不足加上最近事情繁多，所以暂时让copilot总结一个readme出来，之后会补上完整的readme
其次，为了防止本人失忆，router包下面我给每个模块都放了readme，以及util包下面的filter放了example和readme

# AbstractManager

轻量级的 Go 泛型 Service + HTTP 路由管理器，封装了常见的数据库（GORM）与缓存（Redis）操作，以及基于 ServiceManager 的 HTTP 路由快速注册方案，目标是快速搭建标准化的 CRUD / 查询 / 缓存服务。

## 主要功能
- 通用的 ServiceManager（泛型）用于管理单表资源：创建、查询、更新、删除、缓存读写等。
- 将 ServiceManager 与 HTTP 路由集成，提供可注册的路由组（lookup / get / set / writedown 等）。
- 内置 DB/Redis 初始化、连接池管理、以及缓存策略（lookup / writedown）。
- 提供示例代码与方法注册机制，便于在 Gin 等框架下快速构建 API。

## 目录（高层）
- service/ — ServiceManager 与数据库、缓存相关实现与工具
- http_router/ — 将 ServiceManager 封装为路由组并提供 HTTP 处理逻辑
- example/ — 完整的示例（路由组、缓存查询等）
- README.md — （本文件）

---

## 快速开始（最小示例）

1. 初始化 DB 与 Redis（见 service.InitDB / service.InitRedis）
2. 定义你的模型结构体
3. 创建 ServiceManager（泛型）
4. 将 ServiceManager 注入到 HTTP Router（或直接使用路由组构造函数）

示例（伪代码、简化）：

```go
// 定义模型
type User struct {
    ID    uint   `gorm:"primaryKey"`
    Name  string `gorm:"size:100"`
    Email string `gorm:"size:255;uniqueIndex"`
}

// 初始化（伪代码）
dbMgr, _ := service.InitDB()
defer dbMgr.Close()
redisMgr, _ := service.InitRedis()
defer redisMgr.Close()

// 创建 ServiceManager
userSvc := service.NewServiceManager(User{})
userSvc.CacheKeyType = "cache"
userSvc.CacheKeyName = "user"

// 将 service 注入 router（使用 http_router 提供的路由组）
routerGroup := http_router.NewLookupRouterGroup(apiGroup, userSvc)
routerGroup.RegisterDefaultMethods() // 假设有便捷的注册方法
```

---

## 示例：在 Gin 中快速注册（参考仓库示例）
仓库中提供了更完整的示例（包含 Lookup / Set / Get 模块），可以参考下面的示例文件（见仓库原始示例）：

- 完整的 HTTP+Service 快速配置示例（来自仓库）
- Lookup/Cache / Count API 示例（来自仓库）
（这些示例片段已在本页下方以原始文件形式给出，带有 GitHub permalink）

---

## 核心设计与原理

- 泛型 ServiceManager
  - 使用 Go 泛型（type parameter）和 reflect 获取模型名称、表名等元信息，提供对单表资源的统一管理接口（创建、查询、更新、删除、增减等）。
  - ServiceManager 保存与资源相关的配置，如 CacheKeyType、CacheKeyName、ResourceName、TableName 等，便于统一缓存 key 策略。

- DB 与 Redis 管理
  - 提供 InitDB / InitRedis 与全局管理器（DBManager / RedisManager）用于统一初始化与关闭，避免在多个地方重复创建连接池。
  - DB 使用 GORM，支持 logger 等配置；Redis 使用官方客户端或 go-redis。

- 缓存层（Lookup / Writedown）
  - Lookup：优先从 Redis 中读取缓存，如果未命中则从数据库读取并回填缓存（支持按模式、分页、计数等）。
  - Writedown：数据库写（或批量写）后同步刷新缓存，保证缓存的一致性（支持异步写回、版本控制等策略）。
  - 单项（single）与批量（query）操作被分为不同的模块（例如 writedown_single、writedown_query、get_single、get_query 等），以便职责分离。

- HTTP 路由集成
  - 提供一系列路由组（LookupRouterGroup、GetRouterGroup、SetRouterGroup 等），这些路由组内部持有 ServiceManager 的引用并注册相应的 HTTP handler。
  - 支持依赖注入方式创建 HTTPRouterManager：如果你已经有一个 ServiceManager，直接注入可以避免重复创建 DB 连接池。

---

## 常见使用场景
- 快速为单表资源生成 CRUD + 缓存的 HTTP API。
- 对高并发读取场景，使用 Lookup 缓存层减少 DB 压力。
- 需要统一管理表/缓存 key 命名策略的场景（通过 ServiceManager 的 CacheKeyType、CacheKeyName 进行配置）。

---

## 部署建议
- 强烈建议在应用启动阶段统一初始化 DB 与 Redis，并把同一套连接传入不同的 ServiceManager（避免重复连接池）。
- 根据数据访问量合理配置 Redis 和数据库连接池大小。
- 对写多读少场景，考虑缩短缓存 TTL 或在写后及时 writedown。

---

## 我做了什么，接下来建议
- 我已参考仓库中的 service 与 http_router 文档和示例，生成了上面的 README 初稿。
- 建议将此 README.md 放到仓库根目录，并基于示例进一步补充：
  - 在 README 中加入一个完整的可运行示例（main.go + go.mod），并在 CI 中加入 `go vet` / `go test` 的基本检查。
  - 根据需要把示例中涉及的配置（DB/Redis 地址等）抽成 environment variables 并在 README 中列出。

---

附：仓库中原始示例/实现片段（便于复制）
下面我把仓库里与本 README 相关的示例/实现片段以原始文件块形式列出，包含 GitHub permalink，可直接查看或复制。

```go name=http_router/router_model.go url=https://github.com/sukasukasuka123/AbstractManager/blob/d4c01c2b38bf6df4f1548a5fd71d71a4f15c46ab/http_router/router_model.go
/*...*/
// NewHTTPRouterManager 构造函数 (推荐方式：依赖注入)
// 接收一个已经初始化好的 ServiceManager
func NewHTTPRouterManager[T any](svc *serviceManager.ServiceManager[T]) *HTTPRouterManager[T] {
    return &HTTPRouterManager[T]{
        ServiceManager: svc,
    }
}

// NewHTTPRouterManagerFromModel 构造函数 (兼容你原来的方式)
// 接收模型实例，在内部创建新的 ServiceManager
// 注意：如果在多个地方使用，可能会导致创建多个 DB 连接池，请谨慎使用
func NewHTTPRouterManagerFromModel[T any](model T) *HTTPRouterManager[T] {
    return &HTTPRouterManager[T]{
        ServiceManager: serviceManager.NewServiceManager(model),
    }
}
```

```markdown name=service/service_readme.md url=https://github.com/sukasukasuka123/AbstractManager/blob/d4c01c2b38bf6df4f1548a5fd71d71a4f15c46ab/service/service_readme.md
# service 模块 使用说明（之后要跟router模块集成到abstractmanager里面去）
...
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
cctx := context.Background()
```
```

```markdown name=http_router/set_readme.md url=https://github.com/sukasukasuka123/AbstractManager/blob/d4c01c2b38bf6df4f1548a5fd71d71a4f15c46ab/http_router/set_readme.md
# set(为了方便下面称为write) 模块使用文档
...
## 二、完整使用示例

### 2.1 基础配置代码

```go
package main

import (
    "github.com/gin-gonic/gin"
    
    "AbstractManager/http_router"
    "AbstractManager/service"
)

// 数据模型
type User struct {
    ID     uint   `json:"id" gorm:"primaryKey;autoIncrement"`
    Name   string `json:"name" gorm:"size:100"`
    Age    int    `json:"age"`
    Status string `json:"status" gorm:"size:20"`
    Email  string `json:"email" gorm:"size:100;uniqueIndex"`
    Count  int    `json:"count" gorm:"default:0"`
}

func main() {
    r := gin.Default()
    v1 := r.Group("/api/v1")
    
    // 1. 创建 ServiceManager
    userService := service.NewServiceManager(User{})
...
```
```

```markdown name=http_router/cache_get_readme.md url=https://github.com/sukasukasuka123/AbstractManager/blob/d4c01c2b38bf6df4f1548a5fd71d71a4f15c46ab/http_router/cache_get_readme.md
# Lookup 模块使用文档
...
## 三、完整使用示例

### 3.1 基础配置代码

```go
package main

import (
    "context"
    "time"
    
    "AbstractManager/http_router"
    "AbstractManager/service"
    
    "github.com/gin-gonic/gin"
)

// 数据模型
type User struct {
    ID     uint   `json:"id"`
    Name   string `json:"name"`
    Age    int    `json:"age"`
    Status string `json:"status"`
    Email  string `json:"email"`
}

func main() {
    r := gin.Default()
    v1 := r.Group("/api/v1")
    
    // 1. 创建 ServiceManager
    userService := service.NewServiceManager(User{})
    userService.CacheKeyType = "cache"
    userService.CacheKeyName = "user"
    
    // 2. 创建 Lookup 路由组
    userLookup := http_router.NewLookupRouterGroup(v1, userService)
    
    // 3. 注册查询方法
...
```
```
