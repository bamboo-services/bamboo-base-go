package xCtxUtil

import (
	"context"
	"time"

	xCtx "github.com/bamboo-services/bamboo-base-go/context"
	"github.com/bamboo-services/bamboo-base-go/env"
)

// IsDebugMode 判断当前是否处于调试模式。
//
// 该函数通过读取环境变量 `XLF_DEBUG` 来确定调试模式状态。
//
// 返回值:
//   - 返回 `true` 表示处于调试模式。
//   - 返回 `false` 表示不在调试模式。
func IsDebugMode() bool {
	return xEnv.GetEnvBool(xEnv.Debug, false)
}

// CalcOverheadTime 计算当前请求的耗时（微秒级）。
//
// 该函数检查请求是否处于调试模式，如果是，则计算从 `UserStartTimeKey` 到当前时间的耗时。
// 非调试模式下，始终返回 0。
//
// 参数 ctx 表示当前的 `context.Context`，用于访问请求上下文数据。
//
// 返回值为耗时的整数值（单位：微秒），当未启用调试模式时返回 0。
func CalcOverheadTime(ctx context.Context) int64 {
	if IsDebugMode() {
		if startTimeValue := ctx.Value(xCtx.UserStartTimeKey); startTimeValue != nil {
			if startTime, ok := startTimeValue.(time.Time); ok {
				return time.Since(startTime).Microseconds()
			}
		}
	}
	return 0
}

// GetRequestKey 从上下文中获取请求唯一标识键。
//
// 该函数获取当前请求的唯一标识，如果不存在则返回空字符串。
//
// 参数说明:
//   - ctx: `context.Context` 上下文对象
//
// 返回值:
//   - 请求唯一标识字符串
func GetRequestKey(ctx context.Context) string {
	if value := ctx.Value(xCtx.RequestKey); value != nil {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// GetErrorMessage 从上下文中获取错误消息。
//
// 该函数获取当前请求的错误消息，如果不存在则返回空字符串。
//
// 参数说明:
//   - ctx: `context.Context` 上下文对象
//
// 返回值:
//   - 错误消息字符串
func GetErrorMessage(ctx context.Context) string {
	if value := ctx.Value(xCtx.ErrorMessageKey); value != nil {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}
