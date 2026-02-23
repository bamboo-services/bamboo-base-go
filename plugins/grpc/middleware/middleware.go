package xGrpcMiddle

import (
	"strings"
	"sync"

	"google.golang.org/grpc"
)

// UnaryMiddlewareFunc 是服务级一元中间件函数类型，与 gRPC 一元拦截器签名完全一致。
type UnaryMiddlewareFunc = grpc.UnaryServerInterceptor

// StreamMiddlewareFunc 是服务级流式中间件函数类型，与 gRPC 流式拦截器签名完全一致。
type StreamMiddlewareFunc = grpc.StreamServerInterceptor

// 全局注册表，所有服务级中间件通过 UseUnary() 或 UseStream() 注册到此处。
var (
	mu          sync.RWMutex
	unaryStore  = make(map[string][]UnaryMiddlewareFunc)
	streamStore = make(map[string][]StreamMiddlewareFunc)
)

// UseUnary 注册服务级一元中间件到全局注册表。
//
// 该函数采用 Hook 加载方案，在业务端初始化阶段调用，
// 将中间件与指定 gRPC 服务绑定。同一服务多次调用时中间件链按顺序追加。
//
// 参数:
//   - desc: gRPC 服务描述符（值类型），从 desc.ServiceName 提取服务名
//   - middlewares: 要绑定的中间件列表，nil 值会被自动过滤
//
// 使用示例:
//
//	xGrpcMiddle.UseUnary(pb.UserService_ServiceDesc, authMiddleware, logMiddleware)
//	xGrpcMiddle.UseUnary(pb.OrderService_ServiceDesc, rateLimitMiddleware)
func UseUnary(desc grpc.ServiceDesc, middlewares ...UnaryMiddlewareFunc) {
	filtered := filterNilUnary(middlewares)
	if len(filtered) == 0 {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	unaryStore[desc.ServiceName] = append(unaryStore[desc.ServiceName], filtered...)
}

// UseStream 注册服务级流式中间件到全局注册表。
//
// 该函数采用 Hook 加载方案，在业务端初始化阶段调用，
// 将中间件与指定 gRPC 服务绑定。同一服务多次调用时中间件链按顺序追加。
//
// 参数:
//   - desc: gRPC 服务描述符（值类型），从 desc.ServiceName 提取服务名
//   - middlewares: 要绑定的中间件列表，nil 值会被自动过滤
//
// 使用示例:
//
//	xGrpcMiddle.UseStream(pb.FileStream_ServiceDesc, authMiddleware, logMiddleware)
func UseStream(desc grpc.ServiceDesc, middlewares ...StreamMiddlewareFunc) {
	filtered := filterNilStream(middlewares)
	if len(filtered) == 0 {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	streamStore[desc.ServiceName] = append(streamStore[desc.ServiceName], filtered...)
}

// LookupUnary 查找指定服务名绑定的一元中间件链。
//
// 若服务未注册或无中间件，返回 nil。该函数使用 RLock，请求路径零写竞争。
func LookupUnary(serviceName string) []UnaryMiddlewareFunc {
	mu.RLock()
	defer mu.RUnlock()
	return unaryStore[serviceName]
}

// LookupStream 查找指定服务名绑定的流式中间件链。
//
// 若服务未注册或无中间件，返回 nil。该函数使用 RLock，请求路径零写竞争。
func LookupStream(serviceName string) []StreamMiddlewareFunc {
	mu.RLock()
	defer mu.RUnlock()
	return streamStore[serviceName]
}

// ExtractServiceName 从 gRPC FullMethod 中提取服务名。
//
// gRPC FullMethod 格式为 "/package.ServiceName/MethodName"，
// 本函数返回 "package.ServiceName" 部分。
func ExtractServiceName(fullMethod string) string {
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

// reset 重置全局注册表（仅用于测试）。
func reset() {
	mu.Lock()
	defer mu.Unlock()
	unaryStore = make(map[string][]UnaryMiddlewareFunc)
	streamStore = make(map[string][]StreamMiddlewareFunc)
}

// filterNilUnary 过滤切片中的 nil 一元中间件。
func filterNilUnary(middlewares []UnaryMiddlewareFunc) []UnaryMiddlewareFunc {
	filtered := make([]UnaryMiddlewareFunc, 0, len(middlewares))
	for _, mw := range middlewares {
		if mw != nil {
			filtered = append(filtered, mw)
		}
	}
	return filtered
}

// filterNilStream 过滤切片中的 nil 流式中间件。
func filterNilStream(middlewares []StreamMiddlewareFunc) []StreamMiddlewareFunc {
	filtered := make([]StreamMiddlewareFunc, 0, len(middlewares))
	for _, mw := range middlewares {
		if mw != nil {
			filtered = append(filtered, mw)
		}
	}
	return filtered
}
