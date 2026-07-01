package utility

import "github.com/gin-gonic/gin"

// Bind 返回请求绑定工具实例，提供 JSON Body、Query、URI、Header 等多种绑定方式。
//
// 泛型参数 T 为目标结构体类型，绑定失败时自动处理验证错误并中断请求。
//
// 参数说明:
//   - ctx: 当前 HTTP 请求的 Gin 上下文对象
//   - data: 泛型结构体指针，用于接收绑定后的数据
//
// 返回值:
//   - *Binding[T] 绑定工具实例，可链式调用 Data / Query / URI / Header
//
// 使用方式：
//
//	var req CreateUserRequest
//	if data := xUtil.Bind(ctx, &req).Data(); data == nil {
//	    return
//	}
//	// 也支持 Query / URI / Header
//	xUtil.Bind(ctx, &req).Query()
func Bind[T any](ctx *gin.Context, data *T) *Binding[T] {
	return &Binding[T]{
		Context: ctx,
		GetData: data,
	}
}
