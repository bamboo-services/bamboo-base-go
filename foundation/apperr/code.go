package apperr

// Code 表示业务错误码。
//
// 取值为 5 位整数，前 3 位对齐 HTTP 状态分组（如 400xx / 500xx），
// 完整取值关系见 error_code.go 中的预置错误码目录；
// 零值 0 不属于内置目录中的任何错误码。
type Code int

// Output 表示机器可读的错误输出标识。
//
// 通常为大写下划线格式（如 "NOT_FOUND"），供日志检索与调用方程序化识别；
// 零值为空字符串。
type Output string

// Message 表示错误码预定义的对外文案。
//
// 由错误码目录统一维护的中文描述，零值为空字符串。
type Message string

// ErrMessage 表示构造错误时传入的补充错误描述。
//
// 与 [Message] 的预定义文案不同，[ErrMessage] 由调用方按现场情况提供，
// 会拼接到 [*Error.Error] 的输出文本中；零值为空字符串，表示无补充描述。
type ErrMessage string

// String 将 ErrMessage 转换为其底层字符串的表示。
func (e ErrMessage) String() string {
	return string(e)
}

// ErrorCode 保留一个错误码的稳定三元组：机器码、输出标识与对外文案。
//
// 预置目录中的变量均为 *ErrorCode，可在多个错误间共享只读引用；
// [New] 构造时会复制值快照到 [*Error.Code]，
// 调用方后续修改原变量不影响已构造的错误。
type ErrorCode struct {
	Code    Code    // 5 位机器错误码，前 3 位对齐 HTTP 状态分组。
	Output  Output  // 机器可读的输出标识，通常为大写下划线格式。
	Message Message // 对外展示的预定义中文文案。
}

// GetCode 返回错误码的机器码。
// e 为 nil 时会引发 panic，调用方需保证接收器非 nil。
func (e *ErrorCode) GetCode() Code {
	return e.Code
}

// GetOutput 返回错误码的机器可读输出标识。
// e 为 nil 时会引发 panic，调用方需保证接收器非 nil。
func (e *ErrorCode) GetOutput() Output {
	return e.Output
}

// GetMessage 返回错误码的预定义对外文案。
// e 为 nil 时会引发 panic，调用方需保证接收器非 nil。
func (e *ErrorCode) GetMessage() Message {
	return e.Message
}
