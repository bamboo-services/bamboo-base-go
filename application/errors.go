package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

// 应用生命周期、插件工厂、动态组件激活及依赖共享容器相关的错误哨兵。
//
// 哨兵本身仍是原生 error：它们作为错误链的根因保留，
// 调用方继续可以通过 errors.Is 精确判定失败语义；
// 对外暴露的失败结果统一为 [*apperr.Error]，错误码由 [sentinelCodes] 登记。
var (
	ErrAlreadyRun               = errors.New("app has already run")                                // [App] 实例已执行过 [App.Run]；每个实例仅单次运行语义，禁止重复调用。
	ErrAppClosed                = errors.New("app is closed")                                      // 当前 [App] 处于已关闭或正在关机状态（如 [StateStopping]、[StateStopped]），拒绝执行启动或激活新组件。
	ErrRegistrationClosed       = errors.New("app registration closed")                            // 应用已进入启动或运行阶段，组件注册已冻结；[App.RegisterTool]、[App.RegisterWorker]、[App.RegisterComponent] 必须在 [App.Run] 前完成。
	ErrStartupInterrupted       = errors.New("app startup interrupted")                            // 应用启动初始化阶段收到外部中断或系统关机信号。
	ErrWorkerUnexpectedExit     = errors.New("worker unexpected exit")                             // 常驻 Worker 协程在 [StateRunning] 阶段提前退出（即使返回 nil）；看门狗视作异常崩溃并自动触发应用级平滑停机。
	ErrNilFactory               = errors.New("plugin factory cannot be nil")                       // 向注册方法传入 nil 工厂函数时返回。
	ErrNilPlugin                = errors.New("plugin instance cannot be nil")                      // 插件工厂函数返回了 nil 或类型化 nil 实例。
	ErrInvalidActivationParent  = errors.New("invalid activation parent context")                  // 传入的激活上下文不具备合法父级凭据；必须在合法的 Worker 或 Component 运行期上下文中调用 [Activator.Activate]。
	ErrComponentSkipped         = errors.New("component activation skipped due to overlap policy") // 根据 [OverlapSkip] 策略，因前序实例尚未结束而跳过本次激活。
	ErrConcurrencyLimitExceeded = errors.New("component concurrency limit exceeded")               // 根据 [OverlapAllow] 策略，并发激活数已达到 [OverlapPolicy.MaxConcurrency] 上限。
	ErrQueueFull                = errors.New("component queue is full")                            // 根据 [OverlapQueue] 策略，排队等待队列已达到 [OverlapPolicy.MaxQueue] 上限。
	ErrNestedQueueWouldBlock    = errors.New("nested queue activation would block")                // Component 父级在自身执行期间激活排队型组件且目标队列繁忙；为防父子级联死锁，内核立即拒绝该嵌套激活。
	ErrActivationCycle          = errors.New("activation dependency cycle detected")               // 动态组件激活链路中检测到祖先依赖递归环，为避免死循环而拒绝激活。
	ErrContainerNotMounted      = errors.New("shared container not mounted in context")            // 当前上下文未挂载应用共享容器；通常因直接传入未经 [BridgeRequest] 或 [Detach] 初始化的裸 context.Context。
	ErrUndeclaredDependency     = errors.New("undeclared dependency access")                       // 插件在 [Plugin.Start] 启动阶段尝试通过 [Get] 读取未在其 [PluginInfo.Requires] 中显式声明的组件键。
	ErrComponentStopped         = errors.New("component stopped and retired")                      // 被请求的组件实例已执行完成 [Plugin.Stop] 并正常退役，禁止继续调用。
	ErrComponentTypeMismatch    = errors.New("component type mismatch")                            // 容器中存在的组件实例类型与泛型 [Get] 请求的目标类型不兼容。
	ErrComponentNotFound        = errors.New("component not found")                                // 共享容器中不存在指定键的组件条目。
	ErrComponentUnavailable     = errors.New("component provider unavailable")                     // 目标组件提供者发生故障隔离或清理异常，当前处于不可用状态。
	ErrDuplicateExposedKey      = errors.New("duplicate exposed key")                              // 插件暴露了已被其他插件注册占用的重复键。
	ErrExposedKeysMismatch      = errors.New("exposed keys mismatch")                              // 插件实现的 [ctxkey.Exposer.Expose] 返回的键集合与其元数据 [PluginInfo.Provides] 中声明的资源键不一致（多出、缺少或暴露了 nil）。
	ErrInvalidDependencyScope   = errors.New("invalid dependency scope")                           // 静态依赖配置错误，例如普通插件依赖了激发态的 [ModeComponent]。
)

// 特殊上下文取消错误哨兵，调用方可通过 errors.Is(err, context.Canceled) 统一识别。
var (
	ErrParentClosing error = &canceledSentinelError{msg: "parent component closing"}  // 父级组件正在关闭，因此终止子任务的派发或执行；满足 errors.Is(err, context.Canceled)。
	ErrShuttingDown  error = &canceledSentinelError{msg: "application shutting down"} // 宿主应用正在关机，因此拒绝或中断组件激活与执行；满足 errors.Is(err, context.Canceled)。
)

// sentinelCodes 登记应用自有错误哨兵与统一错误码的对应关系。
//
// 哨兵只描述失败语义，错误码只描述失败分类：两者并存不互相替代。
// 哨兵继续作为错误链根因保留，对外失败结果则是携带该码的 [*apperr.Error]。
//
// 分类口径：
//   - 组件与容器访问类失败复用已有的精确框架码；
//   - 生命周期与并发状态类失败归入 [apperr.FrameworkComponentInvalidState]；
//   - 插件声明与装配校验类失败归入 [apperr.FrameworkInternalError]。
var sentinelCodes = map[error]*apperr.ErrorCode{
	// 组件与容器访问类
	ErrContainerNotMounted:   apperr.FrameworkContainerNotMounted,
	ErrUndeclaredDependency:  apperr.FrameworkDependencyUndeclared,
	ErrComponentNotFound:     apperr.FrameworkComponentNotFound,
	ErrComponentTypeMismatch: apperr.FrameworkComponentTypeMismatch,
	ErrComponentStopped:      apperr.FrameworkComponentStopped,
	ErrComponentUnavailable:  apperr.FrameworkComponentUnavailable,

	// 生命周期与并发状态类
	ErrAlreadyRun:               apperr.FrameworkComponentInvalidState,
	ErrAppClosed:                apperr.FrameworkComponentInvalidState,
	ErrRegistrationClosed:       apperr.FrameworkComponentInvalidState,
	ErrStartupInterrupted:       apperr.FrameworkComponentInvalidState,
	ErrInvalidActivationParent:  apperr.FrameworkComponentInvalidState,
	ErrComponentSkipped:         apperr.FrameworkComponentInvalidState,
	ErrConcurrencyLimitExceeded: apperr.FrameworkComponentInvalidState,
	ErrQueueFull:                apperr.FrameworkComponentInvalidState,
	ErrNestedQueueWouldBlock:    apperr.FrameworkComponentInvalidState,
	ErrActivationCycle:          apperr.FrameworkComponentInvalidState,

	// 插件声明与装配校验类
	ErrNilFactory:             apperr.FrameworkInternalError,
	ErrNilPlugin:              apperr.FrameworkInternalError,
	ErrDuplicateExposedKey:    apperr.FrameworkInternalError,
	ErrExposedKeysMismatch:    apperr.FrameworkInternalError,
	ErrInvalidDependencyScope: apperr.FrameworkInternalError,
	ErrEmptyKey:               apperr.FrameworkInternalError,
	ErrInvalidKey:             apperr.FrameworkInternalError,
	ErrRequestKeyShared:       apperr.FrameworkInternalError,
	ErrSelfProvidedKey:        apperr.FrameworkInternalError,
	ErrMissingDependency:      apperr.FrameworkInternalError,
	ErrSelfDependency:         apperr.FrameworkInternalError,
	ErrDependencyCycle:        apperr.FrameworkInternalError,
	ErrComponentHasProvides:   apperr.FrameworkInternalError,
	ErrInvalidOverlapKind:     apperr.FrameworkInternalError,
	ErrInvalidOverlapCapacity: apperr.FrameworkInternalError,
	ErrInvalidOverlapConfig:   apperr.FrameworkInternalError,
	ErrNonComponentHasOverlap: apperr.FrameworkInternalError,
	ErrUnknownMode:            apperr.FrameworkInternalError,

	// 框架诊断类
	ErrWorkerUnexpectedExit: apperr.FrameworkWorkerUnexpectedExit,
}

// newAppErr 以应用自有错误哨兵为根因构造统一错误。
//
// 哨兵作为根因保留，调用方仍可通过 errors.Is 精确判定；
// 错误码取自 [sentinelCodes] 登记的分类。
func newAppErr(sentinel error, msg string) *apperr.Error {
	code, ok := sentinelCodes[sentinel]
	if !ok {
		code = apperr.FrameworkInternalError
	}
	return apperr.From(code, apperr.ErrMessage(msg), sentinel)
}

// normalizeErr 将外部原因归一为应用所属的统一错误。
//
// 归一顺序：
//   - 原因为空或 nil 载体时返回 nil，表示没有失败；
//   - 原因链中已经存在 [*apperr.Error] 时保留其分类与整条原因链，不重新分类；
//   - 取消与超时经 [control.Classify] 归一为控制类错误码，
//     原生 context 哨兵继续作为根因保留，供 errors.Is 精确判定；
//   - 其它普通原因按调用方给出的所属错误码兜底，原始原因链完整保留。
func normalizeErr(code *apperr.ErrorCode, msg apperr.ErrMessage, cause error) *apperr.Error {
	if apperr.IsNil(cause) {
		return nil
	}
	if structured := apperr.Find(cause); structured != nil {
		return apperr.From(&structured.Code, structured.ErrorMessage, cause)
	}
	if classified := control.Classify(cause, control.PhaseOperation); classified != nil {
		return apperr.New(&classified.Code, msg, cause)
	}
	return apperr.From(code, msg, cause)
}

// aggregateErrs 聚合多个统一错误为唯一的失败结果。
//
// 语义：
//   - 没有有效成员时返回 nil；
//   - 只有一个有效成员时原样返回，不额外套壳；
//   - 多个有效成员时，最外层统一错误使用 [apperr.FrameworkMultipleFailures]（50020），
//     避免任意挑选单个子错误码掩盖“多个阶段失败”，全部成员按入参顺序保留为根因，
//     errors.Is 与 errors.As 可逐个遍历。
func aggregateErrs(errs ...*apperr.Error) *apperr.Error {
	kept := make([]*apperr.Error, 0, len(errs))
	for _, appErr := range errs {
		if appErr != nil {
			kept = append(kept, appErr)
		}
	}

	switch len(kept) {
	case 0:
		return nil
	case 1:
		return kept[0]
	}
	return apperr.Join(apperr.FrameworkMultipleFailures, "多个阶段或组件执行失败", kept...)
}

type canceledSentinelError struct {
	msg string
}

func (e *canceledSentinelError) Error() string {
	return e.msg
}

func (e *canceledSentinelError) Is(target error) bool {
	return target == context.Canceled || target == e
}

// HookPanicError 记录内核在调用外部插件生命周期钩子时捕获的 panic 异常。
//
// 内核在调用工厂函数、Start、Expose、Run、Quiesce、Stop 等钩子时均设置了 panic recovery。
// 捕获后包装为该结构体，保留触发阶段、插件主键、实例标识与完整崩溃堆栈，供排障诊断。
type HookPanicError struct {
	PluginKey  ctxkey.Key // 发生 panic 的插件主键 [ctxkey.Key]。
	InstanceID string     // 发生 panic 的具体实例唯一标识；尚未分配实例时允许为空。
	Stage      string     // 发生 panic 的生命周期阶段，取值为 "factory"、"start"、"expose"、"run"、"quiesce" 或 "stop"。
	Value      any        // recover 捕获到的原始 panic 值。
	Stack      []byte     // 发生 panic 时的完整协程调用堆栈。
}

// Error 返回结构化的 panic 诊断错误描述。
func (e *HookPanicError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.InstanceID != "" {
		return fmt.Sprintf("hook panic on plugin %q (instance %q) at stage %q: %v", e.PluginKey, e.InstanceID, e.Stage, e.Value)
	}
	return fmt.Sprintf("hook panic on plugin %q at stage %q: %v", e.PluginKey, e.Stage, e.Value)
}

// Unwrap 返回原始 panic 转换后的错误对象。
// 若 Value 实现了 error 接口则返回该错误，否则返回 nil。
func (e *HookPanicError) Unwrap() error {
	if e == nil {
		return nil
	}
	if err, ok := e.Value.(error); ok {
		return err
	}
	return nil
}

// newHookPanicErr 将捕获的 panic 封装为统一错误。
//
// 诊断对象 [HookPanicError] 作为根因保留，调用方既可以经 errors.As 命中诊断对象，
// 也可以经 errors.Is 命中 panic 值本身携带的原始错误。
func newHookPanicErr(diag *HookPanicError) *apperr.Error {
	if diag == nil {
		return nil
	}
	return apperr.New(apperr.FrameworkHookPanic, apperr.ErrMessage("生命周期钩子 "+diag.Stage+" 阶段发生 panic"), diag)
}
