package filter_translator

import (
	"fmt"

	"gorm.io/gorm"
)

// ========== GORM 过滤器接口 ==========

// GormFilter GORM 过滤器接口
type GormFilter interface {
	BaseFilter
	ApplyGorm(db *gorm.DB) *gorm.DB
}

// ========== GORM Filter 实现 ==========

// GormEqualFilter 等于过滤器
type GormEqualFilter struct {
	*GenericFilter
}

func (f *GormEqualFilter) ApplyGorm(db *gorm.DB) *gorm.DB {
	return db.Where(fmt.Sprintf("%s = ?", f.Field), f.Value)
}

// GormNotEqualFilter 不等于过滤器
type GormNotEqualFilter struct {
	*GenericFilter
}

func (f *GormNotEqualFilter) ApplyGorm(db *gorm.DB) *gorm.DB {
	return db.Where(fmt.Sprintf("%s != ?", f.Field), f.Value)
}

// GormGreaterThanFilter 大于过滤器
type GormGreaterThanFilter struct {
	*GenericFilter
}

func (f *GormGreaterThanFilter) ApplyGorm(db *gorm.DB) *gorm.DB {
	return db.Where(fmt.Sprintf("%s > ?", f.Field), f.Value)
}

// GormGreaterThanOrEqualFilter 大于等于过滤器
type GormGreaterThanOrEqualFilter struct {
	*GenericFilter
}

func (f *GormGreaterThanOrEqualFilter) ApplyGorm(db *gorm.DB) *gorm.DB {
	return db.Where(fmt.Sprintf("%s >= ?", f.Field), f.Value)
}

// GormLessThanFilter 小于过滤器
type GormLessThanFilter struct {
	*GenericFilter
}

func (f *GormLessThanFilter) ApplyGorm(db *gorm.DB) *gorm.DB {
	return db.Where(fmt.Sprintf("%s < ?", f.Field), f.Value)
}

// GormLessThanOrEqualFilter 小于等于过滤器
type GormLessThanOrEqualFilter struct {
	*GenericFilter
}

func (f *GormLessThanOrEqualFilter) ApplyGorm(db *gorm.DB) *gorm.DB {
	return db.Where(fmt.Sprintf("%s <= ?", f.Field), f.Value)
}

// GormLikeFilter 模糊匹配过滤器
type GormLikeFilter struct {
	*GenericFilter
}

func (f *GormLikeFilter) ApplyGorm(db *gorm.DB) *gorm.DB {
	return db.Where(fmt.Sprintf("%s LIKE ?", f.Field), "%"+f.Value.(string)+"%")
}

// GormInFilter IN 过滤器
type GormInFilter struct {
	*GenericInFilter
}

func (f *GormInFilter) ApplyGorm(db *gorm.DB) *gorm.DB {
	return db.Where(fmt.Sprintf("%s IN ?", f.Field), f.Values)
}

// GormBetweenFilter BETWEEN 过滤器
type GormBetweenFilter struct {
	*GenericBetweenFilter
}

func (f *GormBetweenFilter) ApplyGorm(db *gorm.DB) *gorm.DB {
	return db.Where(fmt.Sprintf("%s BETWEEN ? AND ?", f.Field), f.Min, f.Max)
}

// GormIsNullFilter IS NULL 过滤器
type GormIsNullFilter struct {
	*GenericFilter
}

func (f *GormIsNullFilter) ApplyGorm(db *gorm.DB) *gorm.DB {
	return db.Where(fmt.Sprintf("%s IS NULL", f.Field))
}

// GormIsNotNullFilter IS NOT NULL 过滤器
type GormIsNotNullFilter struct {
	*GenericFilter
}

func (f *GormIsNotNullFilter) ApplyGorm(db *gorm.DB) *gorm.DB {
	return db.Where(fmt.Sprintf("%s IS NOT NULL", f.Field))
}

// ========== GORM FilterTranslator 实现 ==========

// GormEqualTranslator 等于翻译器
type GormEqualTranslator struct{}

func (t *GormEqualTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &GormEqualFilter{
		GenericFilter: &GenericFilter{
			Field:    param.Field,
			Operator: "eq",
			Value:    param.Value,
		},
	}, nil
}

func (t *GormEqualTranslator) SupportedOperator() string {
	return "eq"
}

func (t *GormEqualTranslator) Validate(param FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	if param.Value == nil {
		return fmt.Errorf("value cannot be nil")
	}
	return nil
}

// GormNotEqualTranslator 不等于翻译器
type GormNotEqualTranslator struct{}

func (t *GormNotEqualTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &GormNotEqualFilter{
		GenericFilter: &GenericFilter{
			Field:    param.Field,
			Operator: "ne",
			Value:    param.Value,
		},
	}, nil
}

func (t *GormNotEqualTranslator) SupportedOperator() string {
	return "ne"
}

func (t *GormNotEqualTranslator) Validate(param FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	if param.Value == nil {
		return fmt.Errorf("value cannot be nil")
	}
	return nil
}

// GormGreaterThanTranslator 大于翻译器
type GormGreaterThanTranslator struct{}

func (t *GormGreaterThanTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &GormGreaterThanFilter{
		GenericFilter: &GenericFilter{
			Field:    param.Field,
			Operator: "gt",
			Value:    param.Value,
		},
	}, nil
}

func (t *GormGreaterThanTranslator) SupportedOperator() string {
	return "gt"
}

func (t *GormGreaterThanTranslator) Validate(param FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	if param.Value == nil {
		return fmt.Errorf("value cannot be nil")
	}
	return nil
}

// GormGreaterThanOrEqualTranslator 大于等于翻译器
type GormGreaterThanOrEqualTranslator struct{}

func (t *GormGreaterThanOrEqualTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &GormGreaterThanOrEqualFilter{
		GenericFilter: &GenericFilter{
			Field:    param.Field,
			Operator: "gte",
			Value:    param.Value,
		},
	}, nil
}

func (t *GormGreaterThanOrEqualTranslator) SupportedOperator() string {
	return "gte"
}

func (t *GormGreaterThanOrEqualTranslator) Validate(param FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	if param.Value == nil {
		return fmt.Errorf("value cannot be nil")
	}
	return nil
}

// GormLessThanTranslator 小于翻译器
type GormLessThanTranslator struct{}

func (t *GormLessThanTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &GormLessThanFilter{
		GenericFilter: &GenericFilter{
			Field:    param.Field,
			Operator: "lt",
			Value:    param.Value,
		},
	}, nil
}

func (t *GormLessThanTranslator) SupportedOperator() string {
	return "lt"
}

func (t *GormLessThanTranslator) Validate(param FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	if param.Value == nil {
		return fmt.Errorf("value cannot be nil")
	}
	return nil
}

// GormLessThanOrEqualTranslator 小于等于翻译器
type GormLessThanOrEqualTranslator struct{}

func (t *GormLessThanOrEqualTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &GormLessThanOrEqualFilter{
		GenericFilter: &GenericFilter{
			Field:    param.Field,
			Operator: "lte",
			Value:    param.Value,
		},
	}, nil
}

func (t *GormLessThanOrEqualTranslator) SupportedOperator() string {
	return "lte"
}

func (t *GormLessThanOrEqualTranslator) Validate(param FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	if param.Value == nil {
		return fmt.Errorf("value cannot be nil")
	}
	return nil
}

// GormLikeTranslator 模糊匹配翻译器
type GormLikeTranslator struct{}

func (t *GormLikeTranslator) Translate(param FilterParam) (BaseFilter, error) {
	value, ok := param.Value.(string)
	if !ok {
		return nil, fmt.Errorf("value must be string for LIKE operator")
	}
	return &GormLikeFilter{
		GenericFilter: &GenericFilter{
			Field:    param.Field,
			Operator: "like",
			Value:    value,
		},
	}, nil
}

func (t *GormLikeTranslator) SupportedOperator() string {
	return "like"
}

func (t *GormLikeTranslator) Validate(param FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	if _, ok := param.Value.(string); !ok {
		return fmt.Errorf("value must be string")
	}
	return nil
}

// GormInTranslator IN 翻译器
type GormInTranslator struct{}

func (t *GormInTranslator) Translate(param FilterParam) (BaseFilter, error) {
	values, ok := param.Value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("value must be array for IN operator")
	}
	return &GormInFilter{
		GenericInFilter: &GenericInFilter{
			Field:    param.Field,
			Operator: "in",
			Values:   values,
		},
	}, nil
}

func (t *GormInTranslator) SupportedOperator() string {
	return "in"
}

func (t *GormInTranslator) Validate(param FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	values, ok := param.Value.([]interface{})
	if !ok {
		return fmt.Errorf("value must be array")
	}
	if len(values) == 0 {
		return fmt.Errorf("value array cannot be empty")
	}
	return nil
}

// GormBetweenTranslator BETWEEN 翻译器
type GormBetweenTranslator struct{}

func (t *GormBetweenTranslator) Translate(param FilterParam) (BaseFilter, error) {
	values, ok := param.Value.([]interface{})
	if !ok || len(values) != 2 {
		return nil, fmt.Errorf("value must be array with 2 elements for BETWEEN operator")
	}
	return &GormBetweenFilter{
		GenericBetweenFilter: &GenericBetweenFilter{
			Field:    param.Field,
			Operator: "between",
			Min:      values[0],
			Max:      values[1],
		},
	}, nil
}

func (t *GormBetweenTranslator) SupportedOperator() string {
	return "between"
}

func (t *GormBetweenTranslator) Validate(param FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	values, ok := param.Value.([]interface{})
	if !ok {
		return fmt.Errorf("value must be array")
	}
	if len(values) != 2 {
		return fmt.Errorf("value array must contain exactly 2 elements")
	}
	return nil
}

// GormIsNullTranslator IS NULL 翻译器
type GormIsNullTranslator struct{}

func (t *GormIsNullTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &GormIsNullFilter{
		GenericFilter: &GenericFilter{
			Field:    param.Field,
			Operator: "isnull",
			Value:    nil,
		},
	}, nil
}

func (t *GormIsNullTranslator) SupportedOperator() string {
	return "isnull"
}

func (t *GormIsNullTranslator) Validate(param FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	return nil
}

// GormIsNotNullTranslator IS NOT NULL 翻译器
type GormIsNotNullTranslator struct{}

func (t *GormIsNotNullTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &GormIsNotNullFilter{
		GenericFilter: &GenericFilter{
			Field:    param.Field,
			Operator: "isnotnull",
			Value:    nil,
		},
	}, nil
}

func (t *GormIsNotNullTranslator) SupportedOperator() string {
	return "isnotnull"
}

func (t *GormIsNotNullTranslator) Validate(param FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	return nil
}

// ========== GORM 翻译器注册表 ==========

// GormTranslatorRegistry GORM 翻译器注册表
type GormTranslatorRegistry struct {
	translators map[string]FilterTranslator
}

// NewGormTranslatorRegistry 创建 GORM 翻译器注册表
func NewGormTranslatorRegistry() *GormTranslatorRegistry {
	registry := &GormTranslatorRegistry{
		translators: make(map[string]FilterTranslator),
	}

	// 注册所有 GORM 翻译器
	registry.Register(&GormEqualTranslator{})
	registry.Register(&GormNotEqualTranslator{})
	registry.Register(&GormGreaterThanTranslator{})
	registry.Register(&GormGreaterThanOrEqualTranslator{})
	registry.Register(&GormLessThanTranslator{})
	registry.Register(&GormLessThanOrEqualTranslator{})
	registry.Register(&GormLikeTranslator{})
	registry.Register(&GormInTranslator{})
	registry.Register(&GormBetweenTranslator{})
	registry.Register(&GormIsNullTranslator{})
	registry.Register(&GormIsNotNullTranslator{})

	return registry
}

// Register 注册翻译器
func (r *GormTranslatorRegistry) Register(translator FilterTranslator) {
	r.translators[translator.SupportedOperator()] = translator
}

// Translate 翻译前端参数为 Filter
func (r *GormTranslatorRegistry) Translate(param FilterParam) (GormFilter, error) {
	translator, ok := r.translators[param.Operator]
	if !ok {
		return nil, fmt.Errorf("unsupported operator: %s", param.Operator)
	}

	// 验证参数
	if err := translator.Validate(param); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 翻译为 Filter
	baseFilter, err := translator.Translate(param)
	if err != nil {
		return nil, err
	}

	gormFilter, ok := baseFilter.(GormFilter)
	if !ok {
		return nil, fmt.Errorf("translator returned non-GormFilter")
	}

	return gormFilter, nil
}

// TranslateBatch 批量翻译
func (r *GormTranslatorRegistry) TranslateBatch(params []FilterParam) ([]GormFilter, error) {
	filters := make([]GormFilter, 0, len(params))

	for _, param := range params {
		filter, err := r.Translate(param)
		if err != nil {
			return nil, err
		}
		filters = append(filters, filter)
	}

	return filters, nil
}

// GetSupportedOperators 获取所有支持的操作符
func (r *GormTranslatorRegistry) GetSupportedOperators() []string {
	operators := make([]string, 0, len(r.translators))
	for op := range r.translators {
		operators = append(operators, op)
	}
	return operators
}

// ========== GORM 工具函数 ==========

// ApplyGormFilters 应用多个 GORM 过滤器
func ApplyGormFilters(db *gorm.DB, filters []GormFilter) *gorm.DB {
	for _, filter := range filters {
		db = filter.ApplyGorm(db)
	}
	return db
}

// DefaultGormRegistry 默认 GORM 翻译器注册表（全局单例）
var DefaultGormRegistry = NewGormTranslatorRegistry()
