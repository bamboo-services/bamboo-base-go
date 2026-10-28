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

// Registry 是每实例/独立的中间件注册表。
type Registry struct {
	mu          sync.RWMutex
	unaryStore  map[string][]UnaryMiddlewareFunc
	streamStore map[string][]StreamMiddlewareFunc
}

// NewRegistry 创建一个新的中间件注册表实例。
func NewRegistry() *Registry {
	return &Registry{
		unaryStore:  make(map[string][]UnaryMiddlewareFunc),
		streamStore: make(map[string][]StreamMiddlewareFunc),
	}
}

// Clone 创建当前注册表的一份深拷贝快照。
func (r *Registry) Clone() *Registry {
	if r == nil {
		return NewRegistry()
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	cloned := NewRegistry()
	for k, v := range r.unaryStore {
		cloned.unaryStore[k] = append([]UnaryMiddlewareFunc(nil), v...)
	}
	for k, v := range r.streamStore {
		cloned.streamStore[k] = append([]StreamMiddlewareFunc(nil), v...)
	}
	return cloned
}

// UseUnary 向当前注册表注册服务级一元中间件。
func (r *Registry) UseUnary(desc grpc.ServiceDesc, middlewares ...UnaryMiddlewareFunc) {
	filtered := filterNilUnary(middlewares)
	if len(filtered) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.unaryStore[desc.ServiceName] = append(r.unaryStore[desc.ServiceName], filtered...)
}

// UseStream 向当前注册表注册服务级流式中间件。
func (r *Registry) UseStream(desc grpc.ServiceDesc, middlewares ...StreamMiddlewareFunc) {
	filtered := filterNilStream(middlewares)
	if len(filtered) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.streamStore[desc.ServiceName] = append(r.streamStore[desc.ServiceName], filtered...)
}

// LookupUnary 查找当前注册表中指定服务名绑定的一元中间件链。
func (r *Registry) LookupUnary(serviceName string) []UnaryMiddlewareFunc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.unaryStore[serviceName]
}

// LookupStream 查找当前注册表中指定服务名绑定的流式中间件链。
func (r *Registry) LookupStream(serviceName string) []StreamMiddlewareFunc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.streamStore[serviceName]
}

// ExtractServiceName 从 gRPC FullMethod 中提取服务名。
//
// gRPC FullMethod 格式为 "/package.ServiceName/MethodName"，
// 本函数返回 "package.ServiceName" 部分。
func ExtractServiceName(fullMethod string) string {
	if len(fullMethod) > 0 && fullMethod[0] == '/' {
		fullMethod = fullMethod[1:]
	}
	if idx := strings.LastIndex(fullMethod, "/"); idx >= 0 {
		return fullMethod[:idx]
	}
	return fullMethod
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
