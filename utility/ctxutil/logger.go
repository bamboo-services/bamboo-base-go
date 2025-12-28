package xCtxUtil

import (
	xConsts "github.com/bamboo-services/bamboo-base-go/constants"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GetLogger 获取带有请求追踪信息的日志记录器。
//
// 该函数使用全局日志记录器 zap.L()，并从上下文中获取请求 ID 作为 trace 字段，
// 用于在分布式系统中追踪请求链路。
//
// 参数说明:
//   - c: gin.Context 上下文对象，用于获取请求追踪信息
//   - name: 日志记录器的命名空间
//
// 返回值:
//   - *zap.Logger: 带有 trace 字段的日志记录器实例
func GetLogger(c *gin.Context, name string) *zap.Logger {
	logger := zap.L().Named(name)
	// 从上下文获取请求 ID 作为 trace
	if requestID, exists := c.Get(xConsts.ContextRequestKey.String()); exists {
		if trace, ok := requestID.(string); ok {
			return logger.With(zap.String("trace", trace))
		}
	}
	return logger
}

// GetSugarLogger 获取带有请求追踪信息的 Sugar 日志记录器。
//
// 该函数使用全局日志记录器 zap.L()，并从上下文中获取请求 ID 作为 trace 字段，
// 提供更便捷的日志记录 API。
//
// 参数说明:
//   - c: gin.Context 上下文对象，用于获取请求追踪信息
//   - name: 日志记录器的命名空间
//
// 返回值:
//   - *zap.SugaredLogger: 带有 trace 字段的 Sugar 日志记录器实例
func GetSugarLogger(c *gin.Context, name string) *zap.SugaredLogger {
	logger := zap.L().Named(name)
	// 从上下文获取请求 ID 作为 trace
	if requestID, exists := c.Get(xConsts.ContextRequestKey.String()); exists {
		if trace, ok := requestID.(string); ok {
			return logger.With(zap.String("trace", trace)).Sugar()
		}
	}
	return logger.Sugar()
}
