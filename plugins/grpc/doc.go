// Package xGrpc 提供面向 Bamboo 体系的 gRPC Worker 插件与运行时生命周期编排。
//
// 本包基于 Google gRPC 实现，遵循应用框架插件契约，将 gRPC 服务接入
// [application.App] 的统一生命周期状态机。作为 Worker 类型的受管插件，
// 本包承担长驻网络监听、服务注册、请求准入控制、拦截器链编排、请求上下文桥接、
// 链路追踪透传以及优雅排空关机的完整职责。
//
// # 核心类型与生命周期
//
// [Plugin] 是核心插件结构体，实现了 [application.RunnablePlugin] 接口。
// 其生命周期由应用内核严格编排：
//
//   - 构造阶段：通过 [New] 函数配置并返回 [application.PluginInfo] 与构造工厂，
//     构造期不执行任何端口绑定、网络 I/O 或后台 goroutine 启动；
//   - 启动阶段（[Plugin.Start]）：分配 TCP 端口监听，创建原生 [grpc.Server]，
//     注册私有桥接拦截器链与业务 [ServiceRegistrar] 服务，并按配置启用反射；
//   - 运行阶段（[Plugin.Run]）：接收应用层注入的可信生命周期上下文，
//     启动 Server 监听并阻塞运行直至服务停止；
//   - 关闸阶段（[Plugin.Quiesce]）：关闭新请求入站准入闸门，立即拒绝后续新请求，
//     但不中断飞行中的请求处理；
//   - 关机阶段（[Plugin.Stop]）：触发原生优雅关机，协调超时控制与应用强制信号，
//     并严格等待所有处理中请求与流完全排空后释放网络资源。
//
// # 典型用法
//
// 以下代码演示构造 gRPC 插件并挂载至应用的典型流程：
//
//	info, factory := xGrpc.New(
//		xGrpc.WithAddr(":9090"),
//		xGrpc.WithService(func(ctx context.Context, s *grpc.Server) {
//			// 注册业务 Protobuf 服务定义
//			// pb.RegisterUserServiceServer(s, userService)
//		}),
//		xGrpc.WithReflection(true),
//	)
//	app.RegisterWorker(info, factory)
//
// # 拦截器链与请求上下文桥接
//
// 插件在底层拦截器链的最外层装配了一元桥接拦截器与流式桥接拦截器：
//
//   - 准入控制：每个 RPC 接入时检查内部准入闸门，未就绪或已关闸时立即返回
//     包含 [apperr.ServiceUnavailable] 的标准错误；
//   - 追踪透传：优先提取上游 Metadata 中的 "x-request-uuid" 标识，缺失时回退为
//     服务端生成的 UUID，并同时以 SetHeader 与 SetTrailer 写入响应（兼容 v1 规范）；
//   - 耗时统计：在上下文写入请求起始时间 [ctxkey.UserStartTimeKey]，供响应阶段计算耗时；
//   - 容器桥接：通过 [application.BridgeRequest] 将请求专属上下文与应用生命周期
//     及共享组件容器安全绑定；
//   - 级联取消：通过 [context.AfterFunc] 级联应用强制关机取消信号，避免资源悬挂；
//   - 中间件洋葱模型：自动从 RPC FullMethod 中提取服务名，调度通过
//     [xGrpcMiddle.Registry] 绑定的服务级专用中间件。
//
// # 错误安全归一与 Panic 隔离
//
// 拦截器内建顶层 defer-recover 保护机制：
//
//   - 业务 Panic 发生时，自动拦截堆栈并在配置了日志器时输出错误记录，
//     对外转换为包含 [apperr.ServerInternalError] 的安全 gRPC status 响应，
//     绝不泄露内部 SQL、敏感文件路径或 Panic 堆栈至外部调用方；
//   - 业务返回的 [*apperr.Error] 通过 [grpcresult.ToStatusError] 转换为
//     携带安全业务文案及 [generate.BaseResponse] 细节的 gRPC status；
//   - [context.Canceled] 与 [context.DeadlineExceeded] 归一化为标准状态码，
//     避免下游包装错误扩散。
//
// # 并发安全与资源边界
//
// [Plugin] 实例在配置完成并启动后，所有网络请求并发由 gRPC 原生协程池调度，
// 准入闸门与生命周期状态采用互斥锁与条件变量保护，具备并发安全性。
// 中间件注册表在构造时执行深拷贝快照，运行期禁止跨实例共享可变配置。
package xGrpc
