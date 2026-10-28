package httpresult

import (
	"net/http"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/envelope"
	"github.com/gin-gonic/gin"
)

const (
	// DefaultSuccessOutput 是标准成功响应时信封中 output 字段的默认标识短语（"Success"）。
	DefaultSuccessOutput apperr.Output = "Success"
)

// resolveRequestKey 安全地从 ctx.Request.Context() 提取 RequestKey。
func resolveRequestKey(ctx *gin.Context) string {
	if ctx == nil || ctx.Request == nil {
		return ""
	}
	reqCtx := ctx.Request.Context()
	if reqCtx == nil {
		return ""
	}
	return ctxkey.GetRequestKey(reqCtx)
}

// resolveOverhead 安全地从 ctx.Request.Context() 计算耗时微秒。
func resolveOverhead(ctx *gin.Context) int64 {
	if ctx == nil || ctx.Request == nil {
		return 0
	}
	reqCtx := ctx.Request.Context()
	if reqCtx == nil {
		return 0
	}
	return ctxkey.CalcOverheadTime(reqCtx)
}

// resolveHTTPStatus 根据错误码计算 HTTP 状态码。
// 业务标准错误路径只允许 400..599，非错误区间（<400 或 >599，如 1xx/2xx/3xx/负数/6xx）统一回退至 500。
func resolveHTTPStatus(code apperr.Code) int {
	status := int(code / 100)
	if status < http.StatusBadRequest || status > 599 {
		return http.StatusInternalServerError
	}
	return status
}

// buildEnvelope 根据错误构建安全的 envelope.Response。
// 严禁将底层内部 err.Err 暴露至响应中；ErrorMessage 仅使用业务显式指定的安全内容。
func buildEnvelope(ctx *gin.Context, err *apperr.Error) (int, envelope.Response) {
	reqKey := resolveRequestKey(ctx)
	overhead := resolveOverhead(ctx)

	if err == nil {
		fallback := apperr.ServerInternalError
		resp := envelope.Response{
			Context:  reqKey,
			Code:     fallback.Code,
			Output:   fallback.Output,
			Message:  fallback.Message,
			Overhead: overhead,
		}
		return http.StatusInternalServerError, resp
	}

	statusCode := resolveHTTPStatus(err.Code.Code)

	resp := envelope.Response{
		Context:      reqKey,
		Code:         err.Code.Code,
		Output:       err.Code.Output,
		Message:      err.Code.Message,
		ErrorMessage: err.ErrorMessage,
		Overhead:     overhead,
		Data:         err.Data,
	}

	return statusCode, resp
}

// Success 向客户端写回 HTTP 200 成功响应，不携带额外 data 数据载荷。
//
// 响应内容严格包装为 [envelope.Response] 协议信封，包含由请求上下文解析的追踪标识、
// 状态码 200、输出标识 [DefaultSuccessOutput]、调用方传入的 message 提示以及微秒级处理耗时。
// 若当前连接已写入过响应（[gin.ResponseWriter.Written] 返回 true），本函数安全忽略以防重复写入。
func Success(ctx *gin.Context, message string) {
	if ctx == nil || ctx.Writer.Written() {
		return
	}
	resp := envelope.Response{
		Context:  resolveRequestKey(ctx),
		Code:     http.StatusOK,
		Output:   DefaultSuccessOutput,
		Message:  apperr.Message(message),
		Overhead: resolveOverhead(ctx),
	}
	ctx.JSON(http.StatusOK, resp)
}

// SuccessWithData 向客户端写回 HTTP 200 成功响应，并附带业务数据载荷 data。
//
// 载荷数据将赋值给 [envelope.Response.Data] 字段并统一序列化为 JSON。
// 若当前连接已完成响应写入，本函数安全忽略以防止协议错误或重复输出。
func SuccessWithData(ctx *gin.Context, message string, data any) {
	if ctx == nil || ctx.Writer.Written() {
		return
	}
	resp := envelope.Response{
		Context:  resolveRequestKey(ctx),
		Code:     http.StatusOK,
		Output:   DefaultSuccessOutput,
		Message:  apperr.Message(message),
		Overhead: resolveOverhead(ctx),
		Data:     data,
	}
	ctx.JSON(http.StatusOK, resp)
}

// Error 向客户端写回基于 [*apperr.Error] 映射的错误响应及标准 [envelope.Response] JSON 信封。
//
// 该函数执行以下安全与映射规则：
//   - 若传入的 err 为 nil，自动以 [apperr.ServerInternalError] 回退构造 500 响应；
//   - 计算 HTTP 状态码为 err.Code.Code / 100，非 400..599 区间一律回退为 500；
//   - 底层根因错误严格隔离，响应中仅输出安全的 ErrorMessage 与业务错误码；
//   - 若连接已写过响应，本函数安全忽略。
func Error(ctx *gin.Context, err *apperr.Error) {
	if ctx == nil || ctx.Writer.Written() {
		return
	}
	status, resp := buildEnvelope(ctx, err)
	ctx.JSON(status, resp)
}

// AbortError 向客户端写回基于 [*apperr.Error] 映射的错误响应，并中断当前 Gin 请求管道。
//
// 内部调用 [gin.Context.AbortWithStatusJSON] 阻止后续处理函数与后置中间件继续执行。
// 映射算法与安全信封构建规则与 [Error] 完全一致；若连接已写过响应，则仅执行 [gin.Context.Abort]。
func AbortError(ctx *gin.Context, err *apperr.Error) {
	if ctx == nil || ctx.Writer.Written() {
		if ctx != nil {
			ctx.Abort()
		}
		return
	}
	status, resp := buildEnvelope(ctx, err)
	ctx.AbortWithStatusJSON(status, resp)
}
