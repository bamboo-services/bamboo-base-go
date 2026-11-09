// Package positivereturn 验证强类型失败出口的正例可以编译。
package positivereturn

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// StrongFailure 以统一错误指针返回失败结果。
func StrongFailure() *apperr.Error {
	return apperr.New(apperr.FrameworkInternalError, "构造失败")
}
