package apperr

import "reflect"

// chainIdentity 是原因链节点的遍历身份，用于识别真正的环。
//
// typ 始终记录节点的动态类型：同一个底层映射被转换成不同具方法命名类型时，
// 属于不同节点，不会被错误去重。
//
// ptr 用于指针、映射、切片与函数类节点，length 用于区分同底层数组但长度不同的切片，
// value 用于值类型节点；value 只在运行时确认可比较时才会写入，保证它可以安全作为 map 键。
type chainIdentity struct {
	typ    reflect.Type
	ptr    uintptr
	length int
	value  error
}

// chainIdentityOf 返回节点的遍历身份；无法稳定识别时返回 false。
//
// 判定使用 reflect.Value.Comparable：它检查运行时的真实可比较性，
// 而 reflect.Type.Comparable 只反映静态类型。
// 两者在"值类型结构体持有不可比较的接口字段"时结论不同，
// 只有前者能保证本函数返回的身份可安全用作 map 键。
func chainIdentityOf(err error) (chainIdentity, bool) {
	value := reflect.ValueOf(err)
	if !value.IsValid() {
		return chainIdentity{}, false
	}
	typ := value.Type()

	switch value.Kind() {
	case reflect.Pointer, reflect.UnsafePointer, reflect.Func:
		return chainIdentity{typ: typ, ptr: value.Pointer()}, true
	case reflect.Map:
		return chainIdentity{typ: typ, ptr: uintptr(value.UnsafePointer())}, true
	case reflect.Slice:
		return chainIdentity{typ: typ, ptr: value.Pointer(), length: value.Len()}, true
	}

	if !value.Comparable() {
		return chainIdentity{}, false
	}
	return chainIdentity{typ: typ, value: err}, true
}

// projectStructured 通过节点实现的标准 As 钩子提取其暴露的统一错误。
//
// 钩子命中但给出 nil 统一错误时视为未命中：调用方据此继续寻找真实原因。
func projectStructured(node error) (*Error, bool) {
	asserter, ok := node.(interface{ As(any) bool })
	if !ok {
		return nil, false
	}
	var target *Error
	if !asserter.As(&target) {
		return nil, false
	}
	if target == nil {
		return nil, false
	}
	return target, true
}
