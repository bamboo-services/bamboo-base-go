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
//   - *pack.Binding[T] 绑定工具实例，可链式调用 Data / Query / URI / Header
//
// 使用方式：
//
//	var req CreateUserRequest
//	if data := xUtil.Bind(ctx, &req).Data(); data == nil {
//	    return
//	}
//	// 也支持 Query / URI / Header
//	xUtil.Bind(ctx, &req).Query()
func Bind[T any](ctx *gin.Context, data *T) *pack.Binding[T] {
	return &pack.Binding[T]{
		Context: ctx,
		GetData: data,
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

// Parse 返回一个新的 `Parse` 实例，用于执行多种类型的值解析操作。
//
// 该函数通常用于初始化值转换工具，工具内包含丰富的类型解析方法。
func Parse() *pack.Parse {
	return &pack.Parse{}
}

// Valid 返回一个新的 `pack.Valid` 实例，用于执行各种验证任务。
//
// 通过 `Valid` 函数获取的验证器实例，您可以调用其方法完成对手机号、身份证号码、URL、IP 等格式的验证。
// 此函数不接受任何参数，也不会直接返回错误。
func Valid() *pack.Valid {
	return &pack.Valid{}
}

// Function 创建并返回一个新的 Function 实例。
//
// 返回的实例用于提供函数名称获取、方法名称解析等反射相关的工具方法。
func Function() *pack.Function {
	return &pack.Function{}
}
