package main

import "reflect"

type ExpStruct struct {
	Value int
}

type AbstractManager[T any] struct {
	structField T
	typeName    string
}

func TypeName[T any]() string {
	var zero T
	t := reflect.TypeOf(zero)

	// 如果 T 是指针，剥一层
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

func NewAbstractManager[T any](value T) *AbstractManager[T] {
	return &AbstractManager[T]{structField: value}
}

func (am *AbstractManager[T]) GetValue() T {
	return am.structField
}

func (am *AbstractManager[T]) SetValue(value T) {
	am.structField = value
}

func main() {
	expInstance := ExpStruct{Value: 42}
	manager := NewAbstractManager(expInstance)
	value := manager.GetValue()
	println(value.Value) // Output: 42
	manager.SetValue(ExpStruct{Value: 100})
	updatedValue := manager.GetValue()
	println(updatedValue.Value) // Output: 100
}
