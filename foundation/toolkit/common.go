package toolkit

import (
	"reflect"
	"strings"
)

// Ptr 返回指向给定值的新指针。
//
// 该函数接受一个泛型类型的值 data，并返回指向该值的指针。
// 当 data 本身为类型化的 nil 指针时返回 nil；其余输入
// （包括包裹了 nil 的接口值）一律返回非 nil 的新建指针。
//
// 参数说明:
//   - data: 任意类型的输入数据。
//
// 返回值:
//   - 指向输入数据的新指针，或 nil（当输入为类型化 nil 指针时）。
func Ptr[T any](data T) *T {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return nil
	}
	return &data
}

// Val 从指针中安全地获取值，如果指针为 nil 则返回零值。
//
// 该函数提供安全的指针解引用，避免对 nil 指针解引用触发 panic。
//
// 参数说明:
//   - ptr: 指向任意类型的指针，可为 nil。
//
// 返回值:
//   - 指针指向的值；如果指针为 nil，则返回 T 类型的零值。
func Val[T any](ptr *T) T {
	if ptr == nil {
		var zero T
		return zero
	}
	return *ptr
}

// Contains 检查切片中是否包含指定元素。
//
// 该函数使用泛型实现，支持任何可比较的类型，按切片顺序线性查找。
// 空切片或 nil 切片始终返回 false。
//
// 参数说明:
//   - slice: 要搜索的切片。
//   - item: 要查找的元素。
//
// 返回值:
//   - 如果找到元素返回 true，否则返回 false。
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
// 转换前会去除首尾空白并转为小写，因此不区分大小写。
// 识别的取值：true/1/yes/on/enabled 视为 true，
// false/0/no/off/disabled 视为 false。
// 空字符串或无法识别的取值均返回 defaultValue，不会报错。
//
// 参数说明:
//   - str: 要转换的字符串。
//   - defaultValue: 转换失败时的默认值。
//
// 返回值:
//   - 转换后的布尔值；字符串为空或不可识别时返回默认值。
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
