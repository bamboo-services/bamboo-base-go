package ctxkey

import (
	"context"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
)

// IsDebugMode 判断当前是否处于调试模式。
// 从 env.Lookup(env.Debug).Bool() 读取；缺值或解析失败稳健返回 false。
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

// CalcOverheadTime 计算当前请求的耗时（微秒级）。
// 直接从 ctxkey.UserStartTimeKey 读取 time.Time，不依赖全局 Extractor；
// 仅在调试模式下计算；非调试模式、nil 上下文、零值、缺失、类型错误或未来时间均稳健返回 0。
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

// GetRequestKey 从上下文中直接获取请求唯一标识。
// 直接从 ctx.Value(ctxkey.RequestKey) 提取 string；
// nil 上下文、键不存在或类型不匹配时安全返回空字符串。
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

// GetErrorMessage 从上下文中直接获取错误消息。
// 直接从 ctx.Value(ctxkey.ErrorMessageKey) 提取 string；
// nil 上下文、键不存在或类型不匹配时安全返回空字符串。
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
