package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// QueryOptions 查询配置选项
type QueryOptions struct {
	Page     int                    // 页码（从1开始）
	PageSize int                    // 每页数量
	OrderBy  string                 // 排序字段
	Order    string                 // 排序方向（ASC/DESC）
	Preload  []string               // 预加载关联
	Select   []string               // 指定查询字段
	Distinct bool                   // 是否去重
	Group    string                 // 分组字段
	Having   map[string]interface{} // Having 条件
}

// QueryResult 查询结果
type QueryResult[T any] struct {
	Data       []T   // 数据列表
	Total      int64 // 总数
	Page       int   // 当前页
	PageSize   int   // 每页数量
	TotalPages int   // 总页数
}

// GetQuery 条件查询（支持分页）
// queryFunc: 用于构建查询条件的 lambda 函数
func (sm *ServiceManager[T]) GetQuery(
	ctx context.Context,
	queryFunc func(*gorm.DB) *gorm.DB,
	opts *QueryOptions,
) (*QueryResult[T], error) {
	db := GetDB().WithContext(ctx)

	// 设置只读事务隔离级别（READ COMMITTED）
	db = db.Begin()
	defer func() {
		if r := recover(); r != nil {
			db.Rollback()
		}
	}()

	// 应用表名
	db = sm.applyTableName(db)

	// 应用查询条件
	if queryFunc != nil {
		db = queryFunc(db)
	}

	// 统计总数
	var total int64
	countDB := db
	if err := countDB.Model(&sm.Resource).Count(&total).Error; err != nil {
		db.Rollback()
		return nil, fmt.Errorf("failed to count records: %w", err)
	}

	// 应用查询选项
	db = sm.applyQueryOptions(db, opts)

	// 执行查询
	var results []T
	if err := db.Find(&results).Error; err != nil {
		db.Rollback()
		return nil, fmt.Errorf("failed to query records: %w", err)
	}

	// 提交只读事务
	if err := db.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 构建返回结果
	result := &QueryResult[T]{
		Data:  results,
		Total: total,
	}

	if opts != nil && opts.PageSize > 0 {
		result.Page = opts.Page
		result.PageSize = opts.PageSize
		result.TotalPages = int((total + int64(opts.PageSize) - 1) / int64(opts.PageSize))
	}

	return result, nil
}

// GetQueryWithoutTransaction 无事务的条件查询（用于高并发只读场景）
func (sm *ServiceManager[T]) GetQueryWithoutTransaction(
	ctx context.Context,
	queryFunc func(*gorm.DB) *gorm.DB,
	opts *QueryOptions,
) (*QueryResult[T], error) {
	db := GetDB().WithContext(ctx)

	// 应用表名
	db = sm.applyTableName(db)

	// 应用查询条件
	if queryFunc != nil {
		db = queryFunc(db)
	}

	// 统计总数
	var total int64
	countDB := db
	if err := countDB.Model(&sm.Resource).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count records: %w", err)
	}

	// 应用查询选项
	db = sm.applyQueryOptions(db, opts)

	// 执行查询
	var results []T
	if err := db.Find(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}

	// 构建返回结果
	result := &QueryResult[T]{
		Data:  results,
		Total: total,
	}

	if opts != nil && opts.PageSize > 0 {
		result.Page = opts.Page
		result.PageSize = opts.PageSize
		result.TotalPages = int((total + int64(opts.PageSize) - 1) / int64(opts.PageSize))
	}

	return result, nil
}

// applyTableName 应用表名
func (sm *ServiceManager[T]) applyTableName(db *gorm.DB) *gorm.DB {
	tableName := sm.TableName
	if sm.Schema != "" && sm.Schema != "public" {
		tableName = fmt.Sprintf("%s.%s", sm.Schema, sm.TableName)
	}
	return db.Table(tableName)
}

// applyQueryOptions 应用查询选项
func (sm *ServiceManager[T]) applyQueryOptions(db *gorm.DB, opts *QueryOptions) *gorm.DB {
	if opts == nil {
		return db
	}

	// 应用字段选择
	if len(opts.Select) > 0 {
		db = db.Select(opts.Select)
	}

	// 应用去重
	if opts.Distinct {
		db = db.Distinct()
	}

	// 应用分组
	if opts.Group != "" {
		db = db.Group(opts.Group)
	}

	// 应用 Having 条件
	if len(opts.Having) > 0 {
		for key, value := range opts.Having {
			db = db.Having(key, value)
		}
	}

	// 应用排序
	if opts.OrderBy != "" {
		order := "ASC"
		if opts.Order != "" {
			order = opts.Order
		}
		db = db.Order(fmt.Sprintf("%s %s", opts.OrderBy, order))
	}

	// 应用预加载
	for _, preload := range opts.Preload {
		db = db.Preload(preload)
	}

	// 应用分页
	if opts.PageSize > 0 {
		page := opts.Page
		if page < 1 {
			page = 1
		}
		offset := (page - 1) * opts.PageSize
		db = db.Offset(offset).Limit(opts.PageSize)
	}

	return db
}

// CountQuery 条件计数
func (sm *ServiceManager[T]) CountQuery(
	ctx context.Context,
	queryFunc func(*gorm.DB) *gorm.DB,
) (int64, error) {
	db := GetDB().WithContext(ctx)

	// 应用表名
	db = sm.applyTableName(db)

	// 应用查询条件
	if queryFunc != nil {
		db = queryFunc(db)
	}

	var count int64
	if err := db.Model(&sm.Resource).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count records: %w", err)
	}

	return count, nil
}

// ExistsQuery 检查是否存在满足条件的记录
func (sm *ServiceManager[T]) ExistsQuery(
	ctx context.Context,
	queryFunc func(*gorm.DB) *gorm.DB,
) (bool, error) {
	count, err := sm.CountQuery(ctx, queryFunc)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
