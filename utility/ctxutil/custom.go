package xCtxUtil

import (
	"context"
	"fmt"

	xCtx "github.com/bamboo-services/bamboo-base-go/context"
	xError "github.com/bamboo-services/bamboo-base-go/error"
	"github.com/gin-gonic/gin"
)

// MustGet 是一个通用的组件获取函数
// T: 想要获取的组件类型
// key: 注册时使用的 ContextKey
func MustGet[T any](ctx context.Context, key xCtx.ContextKey) T {
	if ginCtx, ok := ctx.(*gin.Context); ok {
		ctx = ginCtx.Request.Context()
	}
	if val := ctx.Value(xCtx.RegNodeKey); val != nil {
		if nodeList, ok := val.(xCtx.ContextNodeList); ok {
			if component := nodeList.Get(key); component != nil {
				if typed, ok := component.(T); ok {
					return typed
				}
			}
		}
	}
	if val := ctx.Value(key); val != nil {
		if typed, ok := val.(T); ok {
			return typed
		}
	}

	errMsg := fmt.Sprintf("SDK 组件缺失: 无法在上下文中找到 Key 为 [%v] 的组件，请确保已正确初始化", key)
	panic(errMsg)
}

// Get 是一个通用的组件获取函数（错误返回版本）。
//
// 读取顺序：
//  1. 优先从 RegNodeKey 聚合的组件 Map 中读取；
//  2. 未命中则回退到普通的 context.Value(key)；
//
// 参数说明:
//   - ctx: context.Context 上下文
//   - key: 注册时使用的 ContextKey
//
// 返回值:
//   - T: 组件实例（失败时为零值）
//   - *xError.Error: 错误信息，成功时为 nil
func Get[T any](ctx context.Context, key xCtx.ContextKey) (T, *xError.Error) {
	var zero T

	if ginCtx, ok := ctx.(*gin.Context); ok {
		ctx = ginCtx.Request.Context()
	}
	if val := ctx.Value(xCtx.RegNodeKey); val != nil {
		if nodeList, ok := val.(xCtx.ContextNodeList); ok {
			if component := nodeList.Get(key); component != nil {
				if typed, ok := component.(T); ok {
					return typed, nil
				}
			}
		}
	}

	if val := ctx.Value(key); val != nil {
		if typed, ok := val.(T); ok {
			return typed, nil
		}
	}

	return zero, &xError.Error{
		ErrorCode:    xError.ServerInternalError,
		ErrorMessage: xError.ErrMessage(fmt.Sprintf("SDK 组件缺失: 无法在上下文中找到 Key 为 [%v] 的组件", key)),
	}
}
