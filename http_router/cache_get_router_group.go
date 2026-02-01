package http_router

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"AbstractManager/service"
	"AbstractManager/util/filter_translator"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// ========== 预定义 Lookup 查询方法 ==========

type LookupQueryMethod[T any] struct {
	Name             string
	Service          *service.ServiceManager[T]
	KeyPattern       string                                                           // Redis 键模式，如 "user:*"
	CacheExpire      time.Duration                                                    // 缓存过期时间
	FallbackToDB     bool                                                             // 是否回源数据库
	CustomFilterFunc func(context.Context, *redis.Client, []string) ([]string, error) // 自定义过滤函数
}

func (m *LookupQueryMethod[T]) Execute(ctx context.Context, filters []filter_translator.RedisFilter) (map[string]*T, []string, error) {
	// 1. 获取所有匹配的键
	redisClient := service.GetRedis()
	allKeys, err := redisClient.Keys(ctx, m.KeyPattern).Result()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get keys: %w", err)
	}

	if len(allKeys) == 0 {
		return make(map[string]*T), []string{}, nil
	}

	// 2. 应用自定义过滤（如果有）
	if m.CustomFilterFunc != nil {
		allKeys, err = m.CustomFilterFunc(ctx, redisClient, allKeys)
		if err != nil {
			return nil, nil, fmt.Errorf("custom filter failed: %w", err)
		}
	}

	// 3. 应用过滤器（使用示例中的 ApplyRedisFilters）
	filteredKeys, err := filter_translator.ApplyRedisFilters(ctx, redisClient, allKeys, filters)
	if err != nil {
		return nil, nil, fmt.Errorf("filter application failed: %w", err)
	}

	// 4. 从缓存查询数据
	opts := &service.LookupQueryOptions{
		KeyPattern:   m.KeyPattern,
		CacheExpire:  m.CacheExpire,
		FallbackToDB: m.FallbackToDB,
	}

	result, err := m.Service.LookupQuery(ctx, filteredKeys, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("lookup query failed: %w", err)
	}

	return result, filteredKeys, nil
}

// ========== Lookup 查询方法注册表 ==========

type LookupMethodRegistry[T any] struct {
	methods map[string]*LookupQueryMethod[T]
}

func NewLookupMethodRegistry[T any]() *LookupMethodRegistry[T] {
	return &LookupMethodRegistry[T]{
		methods: make(map[string]*LookupQueryMethod[T]),
	}
}

func (r *LookupMethodRegistry[T]) Register(method *LookupQueryMethod[T]) {
	r.methods[method.Name] = method
}

func (r *LookupMethodRegistry[T]) Get(name string) (*LookupQueryMethod[T], bool) {
	method, ok := r.methods[name]
	return method, ok
}

// ========== Gin Lookup 路由组 ==========

type LookupRouterGroup[T any] struct {
	RouterGroup        *gin.RouterGroup
	Service            *service.ServiceManager[T]
	MethodRegistry     *LookupMethodRegistry[T]
	TranslatorRegistry *filter_translator.RedisTranslatorRegistry
}

func NewLookupRouterGroup[T any](
	rg *gin.RouterGroup,
	service *service.ServiceManager[T],
) *LookupRouterGroup[T] {
	return &LookupRouterGroup[T]{
		RouterGroup:        rg,
		Service:            service,
		MethodRegistry:     NewLookupMethodRegistry[T](),
		TranslatorRegistry: filter_translator.DefaultRedisRegistry,
	}
}

// ========== 注册预定义 Lookup 查询方法 ==========

func (lrg *LookupRouterGroup[T]) RegisterMethod(
	name string,
	keyPattern string,
	cacheExpire time.Duration,
	fallbackToDB bool,
	customFilterFunc func(context.Context, *redis.Client, []string) ([]string, error),
) {
	method := &LookupQueryMethod[T]{
		Name:             name,
		Service:          lrg.Service,
		KeyPattern:       keyPattern,
		CacheExpire:      cacheExpire,
		FallbackToDB:     fallbackToDB,
		CustomFilterFunc: customFilterFunc,
	}
	lrg.MethodRegistry.Register(method)
}

// RegisterListMethod 注册基础列表方法（查询所有键）
func (lrg *LookupRouterGroup[T]) RegisterListMethod(keyPattern string, cacheExpire time.Duration) {
	lrg.RegisterMethod("list", keyPattern, cacheExpire, false, nil)
}

// RegisterActiveListMethod 注册活跃数据列表（通过自定义过滤器）
func (lrg *LookupRouterGroup[T]) RegisterActiveListMethod(
	keyPattern string,
	cacheExpire time.Duration,
	activeFilterFunc func(context.Context, *redis.Client, []string) ([]string, error),
) {
	lrg.RegisterMethod("active_list", keyPattern, cacheExpire, false, activeFilterFunc)
}

// RegisterFallbackMethod 注册带数据库回源的方法
func (lrg *LookupRouterGroup[T]) RegisterFallbackMethod(keyPattern string, cacheExpire time.Duration) {
	lrg.RegisterMethod("fallback_list", keyPattern, cacheExpire, true, nil)
}

// ========== 路由注册 ==========

func (lrg *LookupRouterGroup[T]) RegisterRoutes(basePath string) {
	lrg.RouterGroup.POST(basePath+"/lookup", lrg.HandleLookup)
	lrg.RouterGroup.GET(basePath+"/:key", lrg.HandleGetByKey)
	lrg.RouterGroup.POST(basePath+"/count", lrg.HandleCount)
	lrg.RouterGroup.POST(basePath+"/invalidate", lrg.HandleInvalidate)
}

// ========== 请求/响应结构 ==========

type LookupRequest struct {
	Method  string                          `json:"method"`
	Filters []filter_translator.FilterParam `json:"filters"`
}

type LookupResponse[T any] struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    map[string]*T `json:"data"`
	Keys    []string      `json:"keys"`
	Count   int           `json:"count"`
}

type LookupCountRequest struct {
	Method  string                          `json:"method"`
	Filters []filter_translator.FilterParam `json:"filters"`
}

type LookupCountResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Count   int    `json:"count"`
}

type InvalidateRequest struct {
	Keys    []string `json:"keys"`    // 精确键列表
	Pattern string   `json:"pattern"` // 或使用模式
}

type InvalidateResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Count   int    `json:"count"` // 删除的键数量
}

// ========== 处理器 ==========

func (lrg *LookupRouterGroup[T]) HandleLookup(c *gin.Context) {
	var req LookupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid request", "error": err.Error()})
		return
	}

	method, ok := lrg.MethodRegistry.Get(req.Method)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": fmt.Sprintf("unknown method: %s", req.Method)})
		return
	}

	// 翻译过滤器
	filters, err := lrg.TranslatorRegistry.TranslateBatch(req.Filters)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid filters", "error": err.Error()})
		return
	}

	// 执行查询
	result, keys, err := method.Execute(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "lookup failed", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, LookupResponse[T]{
		Code:    0,
		Message: "success",
		Data:    result,
		Keys:    keys,
		Count:   len(result),
	})
}

func (lrg *LookupRouterGroup[T]) HandleGetByKey(c *gin.Context) {
	key := c.Param("key")
	ctx := c.Request.Context()

	opts := &service.LookupSingleOptions{
		CacheExpire:  1 * time.Hour,
		FallbackToDB: false,
		Refresh:      false,
	}

	result, err := lrg.Service.LookupSingle(ctx, key, opts)
	if err != nil {
		if err == redis.Nil {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "key not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "lookup failed", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

func (lrg *LookupRouterGroup[T]) HandleCount(c *gin.Context) {
	var req LookupCountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid request", "error": err.Error()})
		return
	}

	method, ok := lrg.MethodRegistry.Get(req.Method)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": fmt.Sprintf("unknown method: %s", req.Method)})
		return
	}

	// 翻译过滤器
	filters, err := lrg.TranslatorRegistry.TranslateBatch(req.Filters)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid filters", "error": err.Error()})
		return
	}

	// 执行查询（只计数，不获取数据）
	_, keys, err := method.Execute(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "count failed", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, LookupCountResponse{
		Code:    0,
		Message: "success",
		Count:   len(keys),
	})
}

func (lrg *LookupRouterGroup[T]) HandleInvalidate(c *gin.Context) {
	var req InvalidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid request", "error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	var deletedCount int

	if req.Pattern != "" {
		// 按模式删除
		if err := lrg.Service.InvalidateCacheByPattern(ctx, req.Pattern); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "invalidate failed", "error": err.Error()})
			return
		}
		// 获取删除的键数量（这里简化处理）
		deletedCount = -1 // 表示按模式删除，无法精确统计
	} else if len(req.Keys) > 0 {
		// 按键列表删除
		if err := lrg.Service.InvalidateCache(ctx, req.Keys...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "invalidate failed", "error": err.Error()})
			return
		}
		deletedCount = len(req.Keys)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "either keys or pattern must be provided"})
		return
	}

	c.JSON(http.StatusOK, InvalidateResponse{
		Code:    0,
		Message: "success",
		Count:   deletedCount,
	})
}

// ========== 便捷方法 ==========

func (lrg *LookupRouterGroup[T]) RegisterCommonMethods(keyPattern string, cacheExpire time.Duration) {
	lrg.RegisterListMethod(keyPattern, cacheExpire)
	lrg.RegisterFallbackMethod(keyPattern, cacheExpire)
}
