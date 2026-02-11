package filter_translator_test

import (
	"context"
	"fmt"
	"log"

	"AbstractManager/util/filter_translator"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ========== 数据模型示例 ==========

type User struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email"`
}

// ========== GORM 使用示例 ==========

func Example_gorm_usage() {
	// 初始化数据库连接
	db, err := gorm.Open(mysql.Open("user:password@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// 前端传来的过滤参数
	frontendParams := []filter_translator.FilterParam{
		{
			Field:    "age",
			Operator: "gt",
			Value:    18,
		},
		{
			Field:    "name",
			Operator: "like",
			Value:    "John",
		},
	}

	// 使用默认注册表翻译过滤器
	filters, err := filter_translator.DefaultGormRegistry.TranslateBatch(frontendParams)
	if err != nil {
		log.Fatal(err)
	}

	// 应用过滤器查询
	var users []User
	query := db.Model(&User{})
	query = filter_translator.ApplyGormFilters(query, filters)

	if err := query.Find(&users).Error; err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d users\n", len(users))
	for _, user := range users {
		fmt.Printf("User: %s, Age: %d\n", user.Name, user.Age)
	}
}

// ========== Redis 使用示例 ==========

func Example_redis_usage() {
	// 初始化 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	ctx := context.Background()

	// 假设 Redis 中存储的数据结构如下：
	// user:1 -> {name: "John Doe", age: "25", email: "john@example.com"}
	// user:2 -> {name: "Jane Smith", age: "30", email: "jane@example.com"}
	// user:3 -> {name: "Bob Johnson", age: "17", email: "bob@example.com"}

	// 前端传来的过滤参数（与 GORM 完全相同的格式！）
	frontendParams := []filter_translator.FilterParam{
		{
			Field:    "age",
			Operator: "gt",
			Value:    18,
		},
		{
			Field:    "name",
			Operator: "like",
			Value:    "John",
		},
	}

	// 使用默认注册表翻译过滤器
	filters, err := filter_translator.DefaultRedisRegistry.TranslateBatch(frontendParams)
	if err != nil {
		log.Fatal(err)
	}

	// 首先获取所有用户的 keys（这里使用 KEYS 命令示例，生产环境建议使用 SCAN）
	allKeys, err := client.Keys(ctx, "user:*").Result()
	if err != nil {
		log.Fatal(err)
	}

	// 应用过滤器
	filteredKeys, err := filter_translator.ApplyRedisFilters(ctx, client, allKeys, filters)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d matching users\n", len(filteredKeys))

	// 获取过滤后的用户信息
	for _, key := range filteredKeys {
		userData, err := client.HGetAll(ctx, key).Result()
		if err != nil {
			log.Printf("Error getting data for key %s: %v\n", key, err)
			continue
		}
		fmt.Printf("Key: %s, Data: %v\n", key, userData)
	}
}

// ========== 更多操作符示例 ==========

func Example_all_operators() {
	// 所有支持的操作符示例（GORM 和 Redis 通用！）
	examples := []filter_translator.FilterParam{
		// 等于
		{Field: "status", Operator: "eq", Value: "active"},

		// 不等于
		{Field: "status", Operator: "ne", Value: "deleted"},

		// 大于
		{Field: "age", Operator: "gt", Value: 18},

		// 大于等于
		{Field: "age", Operator: "gte", Value: 18},

		// 小于
		{Field: "price", Operator: "lt", Value: 100},

		// 小于等于
		{Field: "price", Operator: "lte", Value: 100},

		// 模糊匹配
		{Field: "name", Operator: "like", Value: "John"},

		// IN 操作
		{Field: "category", Operator: "in", Value: []interface{}{"electronics", "books", "toys"}},

		// BETWEEN 操作
		{Field: "price", Operator: "between", Value: []interface{}{10, 100}},

		// IS NULL
		{Field: "deleted_at", Operator: "isnull", Value: nil},

		// IS NOT NULL
		{Field: "email", Operator: "isnotnull", Value: nil},
	}

	fmt.Println("=== GORM Filters ===")
	gormFilters, _ := filter_translator.DefaultGormRegistry.TranslateBatch(examples)
	for i, filter := range gormFilters {
		fmt.Printf("%d. Operator: %s, Field: %s\n", i+1, filter.GetOperator(), filter.GetField())
	}

	fmt.Println("\n=== Redis Filters ===")
	redisFilters, _ := filter_translator.DefaultRedisRegistry.TranslateBatch(examples)
	for i, filter := range redisFilters {
		fmt.Printf("%d. Operator: %s, Field: %s\n", i+1, filter.GetOperator(), filter.GetField())
	}
}

// ========== 自定义翻译器示例 ==========

// CustomRangeFilter 自定义范围过滤器（GORM 版本）
type CustomRangeFilter struct {
	Field string
	Min   int
	Max   int
}

func (f *CustomRangeFilter) ApplyGorm(db *gorm.DB) *gorm.DB {
	return db.Where(fmt.Sprintf("%s >= ? AND %s <= ?", f.Field, f.Field), f.Min, f.Max)
}

func (f *CustomRangeFilter) GetField() string      { return f.Field }
func (f *CustomRangeFilter) GetValue() interface{} { return []interface{}{f.Min, f.Max} }
func (f *CustomRangeFilter) GetOperator() string   { return "range" }

// CustomRangeTranslator 自定义翻译器
type CustomRangeTranslator struct{}

func (t *CustomRangeTranslator) Translate(param filter_translator.FilterParam) (filter_translator.BaseFilter, error) {
	values, ok := param.Value.([]interface{})
	if !ok || len(values) != 2 {
		return nil, fmt.Errorf("range operator requires array with 2 elements")
	}

	min, ok1 := values[0].(int)
	max, ok2 := values[1].(int)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("range values must be integers")
	}

	return &CustomRangeFilter{
		Field: param.Field,
		Min:   min,
		Max:   max,
	}, nil
}

func (t *CustomRangeTranslator) SupportedOperator() string {
	return "range"
}

func (t *CustomRangeTranslator) Validate(param filter_translator.FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	values, ok := param.Value.([]interface{})
	if !ok || len(values) != 2 {
		return fmt.Errorf("value must be array with 2 elements")
	}
	return nil
}

func Example_custom_translator() {
	// 创建自定义注册表
	registry := filter_translator.NewGormTranslatorRegistry()

	// 注册自定义翻译器
	registry.Register(&CustomRangeTranslator{})

	// 使用自定义操作符
	param := filter_translator.FilterParam{
		Field:    "age",
		Operator: "range",
		Value:    []interface{}{18, 65},
	}

	filter, err := registry.Translate(param)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Custom filter created: %s\n", filter.GetOperator())
}

// ========== 获取支持的操作符 ==========

func Example_get_supported_operators() {
	gormOps := filter_translator.DefaultGormRegistry.GetSupportedOperators()
	fmt.Println("GORM supported operators:", gormOps)
	// 前端可以使用这个列表来生成过滤器 UI
}
