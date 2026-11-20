// Package positivefield 验证强类型错误字段的正例可以编译。
package positivefield

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// Holder 持有强类型失败结果。
type Holder struct {
	Err *apperr.Error
}

// NewHolder 用给定失败结果构造持有者。
func NewHolder(appErr *apperr.Error) *Holder {
	return &Holder{Err: appErr}
}
