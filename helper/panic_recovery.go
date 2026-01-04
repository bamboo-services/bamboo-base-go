package xHelper

import (
	"io"
	"log/slog"

	xBase "github.com/bamboo-services/bamboo-base-go"
	xConsts "github.com/bamboo-services/bamboo-base-go/context"
	xError "github.com/bamboo-services/bamboo-base-go/error"
	xLog "github.com/bamboo-services/bamboo-base-go/log"
	"github.com/gin-gonic/gin"
)

// PanicRecovery 提供全局的 Panic 恢复机制。
//
// 该方法返回一个 Gin 中间件，用于捕获处理过程中发生的 Panic，
// 并生成统一结构的 JSON 格式错误响应，便于系统监控和问题排查。
//
// 中间件会优先从上下文 `consts.ErrorCodeKey` 提取错误码信息，
// 若未找到，则返回 `err.ServerInternalError` 为默认错误码。
//
// 参数说明: 无。
//
// 返回值:
//   - 返回一个 `gin.HandlerFunc` 类型的函数，用于注册到 Gin 中间件链中。
//
// 注意: 确保该中间件在输出任何响应前优先被调用，以正确捕获和处理异常。
func PanicRecovery() gin.HandlerFunc {
	return gin.RecoveryWithWriter(io.Discard, func(c *gin.Context, recovered interface{}) {
		// 捕获 Panic 信息
		value, exists := c.Get(xConsts.ErrorCodeKey.String())
		getErrMessage, msgExist := c.Get(xConsts.ErrorMessageKey.String())
		errorCode := xError.ServerInternalError
		if exists && value != nil {
			if ec, ok := value.(*xError.ErrorCode); ok && ec != nil {
				errorCode = ec
			}
		}
		if !msgExist {
			getErrMessage = "未知错误，请稍后再试"
		}

		// 使用 slog 记录日志
		xLog.WithName(xLog.NamedMIDE).Error(c.Request.Context(), "Panic 恢复",
			slog.Uint64("code", uint64(errorCode.Code)),
			slog.String("output", errorCode.GetOutput()),
			slog.String("message", errorCode.GetMessage()),
			slog.String("errorMessage", getErrMessage.(string)),
		)

		c.JSON(int(errorCode.Code/100), xBase.BaseResponse{
			Context:      c.GetString(xConsts.RequestKey.String()),
			Output:       errorCode.GetOutput(),
			Code:         errorCode.Code,
			Message:      errorCode.Message,
			ErrorMessage: xError.ErrMessage(getErrMessage.(string)),
		})
		c.Abort()
	})
}
