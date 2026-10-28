package xGrpcMiddle

import (
	"strings"
	"sync"

	"google.golang.org/grpc"
)

// UnaryMiddlewareFunc 是服务级一元中间件函数类型，签名与原生 [grpc.UnaryServerInterceptor] 完全等价。
//
// 用于在特定 gRPC 服务的作用域内拦截一元调用的请求上下文、入参、响应及错误。
type UnaryMiddlewareFunc = grpc.UnaryServerInterceptor

// StreamMiddlewareFunc 是服务级流式中间件函数类型，签名与原生 [grpc.StreamServerInterceptor] 完全等价。
//
// 用于在特定 gRPC 服务的作用域内拦截流式调用的服务端流实例及全流程交互。
type StreamMiddlewareFunc = grpc.StreamServerInterceptor

// Registry 是服务级中间件的存储与分发注册表。
//
// 该注册表以服务限定名（例如 "package.ServiceName"）为键，分别存储对应服务的一元与流式中间件链。
// 内部状态受 [sync.RWMutex] 读写锁保护，具备并发安全性。
// 支持通过 [Registry.Clone] 生成独立的深拷贝快照，供不同插件实例独享，防止并发污染。
type Registry struct {
	mu          sync.RWMutex
	unaryStore  map[string][]UnaryMiddlewareFunc
	streamStore map[string][]StreamMiddlewareFunc
}

// NewRegistry 创建并返回一个初始化为空的服务级中间件注册表实例。
func NewRegistry() *Registry {
	return &Registry{
		unaryStore:  make(map[string][]UnaryMiddlewareFunc),
		streamStore: make(map[string][]StreamMiddlewareFunc),
	}
}

// Clone 创建并返回当前注册表的一份深拷贝快照。
//
// 若接收器为 nil，安全返回新创建的空注册表实例。
// 内部在读锁保护下将所有服务的中间件切片逐一深拷贝至新实例中，
// 确保后续对原注册表或克隆副本的注册修改互不干扰。该方法可并发安全调用。
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

// UseUnary 向注册表中的指定 gRPC 服务追加一个或多个服务级一元中间件。
//
// 注册依据传入的 [grpc.ServiceDesc.ServiceName] 进行关联。
// 传入的 middlewares 中若包含 nil 元素将被自动过滤；若过滤后列表为空则直接返回。
// 该方法可并发安全调用。
func (r *Registry) UseUnary(desc grpc.ServiceDesc, middlewares ...UnaryMiddlewareFunc) {
	filtered := filterNilUnary(middlewares)
	if len(filtered) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.unaryStore[desc.ServiceName] = append(r.unaryStore[desc.ServiceName], filtered...)
}

// UseStream 向注册表中的指定 gRPC 服务追加一个或多个服务级流式中间件。
//
// 注册依据传入的 [grpc.ServiceDesc.ServiceName] 进行关联。
// 传入的 middlewares 中若包含 nil 元素将被自动过滤；若过滤后列表为空则直接返回。
// 该方法可并发安全调用。
func (r *Registry) UseStream(desc grpc.ServiceDesc, middlewares ...StreamMiddlewareFunc) {
	filtered := filterNilStream(middlewares)
	if len(filtered) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.streamStore[desc.ServiceName] = append(r.streamStore[desc.ServiceName], filtered...)
}

// LookupUnary 查询指定服务名所绑定的一元中间件链切片。
//
// 参数 serviceName 为通过 [ExtractServiceName] 提取的完整服务限定名（如 "package.ServiceName"）。
// 若指定服务未注册任何一元中间件，返回 nil。该方法受读锁保护，并发安全。
func (r *Registry) LookupUnary(serviceName string) []UnaryMiddlewareFunc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.unaryStore[serviceName]
}

// LookupStream 查询指定服务名所绑定的流式中间件链切片。
//
// 参数 serviceName 为通过 [ExtractServiceName] 提取的完整服务限定名（如 "package.ServiceName"）。
// 若指定服务未注册任何流式中间件，返回 nil。该方法受读锁保护，并发安全。
func (r *Registry) LookupStream(serviceName string) []StreamMiddlewareFunc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.streamStore[serviceName]
}

// ExtractServiceName 从 gRPC 调用的 FullMethod 标识符中提取服务限定名。
//
// 标准 gRPC FullMethod 格式为 "/package.ServiceName/MethodName"。
// 本函数会自动剥除开头的斜杠字符，并截取最后一个斜杠前的部分，返回 "package.ServiceName"。
// 若输入不包含斜杠分隔符，则返回去除前导斜杠后的原字符串。
//
// 该函数为纯字符串解析函数，不执行 I/O，并发安全。
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
