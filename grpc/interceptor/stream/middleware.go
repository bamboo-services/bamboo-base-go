package xGrpcIStream

import (
	xGrpcMiddle "github.com/bamboo-services/bamboo-base-go/grpc/middleware"
	"google.golang.org/grpc"
)

// Middleware 返回一个服务级中间件分发拦截器。
//
// 该拦截器根据请求的 FullMethod 解析出服务名，在全局注册表中查找对应的中间件链并依次执行。
// 若服务未注册中间件，直接透传到下一个 handler。
//
// 中间件通过 xGrpcMiddle.UseStream() 注册，在业务端初始化阶段完成绑定。
//
// 中间件链执行遵循洋葱模型，注册顺序 [A, B, C] 的执行顺序为：
//
//	A-enter → B-enter → C-enter → handler → C-exit → B-exit → A-exit
//
// 使用示例：
//
//	xGrpcRunner.New(
//	    xGrpcRunner.WithStreamInterceptors(
//	        xGrpcIStream.Middleware(),
//	    ),
//	)
func Middleware() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		serviceName := xGrpcMiddle.ExtractServiceName(info.FullMethod)
		middlewares := xGrpcMiddle.LookupStream(serviceName)
		if len(middlewares) == 0 {
			return handler(srv, ss)
		}

		return chainStreamMiddlewares(middlewares, info, handler)(srv, ss)
	}
}

// chainStreamMiddlewares 将中间件链包装为单个 StreamHandler。
//
// 通过反向遍历构建嵌套调用链，保证洋葱模型的执行顺序。
func chainStreamMiddlewares(middlewares []xGrpcMiddle.StreamMiddlewareFunc, info *grpc.StreamServerInfo, handler grpc.StreamHandler) grpc.StreamHandler {
	current := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		mw := middlewares[i]
		next := current
		current = func(srv interface{}, ss grpc.ServerStream) error {
			return mw(srv, ss, info, next)
		}
	}
	return current
}
