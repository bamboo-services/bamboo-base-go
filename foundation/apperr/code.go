package apperr

// Code 表示业务错误码（前 3 位对应状态分组，如 400xx / 500xx）。
type Code int

// Output 表示机器可读的错误输出标识（通常为大写下划线格式）。
type Output string

// Message 表示对外预定义的错误文案（中文描述）。
type Message string

// ErrMessage 表示自定义或补充的错误描述消息。
type ErrMessage string

// String 将 ErrMessage 转换为字符串形式的表示。
func (e ErrMessage) String() string {
	return string(e)
}

// ErrorCode 保留稳定的机器码、输出标识与对外文案。
type ErrorCode struct {
	Code    Code
	Output  Output
	Message Message
}

// GetCode 获取错误码。
func (e *ErrorCode) GetCode() Code {
	return e.Code
}

// GetOutput 获取输出标识。
func (e *ErrorCode) GetOutput() Output {
	return e.Output
}

// GetMessage 获取错误信息。
func (e *ErrorCode) GetMessage() Message {
	return e.Message
}
