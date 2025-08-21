package xUtil

import (
	"reflect"
	"strings"
)

// Ptr 返回给定值的指针。
//
// 该函数接受一个泛型类型的值 `data`，并返回其指针。若传入值为 `nil`，则返回 `nil`。
//
// 参数说明:
//   - data: 任意类型的输入数据。
//
// 返回值:
//   - 指向输入数据的指针，或 `nil`（当输入值为 `nil` 时）。
func Ptr[T any](data T) *T {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return nil
	}
	return &data
}

// Val 从指针中安全地获取值，如果指针为 nil 则返回零值。
//
// 该函数提供安全的指针解引用，避免空指针异常。
//
// 参数说明:
//   - ptr: 指向任意类型的指针
//
// 返回值:
//   - 指针指向的值，如果指针为 nil 则返回该类型的零值
func Val[T any](ptr *T) T {
	if ptr == nil {
		var zero T
		return zero
	}
	return *ptr
}

// Contains 检查切片中是否包含指定元素。
//
// 该函数使用泛型实现，支持任何可比较的类型。
//
// 参数说明:
//   - slice: 要搜索的切片
//   - item: 要查找的元素
//
// 返回值:
//   - 如果找到元素返回 true，否则返回 false
func Contains[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// ToBool 将字符串转换为布尔值。
//
// 该函数支持多种布尔值表示：true/false, 1/0, yes/no, on/off 等。
//
// 参数说明:
//   - str: 要转换的字符串
//   - defaultValue: 转换失败时的默认值
//
// 返回值:
//   - 转换后的布尔值，如果转换失败则返回默认值
func ToBool(str string, defaultValue bool) bool {
	if str == "" {
		return defaultValue
	}

	str = strings.ToLower(strings.TrimSpace(str))
	switch str {
	case "true", "1", "yes", "on", "enabled":
		return true
	case "false", "0", "no", "off", "disabled":
		return false
	default:
		return defaultValue
	}
}
