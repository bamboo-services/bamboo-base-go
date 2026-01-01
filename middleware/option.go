package xMiddle

import (
	xLog "github.com/bamboo-services/bamboo-base-go/log"
	"github.com/gin-gonic/gin"
)

// AllowOption 处理 OPTIONS 请求以支持跨域预检请求。
//
// 如果检测到请求方法为 OPTIONS，记录调试日志并终止请求处理，返回 204 状态码。
func AllowOption(ctx *gin.Context) {
	if ctx.Request.Method == "OPTIONS" {
		xLog.WithName(xLog.NamedMIDE).Debug(ctx, "检测到 OPTIONS 请求，继续处理")
		ctx.AbortWithStatus(200)
	}
}
