package http_router

import (
	"fmt"
	"net/http"
	"time"

	"AbstractManager/service"
	"AbstractManager/util/cache_key_builder"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ========== 缓存写入请求/响应结构 ==========

// WritedownSingleRequest 单个缓存写入请求
type WritedownSingleRequest[T any] struct {
	Key        string        `json:"key"`                  // 缓存键
	Data       *T            `json:"data,omitempty"`       // 数据(直接提供)
	ID         interface{}   `json:"id,omitempty"`         // 或通过ID从数据库加载
	Expiration time.Duration `json:"expiration,omitempty"` // 过期时间(秒),默认3600
	Overwrite  bool          `json:"overwrite"`            // 是否覆盖,默认true
	NX         bool          `json:"nx,omitempty"`         // 仅当键不存在时设置
	XX         bool          `json:"xx,omitempty"`         // 仅当键存在时设置
	Async      bool          `json:"async,omitempty"`      // 是否异步写入
}

// WritedownQueryRequest 批量缓存写入请求
type WritedownQueryRequest[T any] struct {
	Data        []T           `json:"data,omitempty"`         // 数据列表(直接提供)
	IDs         []interface{} `json:"ids,omitempty"`          // 或通过ID列表从数据库加载
	LoadAll     bool          `json:"load_all,omitempty"`     // 是否加载全部数据
	KeyTemplate string        `json:"key_template"`           // 键模板,如"cache:user:{id}"
	Expiration  time.Duration `json:"expiration,omitempty"`   // 过期时间(秒),默认3600
	BatchSize   int           `json:"batch_size,omitempty"`   // 批次大小,默认100
	Overwrite   bool          `json:"overwrite"`              // 是否覆盖,默认true
	UsePipeline bool          `json:"use_pipeline,omitempty"` // 是否使用Pipeline(大数据量)
	Incremental bool          `json:"incremental,omitempty"`  // 是否增量更新
}

// WritedownWithLockRequest 带锁的缓存写入请求
type WritedownWithLockRequest struct {
	Key         string        `json:"key"`                    // 缓存键
	ID          interface{}   `json:"id"`                     // 数据库ID
	Expiration  time.Duration `json:"expiration,omitempty"`   // 过期时间(秒),默认3600
	LockTimeout time.Duration `json:"lock_timeout,omitempty"` // 锁超时时间(秒),默认5
}

// WritedownWithVersionRequest 带版本控制的缓存写入请求
type WritedownWithVersionRequest[T any] struct {
	Key        string        `json:"key"`                  // 缓存键
	Data       *T            `json:"data"`                 // 数据
	Version    int64         `json:"version"`              // 版本号
	Expiration time.Duration `json:"expiration,omitempty"` // 过期时间(秒),默认3600
}

// WarmupCacheRequest 缓存预热请求
type WarmupCacheRequest struct {
	KeyTemplate string        `json:"key_template"`         // 键模板
	Limit       int           `json:"limit,omitempty"`      // 预热数量,默认1000
	OrderBy     string        `json:"order_by,omitempty"`   // 排序字段,默认"access_count"
	Expiration  time.Duration `json:"expiration,omitempty"` // 过期时间(秒),默认3600
}

// RefreshCacheRequest 缓存刷新请求
type RefreshCacheRequest struct {
	Key        string        `json:"key"`                  // 缓存键
	ID         interface{}   `json:"id"`                   // 数据库ID
	Expiration time.Duration `json:"expiration,omitempty"` // 过期时间(秒),默认3600
}

// WritedownResponse 缓存写入响应
type WritedownResponse[T any] struct {
	Code         int    `json:"code"`
	Message      string `json:"message"`
	ItemsWritten int    `json:"items_written,omitempty"` // 写入的条目数
	Data         *T     `json:"data,omitempty"`          // 返回的数据(带锁查询时)
}

// ========== 缓存写入路由组 ==========

type WritedownRouterGroup[T any] struct {
	RouterGroup *gin.RouterGroup
	Service     *service.ServiceManager[T]
	KeyBuilder  cache_key_builder.KeyBuilder[T] // 键构建器
}

// WritedownRouterConfig 路由配置选项
type WritedownRouterConfig[T any] struct {
	KeyBuilder cache_key_builder.KeyBuilder[T] // 可选的自定义键构建器
}

// NewWritedownRouterGroup 创建缓存写入路由组
func NewWritedownRouterGroup[T any](
	rg *gin.RouterGroup,
	service *service.ServiceManager[T],
	config ...*WritedownRouterConfig[T],
) *WritedownRouterGroup[T] {
	wdg := &WritedownRouterGroup[T]{
		RouterGroup: rg,
		Service:     service,
	}

	// 如果提供了配置，使用自定义键构建器
	if len(config) > 0 && config[0] != nil && config[0].KeyBuilder != nil {
		wdg.KeyBuilder = config[0].KeyBuilder
	}

	return wdg
}

// SetKeyBuilder 设置键构建器
func (wdg *WritedownRouterGroup[T]) SetKeyBuilder(builder cache_key_builder.KeyBuilder[T]) {
	wdg.KeyBuilder = builder
}

// ========== 路由注册 ==========

func (wdg *WritedownRouterGroup[T]) RegisterRoutes(basePath string) {
	// 单个缓存写入
	wdg.RouterGroup.POST(basePath+"/cache/write", wdg.HandleWritedownSingle)
	wdg.RouterGroup.POST(basePath+"/cache/write-lock", wdg.HandleWritedownWithLock)
	wdg.RouterGroup.POST(basePath+"/cache/write-version", wdg.HandleWritedownWithVersion)
	wdg.RouterGroup.POST(basePath+"/cache/refresh", wdg.HandleRefreshCache)

	// 批量缓存写入
	wdg.RouterGroup.POST(basePath+"/cache/batch-write", wdg.HandleWritedownQuery)
	wdg.RouterGroup.POST(basePath+"/cache/warmup", wdg.HandleWarmupCache)
}

// ========== 单个缓存写入处理器 ==========

// HandleWritedownSingle 处理单个缓存写入
func (wdg *WritedownRouterGroup[T]) HandleWritedownSingle(c *gin.Context) {
	var req WritedownSingleRequest[T]
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, WritedownResponse[T]{
			Code:    400,
			Message: fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	// 验证参数
	if req.Key == "" {
		c.JSON(http.StatusBadRequest, WritedownResponse[T]{
			Code:    400,
			Message: "key cannot be empty",
		})
		return
	}

	// 设置默认值
	if req.Expiration == 0 {
		req.Expiration = 1 * time.Hour
	}

	var data *T
	var err error

	// 获取数据
	if req.Data != nil {
		data = req.Data
	} else if req.ID != nil {
		// 从数据库加载
		data, err = wdg.Service.GetSingle(c.Request.Context(), func(db *gorm.DB) *gorm.DB {
			return db.Where("id = ?", req.ID)
		}, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, WritedownResponse[T]{
				Code:    500,
				Message: fmt.Sprintf("failed to load data: %v", err),
			})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, WritedownResponse[T]{
			Code:    400,
			Message: "either data or id must be provided",
		})
		return
	}

	opts := &service.WritedownSingleOptions{
		Expiration: req.Expiration,
		Overwrite:  req.Overwrite,
		NX:         req.NX,
		XX:         req.XX,
	}

	// 异步写入
	if req.Async {
		wdg.Service.WritedownSingleAsync(c.Request.Context(), req.Key, data, req.Expiration)
		c.JSON(http.StatusOK, WritedownResponse[T]{
			Code:    0,
			Message: "async write initiated",
		})
		return
	}

	// 同步写入
	if err := wdg.Service.WritedownSingle(c.Request.Context(), req.Key, data, opts); err != nil {
		c.JSON(http.StatusInternalServerError, WritedownResponse[T]{
			Code:    500,
			Message: fmt.Sprintf("writedown failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, WritedownResponse[T]{
		Code:         0,
		Message:      "success",
		ItemsWritten: 1,
	})
}

// HandleWritedownWithLock 处理带锁的缓存写入
func (wdg *WritedownRouterGroup[T]) HandleWritedownWithLock(c *gin.Context) {
	var req WritedownWithLockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, WritedownResponse[T]{
			Code:    400,
			Message: fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	if req.Key == "" || req.ID == nil {
		c.JSON(http.StatusBadRequest, WritedownResponse[T]{
			Code:    400,
			Message: "key and id cannot be empty",
		})
		return
	}

	// 设置默认值
	if req.Expiration == 0 {
		req.Expiration = 1 * time.Hour
	}
	if req.LockTimeout == 0 {
		req.LockTimeout = 5 * time.Second
	}

	queryFunc := func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", req.ID)
	}

	data, err := wdg.Service.WritedownSingleWithLock(
		c.Request.Context(),
		req.Key,
		queryFunc,
		req.Expiration,
		req.LockTimeout,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, WritedownResponse[T]{
			Code:    500,
			Message: fmt.Sprintf("writedown with lock failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, WritedownResponse[T]{
		Code:         0,
		Message:      "success",
		Data:         data,
		ItemsWritten: 1,
	})
}

// HandleWritedownWithVersion 处理带版本控制的缓存写入
func (wdg *WritedownRouterGroup[T]) HandleWritedownWithVersion(c *gin.Context) {
	var req WritedownWithVersionRequest[T]
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, WritedownResponse[T]{
			Code:    400,
			Message: fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	if req.Key == "" || req.Data == nil {
		c.JSON(http.StatusBadRequest, WritedownResponse[T]{
			Code:    400,
			Message: "key and data cannot be empty",
		})
		return
	}

	// 设置默认值
	if req.Expiration == 0 {
		req.Expiration = 1 * time.Hour
	}

	err := wdg.Service.WritedownSingleWithVersion(
		c.Request.Context(),
		req.Key,
		req.Data,
		req.Version,
		req.Expiration,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, WritedownResponse[T]{
			Code:    500,
			Message: fmt.Sprintf("writedown with version failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, WritedownResponse[T]{
		Code:         0,
		Message:      "success",
		ItemsWritten: 1,
	})
}

// HandleRefreshCache 处理缓存刷新
func (wdg *WritedownRouterGroup[T]) HandleRefreshCache(c *gin.Context) {
	var req RefreshCacheRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, WritedownResponse[T]{
			Code:    400,
			Message: fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	if req.Key == "" || req.ID == nil {
		c.JSON(http.StatusBadRequest, WritedownResponse[T]{
			Code:    400,
			Message: "key and id cannot be empty",
		})
		return
	}

	// 设置默认值
	if req.Expiration == 0 {
		req.Expiration = 1 * time.Hour
	}

	queryFunc := func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", req.ID)
	}

	err := wdg.Service.RefreshSingleCacheFromDB(
		c.Request.Context(),
		req.Key,
		queryFunc,
		req.Expiration,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, WritedownResponse[T]{
			Code:    500,
			Message: fmt.Sprintf("refresh cache failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, WritedownResponse[T]{
		Code:         0,
		Message:      "success",
		ItemsWritten: 1,
	})
}

// ========== 批量缓存写入处理器 ==========

// HandleWritedownQuery 处理批量缓存写入
func (wdg *WritedownRouterGroup[T]) HandleWritedownQuery(c *gin.Context) {
	var req WritedownQueryRequest[T]
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, WritedownResponse[T]{
			Code:    400,
			Message: fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	if req.KeyTemplate == "" {
		c.JSON(http.StatusBadRequest, WritedownResponse[T]{
			Code:    400,
			Message: "key_template cannot be empty",
		})
		return
	}

	// 设置默认值
	if req.Expiration == 0 {
		req.Expiration = 1 * time.Hour
	}
	if req.BatchSize == 0 {
		req.BatchSize = 100
	}

	// 🔑 使用工具包构建键生成函数
	buildKeyFunc := wdg.buildKeyFuncFromTemplate(req.KeyTemplate)

	var data []T
	var err error

	// 获取数据
	if len(req.Data) > 0 {
		data = req.Data
	} else if len(req.IDs) > 0 {
		// 从数据库加载指定ID的数据
		err = wdg.Service.WritedownQueryByIDs(
			c.Request.Context(),
			req.IDs,
			buildKeyFunc,
			&service.WritedownQueryOptions{
				Expiration: req.Expiration,
				BatchSize:  req.BatchSize,
				Overwrite:  req.Overwrite,
			},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, WritedownResponse[T]{
				Code:    500,
				Message: fmt.Sprintf("writedown by ids failed: %v", err),
			})
			return
		}

		c.JSON(http.StatusOK, WritedownResponse[T]{
			Code:         0,
			Message:      "success",
			ItemsWritten: len(req.IDs),
		})
		return
	} else if req.LoadAll {
		// 加载全部数据
		err = wdg.Service.WritedownAllToCache(
			c.Request.Context(),
			buildKeyFunc,
			&service.WritedownQueryOptions{
				Expiration: req.Expiration,
				BatchSize:  req.BatchSize,
				Overwrite:  req.Overwrite,
			},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, WritedownResponse[T]{
				Code:    500,
				Message: fmt.Sprintf("writedown all failed: %v", err),
			})
			return
		}

		c.JSON(http.StatusOK, WritedownResponse[T]{
			Code:    0,
			Message: "success",
		})
		return
	} else {
		c.JSON(http.StatusBadRequest, WritedownResponse[T]{
			Code:    400,
			Message: "either data, ids, or load_all must be provided",
		})
		return
	}

	opts := &service.WritedownQueryOptions{
		Expiration: req.Expiration,
		BatchSize:  req.BatchSize,
		Overwrite:  req.Overwrite,
	}

	// 选择写入方式
	if req.UsePipeline {
		err = wdg.Service.WritedownWithPipeline(c.Request.Context(), data, buildKeyFunc, opts)
	} else if req.Incremental {
		// 增量更新需要比较函数，这里简化处理
		err = wdg.Service.WritedownIncremental(c.Request.Context(), data, buildKeyFunc, nil, opts)
	} else {
		err = wdg.Service.WritedownQuery(c.Request.Context(), data, buildKeyFunc, opts)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, WritedownResponse[T]{
			Code:    500,
			Message: fmt.Sprintf("writedown query failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, WritedownResponse[T]{
		Code:         0,
		Message:      "success",
		ItemsWritten: len(data),
	})
}

// HandleWarmupCache 处理缓存预热
func (wdg *WritedownRouterGroup[T]) HandleWarmupCache(c *gin.Context) {
	var req WarmupCacheRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, WritedownResponse[T]{
			Code:    400,
			Message: fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	if req.KeyTemplate == "" {
		c.JSON(http.StatusBadRequest, WritedownResponse[T]{
			Code:    400,
			Message: "key_template cannot be empty",
		})
		return
	}

	// 设置默认值
	if req.Expiration == 0 {
		req.Expiration = 1 * time.Hour
	}
	if req.Limit == 0 {
		req.Limit = 1000
	}
	if req.OrderBy == "" {
		req.OrderBy = "access_count"
	}

	// 🔑 使用工具包构建键生成函数
	buildKeyFunc := wdg.buildKeyFuncFromTemplate(req.KeyTemplate)

	queryFunc := func(db *gorm.DB) *gorm.DB {
		return db.Order(fmt.Sprintf("%s DESC", req.OrderBy)).Limit(req.Limit)
	}

	err := wdg.Service.WarmupCache(
		c.Request.Context(),
		queryFunc,
		buildKeyFunc,
		req.Expiration,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, WritedownResponse[T]{
			Code:    500,
			Message: fmt.Sprintf("warmup cache failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, WritedownResponse[T]{
		Code:    0,
		Message: "success",
	})
}

// ========== 辅助方法 ==========

// buildKeyFuncFromTemplate 根据模板构建键生成函数
// 优先使用自定义 KeyBuilder，否则使用默认的模板构建器
func (wdg *WritedownRouterGroup[T]) buildKeyFuncFromTemplate(template string) func(*T) string {
	// 如果设置了自定义 KeyBuilder，使用它
	if wdg.KeyBuilder != nil {
		return cache_key_builder.BuildKeyFunc(wdg.KeyBuilder)
	}

	// 否则使用默认的模板构建器
	builder := cache_key_builder.NewTemplateKeyBuilder[T](template)
	return cache_key_builder.BuildKeyFunc[T](builder)
}
