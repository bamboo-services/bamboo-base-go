package header

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	// ErrMissingAuthorization 表示请求标头中缺少必要的 Authorization 认证标头。
	ErrMissingAuthorization = errors.New("缺少 Authorization 请求头")
	// ErrInvalidAuthorizationFormat 表示 Authorization 请求标头格式错误，必须以 'Bearer ' 前缀开头。
	ErrInvalidAuthorizationFormat = errors.New("请求头 Authorization 格式错误，应该以 'Bearer ' 开头")
)

// GetAuthorization 从 [*gin.Context] 的请求中提取并解析 Bearer 认证令牌。
//
// 该函数严格读取 [HeaderAuthorization] 请求头：
//   - 若请求上下文为 nil、未挂载 Request 或请求头为空，返回 [ErrMissingAuthorization]；
//   - 若请求头格式合法（以 "Bearer " 前缀开头），剥离前缀并返回后续具体的令牌字符串；
//   - 若请求头内容未以 "Bearer " 前缀开头，返回 [ErrInvalidAuthorizationFormat]。
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

// GetToken 从 [*gin.Context] 的请求中获取指定类型的令牌字符串。
//
// 参数 tokenType 支持传入 [HeaderAuthorization] 或 [HeaderRefreshToken]：
//   - 若传入的 tokenType 非上述两者，内部将默认回退使用 [HeaderAuthorization]；
//   - 若请求上下文为 nil、未挂载 Request 或指定标头为空，返回空字符串；
//   - 若获取到的标头值包含 "Bearer " 前缀，会自动去除该前缀后返回纯令牌字符串；
//   - 若不包含该前缀，则原样返回标头内容。
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
