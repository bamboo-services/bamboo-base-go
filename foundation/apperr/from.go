package apperr

// From 把一个原生原因转接为框架统一错误。
//
// 该函数是"nil 安全转接"入口，不是一个失败构造器：
//
//   - cause 为真正的 nil 或 nil 载体（装入 error 接口的 nil 指针等）时返回 nil，
//     不会产生只有 nil 指针的假失败；此时调用方应直接返回真正的 nil 错误。
//   - cause 本身是已构造的 [*Error] 时原样返回该对象：既有错误不因更外层的
//     上下文被重新分类，其错误码、安全描述与业务数据全部保留。
//   - cause 的包装链中已存在统一错误时（由 [Find] 判定，含标准 As 钩子暴露的情况），
//     沿用该错误的分类与安全描述，
//     同时把整条原始原因链保留为新错误的根因：外层包装文本与全部兄弟原因
//     都不会被丢弃，内层统一错误仍可经 errors.As 提取。
//   - 其余未知原因按调用方显式给出的 code 与 msg 包装，根因原样保留；
//     安全描述只来自调用方，不会自动复制 cause 的错误文本。
//
// 返回值在存在真实原因时始终非 nil，可被多 goroutine 并发读取。
func From(code *ErrorCode, msg ErrMessage, cause error) *Error {
	if IsNil(cause) {
		return nil
	}
	if appErr, ok := cause.(*Error); ok {
		return appErr
	}
	if structured := Find(cause); structured != nil {
		return &Error{
			Code:         structured.Code,
			ErrorMessage: structured.ErrorMessage,
			Data:         structured.Data,
			Err:          cause,
		}
	}
	return New(code, msg, cause)
}
