package xAsync

import (
	"context"

	xCtx "github.com/bamboo-services/bamboo-base-go/defined/context"
)

// detachContext 从父上下文中提取组件引用和请求级数据，注入到全新的独立上下文中。
//
// 新上下文基于 context.Background()，因此父上下文的取消不会影响异步任务。
// 复制 RegNodeKey（组件引用）和 RequestKey（请求链路追踪 ID），
// 不复制请求生命周期相关的临时数据（UserStartTimeKey、ErrorCodeKey 等）。
//
// 所有值的读取在同步阶段完成，确保 goroutine 启动时不依赖父上下文。
func detachContext(parentCtx context.Context) (context.Context, context.CancelFunc) {
	ctx := context.Background()

	if parentCtx == nil {
		return context.WithCancel(ctx)
	}

	// 复制组件容器（DB、Redis、Snowflake 等）
	if val := parentCtx.Value(xCtx.RegNodeKey); val != nil {
		if nodeList, ok := val.(xCtx.ContextNodeList); ok {
			ctx = context.WithValue(ctx, xCtx.RegNodeKey, nodeList)
		}
	}

	// 复制请求链路追踪 ID，确保异步任务的日志可追溯至原始请求
	if val := parentCtx.Value(xCtx.RequestKey); val != nil {
		if traceID, ok := val.(string); ok {
			ctx = context.WithValue(ctx, xCtx.RequestKey, traceID)
		}
	}

	return context.WithCancel(ctx)
}
