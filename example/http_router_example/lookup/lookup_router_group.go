package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"AbstractManager/http_router"
	"AbstractManager/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// ========== 数据模型 ==========

type CachedUser struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Age    int    `json:"age"`
	Status string `json:"status"`
	Email  string `json:"email"`
}

type CachedProduct struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Price    int    `json:"price"`
	Stock    int    `json:"stock"`
	Category string `json:"category"`
}

type CachedOrder struct {
	ID       uint   `json:"id"`
	UserID   uint   `json:"user_id"`
	Status   string `json:"status"`
	Amount   int    `json:"amount"`
	CreateAt string `json:"create_at"`
}

// ========== 示例 1: 简单用户缓存查询 ==========

func Example1_SimpleUserLookup() {
	r := gin.Default()
	v1 := r.Group("/api/v1")

	// 创建 ServiceManager
	userService := service.NewServiceManager(CachedUser{})
	userService.CacheKeyType = "cache"
	userService.CacheKeyName = "user"

	// 创建 Lookup 路由组
	userLookup := http_router.NewLookupRouterGroup(v1, userService)

	// 注册基础查询方法
	userLookup.RegisterListMethod("cache:user:*", 1*time.Hour)

	// 注册路由
	userLookup.RegisterRoutes("/users")

	r.Run(":8080")
}

// ========== 示例 2: 自定义过滤器方法 ==========

func Example2_CustomFilterMethods() {
	r := gin.Default()
	v1 := r.Group("/api/v1")

	userService := service.NewServiceManager(CachedUser{})
	userService.CacheKeyType = "cache"
	userService.CacheKeyName = "user"
	userLookup := http_router.NewLookupRouterGroup(v1, userService)

	// 1. 所有用户
	userLookup.RegisterListMethod("cache:user:*", 1*time.Hour)

	// 2. 活跃用户（自定义过滤：status = active）
	userLookup.RegisterActiveListMethod(
		"cache:user:*",
		1*time.Hour,
		func(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
			var activeKeys []string
			for _, key := range keys {
				data, err := client.HGet(ctx, key, "status").Result()
				if err == nil && data == "active" {
					activeKeys = append(activeKeys, key)
				}
			}
			return activeKeys, nil
		},
	)

	// 3. VIP 用户（Age >= 18）
	userLookup.RegisterMethod(
		"vip_users",
		"cache:user:*",
		1*time.Hour,
		false,
		func(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
			var vipKeys []string
			for _, key := range keys {
				ageStr, err := client.HGet(ctx, key, "age").Result()
				if err == nil {
					age, _ := strconv.Atoi(ageStr)
					if age >= 18 {
						vipKeys = append(vipKeys, key)
					}
				}
			}
			return vipKeys, nil
		},
	)

	userLookup.RegisterRoutes("/users")

	r.Run(":8080")
}

// ========== 示例 3: 带数据库回源的查询 ==========

func Example3_FallbackToDatabase() {
	r := gin.Default()
	v1 := r.Group("/api/v1")

	userService := service.NewServiceManager(CachedUser{})
	userService.CacheKeyType = "cache"
	userService.CacheKeyName = "user"
	userLookup := http_router.NewLookupRouterGroup(v1, userService)

	// 注册带回源的方法
	userLookup.RegisterFallbackMethod("cache:user:*", 30*time.Minute)

	userLookup.RegisterRoutes("/users")

	r.Run(":8080")
}

// ========== 示例 4: 产品缓存查询 ==========

func Example4_ProductLookup() {
	r := gin.Default()
	v1 := r.Group("/api/v1")

	productService := service.NewServiceManager(CachedProduct{})
	productService.CacheKeyType = "cache"
	productService.CacheKeyName = "product"
	productLookup := http_router.NewLookupRouterGroup(v1, productService)

	// 所有产品
	productLookup.RegisterListMethod("cache:product:*", 2*time.Hour)

	// 热门产品（价格 > 100）
	productLookup.RegisterMethod(
		"hot_products",
		"cache:product:*",
		2*time.Hour,
		false,
		func(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
			var hotKeys []string
			for _, key := range keys {
				priceStr, err := client.HGet(ctx, key, "price").Result()
				if err == nil {
					price, _ := strconv.Atoi(priceStr)
					if price > 100 {
						hotKeys = append(hotKeys, key)
					}
				}
			}
			return hotKeys, nil
		},
	)

	productLookup.RegisterRoutes("/products")

	r.Run(":8080")
}

// ========== 示例 5: 订单缓存查询 ==========

func Example5_OrderLookup() {
	r := gin.Default()
	v1 := r.Group("/api/v1")

	orderService := service.NewServiceManager(CachedOrder{})
	orderService.CacheKeyType = "cache"
	orderService.CacheKeyName = "order"
	orderLookup := http_router.NewLookupRouterGroup(v1, orderService)

	// 所有订单
	orderLookup.RegisterListMethod("cache:order:*", 30*time.Minute)

	// 待支付订单
	orderLookup.RegisterMethod(
		"pending_payment",
		"cache:order:*",
		30*time.Minute,
		false,
		func(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
			var pendingKeys []string
			for _, key := range keys {
				status, err := client.HGet(ctx, key, "status").Result()
				if err == nil && status == "pending_payment" {
					pendingKeys = append(pendingKeys, key)
				}
			}
			return pendingKeys, nil
		},
	)

	orderLookup.RegisterRoutes("/orders")

	r.Run(":8080")
}

// ========== 示例 6: 完整应用（多资源） ==========

func Example6_FullApplication() {
	r := gin.Default()
	v1 := r.Group("/api/v1")

	setupUserLookupRoutes(v1)
	setupProductLookupRoutes(v1)
	setupOrderLookupRoutes(v1)

	r.Run(":8080")
}

func setupUserLookupRoutes(rg *gin.RouterGroup) {
	userService := service.NewServiceManager(CachedUser{})
	userService.CacheKeyType = "cache"
	userService.CacheKeyName = "user"
	userLookup := http_router.NewLookupRouterGroup(rg, userService)

	userLookup.RegisterCommonMethods("cache:user:*", 1*time.Hour)
	userLookup.RegisterRoutes("/users")
}

func setupProductLookupRoutes(rg *gin.RouterGroup) {
	productService := service.NewServiceManager(CachedProduct{})
	productService.CacheKeyType = "cache"
	productService.CacheKeyName = "product"
	productLookup := http_router.NewLookupRouterGroup(rg, productService)

	productLookup.RegisterCommonMethods("cache:product:*", 2*time.Hour)
	productLookup.RegisterRoutes("/products")
}

func setupOrderLookupRoutes(rg *gin.RouterGroup) {
	orderService := service.NewServiceManager(CachedOrder{})
	orderService.CacheKeyType = "cache"
	orderService.CacheKeyName = "order"
	orderLookup := http_router.NewLookupRouterGroup(rg, orderService)

	orderLookup.RegisterCommonMethods("cache:order:*", 30*time.Minute)
	orderLookup.RegisterRoutes("/orders")
}

// ========== 示例 7: 使用过滤器的高级查询 ==========

func Example7_AdvancedFilterQuery() {
	r := gin.Default()
	v1 := r.Group("/api/v1")

	userService := service.NewServiceManager(CachedUser{})
	userService.CacheKeyType = "cache"
	userService.CacheKeyName = "user"
	userLookup := http_router.NewLookupRouterGroup(v1, userService)

	// 注册基础方法
	userLookup.RegisterListMethod("cache:user:*", 1*time.Hour)

	userLookup.RegisterRoutes("/users")

	r.Run(":8080")

	/*
		前端可以发送如下请求：

		POST /api/v1/users/lookup
		{
		  "method": "list",
		  "filters": [
		    {
		      "field": "age",
		      "operator": "gt",
		      "value": 18
		    },
		    {
		      "field": "name",
		      "operator": "like",
		      "value": "John"
		    },
		    {
		      "field": "status",
		      "operator": "eq",
		      "value": "active"
		    }
		  ]
		}

		返回格式：
		{
		  "code": 0,
		  "message": "success",
		  "data": {
		    "cache:user:1": {
		      "id": 1,
		      "name": "John Doe",
		      "age": 25,
		      "status": "active",
		      "email": "john@example.com"
		    },
		    "cache:user:5": {
		      "id": 5,
		      "name": "Johnny Smith",
		      "age": 30,
		      "status": "active",
		      "email": "johnny@example.com"
		    }
		  },
		  "keys": ["cache:user:1", "cache:user:5"],
		  "count": 2
		}
	*/
}

// ========== 示例 8: 缓存失效管理 ==========

func Example8_CacheInvalidation() {
	r := gin.Default()
	v1 := r.Group("/api/v1")

	userService := service.NewServiceManager(CachedUser{})
	userService.CacheKeyType = "cache"
	userService.CacheKeyName = "user"
	userLookup := http_router.NewLookupRouterGroup(v1, userService)

	userLookup.RegisterListMethod("cache:user:*", 1*time.Hour)
	userLookup.RegisterRoutes("/users")

	r.Run(":8080")

	/*
		缓存失效请求示例：

		1. 按精确键删除：
		POST /api/v1/users/invalidate
		{
		  "keys": ["cache:user:1", "cache:user:2", "cache:user:3"]
		}

		2. 按模式删除：
		POST /api/v1/users/invalidate
		{
		  "pattern": "cache:user:*"
		}

		返回格式：
		{
		  "code": 0,
		  "message": "success",
		  "count": 3
		}
	*/
}

// ========== 示例 9: 获取单个缓存 ==========

func Example9_GetSingleCache() {
	r := gin.Default()
	v1 := r.Group("/api/v1")

	userService := service.NewServiceManager(CachedUser{})
	userService.CacheKeyType = "cache"
	userService.CacheKeyName = "user"
	userLookup := http_router.NewLookupRouterGroup(v1, userService)

	userLookup.RegisterRoutes("/users")

	r.Run(":8080")

	/*
		单个查询请求：

		GET /api/v1/users/cache:user:1

		返回格式：
		{
		  "code": 0,
		  "message": "success",
		  "data": {
		    "id": 1,
		    "name": "John Doe",
		    "age": 25,
		    "status": "active",
		    "email": "john@example.com"
		  }
		}
	*/
}

// ========== 示例 10: 计数查询 ==========

func Example10_CountQuery() {
	r := gin.Default()
	v1 := r.Group("/api/v1")

	userService := service.NewServiceManager(CachedUser{})
	userService.CacheKeyType = "cache"
	userService.CacheKeyName = "user"
	userLookup := http_router.NewLookupRouterGroup(v1, userService)

	userLookup.RegisterListMethod("cache:user:*", 1*time.Hour)
	userLookup.RegisterRoutes("/users")

	r.Run(":8080")

	/*
		计数请求：

		POST /api/v1/users/count
		{
		  "method": "list",
		  "filters": [
		    {
		      "field": "status",
		      "operator": "eq",
		      "value": "active"
		    }
		  ]
		}

		返回格式：
		{
		  "code": 0,
		  "message": "success",
		  "count": 42
		}
	*/
}

// ========== 工具函数：初始化测试数据 ==========

func InitTestCacheData() {
	ctx := context.Background()
	client := service.GetRedis()

	// 初始化用户数据
	users := []CachedUser{
		{ID: 1, Name: "John Doe", Age: 25, Status: "active", Email: "john@example.com"},
		{ID: 2, Name: "Jane Smith", Age: 30, Status: "active", Email: "jane@example.com"},
		{ID: 3, Name: "Bob Johnson", Age: 17, Status: "inactive", Email: "bob@example.com"},
		{ID: 4, Name: "Alice Brown", Age: 28, Status: "active", Email: "alice@example.com"},
		{ID: 5, Name: "Johnny Test", Age: 35, Status: "active", Email: "johnny@example.com"},
	}

	for _, user := range users {
		key := fmt.Sprintf("cache:user:%d", user.ID)
		data, _ := json.Marshal(user)
		client.Set(ctx, key, data, 24*time.Hour)

		// 同时存储为 Hash（用于自定义过滤）
		client.HSet(ctx, key,
			"id", user.ID,
			"name", user.Name,
			"age", user.Age,
			"status", user.Status,
			"email", user.Email,
		)
	}

	// 初始化产品数据
	products := []CachedProduct{
		{ID: 1, Name: "Laptop", Price: 1200, Stock: 50, Category: "electronics"},
		{ID: 2, Name: "Mouse", Price: 25, Stock: 200, Category: "electronics"},
		{ID: 3, Name: "Keyboard", Price: 80, Stock: 150, Category: "electronics"},
		{ID: 4, Name: "Book", Price: 15, Stock: 500, Category: "books"},
	}

	for _, product := range products {
		key := fmt.Sprintf("cache:product:%d", product.ID)
		data, _ := json.Marshal(product)
		client.Set(ctx, key, data, 24*time.Hour)

		client.HSet(ctx, key,
			"id", product.ID,
			"name", product.Name,
			"price", product.Price,
			"stock", product.Stock,
			"category", product.Category,
		)
	}

	fmt.Println("Test cache data initialized successfully!")
}

// ========== 主函数 ==========

func main() {
	// 初始化测试数据
	InitTestCacheData()

	// 运行完整示例
	Example6_FullApplication()
}
