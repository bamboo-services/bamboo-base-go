package xUtil

import (
	pack "github.com/bamboo-services/bamboo-base-go/common/utility/package"
	"github.com/gin-gonic/gin"
)

// Bind 返回请求绑定工具实例，提供 JSON Body、Query、URI、Header 等多种绑定方式。
//
// 泛型参数 T 为目标结构体类型，绑定失败时自动处理验证错误并中断请求。
//
// 参数说明:
//   - ctx: 当前 HTTP 请求的 Gin 上下文对象
//   - data: 泛型结构体指针，用于接收绑定后的数据
//
// 返回值:
//   - *pack.Binding[T] 绑定工具实例，可链式调用 BindData / BindQuery / BindURI / BindHeader
//
// 使用方式：
//
//	var req CreateUserRequest
//	if data := xUtil.Bind(ctx, &req).BindData(); data == nil {
//	    return
//	}
//	// 也支持 BindQuery / BindURI / BindHeader
//	xUtil.Bind(ctx, &req).BindQuery()
func Bind[T any](ctx *gin.Context, data *T) *pack.Binding[T] {
	return &pack.Binding[T]{
		Context: ctx,
		Data:    data,
	}
}

// Encryption 返回加密工具实例，提供 SHA256、MD5 等哈希计算方法。
//
// 使用方式：
//
//	xUtil.Encryption().SHA256("data")
//	xUtil.Encryption().MD5("data")
func Encryption() *pack.Encryption { return &pack.Encryption{} }

// Generate 返回生成工具实例，提供各类随机字符串生成方法。
//
// 使用方式：
//
//	xUtil.Generate().RandomString(32)
func Generate() *pack.Generate { return &pack.Generate{} }

// Password 返回密码工具实例，提供密码加密与验证方法。
//
// 使用方式：
//
//	xUtil.Password().Encrypt("password")
//	xUtil.Password().IsValid("input", "hash")
func Password() *pack.Password { return &pack.Password{} }

// Security 返回安全密钥工具实例，提供安全密钥的生成与验证方法。
//
// 使用方式：
//
//	xUtil.Security().GenerateLongKey()
//	xUtil.Security().GenerateKey()
func Security() *pack.Security { return &pack.Security{} }

// Str 返回字符串工具实例，提供常用的字符串处理方法。
//
// 使用方式：
//
//	xUtil.Str().IsBlank("")
//	xUtil.Str().Mask("13812345678", 3, 4, "*")
func Str() *pack.Str { return &pack.Str{} }

// Timer 返回时间工具实例，提供常用的时间处理方法。
//
// 使用方式：
//
//	xUtil.Timer().Now()
//	xUtil.Timer().Format(t, layout)
func Timer() *pack.Timer { return &pack.Timer{} }
