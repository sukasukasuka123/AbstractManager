package filter_translator

// ========== 通用过滤器接口 ==========

// FilterParam 前端过滤参数（统一格式）
type FilterParam struct {
	Field    string      `json:"field"`    // 字段名
	Operator string      `json:"operator"` // 操作符
	Value    interface{} `json:"value"`    // 值
}

// BaseFilter 基础过滤器接口（抽象）
type BaseFilter interface {
	GetField() string
	GetValue() interface{}
	GetOperator() string
}

// FilterTranslator 过滤器翻译器接口
// 负责将前端参数翻译为后端 Filter
type FilterTranslator interface {
	// Translate 将前端参数翻译为通用 Filter 数据
	Translate(param FilterParam) (BaseFilter, error)

	// SupportedOperator 返回支持的操作符
	SupportedOperator() string

	// Validate 验证参数是否合法
	Validate(param FilterParam) error
}

// ========== 通用过滤器数据结构 ==========

// GenericFilter 通用过滤器（存储过滤器数据）
type GenericFilter struct {
	Field    string
	Operator string
	Value    interface{}
}

func (f *GenericFilter) GetField() string {
	return f.Field
}

func (f *GenericFilter) GetValue() interface{} {
	return f.Value
}

func (f *GenericFilter) GetOperator() string {
	return f.Operator
}

// GenericInFilter IN 过滤器通用数据
type GenericInFilter struct {
	Field    string
	Operator string
	Values   []interface{}
}

func (f *GenericInFilter) GetField() string {
	return f.Field
}

func (f *GenericInFilter) GetValue() interface{} {
	return f.Values
}

func (f *GenericInFilter) GetOperator() string {
	return f.Operator
}

// GenericBetweenFilter BETWEEN 过滤器通用数据
type GenericBetweenFilter struct {
	Field    string
	Operator string
	Min      interface{}
	Max      interface{}
}

func (f *GenericBetweenFilter) GetField() string {
	return f.Field
}

func (f *GenericBetweenFilter) GetValue() interface{} {
	return []interface{}{f.Min, f.Max}
}

func (f *GenericBetweenFilter) GetOperator() string {
	return f.Operator
}
