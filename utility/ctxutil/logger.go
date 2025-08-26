package xCtxUtil

import (
	xConsts "github.com/bamboo-services/bamboo-base-go/constants"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GetLogger 从 `gin.Context` 中获取业务日志记录器。
//
// 该函数优先尝试从上下文中获取 `ContextLoggerBusiness` 对应的日志记录器，
// 如果存在且类型匹配，则返回该日志记录器。
// 如果上下文中未找到 `ContextLoggerBusiness` 或其类型不匹配，则返回默认的 `ContextLogger` 日志记录器。
//
// 参数说明:
//   - c: `gin.Context` 上下文对象，用于存储和获取请求范围内的数据。
//
// 返回值:
//   - 返回一个类型为 `*zap.Logger` 的日志记录器实例，确保始终返回非空的日志对象。
//
// 注意: 确保上下文中已正确设置 `ContextLogger`，否则可能会引发 panic。
func GetLogger(c *gin.Context) *zap.Logger {
	value, exists := c.Get(xConsts.ContextLogger.String())
	if exists {
		if logger, ok := value.(*zap.Logger); ok {
			return logger
		}
	}
	return c.MustGet(xConsts.ContextLogger.String()).(*zap.Logger)
}

// GetSugarLogger 从 `gin.Context` 中获取业务日志记录器的 Sugar 版本。
//
// 该函数首先尝试从上下文中获取 `ContextLoggerBusiness` 对应的日志记录器，
// 如果存在且类型匹配，则返回该日志记录器的 Sugar 版本。
// 如果上下文中未找到 `ContextLoggerBusiness` 或其类型不匹配，则
// 返回默认的 `ContextLogger` 日志记录器的 Sugar 版本。
//
// 参数说明:
//   - c: `gin.Context` 上下文对象，用于存储和获取 请求范围内的数据。
//
// 返回值:
//   - 返回一个类型为 `*zap.SugaredLogger` 的日志记录器实例，确保始终返回非空的日志对象。
func GetSugarLogger(c *gin.Context) *zap.SugaredLogger {
	value, exists := c.Get(xConsts.ContextLogger.String())
	if exists {
		if logger, ok := value.(*zap.Logger); ok {
			return logger.Sugar()
		}
	}
	return c.MustGet(xConsts.ContextLogger.String()).(*zap.Logger).Sugar()
}
