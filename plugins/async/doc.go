// Package async 提供基于有界非阻塞协程池的安全异步任务调度能力。
//
// 本包作为 Bamboo Base Go 框架的共享组件插件（Tool 模式），
// 旨在剥离 HTTP/gRPC 请求的短暂生命周期，为后台批处理、日志归档、
// 异步通知等场景提供受控的并发执行环境。
//
// # 核心类型与调用路径
//
// 应用启动装配时，通过 [New] 构造异步插件定义并注册到 [application.App]：
//
//	info, factory := async.New(
//		async.WithCapacity(1000),
//		async.WithRequires(ctxkey.DatabaseKey),
//	)
//	app.RegisterTool(info, factory)
//
// 启动后，插件将 [Pool] 接口句柄注入应用共享容器，键名为 [PoolKey]。
// 业务 Worker 或请求处理流程中，可直接调用包级辅助函数 [Submit] 提交任务，
// 也可通过 [application.Get] 显式获取 [Pool] 实例调用 [Pool.Submit]。
//
// 任务提交成功后返回 [Task] 句柄，用于观测执行状态、监听完成信号或协同取消。
//
// # 典型用法
//
// 在持有有效应用上下文的业务逻辑中，提交脱钩任务并等待其就绪：
//
//	task, err := async.Submit(ctx, func(taskCtx context.Context) {
//		// taskCtx 继承原请求的追踪元数据与容器视图，但已脱离请求超时
//		doHeavyWork(taskCtx)
//	}, async.WithTaskName("report_generation"))
//	if err != nil {
//		// 处理池满或停机拒绝
//		return err
//	}
//
//	// 若需在当前流程中等待结果（如限时同步等待）：
//	if err := task.WaitFor(5 * time.Second); err != nil {
//		// 超时后任务仍在后台继续执行，不会被强行中止
//	}
//
// # 任务生命周期与上下文脱钩
//
// 提交给 [Pool] 的任务函数接收由 [application.Detach] 生成的独立上下文：
//
//   - 隔离超时与取消：调用方的 HTTP/gRPC 请求提前结束或超时取消，不会中断后台任务；
//   - 继承容器与追踪：保留原上下文中的 [ctxkey.RequestKey] 追踪标识与依赖组件白名单；
//   - 强制取消挂钩：若应用进入强制退出阶段（Force 预算耗尽），系统将自动调用 [Task.Cancel]；
//   - 信号顺序保证：任务结束或 panic 恢复后，[Task.Ctx] 的 Done 信号严格先于 [Task.Done] 关闭。
//
// # 准入控制与并发保证
//
// 插件通过多重防护确保运行安全：
//
//   - 运行态准入：仅在应用处于 [application.StateRunning] 阶段时接受任务。
//     在应用启动中（Starting）或关机中（Stopping/Stopped）均拒绝提交并返回 [ErrInvalidContext] 或 [ErrPoolClosed]；
//   - 容器一致性：校验提交上下文的组件视图必须与当前池绑定的容器一致，杜绝跨 App 误提交；
//   - 容量限制：底层采用非阻塞有界协程池，达到 [WithCapacity] 上限时立即返回 [ErrPoolFull]，不发生协程排队积压；
//   - 并发安全：[Pool] 与 [Task] 的所有公开方法均支持多 goroutine 并发调用。
//
// # 优雅停机与资源协调
//
// 当宿主调用 [Plugin.Stop] 时，插件严格遵循以下协同流程：
//
//   - 立即关闭准入门禁，后续所有提交均返回 [ErrPoolClosed]；
//   - 阻塞等待当前所有在途活跃任务真实执行完毕，不以传入上下文超时虚假退出；
//   - 活跃计数完全归零后，调用底层协程池的释放逻辑归还 worker 内存；
//   - 若任务超时未退出导致关机超时，App 关机协调器将报告未完成实例，保留底层依赖直到进程退出。
//
// # 错误处理
//
// 调用方需针对提交阶段与等待阶段的不同错误做出决策：
//
//   - 提交失败：使用 [errors.Is] 区分 [ErrPoolFull]（触发削峰或降级）、
//     [ErrPoolClosed]（应用正准备停机）与 [ErrNilTaskFunc]（编程错误）；
//   - 运行期 Panic：任务内的 panic 将被自动捕获隔离并记录错误日志，
//     可通过 [Task.Panicked]、[Task.PanicValue] 和 [Task.PanicStack] 进行事后审计；
//   - 等待超时：[Task.Wait] 或 [Task.WaitFor] 超时返回 [context.DeadlineExceeded]，
//     调用方应认识到该超时仅代表调用方放弃等待，任务本身并未被强杀。
package async
