package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

// EntryState 表示共享容器内组件条目的生命周期状态。
type EntryState uint8

// 共享容器条目状态枚举。
const (
	EntryReady       EntryState = iota + 1 // 组件已完成 Start 钩子初始化，处于正常可用状态，允许读取。
	EntryStopped                           // 组件已按契约正常完成 Stop 钩子并退役，禁止继续读取。
	EntryUnavailable                       // 组件提供者发生故障、panic 或清理超时，禁止读取但保留依赖。
)

// InstanceMeta 是插件运行期实例的只读身份元信息。
//
// 该结构体保存在运行期上下文中，可通过 [InstanceMetaFrom] 提取。
// 复制或自行在外部构造此结构不会获得内核激活权限，仅用于日志追踪与诊断。
type InstanceMeta struct {
	InstanceID       string     // 当前运行实例的全局唯一标识。
	PluginKey        ctxkey.Key // 当前实例对应的插件注册主键 [ctxkey.Key]。
	ParentKey        ctxkey.Key // 激活当前实例的父级插件键；顶层 Worker 为空。
	ParentInstanceID string     // 父级运行实例的唯一标识；顶层 Worker 为空。
	TraceID          string     // 当前运行或请求的链路追踪标识。
}

// 内部私有 Context Key 类型，防止外部伪造
type (
	containerCtxKeyType struct{}
	scopeViewCtxKeyType struct{}
	metaCtxKeyType      struct{}
	forceCtxKeyType     struct{}
	budgetCtxKeyType    struct{}
	payloadCtxKeyType   struct{}
	trustedAuthKeyType  struct{}
)

var (
	containerCtxKey containerCtxKeyType
	scopeViewCtxKey scopeViewCtxKeyType
	metaCtxKey      metaCtxKeyType
	forceCtxKey     forceCtxKeyType
	budgetCtxKey    budgetCtxKeyType
	payloadCtxKey   payloadCtxKeyType
	trustedAuthKey  trustedAuthKeyType
)

type trustedAuthToken struct {
	appID      string
	instanceID string
	pluginKey  ctxkey.Key
}

type budgetMetadata struct {
	graceDeadline time.Time
	totalDeadline time.Time
	cleanupLimit  time.Duration
}

// StateFrom 从上下文 ctx 中只读查询当前挂载的应用状态 [AppState]。
//
// 该函数供插件内部（如异步任务池、Cron 调度器）在执行业务前探测宿主 [App] 的运行阶段。
// 插件借此可在 [StateStarting] 或 [StateStopping] 阶段主动拒绝接纳新任务。
// 该查询为纯只读操作，不暴露任何修改 [App] 状态的权限。
//
// # 返回值
//
// 若 ctx 为 nil、未挂载内核共享容器或未配置状态读取器，返回 (0, false)；
// 否则返回当前的应用状态及 true。
func StateFrom(ctx context.Context) (AppState, bool) {
	if ctx == nil {
		return 0, false
	}
	var container *sharedContainer
	if val := ctx.Value(containerCtxKey); val != nil {
		container, _ = val.(*sharedContainer)
	}
	if container == nil || container.stateReader == nil {
		return 0, false
	}
	return container.stateReader(), true
}

// PayloadFrom 从动态组件的上下文 ctx 中提取通过 [WithPayload] 传入的激活参数。
//
// 该函数通常在 [ModeComponent] 插件的 [RunnablePlugin.Run] 钩子内调用。
//
// # 返回值
//
// 若 ctx 为 nil、未挂载有效参数或参数类型无法转换为泛型类型 T，
// 返回类型 T 的零值与 false；转换成功时返回对应的值与 true。
func PayloadFrom[T any](ctx context.Context) (T, bool) {
	var zero T
	if ctx == nil {
		return zero, false
	}
	val := ctx.Value(payloadCtxKey)
	if val == nil {
		return zero, false
	}
	typedVal, ok := val.(T)
	return typedVal, ok
}

// InstanceMetaFrom 从运行期上下文 ctx 中提取当前实例的身份元信息 [InstanceMeta]。
//
// 插件可通过该元信息获取自身的实例标识 [InstanceMeta.InstanceID]、
// 父级组件信息及链路追踪标识 [InstanceMeta.TraceID]。
//
// # 返回值
//
// 若 ctx 为 nil 或未挂载合法的身份元信息，返回零值 [InstanceMeta] 与 false；
// 提取成功时返回对应结构与 true。
func InstanceMetaFrom(ctx context.Context) (InstanceMeta, bool) {
	if ctx == nil {
		return InstanceMeta{}, false
	}
	val := ctx.Value(metaCtxKey)
	if val == nil {
		return InstanceMeta{}, false
	}
	meta, ok := val.(InstanceMeta)
	return meta, ok
}

// BridgeRequest 融合外部请求上下文与应用生命周期上下文，生成请求专用上下文。
//
// 在 Web HTTP 或 gRPC 服务端收到外部请求时，必须通过该函数进行上下文桥接：
//
//   - 外部请求取消与超时（来自 reqCtx）保持生效，客户端断开时仍能正常触发取消；
//   - 应用共享容器、可信父级身份凭据与链路追踪 requestID（来自 appLifetimeCtx）挂载至新上下文，
//     使得请求处理链路能够通过 [Get] 正常访问注册的共享工具与组件；
//   - 拦截并防止将启动期的受限白名单视图 [startScopeView] 升格为完整容器；
//     若检测到传入的 appLifetimeCtx 属于启动期视图，将安全降级返回原生 reqCtx。
//
// 若 reqCtx 为 nil，自动回退使用 [context.Background]。若 appLifetimeCtx 为 nil，
// 则直接返回 reqCtx。
func BridgeRequest(
	reqCtx context.Context,
	appLifetimeCtx context.Context,
	requestID string,
) context.Context {
	if reqCtx == nil {
		reqCtx = context.Background()
	}
	if appLifetimeCtx == nil {
		return reqCtx
	}
	// 校验 appLifetimeCtx 是否为合法运行 Context（必须含有 container 且不能是启动期 startScopeView）
	if appLifetimeCtx.Value(scopeViewCtxKey) != nil {
		// 误传了 startScopeView，不允许升格为完整容器
		return reqCtx
	}
	containerVal := appLifetimeCtx.Value(containerCtxKey)
	if containerVal == nil {
		return reqCtx
	}

	return &bridgedContext{
		reqCtx:         reqCtx,
		appLifetimeCtx: appLifetimeCtx,
		requestID:      requestID,
	}
}

type bridgedContext struct {
	reqCtx         context.Context
	appLifetimeCtx context.Context
	requestID      string
}

func (c *bridgedContext) Deadline() (deadline time.Time, ok bool) {
	return c.reqCtx.Deadline()
}

func (c *bridgedContext) Done() <-chan struct{} {
	return c.reqCtx.Done()
}

func (c *bridgedContext) Err() error {
	return c.reqCtx.Err()
}

func (c *bridgedContext) Value(key any) any {
	// 1. 特殊键拦截：RequestKey 优先返回本次 requestID
	if key == ctxkey.RequestKey || key == ctxkey.RequestKey.String() {
		if c.requestID != "" {
			return c.requestID
		}
	}

	// 2. 内部内核键（容器、身份凭据、InstanceMeta）：从 appLifetimeCtx 取
	switch key.(type) {
	case containerCtxKeyType, metaCtxKeyType, trustedAuthKeyType, forceCtxKeyType:
		return c.appLifetimeCtx.Value(key)
	}

	// 3. ctxkey.Key 检查：若属于共享容器中的键，从 appLifetimeCtx 容器取
	if k, ok := key.(ctxkey.Key); ok {
		if container, ok := c.appLifetimeCtx.Value(containerCtxKey).(*sharedContainer); ok {
			if entry, exists := container.getEntry(k); exists {
				if EntryState(atomic.LoadUint32(&entry.state)) == EntryReady {
					return entry.value
				}
				return nil
			}
		}
	}

	// 4. 其他情况委托给原生 reqCtx
	return c.reqCtx.Value(key)
}

// Detach 剥离请求上下文生成用于后台异步任务的独立上下文。
//
// 适用于协程池、异步计算或后台持久化场景，具有以下行为特征：
//
//   - 阻断父级取消：不继承 parentCtx 的超时与客户端中断取消，生成全新独立取消根；
//   - 继承共享能力：保留应用共享组件容器，后台协程仍可继续通过 [Get] 安全读取组件；
//   - 继承追踪链路：保留父上下文中的 [ctxkey.RequestKey] 链路追踪标识；
//   - 绑定全局强制取消：将上下文关联至宿主应用 Grace 到期的全局强制取消根；
//   - 隔离激活凭据：不继承父级的 Component 动态激活权限，防止越权嵌套激发。
//
// # 资源归属
//
// 返回派生上下文及对应的 [context.CancelFunc]。调用方必须负责在异步任务完成
// 或发生异常时调用 cancel 释放协程树资源。若传入的 parentCtx 包含启动期受限视图，
// 将返回已提前取消的空上下文。
func Detach(parentCtx context.Context) (context.Context, context.CancelFunc) {
	if parentCtx == nil {
		parentCtx = context.Background()
	}

	// 检查是否挂载了容器
	var container *sharedContainer
	if val := parentCtx.Value(containerCtxKey); val != nil {
		container, _ = val.(*sharedContainer)
	}

	// 检查是否是 startScopeView：Detach 不接受启动期受限视图
	if parentCtx.Value(scopeViewCtxKey) != nil {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx, cancel
	}

	// 获取追踪 ID
	var traceID string
	if metaVal := parentCtx.Value(metaCtxKey); metaVal != nil {
		if meta, ok := metaVal.(InstanceMeta); ok {
			traceID = meta.TraceID
		}
	}
	if traceID == "" {
		if reqVal := parentCtx.Value(ctxkey.RequestKey); reqVal != nil {
			if s, ok := reqVal.(string); ok {
				traceID = s
			}
		}
	}

	// 获取全局 forceCtx（当前 App 的强制取消根）
	forceVal := parentCtx.Value(forceCtxKey)
	baseCtx := context.Background()
	if forceCtx, ok := forceVal.(context.Context); ok && forceCtx != nil {
		baseCtx = forceCtx
	}

	detachedCtx, cancel := context.WithCancel(baseCtx)

	ctx := &detachedContext{
		Context:   detachedCtx,
		container: container,
		traceID:   traceID,
	}
	return ctx, cancel
}

type detachedContext struct {
	context.Context
	container *sharedContainer
	traceID   string
}

func (c *detachedContext) Value(key any) any {
	if key == containerCtxKey {
		return c.container
	}
	if key == ctxkey.RequestKey || key == ctxkey.RequestKey.String() {
		if c.traceID != "" {
			return c.traceID
		}
	}
	if k, ok := key.(ctxkey.Key); ok && c.container != nil {
		if entry, ok := c.container.getEntry(k); ok {
			if EntryState(atomic.LoadUint32(&entry.state)) == EntryReady {
				return entry.value
			}
			return nil
		}
	}
	return c.Context.Value(key)
}

// GraceContext 基于关机协调的固定自然排空截止时间派生上下文。
//
// 在插件的 [Quiescer.Quiesce] 或平滑停机逻辑中调用，具有以下契约：
//
//   - 统一截止时限：使用本轮关机协调中统一固定的 GraceDeadline，
//     不会因为某些组件较晚进入停机层级而获得额外的排空时间续期；
//   - 立即过期检测：若调用时排空窗口已经耗尽，返回立即取消的 Context。
//
// 返回派生上下文及对应的取消函数，调用方负责 defer 取消以释放定时器资源。
// 若 stopCtx 未携带预算元信息，默认回退为普通取消上下文。
func GraceContext(stopCtx context.Context) (context.Context, context.CancelFunc) {
	if stopCtx == nil {
		return context.WithCancel(context.Background())
	}
	budgetVal := stopCtx.Value(budgetCtxKey)
	if budgetVal == nil {
		return context.WithCancel(stopCtx)
	}
	budget, ok := budgetVal.(*budgetMetadata)
	if !ok || budget.graceDeadline.IsZero() {
		return context.WithCancel(stopCtx)
	}

	now := time.Now()
	if budget.graceDeadline.Before(now) {
		ctx, cancel := context.WithCancel(stopCtx)
		cancel()
		return ctx, cancel
	}

	return context.WithDeadline(stopCtx, budget.graceDeadline)
}

// CleanupContext 基于资源清理上限派生用于 Stop 钩子的上下文。
//
// 派生上下文的截止时间受限于 [Config.CleanupTimeout]（默认 10s），
// 且绝对不会超过本轮关机协调的总截止时限 [Config.ShutdownTimeout]。
//
// 返回派生上下文及对应的取消函数，调用方负责 defer 取消以释放资源。
// 若当前时间已超出总关机截止时限，返回立即取消的 Context。
func CleanupContext(stopCtx context.Context) (context.Context, context.CancelFunc) {
	if stopCtx == nil {
		return context.WithCancel(context.Background())
	}
	budgetVal := stopCtx.Value(budgetCtxKey)
	if budgetVal == nil {
		return context.WithTimeout(stopCtx, 10*time.Second)
	}
	budget, ok := budgetVal.(*budgetMetadata)
	if !ok {
		return context.WithTimeout(stopCtx, 10*time.Second)
	}

	limit := budget.cleanupLimit
	if limit <= 0 {
		limit = 10 * time.Second
	}
	cleanupDeadline := time.Now().Add(limit)

	if !budget.totalDeadline.IsZero() && budget.totalDeadline.Before(cleanupDeadline) {
		cleanupDeadline = budget.totalDeadline
	}

	if cleanupDeadline.Before(time.Now()) {
		ctx, cancel := context.WithCancel(stopCtx)
		cancel()
		return ctx, cancel
	}

	return context.WithDeadline(stopCtx, cleanupDeadline)
}

// ForceContext 返回当前应用在 Grace 排空到期时的全局强制取消根上下文。
//
// 该上下文仅提供只读的取消通知（Done 通道被触发），不携带共享容器与组件激活凭据，
// 也不向外部暴露 Cancel 函数。协议插件可在启动阶段监听该 Context，
// 作为连接断开的底线兜底信号。若 ctx 为 nil 或未挂载强制取消根，返回 [context.Background]。
func ForceContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	if forceVal := ctx.Value(forceCtxKey); forceVal != nil {
		if forceCtx, ok := forceVal.(context.Context); ok {
			return forceCtx
		}
	}
	return context.Background()
}

func newAppErr(code *apperr.ErrorCode, msg string, cause error) *apperr.Error {
	var errCode apperr.ErrorCode
	if code != nil {
		errCode = *code
	}
	return &apperr.Error{
		Code:         errCode,
		ErrorMessage: apperr.ErrMessage(msg),
		Err:          cause,
	}
}

// Get 从上下文 ctx 挂载的共享容器中按 key 提取类型化组件实例。
//
// 该函数是 Bamboo Base Go v2 框架中提取依赖组件的标准且唯一方式，
// 禁止使用裸 ctx.Value 进行无安全保障的类型断言。
//
// # 读取时序与受限白名单
//
//   - 启动阶段（[Plugin.Start] 钩子）：上下文挂载受限白名单视图。插件只能提取
//     其在 [PluginInfo.Requires] 中显式声明的直接依赖键；越权读取未声明键时，
//     返回携带错误码 [apperr.Forbidden] 与原因 [ErrUndeclaredDependency] 的错误；
//   - 运行阶段（[RunnablePlugin.Run] 与请求处理链路）：上下文挂载完整的只读容器视图。
//     通过 [BridgeRequest] 或 [Detach] 派生的上下文均可读取容器内已就绪的组件；
//   - 停机阶段（[Plugin.Stop] 钩子）：若目标组件已完成 Stop 退役（[EntryStopped]），
//     提取将返回携带错误码 [apperr.Disabled] 的错误；若目标组件提供者发生故障隔离
//     （[EntryUnavailable]），返回携带错误码 [apperr.OperationDenied] 的错误。
//
// # 错误处理
//
// 提取失败时返回类型 T 的零值以及结构化的 [*apperr.Error] 业务错误：
//
//   - 上下文未挂载容器（例如漏调 [BridgeRequest] 或直接传裸 [context.Background]）：
//     返回错误码 [apperr.ParameterEmpty]；
//   - 组件键未找到：返回错误码 [apperr.NotExist]；
//   - 组件实例实际类型与泛型目标类型 T 不兼容：返回错误码 [apperr.ParameterType]。
func Get[T any](ctx context.Context, key ctxkey.Key) (T, *apperr.Error) {
	var zero T
	if ctx == nil {
		return zero, newAppErr(apperr.ParameterEmpty, "上下文未挂载应用 SharedContainer，请检查是否漏调 BridgeRequest/Detach", ErrContainerNotMounted)
	}

	// 1. 检查启动期受限视图
	if viewVal := ctx.Value(scopeViewCtxKey); viewVal != nil {
		view, ok := viewVal.(*startScopeView)
		if ok {
			entry, err := view.getEntry(key)
			if err != nil {
				if errors.Is(err, ErrUndeclaredDependency) {
					return zero, newAppErr(apperr.Forbidden, fmt.Sprintf("插件 [%s] 未在 Requires 中声明依赖 [%s]，禁止在 Start 阶段越权读取", view.ownerKey, key), ErrUndeclaredDependency)
				}
				if errors.Is(err, ErrComponentNotFound) {
					return zero, newAppErr(apperr.NotExist, fmt.Sprintf("组件 [%s] 未找到", key), ErrComponentNotFound)
				}
				return zero, newAppErr(apperr.DeveloperError, err.Error(), err)
			}
			return castEntry[T](key, entry)
		}
	}

	// 2. 检查共享容器
	containerVal := ctx.Value(containerCtxKey)
	if containerVal == nil {
		return zero, newAppErr(apperr.ParameterEmpty, "上下文未挂载应用 SharedContainer，请检查是否漏调 BridgeRequest/Detach", ErrContainerNotMounted)
	}
	container, ok := containerVal.(*sharedContainer)
	if !ok || container == nil {
		return zero, newAppErr(apperr.ParameterEmpty, "上下文未挂载应用 SharedContainer，请检查是否漏调 BridgeRequest/Detach", ErrContainerNotMounted)
	}

	entry, ok := container.getEntry(key)
	if !ok {
		// 容器未命中，尝试回退原生 ctx.Value
		val := ctx.Value(key)
		if val == nil {
			val = ctx.Value(key.String())
		}
		if val != nil {
			typedVal, ok := val.(T)
			if !ok {
				var targetZero T
				return zero, newAppErr(apperr.ParameterType, fmt.Sprintf("组件 [%s] 实际类型为 %T，无法断言为目标类型 %T", key, val, targetZero), ErrComponentTypeMismatch)
			}
			return typedVal, nil
		}
		return zero, newAppErr(apperr.NotExist, fmt.Sprintf("组件 [%s] 未找到", key), ErrComponentNotFound)
	}

	return castEntry[T](key, entry)
}

func castEntry[T any](key ctxkey.Key, entry *containerEntry) (T, *apperr.Error) {
	var zero T
	state := EntryState(atomic.LoadUint32(&entry.state))
	switch state {
	case EntryStopped:
		return zero, newAppErr(apperr.Disabled, fmt.Sprintf("组件 [%s] 已停止卸载，禁止继续访问", key), ErrComponentStopped)
	case EntryUnavailable:
		return zero, newAppErr(apperr.OperationDenied, fmt.Sprintf("组件 [%s] 提供者故障隔离中，禁止访问", key), ErrComponentUnavailable)
	case EntryReady:
		// 正常读取
	default:
		return zero, newAppErr(apperr.InvalidState, fmt.Sprintf("组件 [%s] 状态无效", key), ErrComponentUnavailable)
	}

	if entry.value == nil {
		var targetZero T
		if reflect.ValueOf(targetZero).IsValid() && reflect.TypeOf(targetZero).Kind() != reflect.Pointer && reflect.TypeOf(targetZero).Kind() != reflect.Interface {
			return zero, newAppErr(apperr.ParameterType, fmt.Sprintf("组件 [%s] 实际类型为 nil，无法断言为目标类型 %T", key, targetZero), ErrComponentTypeMismatch)
		}
		return zero, nil
	}

	typedVal, ok := entry.value.(T)
	if !ok {
		var targetZero T
		return zero, newAppErr(apperr.ParameterType, fmt.Sprintf("组件 [%s] 实际类型为 %T，无法断言为目标类型 %T", key, entry.value, targetZero), ErrComponentTypeMismatch)
	}
	return typedVal, nil
}

// MustGet 为 [Get] 的 panic 版本。
//
// 尝试从 ctx 中提取组件，若提取成功返回类型化组件实例；
// 若提取失败（返回 non-nil 的 [*apperr.Error]），则直接 panic 该错误。
//
// 仅建议在应用启动期确信组件必存在、或内部确定性基础设施装配中使用。
// 在处理外部 HTTP/gRPC 请求时，建议优先使用 [Get] 显式处理错误。
func MustGet[T any](ctx context.Context, key ctxkey.Key) T {
	val, err := Get[T](ctx, key)
	if err != nil {
		panic(err)
	}
	return val
}
