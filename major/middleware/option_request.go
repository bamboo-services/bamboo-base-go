package xMiddle

import (
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/gin-gonic/gin"
)

// AllowOption 允许 HTTP OPTIONS 预检请求通过。
// 若检测到请求方法为 OPTIONS，则记录调试日志并终止请求返回 200 状态码。
//
// Deprecated: AllowOption 命名不符合规范，且与 AllowOptionRequest 重复。
// 请改用 AllowOptionRequest，后续版本将移除本函数。
func AllowOption(ctx *gin.Context) {
	if ctx.Request.Method == "OPTIONS" {
		xLog.WithName(xLog.NamedMIDE).Debug(ctx, "检测到 OPTIONS 请求，继续处理")
		ctx.AbortWithStatus(200)
	}
}

// AllowOptionRequest 允许 HTTP OPTIONS 预检请求通过。
// 若检测到请求方法为 OPTIONS，则记录调试日志并终止请求返回 200 状态码。
func AllowOptionRequest(ctx *gin.Context) {
	if ctx.Request.Method == "OPTIONS" {
		xLog.WithName(xLog.NamedMIDE).Debug(ctx, "检测到 OPTIONS 请求，继续处理")
		ctx.AbortWithStatus(200)
	}
}
