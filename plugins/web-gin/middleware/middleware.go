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

// CORS 允许常见跨域请求头部和方法。
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

// Recovery 捕获下游 Handler 的 panic 并安全转换为 httpresult 错误，绝不泄露底层堆栈。
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

// Response 统一响应兜底中间件，未写入响应且有错误或异常未输出时提供标准兜底。
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
