package xError

import (
	xConsts "github.com/bamboo-services/bamboo-base-go/constants"
	awakenCtxUtil "github.com/bamboo-services/bamboo-base-go/utility/ctxutil"
	"github.com/gin-gonic/gin"
)

// NewError 构造并返回一个新的 `Error` 对象。
//
// 该函数用于根据提供的参数创建一个新的 `Error` 实例，同时根据需求记录错误日志。
// 如果参数 `throw` 为 `true`，则会通过上下文日志记录器输出错误信息日志。
//
// 参数说明:
//   - ctx: `*gin.Context` 上下文，用于获取请求范围内的日志记录器。
//   - err: `*ErrorCode` 错误代码结构，包含预定义的错误编码和描述信息。
//   - errorMessage: `string` 自定义错误信息，用于补充或覆盖预定义描述。
//   - getErr: `error` 原始错误对象，用于包装和进一步处理。
//   - throw: `bool` 是否记录并输出错误日志。
//
// 返回值:
//   - `*Error`: 返回一个包含错误代码、描述和上下文信息的 `Error` 对象。
func NewError(ctx *gin.Context, err *ErrorCode, errorMessage string, getErr error, throw bool) *Error {
	newErr := &Error{
		error:        getErr,
		ErrorCode:    err,
		ErrorMessage: errorMessage,
	}
	if throw {
		awakenCtxUtil.GetSugarLogger(ctx).Named(xConsts.LogTHOW).Errorf("[%d]%s | 错误(%s)", err.Code, newErr.ErrorMessage, newErr.error.Error())
	}
	return newErr
}

// NewErrorHasData 构造一个新的包含数据的错误对象。
//
// 该函数根据传入参数创建并返回一个 `*Error` 类型的实例，并根据需要记录错误日志。
//
// 参数说明:
//   - ctx: Gin 上下文，用于记录日志和操作处理。
//   - err: 错误代码的实例，用于描述错误类型及状态码。
//   - errorMessage: 错误的描述性消息。
//   - getErr: 原始 error 类型实例，用于附加到错误对象中。
//   - throw: 是否记录日志。如果为 true，则将错误详细信息记录到日志中。
//   - data: 可选的变长参数，用于将附加数据存储到错误对象中。
//
// 返回值:
//   - 返回一个 `*Error` 类型的错误实例，其中包括错误代码、消息、原始错误及附加数据（如有）。
func NewErrorHasData(ctx *gin.Context, err *ErrorCode, errorMessage string, getErr error, throw bool, data ...interface{}) *Error {
	newErr := &Error{
		error:        getErr,
		ErrorCode:    err,
		ErrorMessage: errorMessage,
	}
	if data != nil && len(data) > 0 {
		newErr.Data = data
	}
	if throw {
		awakenCtxUtil.GetSugarLogger(ctx).Named(xConsts.LogTHOW).Errorf("[%d]%s | 错误(%s)", err.Code, newErr.ErrorMessage, newErr.error.Error())
	}
	return newErr
}
