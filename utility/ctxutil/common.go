package xCtxUtil

import (
	xConsts "github.com/bamboo-services/bamboo-base-go/constants"
	xModels "github.com/bamboo-services/bamboo-base-go/models"
	xUtil "github.com/bamboo-services/bamboo-base-go/utility"
	"github.com/gin-gonic/gin"
	"time"
)

// IsDebugMode 判断当前请求是否处于调试模式。
//
// 该函数通过从上下文中获取 `AwakenConfig` 实例，检查其 `Debug` 字段来确定调试模式状态。
//
// 返回值:
//   - 返回 `true` 表示处于调试模式。
//   - 返回 `false` 表示不在调试模式，或者上下文中未找到对应配置。
func IsDebugMode(c *gin.Context) bool {
	value, exists := c.Get(xConsts.ContextConfig)
	if exists {
		config := value.(xModels.AwakenConfig)
		return config.Awaken.Debug
	} else {
		return false
	}
}

// CalcOverheadTime 计算当前请求的开销时间（以毫秒为单位）。
//
// 该函数通过上下文中的用户起始时间和当前时间的差值计算请求的处理开销，仅在调试模式下生效。
//
// 参数说明:
//   - c: 包含请求上下文的 `*gin.Context` 对象。
//
// 返回值:
//   - 整型指针，表示开销时间（以毫秒为单位）。如果当前不在调试模式，则返回 `nil`。
//
// 注意: 确保 `ContextUserStartTime` 在上下文中已正确设置，否则可能导致异常行为。
func CalcOverheadTime(c *gin.Context) *int64 {
	if IsDebugMode(c) {
		startTime := c.GetTime(xConsts.ContextUserStartTime)
		return xUtil.Ptr(time.Now().Sub(startTime).Milliseconds())
	}
	return nil
}
