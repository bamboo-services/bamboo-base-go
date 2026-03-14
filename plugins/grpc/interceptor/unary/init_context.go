package xGrpcIUnary

import (
	"context"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtx "github.com/bamboo-services/bamboo-base-go/defined/context"
	"google.golang.org/grpc"
)

// InitContext 从主上下文中提取并传播注册节点列表，创建一个 gRPC 一元拦截器。
//
// 该函数从 mainCtx 中检索 RegNodeKey 对应的 ContextNodeList，
// 并将其注入到后续 RPC 请求的上下文中。
//
// 参数 mainCtx 用于提取节点列表的根上下文。
//
// 返回的拦截器确保所有 RPC 调用共享相同的上下文节点链路。
func InitContext(mainCtx context.Context) grpc.UnaryServerInterceptor {
	log := xLog.WithName(xLog.NamedGRPC, "InitContext")

	var ctxNodeList xCtx.ContextNodeList
	if mainCtx != nil {
		if val := mainCtx.Value(xCtx.RegNodeKey); val != nil {
			if nodeList, ok := val.(xCtx.ContextNodeList); ok {
				ctxNodeList = nodeList
			}
		}
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if ctx == nil {
			log.Warn(ctx, "接收到 nil 上下文，使用 context.Background() 替代")
			ctx = context.Background()
		}
		if ctx.Value(xCtx.RegNodeKey) == nil && ctxNodeList != nil {
			ctx = context.WithValue(ctx, xCtx.RegNodeKey, ctxNodeList)
		}
		return handler(ctx, req)
	}
}
