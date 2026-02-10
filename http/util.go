package xHttp

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetAuthorization 从上下文中提取并解析 Authorization 请求头。
//
// 该函数期望请求头包含 Bearer Token 格式的认证信息。
// 如果请求头缺失或格式不正确，将返回相应的错误。
//
// 返回解析后的 Token 字符串和可能的错误。
func GetAuthorization(ctx *gin.Context) (string, error) {
	getHeader := ctx.GetHeader(HeaderAuthorization.String())
	if getHeader == "" {
		return "", fmt.Errorf("缺少 Authorization 请求头")
	}
	if strings.HasPrefix(getHeader, "Bearer ") {
		return getHeader[7:], nil
	}
	return "", fmt.Errorf("请求头 Authorization 格式错误，应该以 'Bearer ' 开头")
}

// GetToken 从 gin.Context 中获取指定类型的 Token 字符串。
//
// 如果 tokenType 不是 HeaderAccessToken 或 HeaderRefreshToken，则默认使用 HeaderAccessToken。
// 如果请求头值包含 "Bearer " 前缀，会自动去除该前缀。
func GetToken(ctx *gin.Context, tokenType Header) string {
	switch tokenType {
	case HeaderAccessToken, HeaderRefreshToken:
	default:
		tokenType = HeaderAccessToken
	}
	getToken := ctx.GetHeader(tokenType.String())
	if strings.HasPrefix(getToken, "Bearer ") {
		return getToken[7:]
	}
	return getToken
}
