package cache_key_builder

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// KeyBuilder 缓存键构建器接口
// 开发者可以实现此接口来自定义键生成逻辑
type KeyBuilder[T any] interface {
	// BuildKey 根据数据生成缓存键
	BuildKey(data *T) string
}

// TemplateKeyBuilder 基于模板的键构建器
// 支持模板语法: "cache:user:{id}" 或 "cache:product:{id}:{category}"
type TemplateKeyBuilder[T any] struct {
	template string
	regex    *regexp.Regexp
}

// NewTemplateKeyBuilder 创建模板键构建器
// 示例模板:
//   - "cache:user:{id}"
//   - "cache:product:{id}:{category}"
//   - "session:{user_id}:{device_id}"
func NewTemplateKeyBuilder[T any](template string) *TemplateKeyBuilder[T] {
	return &TemplateKeyBuilder[T]{
		template: template,
		regex:    regexp.MustCompile(`\{([^}]+)\}`),
	}
}

// BuildKey 实现 KeyBuilder 接口
func (kb *TemplateKeyBuilder[T]) BuildKey(data *T) string {
	if data == nil {
		return kb.template
	}

	key := kb.template
	val := reflect.ValueOf(data)

	// 如果是指针，获取实际值
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// 必须是结构体类型
	if val.Kind() != reflect.Struct {
		return key
	}

	// 查找所有模板变量 {fieldName}
	matches := kb.regex.FindAllStringSubmatch(kb.template, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		placeholder := match[0] // {id}
		fieldName := match[1]   // id

		// 尝试获取字段值
		fieldValue := kb.getFieldValue(val, fieldName)
		key = strings.Replace(key, placeholder, fieldValue, 1)
	}

	return key
}

// getFieldValue 获取结构体字段值（支持嵌套字段）
func (kb *TemplateKeyBuilder[T]) getFieldValue(val reflect.Value, fieldPath string) string {
	// 支持嵌套字段，如 "user.id" 或 "metadata.category"
	parts := strings.Split(fieldPath, ".")
	current := val

	for _, part := range parts {
		// 尝试通过字段名获取
		field := current.FieldByName(part)
		if !field.IsValid() {
			// 尝试大小写不敏感匹配
			field = kb.findFieldCaseInsensitive(current, part)
		}

		if !field.IsValid() {
			return fmt.Sprintf("{%s}", fieldPath)
		}

		current = field
	}

	return fmt.Sprintf("%v", current.Interface())
}

// findFieldCaseInsensitive 大小写不敏感查找字段
func (kb *TemplateKeyBuilder[T]) findFieldCaseInsensitive(val reflect.Value, name string) reflect.Value {
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if strings.EqualFold(field.Name, name) {
			return val.Field(i)
		}
		// 检查 json tag
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			jsonName := strings.Split(jsonTag, ",")[0]
			if strings.EqualFold(jsonName, name) {
				return val.Field(i)
			}
		}
	}
	return reflect.Value{}
}

// FuncKeyBuilder 基于自定义函数的键构建器
type FuncKeyBuilder[T any] struct {
	buildFunc func(*T) string
}

// NewFuncKeyBuilder 创建函数式键构建器
func NewFuncKeyBuilder[T any](buildFunc func(*T) string) *FuncKeyBuilder[T] {
	return &FuncKeyBuilder[T]{
		buildFunc: buildFunc,
	}
}

// BuildKey 实现 KeyBuilder 接口
func (kb *FuncKeyBuilder[T]) BuildKey(data *T) string {
	return kb.buildFunc(data)
}

// PrefixKeyBuilder 前缀键构建器
// 简单在 ID 前加前缀: "prefix:123"
type PrefixKeyBuilder[T any] struct {
	prefix      string
	idExtractor func(*T) interface{}
}

// NewPrefixKeyBuilder 创建前缀键构建器
func NewPrefixKeyBuilder[T any](prefix string, idExtractor func(*T) interface{}) *PrefixKeyBuilder[T] {
	return &PrefixKeyBuilder[T]{
		prefix:      prefix,
		idExtractor: idExtractor,
	}
}

// BuildKey 实现 KeyBuilder 接口
func (kb *PrefixKeyBuilder[T]) BuildKey(data *T) string {
	if data == nil {
		return kb.prefix
	}
	id := kb.idExtractor(data)
	return fmt.Sprintf("%s:%v", kb.prefix, id)
}

// ========== 便捷工具函数 ==========

// BuildKeyFunc 将 KeyBuilder 转换为函数
func BuildKeyFunc[T any](builder KeyBuilder[T]) func(*T) string {
	return func(data *T) string {
		return builder.BuildKey(data)
	}
}

// DefaultKeyBuilder 创建默认的键构建器（基于模板）
func DefaultKeyBuilder[T any](template string) KeyBuilder[T] {
	return NewTemplateKeyBuilder[T](template)
}

// QuickBuildKey 快速构建键的辅助函数
// 用于一次性构建，不需要保存 builder 实例
func QuickBuildKey[T any](template string, data *T) string {
	builder := NewTemplateKeyBuilder[T](template)
	return builder.BuildKey(data)
}
