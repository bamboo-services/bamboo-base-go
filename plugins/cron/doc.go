// Package cron 提供基于 Cron 表达式与固定时间间隔的定时任务调度插件。
//
// 本包作为 Bamboo Base Go 框架的后台 Worker 插件（ModeWorker 模式），
// 专注于定时周期的计算与到点触发。通过集成 [application.Activator]，
// 本包将定时触发与具体业务任务解耦，到点时向应用拓扑中的目标受管组件发起激活请求。
//
// # 核心类型与调用路径
//
// 在应用装配阶段，通过 [New] 配置定时任务规则并注册到宿主 [application.App]：
//
//	info, factory := cron.New(
//		cron.WithActivator(activator),
//		cron.WithCronJob("0 2 * * *", "daily_cleanup_task"),
//		cron.WithIntervalJob(10*time.Minute, "metrics_aggregation_task"),
//	)
//	app.RegisterWorker(info, factory)
//
// 启动时，宿主调用 [Plugin.Start] 完成 cron 表达式解析、间隔校验与调度器初始化；
// 随后由宿主在专用协程中调用 [Plugin.Run] 启动调度事件循环。
//
// # 典型用法
//
// 以下示例展示如何配置包含每日定时与周期循环的 Cron 插件：
//
//	info, factory := cron.New(
//		cron.WithActivator(activator),
//		cron.WithLocation(time.Local),
//		cron.WithCronJob("0 0 * * *", ctxkey.Key("midnight_job")),
//		cron.WithIntervalJob(30*time.Second, ctxkey.Key("heartbeat_job")),
//	)
//	if err := app.RegisterWorker(info, factory); err != nil {
//		return err
//	}
//
// # 任务配置与时区
//
// 插件支持两种作业调度规则：
//
//   - 标准 Cron 表达式：通过 [WithCronJob] 注册标准 5 段式 cron 字符串（分、时、日、月、周）；
//   - 固定时长间隔：通过 [WithIntervalJob] 注册以 [time.Duration] 为周期的定时触发；
//   - 时区控制：默认采用 UTC 时区，可通过 [WithLocation] 指定本地时区或特定时区。
//
// 注意：严禁在 [WithRequires] 中声明作业目标组件的 Key。
// 目标组件属于动态按需激活对象，由 [application.Activator] 负责生命周期调度与并发隔离。
//
// # 调度执行与 Activator 解耦
//
// 调度器触发到点事件时，仅负责调用 [application.Activator.Activate]：
//
//   - 携带触发时间戳：将当前触发时间作为 [application.WithPayload] 注入激活上下文；
//   - 非阻塞解耦：调度器不直接运行具体的业务函数，亦不等待长时间任务执行完毕；
//   - 错误告警：若激活调用返回失败，插件在注入的日志器中记录警告日志，不中断调度循环。
//
// # 停机等待与并发安全
//
// 当宿主触发停机流程时：
//
//   - [Plugin.Quiesce] 率先原子关闭 tick 准入，不再接收新的定时触发；
//   - [Plugin.Stop] 停止底层调度引擎，并阻塞等待当前所有正在飞行中的激活派发调用安全返回；
//   - 若等待过程中传入的上下文超时，[Plugin.Stop] 仍严格保证内部等待结束，并返回包含超时的复合错误；
//   - 多次调用 [Plugin.Stop] 具备幂等性，共享相同退出结果。
//
// # 错误处理
//
// 针对初始化与运行阶段的错误识别：
//
//   - 启动前校验：若未显式注入 [application.Activator]，[Plugin.Start] 立即返回 [ErrNoActivator]；
//   - 规则校验：若 Cron 表达式语法非法或间隔小于等于 0，返回 [ErrInvalidJobSchedule]；
//   - 目标校验：若目标组件 Key 非法，返回 [ErrInvalidJobTarget]。
package cron
