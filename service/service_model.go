package service

import "reflect"

type ServiceManager[T any] struct {
	Resource     T      // 被管理的资源
	ResourceName string // 资源名称
	TableName    string // 表名
	Schema       string // 数据库模式
	CacheKeyType string // 缓存键
	CacheKeyName string // 缓存键名称
}

func getTypeName[T any](value T) string {
	var zero T
	t := reflect.TypeOf(zero)
	// 如果 T 是指针，剥一层
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// NewServiceManager 创建一个新的 ServiceManager 实例
// 通过reflect获取名字自动赋值给ResourceName和TableName还有keyname
func NewServiceManager[T any](resource T) *ServiceManager[T] {
	return &ServiceManager[T]{
		Resource:     resource,
		ResourceName: getTypeName(resource),
		TableName:    getTypeName(resource),
		Schema:       "public",
		CacheKeyType: "none",
		CacheKeyName: getTypeName(resource) + "_key",
	}
}
