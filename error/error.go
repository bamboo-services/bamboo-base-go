package xError

// ErrorInterface 错误接口定义
//
// 定义了系统中错误处理的标准接口，用于统一错误处理方式。
// 该接口继承了标准 error 接口，并扩展了错误代码、错误信息和数据获取功能。
//
// 注意: 实现该接口的类型需要提供完整的错误信息，包括错误代码、错误消息和相关数据。
type ErrorInterface interface {
	Error() string
	GetErrorCode() *ErrorCode
	GetErrorMessage() string
	GetData() interface{}
}

// Error 系统错误结构体
//
// 用于表示系统中的错误信息，包含错误代码、自定义错误消息和相关数据。
// 该结构体实现了 ErrorInterface 接口，提供了统一的错误处理机制。
//
// 字段说明:
//   - ErrorCode: 嵌入的错误代码结构，包含预定义的错误信息
//   - ErrorMessage: 自定义错误消息，用于补充具体的错误描述
//   - Data: 任意类型的错误相关数据，用于传递额外的上下文信息
//
// 注意: 该结构体可用于构造复合错误信息，便于错误追踪和调试。
type Error struct {
	*ErrorCode
	error        error
	ErrorMessage string
	Data         interface{}
}

// Error 实现标准 error 接口
//
// 返回格式化的错误信息字符串，将预定义错误消息和自定义错误消息组合。
// 格式为："{预定义消息}|{自定义消息}"
//
// @return string 格式化的错误信息字符串
func (e *Error) Error() string {
	return e.error.Error()
}

// GetErrorCode 获取错误代码
//
// 返回错误对象中包含的错误代码信息，用于获取预定义的错误类型、代码和消息。
//
// @return *ErrorCode 错误代码结构的指针
func (e *Error) GetErrorCode() *ErrorCode {
	return e.ErrorCode
}

// GetErrorMessage 获取自定义错误消息
//
// 返回错误对象中的自定义错误消息，用于获取具体的错误描述信息。
// 该消息通常用于补充预定义错误信息，提供更详细的错误上下文。
//
// @return string 自定义错误消息字符串
func (e *Error) GetErrorMessage() string {
	return e.ErrorMessage
}

// GetData 获取错误相关数据
//
// 返回错误对象中包含的任意类型数据，用于传递错误相关的上下文信息。
// 该数据可以是任意类型，通常用于调试或提供额外的错误详情。
//
// @return interface{} 错误相关的数据对象
func (e *Error) GetData() interface{} {
	return e.Data
}
