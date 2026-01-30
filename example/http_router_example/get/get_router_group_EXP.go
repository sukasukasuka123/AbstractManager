package main

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"AbstractManager/http_router"
	"AbstractManager/service"
)

// ========== 简单 User 结构体 ==========

type User struct {
	ID     uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	Name   string `json:"name" gorm:"size:100"`
	Age    int    `json:"age"`
	Status string `json:"status" gorm:"size:20"`
}

// ========== 示例 1: 简单用户查询 ==========

func Example1_SimpleUserQuery() {
	r := gin.Default()
	v1 := r.Group("/api/v1")

	// 创建 ServiceManager
	userService := service.NewServiceManager(User{})

	// 创建查询路由组
	userRouter := http_router.NewQueryRouterGroup(v1, userService)

	// 注册常用方法（list, active_list, search）
	userRouter.RegisterCommonMethods(20)

	// 注册路由
	userRouter.RegisterRoutes("/users")

	r.Run(":8080")
}

// ========== 示例 2: 自定义查询方法 ==========

func Example2_CustomQueryMethods() {
	r := gin.Default()
	v1 := r.Group("/api/v1")

	userService := service.NewServiceManager(User{})
	userRouter := http_router.NewQueryRouterGroup(v1, userService)

	// 1. 普通列表
	userRouter.RegisterListMethod(20)

	// 2. VIP 用户列表（这里用 Age >= 18 作为示例条件）
	userRouter.RegisterMethod(
		"vip_list",
		10,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("age >= ?", 18)
		},
		"age",
		"DESC",
	)

	// 3. 最近注册用户（示例用 ID 模拟）
	userRouter.RegisterMethod(
		"recent_users",
		50,
		func(db *gorm.DB) *gorm.DB {
			return db.Order("id DESC")
		},
		"id",
		"DESC",
	)

	// 4. 活跃用户
	userRouter.RegisterActiveListMethod(30)

	userRouter.RegisterRoutes("/users")

	r.Run(":8080")
}

// ========== 示例 3: 完整用户 API ==========

func Example3_CompleteUserAPI() {
	r := gin.Default()
	v1 := r.Group("/api/v1")

	userService := service.NewServiceManager(User{})
	userRouter := http_router.NewQueryRouterGroup(v1, userService)

	// 全部用户列表
	userRouter.RegisterListMethod(20)

	// 活跃用户
	userRouter.RegisterActiveListMethod(20)

	// 待审核用户（Status = pending）
	userRouter.RegisterMethod(
		"pending_users",
		15,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("status = ?", "pending")
		},
		"id",
		"ASC",
	)

	userRouter.RegisterRoutes("/users")

	r.Run(":8080")
}

// ========== 示例 4: 订单查询（模拟订单结构体） ==========

type Order struct {
	ID     uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	Status string `json:"status" gorm:"size:20"`
	Amount int    `json:"amount"`
}

func Example4_OrderQuery() {
	r := gin.Default()
	v1 := r.Group("/api/v1")

	orderService := service.NewServiceManager(Order{})
	orderRouter := http_router.NewQueryRouterGroup(v1, orderService)

	// 全部订单
	orderRouter.RegisterListMethod(30)

	// 待支付订单
	orderRouter.RegisterMethod(
		"pending_payment",
		20,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("status = ?", "pending_payment")
		},
		"id",
		"DESC",
	)

	orderRouter.RegisterRoutes("/orders")

	r.Run(":8080")
}

// ========== 示例 5: 产品查询（模拟产品结构体） ==========

type Product struct {
	ID    uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	Name  string `json:"name" gorm:"size:100"`
	Price int    `json:"price"`
	Stock int    `json:"stock"`
}

func Example5_ProductQuery() {
	r := gin.Default()
	v1 := r.Group("/api/v1")

	productService := service.NewServiceManager(Product{})
	productRouter := http_router.NewQueryRouterGroup(v1, productService)

	// 全部产品
	productRouter.RegisterListMethod(20)

	// 热门产品（Price > 100）
	productRouter.RegisterMethod(
		"hot_products",
		20,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("price > ?", 100)
		},
		"price",
		"DESC",
	)

	productRouter.RegisterRoutes("/products")

	r.Run(":8080")
}

// ========== 示例 6: 多资源完整应用 ==========

func Example6_FullApplication() {
	r := gin.Default()
	v1 := r.Group("/api/v1")

	setupUserRoutes(v1)
	setupOrderRoutes(v1)
	setupProductRoutes(v1)

	r.Run(":8080")
}

func setupUserRoutes(rg *gin.RouterGroup) {
	userService := service.NewServiceManager(User{})
	userRouter := http_router.NewQueryRouterGroup(rg, userService)
	userRouter.RegisterListMethod(20)
	userRouter.RegisterActiveListMethod(20)
	userRouter.RegisterRoutes("/users")
}

func setupOrderRoutes(rg *gin.RouterGroup) {
	orderService := service.NewServiceManager(Order{})
	orderRouter := http_router.NewQueryRouterGroup(rg, orderService)
	orderRouter.RegisterListMethod(30)
	orderRouter.RegisterRoutes("/orders")
}

func setupProductRoutes(rg *gin.RouterGroup) {
	productService := service.NewServiceManager(Product{})
	productRouter := http_router.NewQueryRouterGroup(rg, productService)
	productRouter.RegisterListMethod(20)
	productRouter.RegisterRoutes("/products")
}

// ========== 主函数 ==========

func main() {
	// 运行完整示例
	Example6_FullApplication()
}
