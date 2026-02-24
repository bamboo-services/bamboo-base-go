package pack

import (
	xVaild "github.com/bamboo-services/bamboo-base-go/common/validator"
	"github.com/gin-gonic/gin"
)

type Binding[T any] struct {
	Context *gin.Context // 上下文
	GetData *T           // 转化的数据
}

// Data 绑定请求体数据到指定结构体并进行校验
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
func (u *Binding[T]) Data() *T {
	bindErr := u.Context.ShouldBindBodyWithJSON(&u.GetData)
	if bindErr != nil {
		xVaild.HandleValidationError(u.Context, bindErr)
		u.Context.Abort()
		return nil
	}
	return u.GetData
}

// Query 将查询参数绑定到指定的结构体指针并处理验证错误。
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
func (u *Binding[T]) Query() *T {
	bindErr := u.Context.ShouldBindQuery(u.GetData)
	if bindErr != nil {
		xVaild.HandleValidationError(u.Context, bindErr)
		u.Context.Abort()
		return nil
	}
	return u.GetData
}

// URI 将 URI 路径参数绑定到指定的结构体指针并处理验证错误。
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
func (u *Binding[T]) URI() *T {
	bindErr := u.Context.ShouldBindUri(u.GetData)
	if bindErr != nil {
		xVaild.HandleValidationError(u.Context, bindErr)
		u.Context.Abort()
		return nil
	}
	return u.GetData
}

// Header 将 HTTP 请求头绑定到指定的结构体指针并处理验证错误。
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
func (u *Binding[T]) Header() *T {
	bindErr := u.Context.ShouldBindHeader(u.GetData)
	if bindErr != nil {
		xVaild.HandleValidationError(u.Context, bindErr)
		u.Context.Abort()
		return nil
	}
	return u.GetData
}
