// Package negativefield 验证原生错误不能装入强类型字段。
package negativefield

import (
	"errors"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// Holder 持有强类型失败结果。
type Holder struct {
	Err *apperr.Error
}

// NewHolder 尝试把原生错误装入强类型字段，必须在强类型位置编译失败。
func NewHolder() *Holder {
	return &Holder{Err: errors.New("原生错误不能装入强类型字段")}
}
