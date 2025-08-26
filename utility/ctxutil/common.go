package xCtxUtil

import (
	"time"

	xConsts "github.com/bamboo-services/bamboo-base-go/constants"
	xModels "github.com/bamboo-services/bamboo-base-go/models"
	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
)

// IsDebugMode 判断当前请求是否处于调试模式。
//
// 该函数通过从上下文中获取 `xConfig` 实例，检查其 `Debug` 字段来确定调试模式状态。
//
// 返回值:
//   - 返回 `true` 表示处于调试模式。
//   - 返回 `false` 表示不在调试模式，或者上下文中未找到对应配置。
func IsDebugMode(c *gin.Context) bool {
	value, exists := c.Get(xConsts.ContextConfig.String())
	if exists {
		config := value.(*map[string]interface{})
		var getConfig xModels.Config
		err := mapstructure.Decode(config, &getConfig)
		if err != nil {
			return false
		}
		return getConfig.Xlf.Debug
	} else {
		return false
	}
}

// CalcOverheadTime 计算当前请求的耗时（微秒级）。
//
// 该函数检查请求是否处于调试模式，如果是，则计算从 `ContextUserStartTime` 到当前时间的耗时。
// 非调试模式下，始终返回 0。
//
// 参数 c 表示当前的 `gin.Context`，用于访问请求上下文数据。
//
// 返回值为耗时的整数值（单位：微秒），当未启用调试模式时返回 0。
func CalcOverheadTime(c *gin.Context) int64 {
	if IsDebugMode(c) {
		startTime := c.GetTime(xConsts.ContextUserStartTime.String())
		return time.Now().Sub(startTime).Microseconds()
	}
	return 0
}

// GetConfig 从上下文中获取应用配置。
//
// 该函数从 Gin 上下文中提取应用配置实例，如果配置不存在则记录错误并触发 panic。
//
// 参数说明:
//   - c: `*gin.Context` 上下文对象
//
// 返回值:
//   - `*xModels.Config` 应用配置实例
//
// 注意: 确保配置已正确注入到上下文中
func GetConfig(c *gin.Context) *xModels.Config {
	value, exists := c.Get(xConsts.ContextConfig.String())
	if exists {
		config := value.(*map[string]interface{})
		var getConfig xModels.Config
		err := mapstructure.Decode(config, &getConfig)
		if err != nil {
			return nil
		}
		return &getConfig
	}
	return nil
}

// GetRequestKey 从上下文中获取请求唯一标识键。
//
// 该函数获取当前请求的唯一标识，如果不存在则返回空字符串。
//
// 参数说明:
//   - c: `*gin.Context` 上下文对象
//
// 返回值:
//   - 请求唯一标识字符串
func GetRequestKey(c *gin.Context) string {
	return c.GetString(xConsts.ContextRequestKey.String())
}

// GetErrorMessage 从上下文中获取错误消息。
//
// 该函数获取当前请求的错误消息，如果不存在则返回空字符串。
//
// 参数说明:
//   - c: `*gin.Context` 上下文对象
//
// 返回值:
//   - 错误消息字符串
func GetErrorMessage(c *gin.Context) string {
	return c.GetString(xConsts.ContextErrorMessage.String())
}
