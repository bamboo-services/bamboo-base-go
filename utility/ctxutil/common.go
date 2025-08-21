package xCtxUtil

import (
	xConsts "github.com/bamboo-services/bamboo-base-go/constants"
	xError "github.com/bamboo-services/bamboo-base-go/error"
	xModels "github.com/bamboo-services/bamboo-base-go/models"
	xUtil "github.com/bamboo-services/bamboo-base-go/utility"
	"github.com/gin-gonic/gin"
	"time"
)

// IsDebugMode 判断当前请求是否处于调试模式。
//
// 该函数通过从上下文中获取 `xConfig` 实例，检查其 `Debug` 字段来确定调试模式状态。
//
// 返回值:
//   - 返回 `true` 表示处于调试模式。
//   - 返回 `false` 表示不在调试模式，或者上下文中未找到对应配置。
func IsDebugMode(c *gin.Context) bool {
	value, exists := c.Get(xConsts.ContextConfig)
	if exists {
		config := value.(xModels.Config)
		return config.Xlf.Debug
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
	value, exists := c.Get(xConsts.ContextConfig)
	if exists {
		return value.(*xModels.Config)
	}
	GetLogger(c).Named(xConsts.LogUTIL).Panic("在上下文中找不到应用配置，真的注入成功了吗？")
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
	return c.GetString(xConsts.ContextRequestKey)
}

// GetErrorCode 从上下文中获取错误代码。
//
// 该函数获取当前请求的错误代码，如果不存在则返回 nil。
//
// 参数说明:
//   - c: `*gin.Context` 上下文对象
//
// 返回值:
//   - `*xError.ErrorCode` 错误代码，如果不存在则返回 nil
func GetErrorCode(c *gin.Context) *xError.ErrorCode {
	value, exists := c.Get(xConsts.ContextErrorCode)
	if exists {
		return value.(*xError.ErrorCode)
	}
	return nil
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
	return c.GetString(xConsts.ContextErrorMessage)
}
