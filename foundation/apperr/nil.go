package apperr

import "reflect"

// IsNil 判断一个错误值是否为空。
//
// 除了真正的 nil 接口外，本函数还识别被装入 error 接口的 nil 指针等
// nil-able 载体：这类值的接口比较结果不等于 nil，但其 Error 方法通常
// 已经不可调用。判定只依赖类型与 nil 状态，不会调用 Error 方法，
// 因此可以安全地用于第三方原因的清洗，避免把 nil 载体当作真实原因保留。
//
// 本函数只做空值判定，不构造失败结果，也不改变任何错误的分类；
// 真正的失败结果必须由 [New]、[From] 或 [Join] 显式构造。
func IsNil(err error) bool {
	if err == nil {
		return true
	}
	value := reflect.ValueOf(err)
	switch value.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan, reflect.Interface, reflect.UnsafePointer:
		return value.IsNil()
	default:
		return false
	}
}
