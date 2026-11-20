// Package negativecallback 验证返回原生错误的回调不能赋给强类型失败回调。
package negativecallback

import (
	"errors"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// FailureHandler 是强类型失败回调。
type FailureHandler func() *apperr.Error

// Register 尝试把返回原生错误的回调赋给强类型回调，必须在强类型位置编译失败。
func Register() FailureHandler {
	return func() error { return errors.New("原生错误回调") }
}
