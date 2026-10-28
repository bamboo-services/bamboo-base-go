// Package application 提供 Bamboo Base Go v2 的应用运行时内核与生命周期编排。
//
// 本包作为 Bamboo 服务的基础运行时基座，提供组件注册、Kahn 拓扑排序、
// 启动依赖校验、Worker 看门狗监督、上下文组件注入与逆拓扑平滑关机能力。
// 下游业务应用与各类插件通过本包实现统一的应用装配与生命周期管控。
//
// # 职责与适用场景
//
// 本包解决多组件应用在初始化、运行与退出时的强时序、高并发与依赖解耦问题。
// 调用方通过 [New] 构造 [App] 实例，向其注册工具态插件（[ModeTool]）、
// 常驻型后台服务（[ModeWorker]）与激发态任务组件（[ModeComponent]）。
// 内核在启动阶段计算依赖图拓扑并执行分层初始化，在运行阶段提供类型安全的
// 泛型组件提取 [Get]，并在收到系统信号或主动调用 [App.Shutdown] 时，
// 按照依赖拓扑的逆序优雅平滑停机。
//
// # 核心类型与调用路径
//
// 核心运行时类型及其协作关系如下：
//
//   - [App]：应用级生命周期管理器与状态机核心，负责编排全部静态与动态组件；
//   - [PluginInfo]：组件注册元信息，声明组件主键、依赖项、暴露资源与重入策略；
//   - [Plugin]、[RunnablePlugin]、[Quiescer]：插件生命周期钩子接口；
//   - [Activator]、[RunHandle]：动态组件激活引擎与监督句柄；
//   - [Config]、[Option]：全局关机与超时预算配置；
//   - [AppState]：应用生命周期状态枚举。
//
// 标准调用路径包含以下阶段：
//
//  1. 调用 [New] 传入 [Option] 选项构造 [App]；
//  2. 调用 [App.RegisterTool]、[App.RegisterWorker] 或 [App.RegisterComponent]
//     注册各类插件元信息与工厂函数；
//  3. 调用 [App.Run] 启动应用并阻塞等待系统信号或运行终止；
//  4. 进程退出前若需主动终止，可由独立协程调用 [App.Shutdown]。
//
// # 典型用法
//
// 启动一个包含数据库工具插件与 Web 服务的典型应用：
//
//	app := application.New(
//		application.WithShutdownTimeout(30 * time.Second),
//	)
//
//	// 注册只读共享工具插件
//	_ = app.RegisterTool(application.PluginInfo{
//		Key: ctxkey.DatabaseKey,
//	}, func() application.Plugin {
//		return dbPlugin
//	})
//
//	// 注册依赖数据库的 Web Worker 插件
//	_ = app.RegisterWorker(application.PluginInfo{
//		Key:      ctxkey.WebGinServerKey,
//		Requires: []ctxkey.Key{ctxkey.DatabaseKey},
//	}, func() application.RunnablePlugin {
//		return webPlugin
//	})
//
//	// 启动应用拓扑并阻塞等待信号或停机
//	if err := app.Run(); err != nil {
//		log.Fatalf("app run error: %v", err)
//	}
//
// # 组件形态与注册
//
// 内核通过 [Mode] 区分三种组件形态：
//
//   - [ModeTool]：工具态插件，仅实现 [Plugin] 接口（[Plugin.Start] 与 [Plugin.Stop]），
//     在应用启动时按拓扑顺序初始化并单例常驻，为下游提供共享资源；
//   - [ModeWorker]：常驻型运行时插件，实现 [RunnablePlugin] 接口（含 [RunnablePlugin.Run]），
//     具有常驻事件循环或网络监听（如 Web、gRPC、定时调度），受看门狗监督；
//   - [ModeComponent]：激发态插件，实现 [RunnablePlugin] 接口，不在应用启动时自动实例化，
//     而是由业务或 Worker 通过 [Activator] 在运行期按需动态激活，每次激发分配独立实例。
//
// 所有注册方法（[App.RegisterTool]、[App.RegisterWorker]、[App.RegisterComponent]）
// 必须在 [App.Run] 调用之前完成。在应用已进入运行或停机状态后调用注册方法，
// 均返回 [ErrRegistrationClosed]。工厂函数若为 nil 则返回 [ErrNilFactory]。
//
// # 依赖拓扑与白名单隔离
//
// 应用启动时，内核通过 [PluginInfo.Requires] 与 [PluginInfo.Provides]
// 构建单向无环图，并利用 Kahn 算法计算稳定分层拓扑：
//
//   - 循环依赖检测：若插件间存在依赖环，[App.Run] 立即中止并返回 [ErrDependencyCycle]；
//   - 启动期受限视图：在插件的 [Plugin.Start] 钩子中，上下文挂载严格受限的视图。
//     插件只能通过 [Get] 访问其在 [PluginInfo.Requires] 中显式声明的直接依赖键，
//     越权读取未声明组件将返回携带 [foundation/v2/apperr.Forbidden] 的错误；
//   - 资源原子发布：若插件声明了额外的 [PluginInfo.Provides] 资源键，
//     必须同时实现 [foundation/v2/ctxkey.Exposer] 接口并在 Expose 方法中交付。
//     内核严格校验交付集合，并在启动成功后原子发布至共享容器。
//
// # 组件提取与上下文流转
//
// 在应用运行期间，组件实例通过上下文安全流转与读取：
//
//   - 泛型类型安全提取：使用 [Get] 或 [MustGet] 从 context.Context 中按 [ctxkey.Key]
//     提取组件实例，提取失败时返回结构化业务错误 [*foundation/v2/apperr.Error]。
//     禁止使用裸 ctx.Value 进行不安全的类型断言；
//   - 请求上下文桥接：Web 或 gRPC 接收到外部请求时，使用 [BridgeRequest]
//     将原生请求上下文与应用运行时生命周期上下文融合，注入共享容器、父级可信身份与追踪 ID；
//   - 异步后台任务剥离：在协程池或异步任务中执行后台计算时，使用 [Detach]
//     剥离请求上下文，阻断父请求超时与取消影响，同时完整保留共享容器只读访问与链路追踪 ID；
//   - 应用状态只读探测：插件内部可通过 [StateFrom] 查询宿主当前所处的 [AppState]，
//     从而在启动或停机阶段拒绝接纳新任务，且无法篡改应用状态；
//   - 强制取消根查询：通过 [ForceContext] 获取应用 Grace 到期后的强制取消根通知。
//
// # 运行状态机与生命周期
//
// 应用生命周期由 [AppState] 状态机严格驱动：
//
//   - [StateNew]：初始创建态，接受插件注册；
//   - [StateStarting]：启动初始化态，执行拓扑排序并分层调用各插件的 Start；
//   - [StateRunning]：运行就绪态，冻结共享容器，启动 Worker 并提供服务；
//   - [StateStopping]：平滑停机态，停止接纳新任务，执行排空并逆序调用 Stop；
//   - [StateStopped]：完全停机态，全部静态与动态实例已确认安全停止退出；
//   - [StateIncomplete]：不完全停机态，存在超时未退或 Stop 发生 panic 的残留实例。
//
// [App.Run] 具备单次消耗性语义，多次调用同一实例的 Run 将返回 [ErrAlreadyRun]。
//
// # 看门狗与错误隔离
//
// 内核看门狗协程在应用运行期间持续监督常驻 Worker：
//
//   - 早退故障捕获：在 [StateRunning] 阶段，若某个 Worker 的 Run 异常返回
//     （即使返回 nil），看门狗均将其判定为非预期崩溃，触发应用级自动平滑关机，
//     并记录 [ErrWorkerUnexpectedExit]；
//   - Panic 隔离与诊断：任何插件工厂函数、Start、Expose、Run、Quiesce、Stop 发生 panic，
//     内核均会安全捕获并包装为结构化错误 [HookPanicError]，保留调用阶段、插件主键、
//     实例标识与完整堆栈，防止工作进程裸崩溃。
//
// # 关机协调与超时预算
//
// 应用关机由内建的关机协调器调度，具备幂等性与预算控制：
//
//   - 超时预算体系：通过 [Option] 可配置 [Config.StartTimeout]（单插件启动超时，默认 15s）、
//     [Config.GraceTimeout]（自然排空窗口，默认 20s）、
//     [Config.CleanupTimeout]（纯资源清理上限，默认 10s）与
//     [Config.ShutdownTimeout]（整体关机上限，默认 60s）；
//   - 逆序拓扑分层停机：按照依赖拓扑的严格逆序（顶层使用者先于底层提供者）停机。
//     若底层提供者被活跃的动态任务占用，停机流程将等待占用释放，避免产生悬空调用；
//   - 静默排空：若插件实现了 [Quiescer] 接口，停机时先调用其 Quiesce 拒绝新流量并排空，
//     之后再调用 Stop 进行底层网络与连接池资源关闭；
//   - 残留实例隔离保护：若某个组件在总关机截止前仍未退出，或在 Stop 阶段发生 panic，
//     内核将其记录为 [StuckInstance]，拒绝后续对其调用 Stop，同时保护性保留其依赖的所有
//     底层提供者组件继续存活，避免内存或连接野指针崩溃。最终将应用状态置为 [StateIncomplete]，
//     并返回包含全部残留快照的结构化错误 [StopIncompleteError]。
//
// # 动态组件激发与并发策略
//
// 通过 [App.Activator] 获取的 [Activator] 允许在运行时激发 [ModeComponent] 任务：
//
//   - 动态并发控制：通过 [OverlapPolicy] 配置 [OverlapSkip]（跳过重入）、
//     [OverlapAllow]（有界并发）与 [OverlapQueue]（有界排队等待）；
//   - 树状生命周期级联：动态组件在关闭时先等待其所激发的全部子任务退出，再收尾自身；
//   - 依赖环路防御：当激活链路中检测到递归祖先定义环时，立即拒绝并返回 [ErrActivationCycle]。
//
// # 并发安全与使用边界
//
//   - [App] 实例的 [App.State] 与 [App.Shutdown] 均为线程安全，支持多 goroutine 并发调用；
//   - 插件注册方法（[App.RegisterTool] 等）由互斥锁保护，但必须在单线程装配阶段或 [App.Run] 前完成；
//   - 共享容器在进入 [StateRunning] 阶段后立即冻结，禁止在运行期修改或追加静态组件键；
//   - 传递给 [WithPayload] 的动态负载不执行深拷贝，若涉及可变对象需由调用方自行保证并发安全。
package application
