// Package negativereturn 验证原生错误不能直接作为强类型失败返回。
package negativereturn

import (
	"errors"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// StrongFailure 直接返回原生错误，必须在强类型位置编译失败。
func StrongFailure() *apperr.Error {
	return errors.New("原生错误不能作为强类型失败返回")
}
