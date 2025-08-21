package xResult

import (
	xBase "github.com/bamboo-services/bamboo-base-go"
	xConsts "github.com/bamboo-services/bamboo-base-go/constants"
	xError "github.com/bamboo-services/bamboo-base-go/error"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/utility/ctxutil"
	"github.com/gin-gonic/gin"
)

// Success 向客户端返回 200 状态码的成功响应。
//
// 该函数构造并发送一个标准化的成功响应，包含上下文信息、输出描述、状态码和自定义消息。
//
// 注意:
// - 日志记录器会记录响应状态，日志级别为 `Info`。
// - 确保业务上下文中正确设置日志记录器，否则可能影响日志记录。
func Success(ctx *gin.Context, message string) {
	log := xCtxUtil.GetLogger(ctx)
	log.Named(xConsts.LogRESU).Info("<200>Success | 成功(数据: <nil>)")
	ctx.JSON(200, xBase.BaseResponse{
		Context:  ctx.GetString(xConsts.ContextRequestKey),
		Output:   "Success",
		Code:     200,
		Message:  message,
		Overhead: xCtxUtil.CalcOverheadTime(ctx),
	})
}

// SuccessHasData 构造并返回包含数据的成功响应。
//
// 该函数用于记录成功日志，并通过标准化的 JSON 格式返回 200 状态码的响应。
// 响应中包含请求上下文、状态代码、消息和传入的数据信息。
//
// 参数说明:
//   - ctx: 请求的 `gin.Context` 对象，用于提供上下文信息和响应能力。
//   - message: 响应的描述性消息，通常用于说明操作结果。
//   - data: 可选数据内容，用于携带额外的返回信息。
//
// 注意: 确保调用此函数前，业务上下文中正确设置必要的数据，例如日志记录器。
func SuccessHasData(ctx *gin.Context, message string, data interface{}) {
	log := xCtxUtil.GetLogger(ctx)
	log.Named(xConsts.LogRESU).Sugar().Infof("<200>Success | 成功(数据: %v)", data)
	ctx.JSON(200, xBase.BaseResponse{
		Context:  ctx.GetString(xConsts.ContextRequestKey),
		Output:   "Success",
		Code:     200,
		Message:  message,
		Overhead: xCtxUtil.CalcOverheadTime(ctx),
		Data:     data,
	})
}

// Error 返回通用错误响应。
//
// 该函数用于构造和返回标准化的错误响应，包含错误码、错误信息及相关数据，用于 API 调用失败时的响应处理。
//
// 参数说明:
//   - ctx: `gin.Context` 对象，用于管理请求的上下文。
//   - errorCode: 定义错误的代码、输出和信息的结构化数据类型，标识错误的具体内容。
//   - errorMessage: 自定义错误信息的字符串，用于补充或覆盖 `ErrorCode` 中的默认错误描述。
//   - data: 任意类型的数据，用于返回附加的上下文或调试信息。
//
// 注意: 确保上下文中存在有效的日志记录器，否则可能影响日志记录功能。
func Error(ctx *gin.Context, errorCode *xError.ErrorCode, errorMessage string, data interface{}) {
	log := xCtxUtil.GetSugarLogger(ctx)
	log.Named(xConsts.LogRESU).Warnf("<%d>%s | %s【%s】(数据: %v)", errorCode.Code, errorCode.Output, errorCode.GetMessage(), errorMessage, data)
	ctx.Set(xConsts.ContextErrorCode, errorCode)
	ctx.JSON(int(errorCode.Code/100), xBase.BaseResponse{
		Context:      ctx.GetString(xConsts.ContextRequestKey),
		Output:       errorCode.Output,
		Code:         errorCode.Code,
		Message:      errorCode.Message,
		Overhead:     xCtxUtil.CalcOverheadTime(ctx),
		ErrorMessage: errorMessage,
		Data:         data,
	})
}
