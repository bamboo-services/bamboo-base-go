package toolkit

import (
	"reflect"
	"runtime"
	"strings"
)

// GetFunctionName 获取函数的完整名称（包含包路径）。
//
// 该函数通过反射机制获取传入函数的底层指针，并利用
// runtime.FuncForPC 查询其运行时信息，从而返回函数的完整标识符，
// 例如 "main.myFunc" 或 "github.com/foo/bar.(*Handler).Serve"。
// 匿名函数返回编译器生成的名称。
//
// 参数说明:
//   - fn: 待查询的函数对象，类型为 interface{}。
//
// 返回值:
//   - 函数的完整名称。
//   - 如果传入参数不是函数类型，或者无法获取运行时函数信息，
//     则返回空字符串，不会 panic。
func GetFunctionName(fn interface{}) string {
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
// 该函数通过反射获取传入方法的运行时信息，并对其完整名称进行解析
// 处理：去除包路径、接收者类型（如 (*Receiver) 或 Receiver）以及
// 方法值特有的 -fm 后缀，仅返回纯粹的方法名。传入普通函数同样
// 返回去除包路径后的名称。
//
// 参数说明:
//   - method: 目标方法实例，必须为有效的方法值或函数值。
//
// 返回值:
//   - 解析后的方法名称。
//
// 注意: 该函数不校验入参类型：传入非函数值会使 reflect 的
// Pointer 调用触发 panic，传入 nil 函数值也会因无法定位运行时
// 信息而 panic；调用方必须保证入参是有效的函数或方法值。
func GetMethodName(method interface{}) string {
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
