package pack

import (
	"reflect"
	"runtime"
	"strings"
)

type Function struct{}

// GetFunctionName 获取函数的完整名称（包含包路径）。
//
// 该方法通过反射机制获取传入函数的底层指针，并利用 `runtime.FuncForPC`
// 查询其运行时信息，从而返回函数的完整标识符。
//
// 参数说明:
//   - fn: 待查询的函数对象，类型为 `interface{}`。
//
// 返回值:
//   - string: 函数的完整名称（例如 "main.myFunc"）。
//   - 如果传入参数不是函数类型，或者无法获取运行时函数信息，则返回空字符串。
func (Function) GetFunctionName(fn interface{}) string {
	val := reflect.ValueOf(fn)
	if val.Kind() != reflect.Func {
		return ""
	}

	fnPtr := val.Pointer()
	fnObj := runtime.FuncForPC(fnPtr)
	if fnObj == nil {
		return ""
	}

	return fnObj.Name()
}

// GetMethodName 获取方法的方法名（不包含包路径和接收者类型）。
//
// 该方法通过反射获取传入方法的运行时信息，并对其完整名称进行解析处理。
// 它会去除包路径、接收者类型（如 `(*Receiver)` 或 `Receiver`）以及
// 闭包方法特有的 `-fm` 后缀，仅返回纯粹的方法名称字符串。
//
// 参数说明:
//   - method: 目标方法实例（需为方法引用）。
//
// 返回值:
//   - string: 解析后的方法名称。
//
// 注意: 此方法不会验证传入参数是否为有效的方法，若传入非法参数可能会导致意外结果。
func (Function) GetMethodName(method interface{}) string {
	val := reflect.ValueOf(method)
	pc := val.Pointer()
	fn := runtime.FuncForPC(pc)

	name := fn.Name()
	// 处理格式: (*Receiver).Method 或 Receiver.Method
	if idx := strings.LastIndex(name, "."); idx != -1 {
		name = name[idx+1:]
	}
	// 去掉可能的 -fm 后缀（闭包方法）
	name = strings.TrimSuffix(name, "-fm")

	return name
}
