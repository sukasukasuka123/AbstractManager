package filter_translator

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

// ========== Redis 过滤器接口 ==========

// RedisFilter Redis 过滤器接口
type RedisFilter interface {
	BaseFilter
	// ApplyRedis 应用 Redis 过滤逻辑
	// 返回过滤后的 keys 和可能的错误
	ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error)
}

// RedisFilterFunc 辅助函数类型，用于过滤单个 key 的值
type RedisFilterFunc func(value string) bool

// ========== Redis Filter 实现 ==========

// RedisEqualFilter 等于过滤器
type RedisEqualFilter struct {
	*GenericFilter
}

func (f *RedisEqualFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	return applyRedisFilter(ctx, client, keys, f.Field, func(value string) bool {
		return value == fmt.Sprintf("%v", f.Value)
	})
}

// RedisNotEqualFilter 不等于过滤器
type RedisNotEqualFilter struct {
	*GenericFilter
}

func (f *RedisNotEqualFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	return applyRedisFilter(ctx, client, keys, f.Field, func(value string) bool {
		return value != fmt.Sprintf("%v", f.Value)
	})
}

// RedisGreaterThanFilter 大于过滤器
type RedisGreaterThanFilter struct {
	*GenericFilter
}

func (f *RedisGreaterThanFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	return applyRedisNumericFilter(ctx, client, keys, f.Field, func(value float64) bool {
		targetValue, _ := toFloat64(f.Value)
		return value > targetValue
	})
}

// RedisGreaterThanOrEqualFilter 大于等于过滤器
type RedisGreaterThanOrEqualFilter struct {
	*GenericFilter
}

func (f *RedisGreaterThanOrEqualFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	return applyRedisNumericFilter(ctx, client, keys, f.Field, func(value float64) bool {
		targetValue, _ := toFloat64(f.Value)
		return value >= targetValue
	})
}

// RedisLessThanFilter 小于过滤器
type RedisLessThanFilter struct {
	*GenericFilter
}

func (f *RedisLessThanFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	return applyRedisNumericFilter(ctx, client, keys, f.Field, func(value float64) bool {
		targetValue, _ := toFloat64(f.Value)
		return value < targetValue
	})
}

// RedisLessThanOrEqualFilter 小于等于过滤器
type RedisLessThanOrEqualFilter struct {
	*GenericFilter
}

func (f *RedisLessThanOrEqualFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	return applyRedisNumericFilter(ctx, client, keys, f.Field, func(value float64) bool {
		targetValue, _ := toFloat64(f.Value)
		return value <= targetValue
	})
}

// RedisLikeFilter 模糊匹配过滤器
type RedisLikeFilter struct {
	*GenericFilter
}

func (f *RedisLikeFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	searchStr := strings.ToLower(f.Value.(string))
	return applyRedisFilter(ctx, client, keys, f.Field, func(value string) bool {
		return strings.Contains(strings.ToLower(value), searchStr)
	})
}

// RedisInFilter IN 过滤器
type RedisInFilter struct {
	*GenericInFilter
}

func (f *RedisInFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	// 构建值集合用于快速查找
	valueSet := make(map[string]bool)
	for _, v := range f.Values {
		valueSet[fmt.Sprintf("%v", v)] = true
	}

	return applyRedisFilter(ctx, client, keys, f.Field, func(value string) bool {
		return valueSet[value]
	})
}

// RedisBetweenFilter BETWEEN 过滤器
type RedisBetweenFilter struct {
	*GenericBetweenFilter
}

func (f *RedisBetweenFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	minValue, _ := toFloat64(f.Min)
	maxValue, _ := toFloat64(f.Max)

	return applyRedisNumericFilter(ctx, client, keys, f.Field, func(value float64) bool {
		return value >= minValue && value <= maxValue
	})
}

// RedisIsNullFilter IS NULL 过滤器（检查字段是否不存在）
type RedisIsNullFilter struct {
	*GenericFilter
}

func (f *RedisIsNullFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	result := make([]string, 0)

	for _, key := range keys {
		exists, err := client.HExists(ctx, key, f.Field).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to check field existence for key %s: %w", key, err)
		}
		// 字段不存在才符合条件
		if !exists {
			result = append(result, key)
		}
	}

	return result, nil
}

// RedisIsNotNullFilter IS NOT NULL 过滤器（检查字段是否存在）
type RedisIsNotNullFilter struct {
	*GenericFilter
}

func (f *RedisIsNotNullFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	result := make([]string, 0)

	for _, key := range keys {
		exists, err := client.HExists(ctx, key, f.Field).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to check field existence for key %s: %w", key, err)
		}
		// 字段存在才符合条件
		if exists {
			result = append(result, key)
		}
	}

	return result, nil
}

// ========== Redis 辅助函数 ==========

// applyRedisFilter 应用 Redis 字符串过滤
func applyRedisFilter(ctx context.Context, client *redis.Client, keys []string, field string, filterFunc RedisFilterFunc) ([]string, error) {
	result := make([]string, 0)

	for _, key := range keys {
		value, err := client.HGet(ctx, key, field).Result()
		if err == redis.Nil {
			// 字段不存在，跳过
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get field %s from key %s: %w", field, key, err)
		}

		if filterFunc(value) {
			result = append(result, key)
		}
	}

	return result, nil
}

// applyRedisNumericFilter 应用 Redis 数值过滤
func applyRedisNumericFilter(ctx context.Context, client *redis.Client, keys []string, field string, filterFunc func(float64) bool) ([]string, error) {
	result := make([]string, 0)

	for _, key := range keys {
		value, err := client.HGet(ctx, key, field).Result()
		if err == redis.Nil {
			// 字段不存在，跳过
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get field %s from key %s: %w", field, key, err)
		}

		numValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			// 无法转换为数值，跳过
			continue
		}

		if filterFunc(numValue) {
			result = append(result, key)
		}
	}

	return result, nil
}

// toFloat64 将 interface{} 转换为 float64
func toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case string:
		return strconv.ParseFloat(val, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

// ========== Redis FilterTranslator 实现 ==========

// RedisEqualTranslator 等于翻译器
type RedisEqualTranslator struct{}

func (t *RedisEqualTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &RedisEqualFilter{
		GenericFilter: &GenericFilter{
			Field:    param.Field,
			Operator: "eq",
			Value:    param.Value,
		},
	}, nil
}

func (t *RedisEqualTranslator) SupportedOperator() string {
	return "eq"
}

func (t *RedisEqualTranslator) Validate(param FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	if param.Value == nil {
		return fmt.Errorf("value cannot be nil")
	}
	return nil
}

// RedisNotEqualTranslator 不等于翻译器
type RedisNotEqualTranslator struct{}

func (t *RedisNotEqualTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &RedisNotEqualFilter{
		GenericFilter: &GenericFilter{
			Field:    param.Field,
			Operator: "ne",
			Value:    param.Value,
		},
	}, nil
}

func (t *RedisNotEqualTranslator) SupportedOperator() string {
	return "ne"
}

func (t *RedisNotEqualTranslator) Validate(param FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	if param.Value == nil {
		return fmt.Errorf("value cannot be nil")
	}
	return nil
}

// RedisGreaterThanTranslator 大于翻译器
type RedisGreaterThanTranslator struct{}

func (t *RedisGreaterThanTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &RedisGreaterThanFilter{
		GenericFilter: &GenericFilter{
			Field:    param.Field,
			Operator: "gt",
			Value:    param.Value,
		},
	}, nil
}

func (t *RedisGreaterThanTranslator) SupportedOperator() string {
	return "gt"
}

func (t *RedisGreaterThanTranslator) Validate(param FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	if param.Value == nil {
		return fmt.Errorf("value cannot be nil")
	}
	return nil
}

// RedisGreaterThanOrEqualTranslator 大于等于翻译器
type RedisGreaterThanOrEqualTranslator struct{}

func (t *RedisGreaterThanOrEqualTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &RedisGreaterThanOrEqualFilter{
		GenericFilter: &GenericFilter{
			Field:    param.Field,
			Operator: "gte",
			Value:    param.Value,
		},
	}, nil
}

func (t *RedisGreaterThanOrEqualTranslator) SupportedOperator() string {
	return "gte"
}

func (t *RedisGreaterThanOrEqualTranslator) Validate(param FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	if param.Value == nil {
		return fmt.Errorf("value cannot be nil")
	}
	return nil
}

// RedisLessThanTranslator 小于翻译器
type RedisLessThanTranslator struct{}

func (t *RedisLessThanTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &RedisLessThanFilter{
		GenericFilter: &GenericFilter{
			Field:    param.Field,
			Operator: "lt",
			Value:    param.Value,
		},
	}, nil
}

func (t *RedisLessThanTranslator) SupportedOperator() string {
	return "lt"
}

func (t *RedisLessThanTranslator) Validate(param FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	if param.Value == nil {
		return fmt.Errorf("value cannot be nil")
	}
	return nil
}

// RedisLessThanOrEqualTranslator 小于等于翻译器
type RedisLessThanOrEqualTranslator struct{}

func (t *RedisLessThanOrEqualTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &RedisLessThanOrEqualFilter{
		GenericFilter: &GenericFilter{
			Field:    param.Field,
			Operator: "lte",
			Value:    param.Value,
		},
	}, nil
}

func (t *RedisLessThanOrEqualTranslator) SupportedOperator() string {
	return "lte"
}

func (t *RedisLessThanOrEqualTranslator) Validate(param FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	if param.Value == nil {
		return fmt.Errorf("value cannot be nil")
	}
	return nil
}

// RedisLikeTranslator 模糊匹配翻译器
type RedisLikeTranslator struct{}

func (t *RedisLikeTranslator) Translate(param FilterParam) (BaseFilter, error) {
	value, ok := param.Value.(string)
	if !ok {
		return nil, fmt.Errorf("value must be string for LIKE operator")
	}
	return &RedisLikeFilter{
		GenericFilter: &GenericFilter{
			Field:    param.Field,
			Operator: "like",
			Value:    value,
		},
	}, nil
}

func (t *RedisLikeTranslator) SupportedOperator() string {
	return "like"
}

func (t *RedisLikeTranslator) Validate(param FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	if _, ok := param.Value.(string); !ok {
		return fmt.Errorf("value must be string")
	}
	return nil
}

// RedisInTranslator IN 翻译器
type RedisInTranslator struct{}

func (t *RedisInTranslator) Translate(param FilterParam) (BaseFilter, error) {
	values, ok := param.Value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("value must be array for IN operator")
	}
	return &RedisInFilter{
		GenericInFilter: &GenericInFilter{
			Field:    param.Field,
			Operator: "in",
			Values:   values,
		},
	}, nil
}

func (t *RedisInTranslator) SupportedOperator() string {
	return "in"
}

func (t *RedisInTranslator) Validate(param FilterParam) error {
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

// RedisBetweenTranslator BETWEEN 翻译器
type RedisBetweenTranslator struct{}

func (t *RedisBetweenTranslator) Translate(param FilterParam) (BaseFilter, error) {
	values, ok := param.Value.([]interface{})
	if !ok || len(values) != 2 {
		return nil, fmt.Errorf("value must be array with 2 elements for BETWEEN operator")
	}
	return &RedisBetweenFilter{
		GenericBetweenFilter: &GenericBetweenFilter{
			Field:    param.Field,
			Operator: "between",
			Min:      values[0],
			Max:      values[1],
		},
	}, nil
}

func (t *RedisBetweenTranslator) SupportedOperator() string {
	return "between"
}

func (t *RedisBetweenTranslator) Validate(param FilterParam) error {
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

// RedisIsNullTranslator IS NULL 翻译器
type RedisIsNullTranslator struct{}

func (t *RedisIsNullTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &RedisIsNullFilter{
		GenericFilter: &GenericFilter{
			Field:    param.Field,
			Operator: "isnull",
			Value:    nil,
		},
	}, nil
}

func (t *RedisIsNullTranslator) SupportedOperator() string {
	return "isnull"
}

func (t *RedisIsNullTranslator) Validate(param FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	return nil
}

// RedisIsNotNullTranslator IS NOT NULL 翻译器
type RedisIsNotNullTranslator struct{}

func (t *RedisIsNotNullTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &RedisIsNotNullFilter{
		GenericFilter: &GenericFilter{
			Field:    param.Field,
			Operator: "isnotnull",
			Value:    nil,
		},
	}, nil
}

func (t *RedisIsNotNullTranslator) SupportedOperator() string {
	return "isnotnull"
}

func (t *RedisIsNotNullTranslator) Validate(param FilterParam) error {
	if param.Field == "" {
		return fmt.Errorf("field cannot be empty")
	}
	return nil
}

// ========== Redis 翻译器注册表 ==========

// RedisTranslatorRegistry Redis 翻译器注册表
type RedisTranslatorRegistry struct {
	translators map[string]FilterTranslator
}

// NewRedisTranslatorRegistry 创建 Redis 翻译器注册表
func NewRedisTranslatorRegistry() *RedisTranslatorRegistry {
	registry := &RedisTranslatorRegistry{
		translators: make(map[string]FilterTranslator),
	}

	// 注册所有 Redis 翻译器
	registry.Register(&RedisEqualTranslator{})
	registry.Register(&RedisNotEqualTranslator{})
	registry.Register(&RedisGreaterThanTranslator{})
	registry.Register(&RedisGreaterThanOrEqualTranslator{})
	registry.Register(&RedisLessThanTranslator{})
	registry.Register(&RedisLessThanOrEqualTranslator{})
	registry.Register(&RedisLikeTranslator{})
	registry.Register(&RedisInTranslator{})
	registry.Register(&RedisBetweenTranslator{})
	registry.Register(&RedisIsNullTranslator{})
	registry.Register(&RedisIsNotNullTranslator{})

	return registry
}

// Register 注册翻译器
func (r *RedisTranslatorRegistry) Register(translator FilterTranslator) {
	r.translators[translator.SupportedOperator()] = translator
}

// Translate 翻译前端参数为 Filter
func (r *RedisTranslatorRegistry) Translate(param FilterParam) (RedisFilter, error) {
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

	redisFilter, ok := baseFilter.(RedisFilter)
	if !ok {
		return nil, fmt.Errorf("translator returned non-RedisFilter")
	}

	return redisFilter, nil
}

// TranslateBatch 批量翻译
func (r *RedisTranslatorRegistry) TranslateBatch(params []FilterParam) ([]RedisFilter, error) {
	filters := make([]RedisFilter, 0, len(params))

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
func (r *RedisTranslatorRegistry) GetSupportedOperators() []string {
	operators := make([]string, 0, len(r.translators))
	for op := range r.translators {
		operators = append(operators, op)
	}
	return operators
}

// ========== Redis 工具函数 ==========

// ApplyRedisFilters 应用多个 Redis 过滤器
// 注意：需要先获取初始的 keys 列表（例如通过 KEYS 或 SCAN 命令）
func ApplyRedisFilters(ctx context.Context, client *redis.Client, initialKeys []string, filters []RedisFilter) ([]string, error) {
	keys := initialKeys

	for _, filter := range filters {
		filteredKeys, err := filter.ApplyRedis(ctx, client, keys)
		if err != nil {
			return nil, err
		}
		keys = filteredKeys
	}

	return keys, nil
}

// DefaultRedisRegistry 默认 Redis 翻译器注册表（全局单例）
var DefaultRedisRegistry = NewRedisTranslatorRegistry()
