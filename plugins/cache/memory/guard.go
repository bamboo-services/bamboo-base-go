package memory

import (
	"errors"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver"
)

// errNotInitialized 是缓存实例不可用时的稳定原因。
var errNotInitialized = errors.New("memory: cache instance is not initialized")

// notInitializedErr 返回缓存实例不可用时的统一错误。
//
// 用于 nil 接收器或未注入 [Store] 的场景：调用方在错误路径上得到
// [driver.OperationFailed]（50039），而不是触发 panic。
func notInitializedErr() *apperr.Error {
	return apperr.New(driver.OperationFailed, "缓存实例未初始化", errNotInitialized)
}
