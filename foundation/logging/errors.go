package logging

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// 本文件集中本包自有失败出口的错误构造规则。
//
// 分类只依据实际失败分支显式给出，不解析错误文本、不按错误来源推断：
//
//   - 日志处理器缺失：HandlerInvalid（50028）；
//   - 轮转写入器配置非法：ConfigInvalid（50029）；
//   - 日志写入失败（控制台/文件双写、底层写入、写入器已关闭）：WriteFailed（50030）；
//   - 日志轮转失败（目录创建、文件打开、跨天重命名、按大小切割、归档与关闭）：RotationFailed（50031）。
//
// 安全描述一律由本包给出，不复制原始 IO 错误文本；原生错误只作为根因保留，
// 供 errors.Is 与 errors.As 继续识别。

// errHandlerInvalid 构造日志处理器缺失错误（50028）。
//
// 该错误没有原生根因可保留：缺失的处理器是调用方传入的配置事实。
func errHandlerInvalid() *apperr.Error {
	return apperr.New(HandlerInvalid, "日志处理器缺失")
}

// errConfigInvalid 构造配置非法错误（50029）。
//
// 用于纯配置问题，例如文件名包含路径分隔符；该类问题没有原生 cause 可保留。
func errConfigInvalid(msg apperr.ErrMessage) *apperr.Error {
	return apperr.New(ConfigInvalid, msg)
}

// errWriteFailed 构造日志写入失败错误（50030），保留原始根因。
//
// cause 为 nil 时返回 nil，表示该写入路径没有失败。
func errWriteFailed(msg apperr.ErrMessage, cause error) *apperr.Error {
	return apperr.From(WriteFailed, msg, cause)
}

// errRotationFailed 构造日志轮转失败错误（50031），保留原始根因。
//
// cause 为 nil 时返回 nil，表示该轮转路径没有失败。
func errRotationFailed(msg apperr.ErrMessage, cause error) *apperr.Error {
	return apperr.From(RotationFailed, msg, cause)
}
