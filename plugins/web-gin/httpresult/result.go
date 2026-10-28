package httpresult

import (
	"net/http"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/envelope"
	"github.com/gin-gonic/gin"
)

const (
	// DefaultSuccessOutput 是纯成功响应的默认输出标识。
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

// Success 向客户端返回 200 成功响应，包含 Context、Code、Output、Message 与可选 Overhead。
// 避免重复写入响应。
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

// SuccessWithData 向客户端返回 200 成功响应并附带数据。
// 避免重复写入响应。
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

// Error 向客户端写回基于 apperr.Error 映射的状态码与标准 Response JSON 响应。
// 避免重复写入响应。
func Error(ctx *gin.Context, err *apperr.Error) {
	if ctx == nil || ctx.Writer.Written() {
		return
	}
	status, resp := buildEnvelope(ctx, err)
	ctx.JSON(status, resp)
}

// AbortError 终止当前 gin 请求处理并写回基于 apperr.Error 映射的错误响应。
// 与 AbortWithStatusJSON 语义一致，避免重复写入响应。
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
