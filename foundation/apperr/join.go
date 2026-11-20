package apperr

import "errors"

// Join 聚合多个框架统一错误。
//
// 聚合语义：
//
//   - 没有有效失败时返回 nil，不产生假失败。
//   - 只有一个有效失败时原样返回该对象，不额外套壳。
//   - 多个有效失败时构造新的统一错误：错误码由调用方显式给出，
//     本函数不会自动改用 [FrameworkMultipleFailures]，也不会把取消等控制原因
//     特殊吞掉；全部子失败按入参顺序保留为根因，可经 errors.Is 与 errors.As 遍历。
//
// 入参中的 nil 指针会被跳过；其余非 nil 诊断对象一律保留，不会被当作空值丢弃。
func Join(code *ErrorCode, msg ErrMessage, errs ...*Error) *Error {
	kept := make([]*Error, 0, len(errs))
	for _, appErr := range errs {
		if appErr == nil {
			continue
		}
		kept = append(kept, appErr)
	}

	switch len(kept) {
	case 0:
		return nil
	case 1:
		return kept[0]
	}

	causes := make([]error, 0, len(kept))
	for _, appErr := range kept {
		causes = append(causes, appErr)
	}
	return &Error{
		Code:         codeSnapshot(code),
		ErrorMessage: msg,
		Err:          errors.Join(causes...),
	}
}
