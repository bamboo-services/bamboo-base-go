// Package negativejoin 验证原生错误不能直接进入真实Join API的强类型成员。
package negativejoin

import (
	"errors"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

func JoinNative() *apperr.Error {
	return apperr.Join(apperr.FrameworkMultipleFailures, "", errors.New("原生错误必须先归一"))
}
