package apperr

import "errors"

// New 构造统一框架错误。
// 不会自动记录日志，不包含 ctx/isLog 参数；日志由调用方或协议兜底显式记录。
// 复制 *code 为值，code 为 nil 时安全默认 ServerInternalError（50000）。
// errs 仅保留实际非 nil 的底层根因，单个直接赋予 Err，多个使用 errors.Join 合并；
// 若无非 nil 根因，则 Err 保持 nil，Unwrap() 返回 nil。
func New(code *ErrorCode, msg ErrMessage, errs ...error) *Error {
	targetCode := ServerInternalError
	if code != nil {
		targetCode = code
	}

	var nonNilErrs []error
	for _, err := range errs {
		if err != nil {
			nonNilErrs = append(nonNilErrs, err)
		}
	}

	var cause error
	switch len(nonNilErrs) {
	case 0:
		cause = nil
	case 1:
		cause = nonNilErrs[0]
	default:
		cause = errors.Join(nonNilErrs...)
	}

	return &Error{
		Code:         *targetCode,
		ErrorMessage: msg,
		Err:          cause,
	}
}
