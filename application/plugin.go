package application

import (
	"context"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

// AppState 描述应用内核全局生命周期状态。
//
// 状态机单向流转，描述的是宿主 [App] 实例的整体状态，
// 不代表某个动态任务组件或业务请求的执行结果。
type AppState uint8

// 应用生命周期状态枚举。
const (
	// StateNew 是初始创建状态，此时允许调用各类注册方法注册插件。
	StateNew AppState = iota + 1

	// StateStarting 是启动初始化状态，内核正在计算拓扑并分层调用各插件的 Start 钩子。
	StateStarting

	// StateRunning 是运行就绪状态，共享容器已冻结，常驻 Worker 已启动运行并受看门狗监督。
	StateRunning

	// StateStopping 是平滑停机状态，已停止接纳新任务，正在逆序执行 Quiesce 排空与 Stop 资源清理。
	StateStopping

	// StateStopped 是完全停机状态，表明所有静态组件与动态实例均已确认安全退出。
	StateStopped

	// StateIncomplete 是不完全停机状态，表明在关机总截止时仍有组件超时未退，
	// 或 Stop 钩子发生 panic，相关实例已被作为 [StuckInstance] 保护隔离。
	StateIncomplete
)

// Mode 标识插件的三种运行形态。
//
// 每种形态对应不同的生命周期契约与实例化时序，在调用 [App.RegisterTool]、
// [App.RegisterWorker] 或 [App.RegisterComponent] 时由框架自动绑定。
type Mode uint8

// 插件运行形态枚举。
const (
	// ModeTool 是工具/注入态插件。
	// 仅实现 [Plugin] 接口（Start 与 Stop），在应用级为全局单实例，
	// 在启动阶段初始化并常驻，为下游提供只读共享能力。
	ModeTool Mode = iota + 1

	// ModeWorker 是常驻型后台服务插件。
	// 实现 [RunnablePlugin] 接口（含 Run 方法），在应用级为全局单实例，
	// 在启动完成后由看门狗启动独立协程长期运行，退出将触发应用级自动停机。
	ModeWorker

	// ModeComponent 是按需激发的任务型组件插件。
	// 实现 [RunnablePlugin] 接口，不在应用启动时自动实例化，
	// 而是通过 [Activator.Activate] 在运行期按需动态激发，每次激活创建独立实例。
	ModeComponent
)

// OverlapKind 定义 [ModeComponent] 在上一轮任务尚未结束时的重入并发策略。
type OverlapKind uint8

// 动态组件重入并发策略枚举。
const (
	// OverlapSkip 表示跳过策略（默认策略）。
	// 当该组件已有实例正在执行时，新的激活请求将直接被拒绝并返回 [ErrComponentSkipped]。
	OverlapSkip OverlapKind = iota + 1

	// OverlapAllow 表示有界并发策略。
	// 允许在 [OverlapPolicy.MaxConcurrency] 配额内并发创建并执行多个实例，
	// 超过上限时返回 [ErrConcurrencyLimitExceeded]。
	OverlapAllow

	// OverlapQueue 表示有界排队策略。
	// 当无可用执行槽时，新的请求进入 FIFO 等待队列，队列容量受 [OverlapPolicy.MaxQueue] 约束；
	// 队列满时返回 [ErrQueueFull]。
	OverlapQueue
)

// OverlapPolicy 描述 [ModeComponent] 插件的并发与重入配额策略。
//
// 仅对 [ModeComponent] 生效，Tool 或 Worker 插件配置该策略将在建图时报错。
// 零值默认为 [OverlapSkip] 策略。
type OverlapPolicy struct {
	// Kind 是重入策略类型，支持 [OverlapSkip]、[OverlapAllow] 或 [OverlapQueue]。
	Kind OverlapKind

	// MaxConcurrency 是最大并发实例数。仅在 Kind 为 [OverlapAllow] 时生效，取值必须 >= 1。
	MaxConcurrency int

	// MaxQueue 是等待队列的最大容量。仅在 Kind 为 [OverlapQueue] 时生效，取值必须 >= 1（不含正在执行的执行槽）。
	MaxQueue int
}

// PluginInfo 是插件组件注册定义的唯一元信息声明。
//
// 声明组件的主键、名称、暴露资源、所需依赖及重入策略。
// 所有键均使用强类型 [ctxkey.Key]，禁止使用空键或未通过校验的非法键。
type PluginInfo struct {
	// Key 是全局唯一的插件标识键 [ctxkey.Key]。该键在容器中默认映射为插件实例本身。
	Key ctxkey.Key

	// Name 是人类可读的组件名称，用于日志输出与排障诊断报告。
	Name string

	// Mode 是插件运行形态，由 RegisterTool/RegisterWorker/RegisterComponent 自动绑定。
	Mode Mode

	// Provides 是静态声明由本插件额外暴露的共享资源键集合（如连接池、配置对象等）。
	// 声明了 Provides 的插件必须实现 [ctxkey.Exposer] 接口；[ModeComponent] 必须为空。
	Provides []ctxkey.Key

	// Requires 是本插件所必需的依赖键集合（插件主键或暴露的资源键）。
	// 启动期上下文将基于 Requires 构建受限白名单视图 [startScopeView]，越权访问将失败。
	Requires []ctxkey.Key

	// Overlap 是动态组件的重入策略配额，仅对 [ModeComponent] 生效；零值默认为 [OverlapSkip]。
	Overlap OverlapPolicy
}

// Plugin 是所有插件（包含纯工具插件）必须实现的生命周期底线契约接口。
//
// 规范要求插件在钩子中严格遵循传入 Context 的超时与取消信号，且返回后不得遗留未受管协程。
type Plugin interface {
	// Start 执行组件的资源准备与监听绑定。
	//
	// 入参 ctx 是启动期受限上下文（挂载基于 Requires 的白名单视图与独立超时）。
	// 插件在此阶段可建立数据库连接、加载配置或监听端口。
	// 必须遵守 ctx 的取消与超时；返回 non-nil 错误或发生 panic 将中止启动流程并触发回滚停机。
	Start(ctx context.Context) error

	// Stop 执行组件的排空收尾与底层物理资源释放（如关闭连接池、注销监听器）。
	//
	// 入参 ctx 携带本轮关机协调的预算时限。
	// 方法返回后不得遗留由插件自行管理的后台协程。
	// Stop 必须具备幂等性与超时感知能力，且禁止在 Stop 内部发生 panic。
	Stop(ctx context.Context) error
}

// RunnablePlugin 是具备长期执行体的插件接口，由 [ModeWorker] 与 [ModeComponent] 共用。
//
// 继承基础 [Plugin] 接口，并增加阻塞式业务执行体 [RunnablePlugin.Run]。
type RunnablePlugin interface {
	Plugin

	// Run 是组件的阻塞式业务执行体。
	//
	// 入参 ctx 是运行期上下文（挂载完整共享容器、可信父级身份凭据与追踪 ID）。
	// 对于 Worker，Run 应当保持阻塞运行（如事件循环或监听处理），直至 ctx 被取消或发生不可恢复错误；
	// 若 Worker 在运行期间提前非预期返回，看门狗将判定为崩溃并触发应用级关机。
	// 对于 Component，Run 代表一次单次任务执行，执行完成后正常退出。
	Run(ctx context.Context) error
}

// Quiescer 是供 Worker 插件按需可选实现的静默下线扩展接口。
//
// 若插件实现了此接口，关机协调器将在逆拓扑停机时优先调用 [Quiescer.Quiesce]，
// 随后再调用 [Plugin.Stop]。
type Quiescer interface {
	// Quiesce 执行业务流量的静默排空。
	//
	// 插件在此阶段应立即拒绝新的外部接入请求（例如向网关返回健康检查失败或返回 HTTP 503），
	// 并等待当前已接纳的在途请求处理完毕。入参 ctx 携带固定自然排空窗口 [Config.GraceTimeout]。
	// 达到超时或排空完成后返回。
	Quiesce(ctx context.Context) error
}

// Outcome 表示动态组件运行实例的最终归宿分类。
type Outcome uint8

// 实例运行归宿分类枚举。
const (
	// OutcomeSuccess 表示实例执行成功，且所有必要的清理与 Stop 均已确认安全完成。
	OutcomeSuccess Outcome = iota + 1

	// OutcomeFailed 表示实例在 Start、Run 或 Stop 阶段执行失败或发生 panic。
	// 具体错误记录在 [RunResult] 的对应字段中。
	OutcomeFailed

	// OutcomeCancelled 表示实例尚未派发即被取消，或在执行期间响应取消并完成真实收尾。
	OutcomeCancelled
)

// RunResult 汇总动态组件一次实例运行的完整状态账单。
//
// 该结构体由 [RunHandle.Wait] 返回，详细记录执行各阶段的耗时与具体错误。
type RunResult struct {
	// InstanceID 是该次运行实例的唯一标识。
	InstanceID string

	// PluginKey 是该次运行对应的插件注册键 [ctxkey.Key]。
	PluginKey ctxkey.Key

	// Outcome 是最终归宿判定，取值为 [OutcomeSuccess]、[OutcomeFailed] 或 [OutcomeCancelled]。
	Outcome Outcome

	// Dispatched 表示该实例是否已经取得执行槽并进入构造监督流程（不代表 Start 或 Run 一定执行成功）。
	Dispatched bool

	// CancelRequested 表示内核是否向该实例发出了取消请求（包含未派发前取消与在途取消）。
	CancelRequested bool

	// CancelErr 是未执行 Run 时的取消原因；正常执行 Run 时不以此字段冒充 RunErr。
	CancelErr error

	// FactoryErr 记录调用工厂函数时的 panic 或无效实例错误；发生此错误时不调用 Stop。
	FactoryErr error

	// StartErr 记录执行 [Plugin.Start] 钩子时返回的错误或捕获的 panic。
	StartErr error

	// ExposeErr 记录执行 Expose 校验或发布时的错误。
	ExposeErr error

	// QuiesceErr 记录执行 [Quiescer.Quiesce] 时的错误或 panic。
	QuiesceErr error

	// RunErr 记录执行 [RunnablePlugin.Run] 时返回的原始错误或 panic。
	RunErr error

	// StopErr 记录执行 [Plugin.Stop] 清理钩子时返回的错误或 panic。
	StopErr error
}

// RunHandle 是内核返回给激活调用方的动态组件运行监督句柄。
//
// 调用方通过该句柄跟踪组件执行进度与获取最终状态账单。
type RunHandle interface {
	// InstanceID 返回当前运行实例的唯一标识。
	InstanceID() string

	// Wait 阻塞等待组件实例执行完毕，并返回完整的运行账单 [RunResult] 与错误。
	//
	// 入参 ctx 控制等待本身的超时或取消，若 ctx 超时或取消，返回零值 [RunResult] 与 ctx.Err()；
	// 若实例本身执行失败或取消，Wait 将返回真实的 [RunResult] 以及由账单计算得到的综合错误。
	Wait(ctx context.Context) (RunResult, error)
}

// ActivateOption 描述激活动态组件时的可选参数配置函数。
type ActivateOption func(*activateConfig)

type activateConfig struct {
	payload any
}

// WithPayload 设置本次组件激活所携带的输入参数。
//
// 参数保存在激活上下文中，组件可在其 [RunnablePlugin.Run] 内部通过 [PayloadFrom] 提取。
// 内核不对 payload 执行深拷贝，若涉及并发共享对象，调用方须自行保证线程安全。
func WithPayload(payload any) ActivateOption {
	return func(c *activateConfig) {
		c.payload = payload
	}
}

// Activator 供常驻 Worker 或业务在运行期按需动态激发 [ModeComponent] 组件实例。
//
// 实例通常通过 [App.Activator] 获取。
type Activator interface {
	// Activate 按指定插件键 compKey 激发一个组件实例。
	//
	// # 入参约束
	//
	//   - ctx 必须是具备可信父级身份凭据的合法运行期上下文（例如 Worker 的 lifetimeCtx）；
	//     若 ctx 未挂载身份或父级已关闭，返回 [ErrInvalidActivationParent]；
	//   - compKey 必须对应已通过 [App.RegisterComponent] 注册的组件键；未找到返回 [ErrComponentNotFound]；
	//   - opts 可选配置激活负载 [WithPayload]。
	//
	// # 并发与重入决策
	//
	// 内核根据组件注册时声明的 [OverlapPolicy] 决定立即派发、进入等待队列或直接拒绝：
	//
	//   - 若处于 [OverlapSkip] 且已有实例在执行，返回 [ErrComponentSkipped]；
	//   - 若处于 [OverlapAllow] 且并发已满，返回 [ErrConcurrencyLimitExceeded]；
	//   - 若处于 [OverlapQueue] 且队列已满，返回 [ErrQueueFull]；
	//   - 若在激活链路上检测到祖先依赖递归环，返回 [ErrActivationCycle]。
	//
	// 成功接纳后，返回用于跟踪执行的 [RunHandle]。
	Activate(
		ctx context.Context,
		compKey ctxkey.Key,
		opts ...ActivateOption,
	) (RunHandle, error)
}
