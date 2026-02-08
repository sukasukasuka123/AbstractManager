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

// ========== Lookup 路由组 ==========

type LookupRouterGroup[T any] struct {
	RouterGroup        *gin.RouterGroup
	Service            *service.ServiceManager[T]
	TranslatorRegistry *filter_translator.RedisTranslatorRegistry

	// 预定义的查询方法配置
	defaultKeyPattern  string
	defaultCacheExpire time.Duration
	customFilterFunc   func(context.Context, *redis.Client, []string) ([]string, error)
}

func NewLookupRouterGroup[T any](
	rg *gin.RouterGroup,
	service *service.ServiceManager[T],
) *LookupRouterGroup[T] {
	return &LookupRouterGroup[T]{
		RouterGroup:        rg,
		Service:            service,
		TranslatorRegistry: filter_translator.DefaultRedisRegistry,
		defaultCacheExpire: 1 * time.Hour, // 默认1小时
	}
}

// ========== 配置方法（在注册路由前调用）==========

// SetDefaults 设置默认的 key 模式和缓存过期时间
func (lrg *LookupRouterGroup[T]) SetDefaults(keyPattern string, cacheExpire time.Duration) *LookupRouterGroup[T] {
	lrg.defaultKeyPattern = keyPattern
	lrg.defaultCacheExpire = cacheExpire
	return lrg
}

// SetCustomFilter 设置自定义过滤函数（如活跃用户过滤）
func (lrg *LookupRouterGroup[T]) SetCustomFilter(
	filterFunc func(context.Context, *redis.Client, []string) ([]string, error),
) *LookupRouterGroup[T] {
	lrg.customFilterFunc = filterFunc
	return lrg
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
	KeyPattern      string                          `json:"key_pattern"`       // 可选，覆盖默认 key 模式
	Filters         []filter_translator.FilterParam `json:"filters"`           // 过滤条件
	UseCustomFilter bool                            `json:"use_custom_filter"` // 是否使用自定义过滤器
	FallbackToDB    bool                            `json:"fallback_db"`       // 是否回源数据库
}

type LookupResponse[T any] struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    map[string]*T `json:"data"`
	Keys    []string      `json:"keys"`
	Count   int           `json:"count"`
}

type LookupCountRequest struct {
	KeyPattern      string                          `json:"key_pattern"`
	Filters         []filter_translator.FilterParam `json:"filters"`
	UseCustomFilter bool                            `json:"use_custom_filter"`
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

// ========== 核心查询逻辑 ==========

func (lrg *LookupRouterGroup[T]) executeLookup(
	ctx context.Context,
	keyPattern string,
	filters []filter_translator.FilterParam,
	useCustomFilter bool,
	fallbackToDB bool,
) (map[string]*T, []string, error) {

	// 1. 获取所有匹配的键
	redisClient := service.GetRedis()
	allKeys, err := redisClient.Keys(ctx, keyPattern).Result()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get keys: %w", err)
	}

	if len(allKeys) == 0 {
		return make(map[string]*T), []string{}, nil
	}

	// 2. 应用自定义过滤（如果启用）
	if useCustomFilter && lrg.customFilterFunc != nil {
		allKeys, err = lrg.customFilterFunc(ctx, redisClient, allKeys)
		if err != nil {
			return nil, nil, fmt.Errorf("custom filter failed: %w", err)
		}
	}

	// 3. 翻译并应用通用过滤器
	if len(filters) > 0 {
		redisFilters, err := lrg.TranslatorRegistry.TranslateBatch(filters)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid filters: %w", err)
		}

		allKeys, err = filter_translator.ApplyRedisFilters(ctx, redisClient, allKeys, redisFilters)
		if err != nil {
			return nil, nil, fmt.Errorf("filter application failed: %w", err)
		}
	}

	// 4. 从缓存查询数据
	opts := &service.LookupQueryOptions{
		KeyPattern:   keyPattern,
		CacheExpire:  lrg.defaultCacheExpire,
		FallbackToDB: fallbackToDB,
	}

	result, err := lrg.Service.LookupQuery(ctx, allKeys, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("lookup query failed: %w", err)
	}

	return result, allKeys, nil
}

// ========== HTTP 处理器 ==========

func (lrg *LookupRouterGroup[T]) HandleLookup(c *gin.Context) {
	var req LookupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request",
			"error":   err.Error(),
		})
		return
	}

	// 使用请求中的 key pattern，如果没有则使用默认值
	keyPattern := req.KeyPattern
	if keyPattern == "" {
		keyPattern = lrg.defaultKeyPattern
	}

	if keyPattern == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "key_pattern is required (or set default via SetDefaults)",
		})
		return
	}

	// 执行查询
	result, keys, err := lrg.executeLookup(
		c.Request.Context(),
		keyPattern,
		req.Filters,
		req.UseCustomFilter,
		req.FallbackToDB,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "lookup failed",
			"error":   err.Error(),
		})
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
		CacheExpire:  lrg.defaultCacheExpire,
		FallbackToDB: false,
		Refresh:      false,
	}

	result, err := lrg.Service.LookupSingle(ctx, key, opts)
	if err != nil {
		if err == redis.Nil {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "key not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "lookup failed",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

func (lrg *LookupRouterGroup[T]) HandleCount(c *gin.Context) {
	var req LookupCountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request",
			"error":   err.Error(),
		})
		return
	}

	keyPattern := req.KeyPattern
	if keyPattern == "" {
		keyPattern = lrg.defaultKeyPattern
	}

	if keyPattern == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "key_pattern is required",
		})
		return
	}

	// 执行查询（只需要 keys，不需要数据）
	_, keys, err := lrg.executeLookup(
		c.Request.Context(),
		keyPattern,
		req.Filters,
		req.UseCustomFilter,
		false, // 计数不需要回源
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "count failed",
			"error":   err.Error(),
		})
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
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request",
			"error":   err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	var deletedCount int

	if req.Pattern != "" {
		// 按模式删除
		if err := lrg.Service.InvalidateCacheByPattern(ctx, req.Pattern); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "invalidate failed",
				"error":   err.Error(),
			})
			return
		}
		deletedCount = -1 // -1 表示按模式删除，无法精确统计
	} else if len(req.Keys) > 0 {
		// 按键列表删除
		if err := lrg.Service.InvalidateCache(ctx, req.Keys...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "invalidate failed",
				"error":   err.Error(),
			})
			return
		}
		deletedCount = len(req.Keys)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "either keys or pattern must be provided",
		})
		return
	}

	c.JSON(http.StatusOK, InvalidateResponse{
		Code:    0,
		Message: "success",
		Count:   deletedCount,
	})
}
