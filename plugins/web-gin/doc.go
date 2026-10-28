// Package webgin 提供基于 Gin 框架构建的 Web HTTP 服务插件与运行时契约。
//
// 本包为应用框架提供标准的 HTTP 服务端 Worker 实现，支持路由挂载、
// 独立验证器扩展、请求级上下文桥接、双向追踪标识注入与优雅关机协调。
// 每个插件实例持有独立的 Gin 引擎与绑定器，不修改全局 Gin 运行模式，
// 也不依赖任何全局单例对象。
//
// # 核心概念与调用路径
//
// 本插件以 [application.ModeWorker] 模式接入应用框架。
// 核心装配与启动路径如下：
//
//   - 插件构造：调用 [New] 并传入可选配置 [Option]（例如 [WithAddr]、
//     [WithRoute]、[WithRequires]），获得插件注册描述 [application.PluginInfo]
//     以及用于延迟实例化的工厂函数；
//   - 实例准备：框架拓扑就绪后调用 [Plugin.Start]，完成私有 Gin 引擎创建、
//     独立验证器与 [bind.Binder] 实例化、内置中间件装配及 TCP 端口监听；
//   - 运行服务：框架调度协程进入 [Plugin.Run]，封存应用生命周期上下文并阻塞服务；
//   - 请求桥接：请求到达时通过内部中间件执行准入检查、生成服务端追踪 UUID、
//     注入 [bind.Binder] 并通过 [BridgeRequest] 挂载应用上下文与耗时起始标记；
//   - 平滑退出：框架协调停机时通过 [Plugin.Quiesce] 关闭准入门禁，
//     随后通过 [Plugin.Stop] 协调 HTTP 服务关机与未决请求排空。
//
// # 典型用法
//
// 下面的示例展示在业务服务装配阶段创建 Gin Worker 并挂载业务路由：
//
//	info, factory := webgin.New(
//		webgin.WithAddr("127.0.0.1:8080"),
//		webgin.WithName("UserWebService"),
//		webgin.WithRoute(func(ctx context.Context, engine *gin.Engine) {
//			engine.GET("/ping", func(c *gin.Context) {
//				httpresult.Success(c, "pong")
//			})
//		}),
//	)
//	app.RegisterWorker(info, factory)
//
// # 生命周期与优雅关机
//
// [Plugin] 完整实现了 [application.RunnablePlugin] 接口定义的运行态契约：
//
//   - 启动保障：[Plugin.Start] 初始化失败时会自动撤销已创建的监听与资源，
//     若服务已处于启动中或已启动状态，再次调用将返回 [ErrServerAlreadyStarted]；
//   - 运行隔离：[Plugin.Run] 要求传入非 nil 的可信生命周期 Context，
//     若未完成 [Plugin.Start] 即调用 [Plugin.Run]，将直接返回错误；
//   - 关闸停机：[Plugin.Quiesce] 仅关闭请求准入门禁，后续新请求将收到 503 响应，
//     正在处理中的请求不受影响；
//   - 优雅终止：[Plugin.Stop] 先确保入口关闸，再触发底层 HTTP 服务的平滑停机。
//     若停机等待上下文超时或收到强制退出信号，将强制切断传输层连接；
//     同时始终等待当前已接纳的飞行中请求完全退出后才返回，绝不提前虚假返回。
//
// # 请求追踪与上下文桥接
//
// 本插件遵循严格的服务端追踪标识生成规则（保留 v1 行为）：
//
//   - 标识生成：每次 HTTP 请求均由服务端独立生成全新的 UUID，
//     不采纳客户端通过请求头透传的伪造标识；
//   - 标头回写：生成的追踪标识会立即回写到响应标头 [header.HeaderRequestUUID]，
//     确保客户端收到的响应头与响应 JSON 信封中的 context 字段严格一致；
//   - 上下文派生：通过 [BridgeRequest] 将请求上下文与生命周期上下文关联，
//     挂载 [ctxkey.UserStartTimeKey] 作为耗时统计起点，
//     并挂载应用容器视图与实例凭据。
//
// # 错误处理与超时设置
//
// 插件提供了可预期的标准错误值：
//
//   - [ErrServerAlreadyStarted]：服务已启动或处于启动中；
//   - [ErrServerClosed]：服务处于关机或已停止状态，拒绝接纳新请求；
//   - [ErrNoLifetimeContext]：缺少有效的应用生命周期上下文；
//   - [ErrStopTimeout]：服务停机等待超时。
//
// 可通过 [WithTimeouts] 配置 HTTP 读、写、空闲及停机等待超时时间。
//
// # 并发安全
//
// [Plugin] 内部使用读写锁与原子协调机制保证状态迁移与请求准入的并发安全。
// [Plugin.Addr]、[Plugin.Quiesce]、[Plugin.Stop] 支持被多个 goroutine
// 安全并发调用；多次调用 [Plugin.Stop] 将共享同一次停机执行结果。
package webgin
