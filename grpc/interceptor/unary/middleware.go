package xGrpcIUnary

import (
	"context"
	"strings"

	xGrpcMiddle "github.com/bamboo-services/bamboo-base-go/grpc/middleware"
	"google.golang.org/grpc"
)

// Middleware 返回一个服务级中间件分发拦截器。
//
// 该拦截器根据请求的 FullMethod 解析出服务名，在全局注册表中查找对应的中间件链并依次执行。
// 若服务未注册中间件，直接透传到下一个 handler。
//
// 中间件通过 xGrpcMiddle.UseUnary() 注册，在业务端初始化阶段完成绑定。
//
// 中间件链执行遵循洋葱模型，注册顺序 [A, B, C] 的执行顺序为：
//
//	A-enter → B-enter → C-enter → handler → C-exit → B-exit → A-exit
//
// 使用示例：
//
//	xGrpcRunner.New(
//	    xGrpcRunner.WithUnaryInterceptors(
//	        xGrpcIUnary.Middleware(),
//	        xGrpcIUnary.ResponseBuilder(),
//	    ),
//	)
func Middleware() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		serviceName := extractServiceName(info.FullMethod)
		middlewares := xGrpcMiddle.LookupUnary(serviceName)
		if len(middlewares) == 0 {
			return handler(ctx, req)
		}

		return chainMiddlewares(middlewares, info, handler)(ctx, req)
	}
}

// chainMiddlewares 将中间件链包装为单个 UnaryHandler。
//
// 通过反向遍历构建嵌套调用链，保证洋葱模型的执行顺序。
func chainMiddlewares(middlewares []xGrpcMiddle.UnaryMiddlewareFunc, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) grpc.UnaryHandler {
	current := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		mw := middlewares[i]
		next := current
		current = func(ctx context.Context, req interface{}) (interface{}, error) {
			return mw(ctx, req, info, next)
		}
	}
	return current
}

// extractServiceName 从 gRPC FullMethod 中提取服务名。
//
// gRPC FullMethod 格式为 "/package.ServiceName/MethodName"，
// 本函数返回 "package.ServiceName" 部分。
func extractServiceName(fullMethod string) string {
	// 移除前导 "/"
	if len(fullMethod) > 0 && fullMethod[0] == '/' {
		fullMethod = fullMethod[1:]
	}
	// 截取最后一个 "/" 之前的部分
	if idx := strings.LastIndex(fullMethod, "/"); idx >= 0 {
		return fullMethod[:idx]
	}
	return fullMethod
}
