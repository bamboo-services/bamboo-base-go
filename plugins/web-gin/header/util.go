package header

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	// ErrMissingAuthorization 表示缺少 Authorization 请求头。
	ErrMissingAuthorization = errors.New("缺少 Authorization 请求头")
	// ErrInvalidAuthorizationFormat 表示 Authorization 格式错误。
	ErrInvalidAuthorizationFormat = errors.New("请求头 Authorization 格式错误，应该以 'Bearer ' 开头")
)

// GetAuthorization 从 gin.Context 中提取并解析 Authorization 请求头。
// 期望请求头包含 Bearer Token 格式的认证信息。
func GetAuthorization(ctx *gin.Context) (string, error) {
	if ctx == nil || ctx.Request == nil {
		return "", ErrMissingAuthorization
	}
	getHeader := ctx.GetHeader(HeaderAuthorization.String())
	if getHeader == "" {
		return "", ErrMissingAuthorization
	}
	if strings.HasPrefix(getHeader, "Bearer ") {
		return getHeader[7:], nil
	}
	return "", ErrInvalidAuthorizationFormat
}

// GetToken 从 gin.Context 中获取指定类型的 Token 字符串。
// 如果 tokenType 不是 HeaderAuthorization 或 HeaderRefreshToken，则默认使用 HeaderAuthorization。
// 如果请求头值包含 "Bearer " 前缀，会自动去除该前缀。
func GetToken(ctx *gin.Context, tokenType Header) string {
	if ctx == nil || ctx.Request == nil {
		return ""
	}
	switch tokenType {
	case HeaderAuthorization, HeaderRefreshToken:
	default:
		tokenType = HeaderAuthorization
	}
	getToken := ctx.GetHeader(tokenType.String())
	if strings.HasPrefix(getToken, "Bearer ") {
		return getToken[7:]
	}
	return getToken
}
