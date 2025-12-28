package xMiddle

import (
	"errors"

	xConsts "github.com/bamboo-services/bamboo-base-go/constants"
	xError "github.com/bamboo-services/bamboo-base-go/error"
	xResult "github.com/bamboo-services/bamboo-base-go/result"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/utility/ctxutil"
	"github.com/gin-gonic/gin"
)

// ResponseMiddleware 统一响应中间件
//
// 用于在 HTTP 请求的响应阶段检查是否已写入响应，
// 如果未写入且存在错误，则按标准错误结构化输出；否则不进行任何响应操作。
//
// 详细描述:
// - 当 `ctx.Writer.Written()` 返回 true，表示响应已写入，函数直接返回。
// - 如果 `ctx.Errors` 存在错误列表，将解析最后一个错误。
// - 优先检查是否为 `xError.Error` 类型的错误，从中提取错误码、消息和数据进行格式化输出。
// - 若非上述类型错误则返回通用的服务器内部错误 (`xError.ServerInternalError`)。
//
// 注意:
// - 确保所有错误信息通过 `ctx.Errors` 提供适当的上下文。
// - 避免在链式中间件或控制器中重复写入响应。
func ResponseMiddleware(ctx *gin.Context) {
	ctx.Next()

	// 继续执行下一个中间件或处理函数
	log := xCtxUtil.GetSugarLogger(ctx, xConsts.LogMIDE)

	// 获取检查是否存在 buffer
	if !ctx.Writer.Written() {
		// 如果存在错误输出错误内容
		if ctx.Errors != nil && len(ctx.Errors) > 0 {
			var getErr *xError.Error
			if errors.As(ctx.Errors.Last(), &getErr) {
				xResult.Error(
					ctx, getErr.ErrorCode,
					getErr.ErrorMessage,
					getErr.Data,
				)
			} else {
				xResult.Error(
					ctx, xError.ServerError,
					xError.ErrMessage(ctx.GetString(xConsts.ContextErrorMessage.String())),
					ctx.Errors.Last(),
				)
			}
		} else {
			xResult.Error(
				ctx, xError.DeveloperError,
				"没有正常输出信息或报错信息，请检查代码逻辑「开发者错误」",
				nil,
			)
		}
	}

	// 记录接口响应时间
	if xCtxUtil.IsDebugMode(ctx) {
		if xCtxUtil.CalcOverheadTime(ctx) > 1000 {
			log.Debugf("接口耗时: %dms", xCtxUtil.CalcOverheadTime(ctx)/1000)
		} else {
			log.Debugf("接口耗时: %dµs", xCtxUtil.CalcOverheadTime(ctx))
		}
	}
}
