package apperr

import "fmt"

// Error 是框架统一错误对象。
//
// 携带错误码快照、可选的补充描述、可选的业务数据与底层根因，
// 实现 error 接口并可被 errors.Is / errors.As 识别。
// 构造入口是 [New]；构造后不再修改字段的实例可被多 goroutine 并发读取。
type Error struct {
	// Code 是构造时复制的错误码值快照，始终参与 JSON 序列化。
	Code ErrorCode `json:"code"`

	// ErrorMessage 是调用方提供的补充描述，为空时序列化省略。
	ErrorMessage ErrMessage `json:"error_message,omitempty"`

	// Data 是可选的业务数据载荷（如关联资源标识），由协议层透传；
	// 为空时序列化省略，内容约束由调用方负责。
	Data any `json:"data,omitempty"`

	// Err 是底层根因，通过 json:"-" 忽略，不会随 JSON 序列化对外泄漏；
	// 经 [New] 构造时由非 nil 根因汇总而来，可经 [Error.Unwrap] 暴露。
	Err error `json:"-"`
}

// Error 按固定优先级拼接错误文本，实现标准 error 接口。
//
// 拼接顺序为：错误码预定义文案（为空时退用输出标识）、
// 补充描述 [Error.ErrorMessage]、底层根因 [Error.Err]，
// 各段之间以 ": " 分隔；全部为空时退化为 "error code N"。
// e 为 nil 时安全返回空字符串。
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	base := string(e.Code.Message)
	if base == "" {
		base = string(e.Code.Output)
	}
	if e.ErrorMessage != "" {
		if base != "" {
			base = base + ": " + string(e.ErrorMessage)
		} else {
			base = string(e.ErrorMessage)
		}
	}
	if e.Err != nil {
		if base != "" {
			return fmt.Sprintf("%s: %v", base, e.Err)
		}
		return e.Err.Error()
	}
	if base != "" {
		return base
	}
	return fmt.Sprintf("error code %d", e.Code.Code)
}

// Unwrap 暴露底层根因，供 errors.Is 与 errors.As 判定。
//
// 返回 [Error.Err]；e 为 nil 或未携带根因时返回 nil，
// 调用方对 nil 结果不应假定任何根因存在。
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
