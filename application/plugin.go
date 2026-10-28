package application

import (
	"context"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

// AppState 描述本轮 App，不代表某个任务的业务结果。
type AppState uint8

// AppState 枚举常量定义。
const (
	StateNew AppState = iota + 1
	StateStarting
	StateRunning
	StateStopping
	StateStopped
	StateIncomplete
)

// Mode 标识插件的三种运行形态。
type Mode uint8

// Mode 枚举常量定义。
const (
	ModeTool      Mode = iota + 1 // 工具/注入态：仅 Start/Stop，应用级单实例
	ModeWorker                    // 常驻型运行时：Start/Run/Stop，应用级单实例
	ModeComponent                 // 激发态运行时：Start/Run/Stop，每次激活新建实例
)

// OverlapKind 定义 Component 在上一轮尚未结束时的重入策略。
type OverlapKind uint8

// OverlapKind 枚举常量定义。
const (
	OverlapSkip  OverlapKind = iota + 1 // 默认：跳过本次激活
	OverlapAllow                        // 允许有界并发（受 MaxConcurrency 限制）
	OverlapQueue                        // 有界排队等待（受 MaxQueue 限制）
)

// OverlapPolicy 描述 Component 的并发与重入配额。
type OverlapPolicy struct {
	Kind           OverlapKind
	MaxConcurrency int // 仅 OverlapAllow 使用，必须 >= 1
	MaxQueue       int // 仅 OverlapQueue 使用，必须 >= 1；不含单个执行槽
}

// PluginInfo 是注册定义的唯一元信息来源；插件主键与资源键都使用 ctxkey.Key。
type PluginInfo struct {
	Key      ctxkey.Key    // 全局唯一插件标识；该键默认保存插件实例
	Name     string        // 人类可读名称，用于日志与排障报告
	Mode     Mode          // 由 RegisterWorker/Component/Tool 自动绑定并校验
	Provides []ctxkey.Key  // 静态声明额外暴露的资源键；Component 必须为空
	Requires []ctxkey.Key  // 所需插件主键或资源键；启动前解析到 Tool/Worker 提供者
	Overlap  OverlapPolicy // 仅 ModeComponent 生效，零值默认采用 OverlapSkip
}

// Plugin 是所有插件（含纯工具插件）必须实现的生命周期底线契约。
type Plugin interface {
	Start(ctx context.Context) error // 资源准备与端口监听；接收受限 startCtx
	Stop(ctx context.Context) error  // 真实排空与资源清理；返回后不得遗留自管执行
}

// RunnablePlugin 由 Worker 与 Component 共用。
type RunnablePlugin interface {
	Plugin
	Run(ctx context.Context) error // 阻塞执行体；接收实例 lifetimeCtx
}

// Quiescer 是 Worker 按需可选实现的静默扩展接口。
type Quiescer interface {
	Quiesce(ctx context.Context) error
}

// Outcome 表示运行实例的最终归宿分类。
type Outcome uint8

// Outcome 枚举常量定义。
const (
	OutcomeSuccess   Outcome = iota + 1 // 执行成功且需要执行的清理均真实完成
	OutcomeFailed                       // Start/Run/Stop 失败；实际错误保留在结果中
	OutcomeCancelled                    // 未派发已取消，或执行明确响应取消且真实收尾
)

// RunResult 汇总一次实例运行的完整账单。
type RunResult struct {
	InstanceID      string
	PluginKey       ctxkey.Key
	Outcome         Outcome
	Dispatched      bool  // 已取得执行槽、进入构造监督；不等于 Start/Run 一定执行
	CancelRequested bool  // 是否收到内核取消请求，包含尚未派发就被取消
	CancelErr       error // 未执行 Run 时的取消原因，不冒充 RunErr
	FactoryErr      error // 工厂 panic 或返回无效实例；没有对象时不调用 Stop
	StartErr        error // Start 返回错误或 panic
	ExposeErr       error // Expose panic 或交付校验失败
	QuiesceErr      error // 静默钩子返回错误或 panic
	RunErr          error // Run 返回的原始错误
	StopErr         error // Stop 返回的清理错误
}

// RunHandle 是内核返回的运行监督句柄（非插件接口）。
type RunHandle interface {
	InstanceID() string
	Wait(ctx context.Context) (RunResult, error)
}

// ActivateOption 描述激活 Component 时的可选参数。
type ActivateOption func(*activateConfig)

type activateConfig struct {
	payload any
}

// WithPayload 保存本次接纳参数，不自动深拷贝可变对象。
func WithPayload(payload any) ActivateOption {
	return func(c *activateConfig) {
		c.payload = payload
	}
}

// Activator 供 Worker 或业务按需激发 Component 实例。
type Activator interface {
	Activate(
		ctx context.Context,
		compKey ctxkey.Key,
		opts ...ActivateOption,
	) (RunHandle, error)
}
