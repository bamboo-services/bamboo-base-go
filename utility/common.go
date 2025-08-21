package xUtil

import "reflect"

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
