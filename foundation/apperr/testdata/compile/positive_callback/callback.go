// Package positivecallback 验证强类型失败回调的正例可以编译。
package positivecallback

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// FailureHandler 是强类型失败回调。
type FailureHandler func() *apperr.Error

// Register 注册并返回强类型失败回调。
func Register(handler FailureHandler) FailureHandler {
	return handler
}
