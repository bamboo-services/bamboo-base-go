package apperr

import "errors"

// New 构造统一的框架错误。
//
// 这是纯构造函数：不自动记录日志，也不包含 ctx/isLog 参数，
// 日志由调用方或协议层兑底显式记录。
//
// 参数与快照语义：
//
//   - code：选定的错误码；构造时复制 *code 为值快照，
//     后续修改原变量不影响已构造的错误；
//     code 为 nil 时安全回退为 [FrameworkInternalError]（50000）。
//   - msg：可选的补充描述，写入 [Error.ErrorMessage]，可为空。
//   - errs：底层根因列表，仅保留实际非 nil 且非 nil 载体的项
//     （装入 error 接口的 nil 指针等由 [IsNil] 识别并过滤，
//     过滤过程不会调用其 Error 方法）；单个根因直接赋予
//     [Error.Err]，多个根因使用 errors.Join 合并，
//     若无有效根因则 [Error.Err] 保持 nil、[Error.Unwrap] 返回 nil。
//
// 返回值始终为非 nil 的 [*Error]，可被多 goroutine 并发读取。
// 需要"无原因即无失败"语义时使用 [From] 或 [Join]。
func New(code *ErrorCode, msg ErrMessage, errs ...error) *Error {
	var nonNilErrs []error
	for _, err := range errs {
		if !IsNil(err) {
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
		Code:         codeSnapshot(code),
		ErrorMessage: msg,
		Err:          cause,
	}
}

// codeSnapshot 复制错误码的值快照；code 为 nil 时回退为框架内部错误。
func codeSnapshot(code *ErrorCode) ErrorCode {
	if code == nil {
		return *FrameworkInternalError
	}
	return *code
}
