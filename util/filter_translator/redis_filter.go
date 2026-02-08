package filter_translator

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

// ========== Redis 过滤器接口 ==========

type RedisFilter interface {
	BaseFilter
	// ApplyRedis 应用 Redis 过滤逻辑，返回过滤后的 keys
	ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error)
}

// RedisFilterFunc 辅助函数类型
type RedisFilterFunc func(value interface{}) bool

// ========== Redis 核心辅助函数 (性能优化版) ==========

// applyRedisBatchFilter 通用批量过滤器：使用 MGET 获取数据并在内存中解析 JSON
func applyRedisBatchFilter(ctx context.Context, client *redis.Client, keys []string, field string, filterFunc RedisFilterFunc) ([]string, error) {
	if len(keys) == 0 {
		return []string{}, nil
	}

	// 1. 使用 MGET 一次性获取所有 String 值 (解决 WRONGTYPE 问题)
	values, err := client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("redis MGET failed: %w", err)
	}

	result := make([]string, 0)
	for i, val := range values {
		if val == nil {
			continue // Key 不存在
		}

		jsonStr, ok := val.(string)
		if !ok {
			continue // 类型不是 string，跳过
		}

		// 2. 解析 JSON
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			continue // JSON 格式错误
		}

		// 3. 提取字段逻辑：优先提取顶层，若无则看是否在 data 嵌套里
		fieldVal, exists := data[field]
		if !exists {
			if innerData, ok := data["data"].(map[string]interface{}); ok {
				fieldVal, exists = innerData[field]
			}
		}

		if exists && filterFunc(fieldVal) {
			result = append(result, keys[i])
		}
	}
	return result, nil
}

// ========== Redis Filter 具体实现 ==========

type RedisEqualFilter struct{ *GenericFilter }

func (f *RedisEqualFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	return applyRedisBatchFilter(ctx, client, keys, f.Field, func(val interface{}) bool {
		return fmt.Sprintf("%v", val) == fmt.Sprintf("%v", f.Value)
	})
}

type RedisNotEqualFilter struct{ *GenericFilter }

func (f *RedisNotEqualFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	return applyRedisBatchFilter(ctx, client, keys, f.Field, func(val interface{}) bool {
		return fmt.Sprintf("%v", val) != fmt.Sprintf("%v", f.Value)
	})
}

type RedisGreaterThanFilter struct{ *GenericFilter }

func (f *RedisGreaterThanFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	target, _ := toFloat64(f.Value)
	return applyRedisBatchFilter(ctx, client, keys, f.Field, func(val interface{}) bool {
		v, err := toFloat64(val)
		return err == nil && v > target
	})
}

type RedisGreaterThanOrEqualFilter struct{ *GenericFilter }

func (f *RedisGreaterThanOrEqualFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	target, _ := toFloat64(f.Value)
	return applyRedisBatchFilter(ctx, client, keys, f.Field, func(val interface{}) bool {
		v, err := toFloat64(val)
		return err == nil && v >= target
	})
}

type RedisLessThanFilter struct{ *GenericFilter }

func (f *RedisLessThanFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	target, _ := toFloat64(f.Value)
	return applyRedisBatchFilter(ctx, client, keys, f.Field, func(val interface{}) bool {
		v, err := toFloat64(val)
		return err == nil && v < target
	})
}

type RedisLessThanOrEqualFilter struct{ *GenericFilter }

func (f *RedisLessThanOrEqualFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	target, _ := toFloat64(f.Value)
	return applyRedisBatchFilter(ctx, client, keys, f.Field, func(val interface{}) bool {
		v, err := toFloat64(val)
		return err == nil && v <= target
	})
}

type RedisLikeFilter struct{ *GenericFilter }

func (f *RedisLikeFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	search := strings.ToLower(f.Value.(string))
	return applyRedisBatchFilter(ctx, client, keys, f.Field, func(val interface{}) bool {
		return strings.Contains(strings.ToLower(fmt.Sprintf("%v", val)), search)
	})
}

type RedisInFilter struct{ *GenericInFilter }

func (f *RedisInFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	valueSet := make(map[string]bool)
	for _, v := range f.Values {
		valueSet[fmt.Sprintf("%v", v)] = true
	}
	return applyRedisBatchFilter(ctx, client, keys, f.Field, func(val interface{}) bool {
		return valueSet[fmt.Sprintf("%v", val)]
	})
}

type RedisBetweenFilter struct{ *GenericBetweenFilter }

func (f *RedisBetweenFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	minV, _ := toFloat64(f.Min)
	maxV, _ := toFloat64(f.Max)
	return applyRedisBatchFilter(ctx, client, keys, f.Field, func(val interface{}) bool {
		v, err := toFloat64(val)
		return err == nil && v >= minV && v <= maxV
	})
}

type RedisIsNullFilter struct{ *GenericFilter }

func (f *RedisIsNullFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	return applyRedisBatchFilter(ctx, client, keys, f.Field, func(val interface{}) bool {
		return val == nil
	})
}

type RedisIsNotNullFilter struct{ *GenericFilter }

func (f *RedisIsNotNullFilter) ApplyRedis(ctx context.Context, client *redis.Client, keys []string) ([]string, error) {
	return applyRedisBatchFilter(ctx, client, keys, f.Field, func(val interface{}) bool {
		return val != nil
	})
}

// ========== 工具函数 ==========

func toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case string:
		return strconv.ParseFloat(val, 64)
	default:
		return 0, fmt.Errorf("unknown type")
	}
}

// ========== Redis Translator 实现 ==========

type RedisEqualTranslator struct{}

func (t *RedisEqualTranslator) SupportedOperator() string { return "=" }
func (t *RedisEqualTranslator) Validate(param FilterParam) error {
	if param.Field == "" || param.Value == nil {
		return fmt.Errorf("invalid params")
	}
	return nil
}
func (t *RedisEqualTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &RedisEqualFilter{GenericFilter: &GenericFilter{Field: param.Field, Operator: "=", Value: param.Value}}, nil
}

type RedisNotEqualTranslator struct{}

func (t *RedisNotEqualTranslator) SupportedOperator() string { return "!=" }
func (t *RedisNotEqualTranslator) Validate(param FilterParam) error {
	if param.Field == "" || param.Value == nil {
		return fmt.Errorf("invalid params")
	}
	return nil
}
func (t *RedisNotEqualTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &RedisNotEqualFilter{GenericFilter: &GenericFilter{Field: param.Field, Operator: "!=", Value: param.Value}}, nil
}

type RedisGreaterThanTranslator struct{}

func (t *RedisGreaterThanTranslator) SupportedOperator() string { return ">" }
func (t *RedisGreaterThanTranslator) Validate(param FilterParam) error {
	if param.Field == "" || param.Value == nil {
		return fmt.Errorf("invalid params")
	}
	return nil
}
func (t *RedisGreaterThanTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &RedisGreaterThanFilter{GenericFilter: &GenericFilter{Field: param.Field, Operator: ">", Value: param.Value}}, nil
}

type RedisGreaterThanOrEqualTranslator struct{}

func (t *RedisGreaterThanOrEqualTranslator) SupportedOperator() string { return ">=" }
func (t *RedisGreaterThanOrEqualTranslator) Validate(param FilterParam) error {
	if param.Field == "" || param.Value == nil {
		return fmt.Errorf("invalid params")
	}
	return nil
}
func (t *RedisGreaterThanOrEqualTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &RedisGreaterThanOrEqualFilter{GenericFilter: &GenericFilter{Field: param.Field, Operator: ">=", Value: param.Value}}, nil
}

type RedisLessThanTranslator struct{}

func (t *RedisLessThanTranslator) SupportedOperator() string { return "<" }
func (t *RedisLessThanTranslator) Validate(param FilterParam) error {
	if param.Field == "" || param.Value == nil {
		return fmt.Errorf("invalid params")
	}
	return nil
}
func (t *RedisLessThanTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &RedisLessThanFilter{GenericFilter: &GenericFilter{Field: param.Field, Operator: "<", Value: param.Value}}, nil
}

type RedisLessThanOrEqualTranslator struct{}

func (t *RedisLessThanOrEqualTranslator) SupportedOperator() string { return "<=" }
func (t *RedisLessThanOrEqualTranslator) Validate(param FilterParam) error {
	if param.Field == "" || param.Value == nil {
		return fmt.Errorf("invalid params")
	}
	return nil
}
func (t *RedisLessThanOrEqualTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &RedisLessThanOrEqualFilter{GenericFilter: &GenericFilter{Field: param.Field, Operator: "<=", Value: param.Value}}, nil
}

type RedisLikeTranslator struct{}

func (t *RedisLikeTranslator) SupportedOperator() string { return "like" }
func (t *RedisLikeTranslator) Validate(param FilterParam) error {
	if _, ok := param.Value.(string); !ok {
		return fmt.Errorf("value must be string")
	}
	return nil
}
func (t *RedisLikeTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &RedisLikeFilter{GenericFilter: &GenericFilter{Field: param.Field, Operator: "like", Value: param.Value}}, nil
}

type RedisInTranslator struct{}

func (t *RedisInTranslator) SupportedOperator() string { return "in" }
func (t *RedisInTranslator) Validate(param FilterParam) error {
	if _, ok := param.Value.([]interface{}); !ok {
		return fmt.Errorf("value must be array")
	}
	return nil
}
func (t *RedisInTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &RedisInFilter{GenericInFilter: &GenericInFilter{Field: param.Field, Operator: "in", Values: param.Value.([]interface{})}}, nil
}

type RedisBetweenTranslator struct{}

func (t *RedisBetweenTranslator) SupportedOperator() string { return "between" }
func (t *RedisBetweenTranslator) Validate(param FilterParam) error {
	v, ok := param.Value.([]interface{})
	if !ok || len(v) != 2 {
		return fmt.Errorf("between needs 2 values")
	}
	return nil
}
func (t *RedisBetweenTranslator) Translate(param FilterParam) (BaseFilter, error) {
	v := param.Value.([]interface{})
	return &RedisBetweenFilter{GenericBetweenFilter: &GenericBetweenFilter{Field: param.Field, Operator: "between", Min: v[0], Max: v[1]}}, nil
}

type RedisIsNullTranslator struct{}

func (t *RedisIsNullTranslator) SupportedOperator() string        { return "isnull" }
func (t *RedisIsNullTranslator) Validate(param FilterParam) error { return nil }
func (t *RedisIsNullTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &RedisIsNullFilter{GenericFilter: &GenericFilter{Field: param.Field, Operator: "isnull"}}, nil
}

type RedisIsNotNullTranslator struct{}

func (t *RedisIsNotNullTranslator) SupportedOperator() string        { return "isnotnull" }
func (t *RedisIsNotNullTranslator) Validate(param FilterParam) error { return nil }
func (t *RedisIsNotNullTranslator) Translate(param FilterParam) (BaseFilter, error) {
	return &RedisIsNotNullFilter{GenericFilter: &GenericFilter{Field: param.Field, Operator: "isnotnull"}}, nil
}

// ========== Redis 翻译器注册表 ==========

type RedisTranslatorRegistry struct {
	translators map[string]FilterTranslator
}

func NewRedisTranslatorRegistry() *RedisTranslatorRegistry {
	registry := &RedisTranslatorRegistry{translators: make(map[string]FilterTranslator)}
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

func (r *RedisTranslatorRegistry) Register(translator FilterTranslator) {
	r.translators[translator.SupportedOperator()] = translator
}

func (r *RedisTranslatorRegistry) Translate(param FilterParam) (RedisFilter, error) {
	t, ok := r.translators[param.Operator]
	if !ok {
		return nil, fmt.Errorf("unsupported operator: %s", param.Operator)
	}
	if err := t.Validate(param); err != nil {
		return nil, err
	}
	base, err := t.Translate(param)
	if err != nil {
		return nil, err
	}
	return base.(RedisFilter), nil
}

func (r *RedisTranslatorRegistry) TranslateBatch(params []FilterParam) ([]RedisFilter, error) {
	filters := make([]RedisFilter, 0, len(params))
	for _, p := range params {
		f, err := r.Translate(p)
		if err != nil {
			return nil, err
		}
		filters = append(filters, f)
	}
	return filters, nil
}

// ApplyRedisFilters 外部调用入口
func ApplyRedisFilters(ctx context.Context, client *redis.Client, initialKeys []string, filters []RedisFilter) ([]string, error) {
	keys := initialKeys
	for _, filter := range filters {
		filtered, err := filter.ApplyRedis(ctx, client, keys)
		if err != nil {
			return nil, err
		}
		keys = filtered
	}
	return keys, nil
}

var DefaultRedisRegistry = NewRedisTranslatorRegistry()
