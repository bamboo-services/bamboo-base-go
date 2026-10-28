package ctxkey

import (
	"context"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
)

// IsDebugMode 判断当前进程是否处于调试模式。
//
// 从环境变量 APP_DEBUG（[env.Debug]）读取并解析布尔值；
// 变量未设置或解析失败时稳健返回 false。
func IsDebugMode() bool {
	val, ok := env.Lookup(env.Debug)
	if !ok {
		return false
	}
	b, err := val.Bool()
	if err != nil {
		return false
	}
	return b
}

// CalcOverheadTime 计算当前请求的耗时，单位微秒。
//
// 直接从 [UserStartTimeKey] 读取 time.Time 类型的起始时刻，
// 不依赖任何全局提取器；仅在调试模式（[IsDebugMode] 为 true）下计算。
// 非调试模式、nil 上下文、键缺失、值类型错误、零值时间
// 或起始时刻晚于当前时间时，均稳健返回 0。
func CalcOverheadTime(ctx context.Context) int64 {
	if !IsDebugMode() || ctx == nil {
		return 0
	}
	val := ctx.Value(UserStartTimeKey)
	if val == nil {
		return 0
	}
	startTime, ok := val.(time.Time)
	if !ok || startTime.IsZero() {
		return 0
	}
	now := time.Now()
	if startTime.After(now) {
		return 0
	}
	diff := now.Sub(startTime).Microseconds()
	if diff < 0 {
		return 0
	}
	return diff
}

// GetRequestKey 从上下文中读取请求唯一标识。
//
// 直接从 [RequestKey] 提取 string 值；nil 上下文、键不存在
// 或值类型不匹配时安全返回空字符串。
func GetRequestKey(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	val := ctx.Value(RequestKey)
	if val == nil {
		return ""
	}
	str, ok := val.(string)
	if !ok {
		return ""
	}
	return str
}

// GetErrorMessage 从上下文中读取错误消息。
//
// 直接从 [ErrorMessageKey] 提取 string 值；nil 上下文、键不存在
// 或值类型不匹配时安全返回空字符串。
func GetErrorMessage(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	val := ctx.Value(ErrorMessageKey)
	if val == nil {
		return ""
	}
	str, ok := val.(string)
	if !ok {
		return ""
	}
	return str
}
