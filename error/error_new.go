package xError

import (
	"errors"
	"log/slog"

	"github.com/gin-gonic/gin"
)

// NewError 创建一个新的错误对象。
//
// 参数说明:
//   - ctx: `gin.Context` 请求上下文，用于日志记录。
//   - err: 错误代码对象，包含预定义的错误信息。
//   - errorMessage: 自定义错误消息，用于补充具体的错误描述。
//   - throw: 是否立即记录该错误，true 表示记录日志。
//   - getErr: 可选的 error 参数，用于指定实际的错误详情。
//
// 返回值:
//   - 返回指向 `Error` 对象的指针，包含完整的错误信息。
func NewError(ctx *gin.Context, err *ErrorCode, errorMessage ErrMessage, throw bool, getErr ...error) *Error {
	newErr := &Error{
		ErrorCode:    err,
		ErrorMessage: errorMessage,
	}
	if len(getErr) > 0 {
		newErr.error = getErr[0]
	} else {
		newErr.error = errors.New(errorMessage.String())
	}
	if throw {
		slog.ErrorContext(ctx.Request.Context(), "业务错误",
			"code", err.Code,
			"message", newErr.ErrorMessage,
			"error", newErr.error.Error(),
		)
	}
	return newErr
}

// NewErrorHasData 创建一个包含错误数据的自定义错误实例。
//
// 参数 ctx 表示 gin.Context 上下文对象，记录日志时使用。
// 参数 err 表示错误码对象，包含预定义的错误信息。
// 参数 errorMessage 表示自定义错误消息，补充描述具体错误。
// 参数 throw 表示是否抛出错误日志，为 true 时记录错误日志。
// 参数 getErr 表示实际错误对象，用于包装具体错误。
// 参数 data 表示错误相关的上下文数据，可选。
//
// 返回值为 *Error 类型，包含错误码、自定义消息、原始错误以及上下文数据。
func NewErrorHasData(ctx *gin.Context, err *ErrorCode, errorMessage ErrMessage, throw bool, getErr error, data ...interface{}) *Error {
	newErr := &Error{
		error:        getErr,
		ErrorCode:    err,
		ErrorMessage: errorMessage,
	}
	if data != nil && len(data) > 0 {
		newErr.Data = data
	}
	if throw {
		slog.ErrorContext(ctx.Request.Context(), "业务错误",
			"code", err.Code,
			"message", newErr.ErrorMessage,
			"error", newErr.error.Error(),
		)
	}
	return newErr
}
