// Package xGrpcMiddle 提供面向 gRPC 服务的服务级中间件注册表与方法路由解析。
//
// 本包允许按具体 gRPC Service 维度配置一元（Unary）与流式（Stream）中间件链，
// 改变了原生 gRPC 拦截器只能全局生效的局限。在 RPC 入口拦截器执行时，
// 插件会提取当前调用的服务名并动态调度执行目标服务专属的中间件链。
//
// # 服务名提取与匹配
//
// gRPC 的 FullMethod 遵循 "/package.ServiceName/MethodName" 规范。
// [ExtractServiceName] 函数负责解析 FullMethod 并剥离具体方法名，
// 提取权威的服务限定名（如 "package.ServiceName"），作为中间件查找的键。
//
// # 典型用法
//
// 创建注册表并为指定服务绑定一元与流式中间件：
//
//	reg := xGrpcMiddle.NewRegistry()
//
//	// 为 UserService 服务挂载专属鉴权中间件
//	reg.UseUnary(pb.UserService_ServiceDesc, authUnaryMiddleware)
//	reg.UseStream(pb.UserService_ServiceDesc, authStreamMiddleware)
//
//	// 将注册表配置到 gRPC 插件中
//	info, factory := xGrpc.New(
//		xGrpc.WithMiddlewareRegistry(reg),
//	)
//
// # 并发安全与快照隔离
//
// [Registry] 内部基于 [sync.RWMutex] 保护，支持注册与查询的并发安全执行。
// 通过 [Registry.Clone] 可生成一份独立的深拷贝快照，gRPC 插件在构造时会自动克隆快照，
// 确保每个插件实例拥有独立的中间件存储，避免多实例运行期互相干扰。
package xGrpcMiddle
