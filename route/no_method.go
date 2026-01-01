package xRoute

import (
	"fmt"
	"log/slog"

	xError "github.com/bamboo-services/bamboo-base-go/error"
	xLog "github.com/bamboo-services/bamboo-base-go/log"
	xResult "github.com/bamboo-services/bamboo-base-go/result"
	"github.com/gin-gonic/gin"
)

// NoMethod 处理请求方法不被允许的情况。
//
// 当访问的路由存在但使用了不被允许的 HTTP 方法时，该方法会返回一个标准化的 405 响应，包含详细的错误信息。
//
// 参数说明:
//   - ctx: `*gin.Context` 上下文对象，包含请求和响应的信息。
//
// 注意: 此方法用于全局方法不匹配的处理，需通过 `router.NoMethod` 绑定使用。
func NoMethod(ctx *gin.Context) {
	xLog.WithName(xLog.NamedROUT).Warn(ctx.Request.Context(), "请求方法不被允许",
		slog.String("method", ctx.Request.Method),
		slog.String("path", ctx.Request.URL.Path),
	)
	xResult.Error(
		ctx, xError.MethodNotAllowed,
		xError.ErrMessage(fmt.Sprintf(
			"请求方法 [%s] 不被允许访问路径 %s，请检查 API 文档中支持的 HTTP 方法",
			ctx.Request.Method,
			ctx.Request.URL.Path,
		)),
		nil,
	)
}
