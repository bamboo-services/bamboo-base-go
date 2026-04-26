package xAsync

import (
	"context"

	xCtx "github.com/bamboo-services/bamboo-base-go/defined/context"
)

// detachContext 从父上下文中提取 RegNodeKey 组件容器，注入到全新的独立上下文中。
//
// 新上下文基于 context.Background()，因此父上下文的取消不会影响异步任务。
// 仅复制 RegNodeKey（组件引用），不复制请求级数据（RequestKey、UserStartTimeKey 等）。
func detachContext(parentCtx context.Context) (context.Context, context.CancelFunc) {
	ctx := context.Background()

	if parentCtx != nil {
		if val := parentCtx.Value(xCtx.RegNodeKey); val != nil {
			if nodeList, ok := val.(xCtx.ContextNodeList); ok {
				ctx = context.WithValue(ctx, xCtx.RegNodeKey, nodeList)
			}
		}
	}

	return context.WithCancel(ctx)
}
