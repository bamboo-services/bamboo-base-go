// Package negativepluginreturn 验证原生错误不能直接作为强类型失败返回。
package negativepluginreturn

import (
	"context"
	"errors"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// WeakPlugin 的 Start 声明强类型失败出口，却返回原生错误，必须编译失败。
type WeakPlugin struct{}

// Start 必须在强类型位置因类型不匹配而编译失败。
func (p *WeakPlugin) Start(ctx context.Context) *apperr.Error {
	return errors.New("原生错误不能作为强类型失败返回")
}
