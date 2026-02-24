package xMiddle

import "github.com/gin-gonic/gin"

// ReleaseAllCors 设置跨域请求的头部信息，允许所有来源的请求，并支持常用的 HTTP 方法和头部。
func ReleaseAllCors(ctx *gin.Context) {
	ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	ctx.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	ctx.Next()
}
