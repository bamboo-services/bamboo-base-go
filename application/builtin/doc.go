// Package builtin 提供与 [application.App] 紧密绑定的内置受管基础组件。
//
// 本包为应用运行时提供日志与雪花发号器两项核心受管能力。所有组件均以
// [application.ModeTool] 工具模式运行，由应用生命周期协调器统一管理启动、
// 依赖拓扑注入与优雅关闭。
//
// # 核心组件与导出键
//
// 本包提供两套独立的受管组件构造器：
//
//   - 日志组件：通过 [NewLogger] 创建，插件标识为 [LoggerPluginKey]，
//     向容器导出 [*logging.Logger] 实例，上下文键为 [LoggerKey]。
//   - 雪花发号器组件：通过 [NewSnowflake] 创建，插件标识为 [SnowflakePluginKey]，
//     向容器导出 [*snowflake.Node] 实例，上下文键为 [ctxkey.SnowflakeNodeKey]。
//
// # 隔离性与设计原则
//
// 本包内的所有组件严格遵循以下运行时设计原则：
//
//   - 实例级隔离：各 [application.App] 拥有完全独立的组件实例，
//     日志组件绝不修改全局 [slog.Default]，雪花发号器不注册任何全局节点。
//     多个 App 实例在同进程内可并行启动与独立关闭，互不干扰。
//   - 延迟生效与纯构造：构造函数 [NewLogger] 与 [NewSnowflake] 及其返回的
//     工厂闭包均为纯构造函数，不执行任何磁盘 I/O、目录创建、协程启动或参数校验。
//     所有耗时初始化、文件句柄打开与参数边界校验均推迟至受管生命周期的
//     Start 阶段执行。
//   - 类型安全提取：服务内部禁止使用裸类型断言。下游 Worker 或 Tool
//     必须通过 [application.Get] 泛型方法从上下文安全提取实例。
//
// # 典型用法
//
// 下面的示例展示如何在装配应用时注册内置日志与雪花发号器组件，并在下游
// Worker 内部提取使用：
//
//	func setupApp() (*application.App, error) {
//		app := application.New()
//
//		// 1. 注册内置受管日志组件
//		logInfo, logFac := builtin.NewLogger(builtin.LoggerConfig{
//			Rotator: logging.RotatorConfig{
//				Dir:      "./logs",
//				BaseName: "app",
//				Ext:      ".log",
//			},
//			Console: os.Stdout,
//			Level:   slog.LevelInfo,
//		})
//		if err := app.RegisterTool(logInfo, logFac); err != nil {
//			return nil, err
//		}
//
//		// 2. 注册内置受管雪花发号器组件（数据中心 1，节点 1）
//		sfInfo, sfFac := builtin.NewSnowflake(1, 1)
//		if err := app.RegisterTool(sfInfo, sfFac); err != nil {
//			return nil, err
//		}
//
//		return app, nil
//	}
//
// 下游组件在自身 Start 或 Run 阶段通过 Context 提取组件：
//
//	func (w *myWorker) Start(ctx context.Context) error {
//		logger, err := application.Get[*logging.Logger](ctx, builtin.LoggerKey)
//		if err != nil {
//			return err
//		}
//		node, err := application.Get[*snowflake.Node](ctx, ctxkey.SnowflakeNodeKey)
//		if err != nil {
//			return err
//		}
//		w.logger = logger
//		w.node = node
//		return nil
//	}
//
// # 生命周期与资源归属
//
// 内置组件的完整生命周期受应用容器托管：
//
//   - 启动阶段（Start）：按照依赖拓扑顺序调用。日志组件在此阶段真实创建目录并
//     打开 [logging.RotatingWriter]；雪花发号器校验节点 ID（数据中心与节点 ID
//     取值必须在 0 到 7 之间），若校验失败则立即中断启动流程并向应用返回错误。
//   - 停止阶段（Stop）：按照逆拓扑顺序调用。日志组件在此阶段真实调用底层
//     写入器的 Close 方法，刷新缓冲区并安全关闭日志归档文件；雪花发号器清理
//     内部节点引用。日志组件的 Stop 方法具备幂等性（内部通过 [sync.Once] 保护），
//     且无论传入的 context 是否已被取消或超时，均会真实等待底层 I/O 资源释放完毕，
//     确保在应用完全终止前所有在途日志均已落盘。
//
// # 错误处理与使用边界
//
// 调用方使用本包组件时须注意以下约束：
//
//   - 键命名空间：日志上下文键 [LoggerKey] 由本包独立定义，
//     [foundation/ctxkey] 基础包中不包含 LoggerKey，调用方请勿混淆；
//     雪花算法发号器键使用基础原语定义的 [ctxkey.SnowflakeNodeKey]。
//   - 节点 ID 约束：雪花算法的数据中心 ID 与节点 ID 各占 3 位，
//     合法取值范围均为 [0, 7]。超出范围不会在 [NewSnowflake] 构造时报错，
//     而会在组件生命周期的 Start 阶段返回错误并导致应用启动失败。
package builtin
