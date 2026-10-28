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

// EntryState 枚举常量定义。
const (
	EntryReady       EntryState = iota + 1 // 已完成 Start，可正常读取
	EntryStopped                           // Stop 按契约正常返回后退役，拒绝读取
	EntryUnavailable                       // 提供者故障或清理无法确认，拒读但仍保留依赖
)

// InstanceMeta 是身份的只读描述；复制或自行构造此结构不会获得激活权限。
type InstanceMeta struct {
	InstanceID       string
	PluginKey        ctxkey.Key
	ParentKey        ctxkey.Key // 顶层 Worker 为空，其余为父插件键
	ParentInstanceID string     // 实例树以此关联，顶层 Worker 为空
	TraceID          string     // 当前运行或请求的 RequestKey
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

// StateFrom 从 ctx 中只读查询当前挂载的 AppState。
// 若 ctx 为 nil、未挂载内核 sharedContainer 或未配置 stateReader，返回 (0, false)。
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

// PayloadFrom 从 Component 的 ctx 中提取 Activate 时传入的参数。
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

// InstanceMetaFrom 从运行期 ctx 中提取当前实例的身份元信息。
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

// BridgeRequest 挂载应用容器、可信父实例身份与 requestID。
// 请求取消仍来自 reqCtx；应用强制取消由协议插件合入其请求根 Context。
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

// Detach 只保留容器与追踪 ID，不继承临时数据或父实例激活凭据。
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

// GraceContext 使用本轮固定排空截止，不因晚进入某一层而续期。
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

// CleanupContext 在协调期间不越过总截止；已接纳实例的晚到收尾使用独立上限。
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

// ForceContext 只提供当前 App 的 Grace 到期取消通知，不挂容器/身份，也不暴露 Cancel。
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

// Get 从 ctx 挂载的 SharedContainer 中按 key 读取类型化组件实例
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

// MustGet 为 Get 的 panic 版本
func MustGet[T any](ctx context.Context, key ctxkey.Key) T {
	val, err := Get[T](ctx, key)
	if err != nil {
		panic(err)
	}
	return val
}
