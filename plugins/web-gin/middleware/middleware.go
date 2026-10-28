package middleware

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/header"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/httpresult"
	"github.com/gin-gonic/gin"
)

// CORS 构造跨域请求处理中间件。
//
// 该中间件放行常见 HTTP 方法（GET、POST、PUT、DELETE、PATCH、OPTIONS），
// 允许 Content-Type、Authorization 与 X-Request-Id 请求标头，
// 并将 Content-Length 与 [header.HeaderRequestUUID] 暴露给客户端脚本。
// 当收到预检请求（OPTIONS 方法）时，直接以 204 [http.StatusNoContent] 响应并中断请求管道。
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-Id")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, "+header.HeaderRequestUUID.String())

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Recovery 构造全局 panic 捕获与安全恢复中间件。
//
// 参数 logger 允许为 nil；非 nil 时将异常原因记录到专用日志中。
// 该中间件通过 defer recover() 拦截下游所有中间件与业务处理函数中发生的 panic：
//   - 将 panic 转换为 500 [apperr.ServerInternalError] 业务错误；
//   - 提示语固定为“系统内部异常”，绝不泄漏原始堆栈或内部调用路径；
//   - 内部通过 [httpresult.AbortError] 写回响应并调用 [gin.Context.Abort] 中断后续逻辑。
func Recovery(logger *logging.LogNamedLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				if logger != nil {
					logger.Error(c.Request.Context(), fmt.Sprintf("webgin: 请求处理 panic: %v", r))
				}
				panicErr := apperr.New(apperr.ServerInternalError, "系统内部异常", fmt.Errorf("panic: %v", r))
				httpresult.AbortError(c, panicErr)
			}
		}()
		c.Next()
	}
}

// Response 构造统一响应后置检查与兜底中间件。
//
// 该中间件在下游处理函数执行完毕后检查响应写入状态：
//   - 若下游已调用写入方法（[gin.ResponseWriter.Written] 为 true），则直接放行；
//   - 若下游未写入且 [gin.Context.Errors] 存在错误，优先按最后一个错误映射输出；
//   - 若下游未写入且无错误，且当前状态码非 301 或 302 重定向，
//     自动输出 [apperr.DeveloperError] 提示开发者检查代码逻辑。
func Response() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if !c.Writer.Written() {
			if len(c.Errors) > 0 {
				lastErr := c.Errors.Last().Err
				var appErr *apperr.Error
				if errors.As(lastErr, &appErr) {
					httpresult.Error(c, appErr)
				} else {
					httpresult.Error(c, apperr.New(apperr.ServerInternalError, "系统内部异常", lastErr))
				}
				c.Abort()
				return
			}

			// 无错误也未写入任何内容（排查重定向 301/302）
			status := c.Writer.Status()
			if status != http.StatusMovedPermanently && status != http.StatusFound {
				httpresult.Error(c, apperr.New(apperr.DeveloperError, "没有正常输出信息或报错信息，请检查代码逻辑「开发者错误」", nil))
				c.Abort()
			}
		}
	}
}
