package xGrpcIStream

import (
	"context"

	xCtx "github.com/bamboo-services/bamboo-base-go/context"
	"google.golang.org/grpc"
)

// wrappedServerStream 包装 grpc.ServerStream 以支持自定义上下文。
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context 返回包装后的上下文。
func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// InitContext 从主上下文中提取并传播注册节点列表，创建一个 gRPC 流式拦截器。
//
// 该函数从 mainCtx 中检索 RegNodeKey 对应的 ContextNodeList，
// 并将其注入到后续 RPC 请求的上下文中。
//
// 参数 mainCtx 用于提取节点列表的根上下文。
//
// 返回的拦截器确保所有 RPC 调用共享相同的上下文节点链路。
func InitContext(mainCtx context.Context) grpc.StreamServerInterceptor {
	var ctxNodeList xCtx.ContextNodeList
	if val := mainCtx.Value(xCtx.RegNodeKey); val != nil {
		if nodeList, ok := val.(xCtx.ContextNodeList); ok {
			ctxNodeList = nodeList
		}
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		newCtx := context.WithValue(ss.Context(), xCtx.RegNodeKey, ctxNodeList)
		wrapped := &wrappedServerStream{ServerStream: ss, ctx: newCtx}
		return handler(srv, wrapped)
	}
}
