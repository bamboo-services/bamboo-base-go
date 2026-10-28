package apperr

import "fmt"

// Error 是框架统一错误对象。
// Err 内部错误通过 json:"-" 忽略，不得直接对外序列化泄漏。
type Error struct {
	Code         ErrorCode  `json:"code"`
	ErrorMessage ErrMessage `json:"error_message,omitempty"`
	Data         any        `json:"data,omitempty"`
	Err          error      `json:"-"`
}

// Error 实现标准 error 接口。
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

// Unwrap 暴露底层根因 error，供 errors.Is 与 errors.As 判定。
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
