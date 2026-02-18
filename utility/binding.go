package xUtil

import (
	xVaild "github.com/bamboo-services/bamboo-base-go/validator"
	"github.com/gin-gonic/gin"
)

// BindData 绑定请求体数据到指定结构体并进行校验
//
// 该函数用于从 HTTP 请求体中绑定 JSON 数据到泛型结构体 `data`，并根据绑定结果处理错误逻辑。
// 如果绑定失败，会调用 `HandleValidationError` 输出错误信息，并终止后续处理。
//
// 参数说明:
//   - data: 泛型结构体指针，用于接收绑定后的数据。
//   - ctx: 当前 HTTP 请求的上下文对象。
//
// 返回值:
//   - 返回泛型结构体指针，当绑定成功时为 `&data`，否则为 `nil`。
//
// 注意: 该函数仅支持 JSON 格式的请求体数据。异常情况会结束当前 HTTP 请求生命周期。
func BindData[T any](ctx *gin.Context, data *T) *T {
	bindErr := ctx.ShouldBindBodyWithJSON(&data)
	if bindErr != nil {
		xVaild.HandleValidationError(ctx, bindErr)
		ctx.Abort()
		return nil
	}
	return data
}

// BindQuery 将查询参数绑定到指定的结构体指针并处理验证错误。
//
// 如果绑定或验证失败，会通过 HandleValidationError 统一处理错误响应并中断请求上下文。
// 成功时返回填充了数据的指针，失败时返回 nil。
//
// 参数说明:
//   - data: 泛型结构体指针，用于接收绑定后的数据。
//   - ctx: 当前 HTTP 请求的上下文对象。
//
// 返回值:
//   - 返回泛型结构体指针，当绑定成功时为 `&data`，否则为 `nil`。
func BindQuery[T any](ctx *gin.Context, data *T) *T {
	bindErr := ctx.ShouldBindQuery(data)
	if bindErr != nil {
		xVaild.HandleValidationError(ctx, bindErr)
		ctx.Abort()
		return nil
	}
	return data
}

// BindURI 将 URI 路径参数绑定到指定的结构体指针并处理验证错误。
//
// 如果绑定或验证失败，会通过 HandleValidationError 统一处理错误响应并中断请求上下文。
// 成功时返回填充了数据的指针，失败时返回 nil。
//
// 参数说明:
//   - data: 泛型结构体指针，用于接收绑定后的数据。
//   - ctx: 当前 HTTP 请求的上下文对象。
//
// 返回值:
//   - 返回泛型结构体指针，当绑定成功时为 `&data`，否则为 `nil`。
func BindURI[T any](ctx *gin.Context, data *T) *T {
	bindErr := ctx.ShouldBindUri(data)
	if bindErr != nil {
		xVaild.HandleValidationError(ctx, bindErr)
		ctx.Abort()
		return nil
	}
	return data
}

// BindHeader 将 HTTP 请求头绑定到指定的结构体指针并处理验证错误。
//
// 如果绑定或验证失败，会通过 HandleValidationError 统一处理错误响应并中断请求上下文。
// 成功时返回填充了数据的指针，失败时返回 nil。
//
// 参数说明:
//   - data: 泛型结构体指针，用于接收绑定后的数据。
//   - ctx: 当前 HTTP 请求的上下文对象。
//
// 返回值:
//   - 返回泛型结构体指针，当绑定成功时为 `&data`，否则为 `nil`。
func BindHeader[T any](ctx *gin.Context, data *T) *T {
	bindErr := ctx.ShouldBindHeader(data)
	if bindErr != nil {
		xVaild.HandleValidationError(ctx, bindErr)
		ctx.Abort()
		return nil
	}
	return data
}
