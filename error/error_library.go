package xError

import (
	xConsts "github.com/bamboo-services/bamboo-base-go/constants"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/utility/ctxutil"
	"github.com/gin-gonic/gin"
)

// NewInternalServerError 创建一个包含服务器内部错误信息的 Error 对象并记录错误日志。
//
// 参数 ctx 表示 gin 的上下文，用于获取日志记录器以记录错误信息。
// 参数 errMessage 为自定义错误消息，描述具体的错误上下文。
// 参数 err 为实际错误，用于提供底层的错误信息。
//
// 返回一个指向 Error 的指针，包含服务器内部错误代码和相关信息。
func NewInternalServerError(ctx *gin.Context, errMessage ErrMessage, err error) *Error {
	newErr := &Error{
		error:        err,
		ErrorCode:    ServerError,
		ErrorMessage: errMessage,
	}
	xCtxUtil.GetSugarLogger(ctx, xConsts.LogTHOW).Errorf("[%d]%s | 错误(%s)", newErr.Code, newErr.ErrorMessage, newErr.error.Error())
	return newErr
}
