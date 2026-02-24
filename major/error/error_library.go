package xError

import (
	"context"
	"log/slog"
)

// NewInternalServerError 创建一个包含服务器内部错误信息的 Error 对象并记录错误日志。
//
// 参数 ctx 表示标准 context，用于获取日志记录器以记录错误信息。
// 参数 errMessage 为自定义错误消息，描述具体的错误上下文。
// 参数 err 为实际错误，用于提供底层的错误信息。
//
// 返回一个指向 Error 的指针，包含服务器内部错误代码和相关信息。
func NewInternalServerError(ctx context.Context, errMessage ErrMessage, err error) *Error {
	newErr := &Error{
		error:        err,
		ErrorCode:    ServerInternalError,
		ErrorMessage: errMessage,
	}
	slog.ErrorContext(ctx, "服务器内部错误",
		"code", newErr.Code,
		"message", newErr.ErrorMessage,
		"error", newErr.error.Error(),
	)
	return newErr
}
