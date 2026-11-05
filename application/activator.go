package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

// ==============================
// 内部任务状态与引擎结构
// ==============================

// componentQueueItem 描述排队中尚未派发的任务。
type componentQueueItem struct {
	instanceID string         // 分配给该次激活实例的唯一 ID（如 compKey_timestamp）
	parentID   string         // 发起激活的父级实例唯一 ID
	parentKey  ctxkey.Key     // 发起激活的父级插件键
	compKey    ctxkey.Key     // 被激发的组件唯一标识键
	handle     *runHandleImpl // 交付给调用方的运行句柄
	payload    any            // 透传给组件的业务激活数据
	deps       []ctxkey.Key   // 该组件声明依赖的外部资源键集合

	// 取消与派发状态分支
	mu          sync.Mutex         // 保护当前排队项内部状态的互斥锁
	stage       string             // 当前生命周期阶段（queued/factory/start/run/stop/stop-panic）
	dispatched  bool               // 标识是否已成功分配槽位并派发执行
	cancelled   bool               // 标识是否在派发前被取消
	cancelErr   error              // 记录排队期取消的原因错误
	runCancel   context.CancelFunc // 派发后用于级联取消执行的 CancelFunc
	executeDone chan struct{}      // 标识整个执行流程完全结束的信号通道
}

// componentSlotState 追踪每个 Component 注册键的重入与并发配额。
type componentSlotState struct {
	mu           sync.Mutex            // 保护槽位并发与排队队列的互斥锁
	policy       OverlapPolicy         // 注册时声明的并发与重入策略
	activeCount  int                   // 当前正在并发执行的组件实例数量
	queuedItems  []*componentQueueItem // 排队等待执行槽位的任务队列
	wakeWorkerCh chan struct{}         // 用于通知等待槽位空闲的调度通道
}

// parentState 记录存活的父级（Worker 或 Component）状态。
type parentState struct {
	mu               sync.Mutex // 保护父级状态与子树计数的互斥锁
	instanceID       string     // 父级实例的全局唯一标识
	pluginKey        ctxkey.Key // 父级组件的插件标识键
	parentID         string     // 若父级自身也是 Component，指向其上层祖先 ID
	isComponent      bool       // 标识父级是否为 Component（用于嵌套激活规则校验）
	closed           bool       // 标识父级是否已触发关闭流程
	activeCancelSent bool       // 标识是否已向所有存活子任务发送取消信号
	closeCause       error      // 记录父级关闭的诱因错误

	// 子树计数
	activeCount    int                            // 属于该父级的活跃与排队子任务总数
	zeroCh         chan struct{}                  // 当 activeCount 降为 0 时关闭的归零等待通道
	queuedItemMap  map[string]*componentQueueItem // 属于该父级且尚未派发的排队项索引
	runningCancels map[string]context.CancelFunc  // 属于该父级且正在运行的任务取消函数索引
}

func newParentState(instanceID string, pluginKey ctxkey.Key, parentID string, isComponent bool) *parentState {
	ch := make(chan struct{})
	close(ch)
	return &parentState{
		instanceID:     instanceID,
		pluginKey:      pluginKey,
		parentID:       parentID,
		isComponent:    isComponent,
		zeroCh:         ch,
		queuedItemMap:  make(map[string]*componentQueueItem),
		runningCancels: make(map[string]context.CancelFunc),
	}
}

func (p *parentState) inc() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.activeCount == 0 {
		p.zeroCh = make(chan struct{})
	}
	p.activeCount++
}

func (p *parentState) dec() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.activeCount--
	if p.activeCount == 0 {
		close(p.zeroCh)
	}
}

// componentEngine 是 Component 运行时的核心调度器。
type componentEngine struct {
	app *App // 所属的应用实例指针

	mu        sync.RWMutex                         // 保护引擎定义、槽位与全局状态的读写锁
	stopped   bool                                 // 标识调度引擎是否已停止接收新任务
	stopErr   error                                // 引擎停止时的错误原因
	slots     map[ctxkey.Key]*componentSlotState   // 组件键到并发槽位状态的映射表
	parents   map[string]*parentState              // 存活父级实例 ID 到父级状态的映射表
	compDefs  map[ctxkey.Key]PluginInfo            // 已注册的组件元信息定义表
	compFacs  map[ctxkey.Key]func() RunnablePlugin // 已注册的组件工厂函数表
	liveItems map[string]*componentQueueItem       // 存活与执行中的任务字典

	// 依赖占用集合追踪：供集成 owner 查询真实占用情况
	retainedDepsMu    sync.RWMutex       // 保护依赖占用计数的读写锁
	retainedDeps      map[ctxkey.Key]int // 依赖的提供者/资源键 -> 引用计数
	dependencyChanged chan struct{}      // 依赖占用发生变化时的通知通道
}

func newComponentEngine(a *App) *componentEngine {
	e := &componentEngine{
		app:               a,
		slots:             make(map[ctxkey.Key]*componentSlotState),
		parents:           make(map[string]*parentState),
		compDefs:          make(map[ctxkey.Key]PluginInfo),
		compFacs:          make(map[ctxkey.Key]func() RunnablePlugin),
		liveItems:         make(map[string]*componentQueueItem),
		retainedDeps:      make(map[ctxkey.Key]int),
		dependencyChanged: make(chan struct{}),
	}

	// 从 a.defs 载入 Component 定义
	if a != nil {
		a.regMu.Lock()
		for _, d := range a.defs {
			if d.info.Mode == ModeComponent {
				e.compDefs[d.info.Key] = d.info
				e.compFacs[d.info.Key] = d.compFac
				e.slots[d.info.Key] = &componentSlotState{
					policy:       d.info.Overlap,
					wakeWorkerCh: make(chan struct{}, 1),
				}
			}
		}
		a.regMu.Unlock()
	}

	return e
}

// ==============================
// 引擎构造与组件登记
// ==============================

// RegisterComponentDef 供测试 Harness 或 App 初始化前动态登记组件定义。
//
// 该方法受互斥锁保护，并发安全。
//
// # 使用场景与契约
//
//   - info 必须声明有效的 [PluginInfo.Key]，方法内部自动将其 Mode 强制归一化为 [ModeComponent]；
//   - fac 是在组件激活时产生全新 [RunnablePlugin] 实例的工厂函数闭包；
//   - 主要供内部装配（如 [App.RegisterComponent]）及模块自测 Harness 在运行前完成定义注册；
//   - 若组件对应的槽位状态尚未初始化，内部会自动根据 info.Overlap 策略分配槽位并初始化调度通道。
func (e *componentEngine) RegisterComponentDef(info PluginInfo, fac func() RunnablePlugin) {
	e.mu.Lock()
	defer e.mu.Unlock()
	info.Mode = ModeComponent
	e.compDefs[info.Key] = info
	e.compFacs[info.Key] = fac
	if _, ok := e.slots[info.Key]; !ok {
		e.slots[info.Key] = &componentSlotState{
			policy:       info.Overlap,
			wakeWorkerCh: make(chan struct{}, 1),
		}
	}
}

// registerWorker 登记存活的 Worker 父级（必须具有有效可信身份）。
func (e *componentEngine) registerWorker(instanceID string, key ctxkey.Key) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.parents[instanceID] = newParentState(instanceID, key, "", false)
}

// ==============================
// 依赖追踪与状态查询
// ==============================

// GetRetainedDependencies 快照查询当前所有处于执行状态的 Component 所占用的真实外部依赖集合。
//
// 返回以资源键为 Key、当前引用计数为 Value 的字典浅拷贝。
//
// # 并发与使用契约
//
//   - 该方法受内部读锁保护，并发安全；
//   - 供关机协调器（[shutdownCoordinator]）或运行时拓扑监控查询外部 Tool 依赖是否仍被占用；
//   - 外部调用者修改返回的 map 不会影响引擎内部真实的依赖计数。
func (e *componentEngine) GetRetainedDependencies() map[ctxkey.Key]int {
	e.retainedDepsMu.RLock()
	defer e.retainedDepsMu.RUnlock()
	copied := make(map[ctxkey.Key]int, len(e.retainedDeps))
	for k, v := range e.retainedDeps {
		copied[k] = v
	}
	return copied
}

// dependencySnapshot 原子快照依赖占用计数和变化通知通道，供 coordinator 无轮询等待
func (e *componentEngine) dependencySnapshot() (map[ctxkey.Key]int, <-chan struct{}) {
	e.retainedDepsMu.Lock()
	defer e.retainedDepsMu.Unlock()
	copied := make(map[ctxkey.Key]int, len(e.retainedDeps))
	for k, v := range e.retainedDeps {
		copied[k] = v
	}
	ch := e.dependencyChanged
	return copied, ch
}

func (e *componentEngine) retainDeps(deps []ctxkey.Key) {
	if len(deps) == 0 {
		return
	}
	e.retainedDepsMu.Lock()
	defer e.retainedDepsMu.Unlock()
	for _, k := range deps {
		e.retainedDeps[k]++
	}
	close(e.dependencyChanged)
	e.dependencyChanged = make(chan struct{})
}

func (e *componentEngine) releaseDeps(deps []ctxkey.Key) {
	if len(deps) == 0 {
		return
	}
	e.retainedDepsMu.Lock()
	defer e.retainedDepsMu.Unlock()
	for _, k := range deps {
		e.retainedDeps[k]--
		if e.retainedDeps[k] <= 0 {
			delete(e.retainedDeps, k)
		}
	}
	close(e.dependencyChanged)
	e.dependencyChanged = make(chan struct{})
}

// setTaskStage 更新任务阶段
func (e *componentEngine) setTaskStage(item *componentQueueItem, stage string) {
	if item == nil {
		return
	}
	item.mu.Lock()
	item.stage = stage
	item.mu.Unlock()
}

// forgetTask 从 liveItems 中注销任务，不释放占用，且不在 item 锁内调用
func (e *componentEngine) forgetTask(item *componentQueueItem) {
	if item == nil {
		return
	}
	e.mu.Lock()
	delete(e.liveItems, item.instanceID)
	e.mu.Unlock()
}

// snapshotLiveTasks 收集存活且已派发的任务快照，避免持 e.mu 等待 item.mu 造成锁倒置
func (e *componentEngine) snapshotLiveTasks() []StuckInstance {
	e.mu.Lock()
	items := make([]*componentQueueItem, 0, len(e.liveItems))
	for _, it := range e.liveItems {
		items = append(items, it)
	}
	e.mu.Unlock()

	var stuckList []StuckInstance
	for _, it := range items {
		it.mu.Lock()
		if !it.dispatched {
			it.mu.Unlock()
			continue
		}
		stage := it.stage
		it.mu.Unlock()

		depsCopy := make([]ctxkey.Key, len(it.deps))
		copy(depsCopy, it.deps)

		stuckList = append(stuckList, StuckInstance{
			PluginKey:    it.compKey,
			InstanceID:   it.instanceID,
			Stage:        stage,
			RetainedDeps: depsCopy,
			Err:          context.DeadlineExceeded,
		})
	}
	return stuckList
}

// closeParent 关闭指定父级：阻止新接纳，未派发项取消，向运行中在途任务发出取消
func (e *componentEngine) closeParent(instanceID string, cause error) {
	e.mu.Lock()
	p, ok := e.parents[instanceID]
	e.mu.Unlock()
	if !ok {
		return
	}

	if cause == nil {
		cause = ErrParentClosing
	}

	p.mu.Lock()
	p.closed = true
	p.closeCause = cause

	// 收集待取消的未派发排队项（仅收集一次未清空的排队项）
	var itemsToCancel []*componentQueueItem
	for _, it := range p.queuedItemMap {
		itemsToCancel = append(itemsToCancel, it)
	}
	p.queuedItemMap = make(map[string]*componentQueueItem)

	// 若尚未向运行中任务发出取消，则在本次发出，保证只发一次
	var cancels []context.CancelFunc
	if !p.activeCancelSent {
		p.activeCancelSent = true
		for _, cancel := range p.runningCancels {
			cancels = append(cancels, cancel)
		}
	}
	p.mu.Unlock()

	// 触发未派发排队项的取消
	for _, it := range itemsToCancel {
		e.cancelUndispatchedItem(it, cause)
	}

	// 触发在途运行项取消
	for _, cancel := range cancels {
		cancel()
	}
}

// waitParent 仅观察真实子树归零，ctx 截止不伪归零
func (e *componentEngine) waitParent(ctx context.Context, parentID string) error {
	e.mu.RLock()
	p, ok := e.parents[parentID]
	e.mu.RUnlock()
	if !ok {
		return nil
	}

	p.mu.Lock()
	ch := p.zeroCh
	p.mu.Unlock()

	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// requestStopAll 全局关闸：关闭所有新接纳，标父closed，取消所有未派发项，但不cancel runningCancels（保证优雅停机Grace）
func (e *componentEngine) requestStopAll(cause error) {
	if cause == nil {
		cause = ErrShuttingDown
	}

	var itemsToCancel []*componentQueueItem

	e.mu.Lock()
	if e.stopped {
		e.mu.Unlock()
		return
	}
	e.stopped = true
	e.stopErr = cause

	for _, p := range e.parents {
		p.mu.Lock()
		p.closed = true
		p.closeCause = cause
		for _, it := range p.queuedItemMap {
			itemsToCancel = append(itemsToCancel, it)
		}
		p.queuedItemMap = make(map[string]*componentQueueItem)
		p.mu.Unlock()
	}
	e.mu.Unlock()

	for _, it := range itemsToCancel {
		e.cancelUndispatchedItem(it, cause)
	}
}

func (e *componentEngine) cancelUndispatchedItem(it *componentQueueItem, cause error) {
	it.mu.Lock()
	if it.dispatched || it.cancelled {
		it.mu.Unlock()
		return
	}
	it.cancelled = true
	it.cancelErr = cause
	it.mu.Unlock()

	// 从父级与全局依赖中退还一次占用
	e.mu.RLock()
	p, ok := e.parents[it.parentID]
	e.mu.RUnlock()
	if ok {
		p.dec()
	}
	e.releaseDeps(it.deps)

	// 排队取消真 release 后仅删除对应 liveItems
	e.mu.Lock()
	delete(e.liveItems, it.instanceID)
	e.mu.Unlock()

	// 发布未派发取消结果：Dispatched=false, CancelRequested=true, CancelErr=cause, RunErr=nil
	it.handle.publishResult(RunResult{
		InstanceID:      it.instanceID,
		PluginKey:       it.compKey,
		Outcome:         OutcomeCancelled,
		Dispatched:      false,
		CancelRequested: true,
		CancelErr:       cause,
	})
}

// onSlotReleased 当一个执行槽执行完成后调用，释放活跃数并调度下一个排队项
func (e *componentEngine) onSlotReleased(compKey ctxkey.Key, slot *componentSlotState) {
	var nextItem *componentQueueItem
	var rejectedItems []*componentQueueItem
	var rejectCauses []error

	// 派发的合法性检查+dispatched标记用e.mu读锁覆盖，与全局/父级关闸写锁线性化，避免持slot.mu再取p.mu倒置死锁
	e.mu.RLock()
	slot.mu.Lock()
	slot.activeCount--

	for len(slot.queuedItems) > 0 {
		head := slot.queuedItems[0]
		slot.queuedItems = slot.queuedItems[1:]

		head.mu.Lock()
		if head.cancelled {
			head.mu.Unlock()
			continue
		}

		// 检查全局停机或父级关闭状态
		var shouldReject bool
		var rejectCause error
		if e.stopped {
			shouldReject = true
			rejectCause = e.stopErr
			if rejectCause == nil {
				rejectCause = ErrShuttingDown
			}
		} else if p, ok := e.parents[head.parentID]; ok {
			p.mu.Lock()
			if p.closed {
				shouldReject = true
				rejectCause = p.closeCause
				if rejectCause == nil {
					rejectCause = ErrParentClosing
				}
			}
			p.mu.Unlock()
		} else {
			shouldReject = true
			rejectCause = ErrInvalidActivationParent
		}

		if shouldReject {
			head.mu.Unlock()
			rejectedItems = append(rejectedItems, head)
			rejectCauses = append(rejectCauses, rejectCause)
			continue
		}

		// 合法可派发：标记dispatched
		head.dispatched = true
		head.mu.Unlock()

		slot.activeCount++
		nextItem = head
		break
	}

	var def PluginInfo
	var fac func() RunnablePlugin
	if nextItem != nil {
		def = e.compDefs[nextItem.compKey]
		fac = e.compFacs[nextItem.compKey]
		if p, ok := e.parents[nextItem.parentID]; ok {
			p.mu.Lock()
			delete(p.queuedItemMap, nextItem.instanceID)
			p.mu.Unlock()
		}
	}
	slot.mu.Unlock()
	e.mu.RUnlock()

	// 锁外释放被拒派的项，保持once-release
	for i, item := range rejectedItems {
		e.cancelUndispatchedItem(item, rejectCauses[i])
	}

	if nextItem != nil {
		go e.executePipeline(nextItem, def, fac, slot)
	}
}

// hasAncestorCycle 检查激活链路是否存在祖先定义环
func (e *componentEngine) hasAncestorCycle(parentID string, targetKey ctxkey.Key) bool {
	currID := parentID
	visited := make(map[string]struct{})
	for currID != "" {
		if _, ok := visited[currID]; ok {
			break
		}
		visited[currID] = struct{}{}

		p, ok := e.parents[currID]
		if !ok {
			break
		}
		if p.pluginKey == targetKey {
			return true
		}
		if !p.isComponent {
			// Worker 为根，不再有父级
			break
		}
		currID = p.parentID
	}
	return false
}

// ==============================
// 动态激活核心入口 (Activator 实现)
// ==============================

// Activate 按指定组件键 compKey 动态激活并调度执行一个组件实例，实现 [Activator] 接口。
//
// # 准入与校验契约
//
//   - ctx 必须是携带可信父级身份凭据（[trustedAuthKey]）的非空上下文；若凭据无效或父级已关闭，
//     立即返回 [ErrInvalidActivationParent]；
//   - 若底层 App 处于停止或已关闭状态（[StateStopping]、[StateStopped]），返回 [ErrAppClosed]；
//   - 若 compKey 未在引擎中注册，返回 [ErrComponentNotFound]；
//   - 若在激活调用链中检测到祖先依赖递归环，返回 [ErrActivationCycle]。
//
// # 并发与重入决策
//
//   - 根据注册时声明的 [OverlapPolicy] 决定立即派发执行、进入等待队列或直接拒绝：
//   - [OverlapSkip]：已有实例在执行时立即拒绝，返回 [ErrComponentSkipped]；
//   - [OverlapAllow]：并发数达到限制时拒绝，返回 [ErrConcurrencyLimitExceeded]；
//   - [OverlapQueue]：若存在执行槽立即派发，否则若队列未满则排队，满则返回 [ErrQueueFull]；
//     特别地，若为 Component 嵌套激活且队列繁忙，返回 [ErrNestedQueueWouldBlock] 避免死锁。
//
// 成功接纳后返回可用于取消或等待结果的 [RunHandle]。
func (e *componentEngine) Activate(ctx context.Context, compKey ctxkey.Key, opts ...ActivateOption) (RunHandle, error) {
	// 1. ctx 空安全与接纳前取消检查
	if ctx == nil {
		return nil, ErrInvalidActivationParent
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// 2. 状态检查：必须在 Running 状态下接受新任务
	if e.app != nil {
		state := e.app.State()
		if state != StateRunning {
			if state == StateStopping || state == StateStopped || state == StateIncomplete {
				return nil, ErrAppClosed
			}
			return nil, ErrInvalidActivationParent
		}
	}

	// 3. 校验可信身份：提取 trustedAuthToken，严格校验 appID 匹配且在 parents 中存活且未 closed
	tokenVal := ctx.Value(trustedAuthKey)
	if tokenVal == nil {
		return nil, ErrInvalidActivationParent
	}
	token, ok := tokenVal.(trustedAuthToken)
	if !ok {
		return nil, ErrInvalidActivationParent
	}
	if e.app != nil && token.appID != e.app.appID {
		return nil, ErrInvalidActivationParent
	}

	// 4. 解析可选配置 WithPayload
	cfg := activateConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	// 5. 准入与重入决策（在引擎锁内完成）
	e.mu.Lock()
	if e.stopped {
		e.mu.Unlock()
		return nil, ErrAppClosed
	}

	parentSt, parentFound := e.parents[token.instanceID]
	if !parentFound {
		e.mu.Unlock()
		return nil, ErrInvalidActivationParent
	}

	parentSt.mu.Lock()
	if parentSt.closed {
		parentSt.mu.Unlock()
		e.mu.Unlock()
		return nil, ErrInvalidActivationParent
	}
	parentSt.mu.Unlock()

	// 祖先定义环检测：若链路中有任一祖先匹配当前组件定义，拒绝 ErrActivationCycle
	if e.hasAncestorCycle(token.instanceID, compKey) {
		e.mu.Unlock()
		return nil, ErrActivationCycle
	}

	// 检查组件定义是否存在
	def, defFound := e.compDefs[compKey]
	fac, facFound := e.compFacs[compKey]
	if !defFound || !facFound {
		e.mu.Unlock()
		return nil, fmt.Errorf("%w: %s", ErrComponentNotFound, compKey)
	}

	slot, slotFound := e.slots[compKey]
	if !slotFound {
		slot = &componentSlotState{
			policy:       def.Overlap,
			wakeWorkerCh: make(chan struct{}, 1),
		}
		e.slots[compKey] = slot
	}

	slot.mu.Lock()

	// 计算策略参数
	policyKind := slot.policy.Kind
	if policyKind == 0 {
		policyKind = OverlapSkip
	}

	var shouldDispatchNow bool
	var shouldQueue bool

	switch policyKind {
	case OverlapSkip:
		if slot.activeCount >= 1 {
			slot.mu.Unlock()
			e.mu.Unlock()
			return nil, ErrComponentSkipped
		}
		slot.activeCount++
		shouldDispatchNow = true

	case OverlapAllow:
		maxConcurrency := slot.policy.MaxConcurrency
		if maxConcurrency <= 0 {
			maxConcurrency = 1
		}
		if slot.activeCount >= maxConcurrency {
			slot.mu.Unlock()
			e.mu.Unlock()
			return nil, ErrConcurrencyLimitExceeded
		}
		slot.activeCount++
		shouldDispatchNow = true

	case OverlapQueue:
		// 清理已取消或已终结的排队项，保持真实等待项 FIFO
		filtered := slot.queuedItems[:0]
		for _, it := range slot.queuedItems {
			it.mu.Lock()
			cancelled := it.cancelled || it.dispatched
			it.mu.Unlock()
			if !cancelled {
				filtered = append(filtered, it)
			}
		}
		slot.queuedItems = filtered

		// 嵌套激活规则：如果是 Component 父级激活，且目标 Queue 繁忙（有执行槽或有等待项），立即拒绝
		if parentSt.isComponent && (slot.activeCount > 0 || len(slot.queuedItems) > 0) {
			slot.mu.Unlock()
			e.mu.Unlock()
			return nil, ErrNestedQueueWouldBlock
		}

		if slot.activeCount == 0 && len(slot.queuedItems) == 0 {
			// 空闲且无排队：立即派发
			slot.activeCount++
			shouldDispatchNow = true
		} else {
			// 需要排队：检查等待队列容量 MaxQueue（仅统计真实排队项）
			maxQueue := slot.policy.MaxQueue
			if maxQueue <= 0 {
				maxQueue = 1
			}
			if len(slot.queuedItems) >= maxQueue {
				slot.mu.Unlock()
				e.mu.Unlock()
				return nil, ErrQueueFull
			}
			shouldQueue = true
		}
	}

	// 接纳即分配唯一 instanceID，构建句柄，登记父子树计数，记录依赖占用
	instID := fmt.Sprintf("%s_%d", compKey, time.Now().UnixNano())
	handle := newRunHandle(instID, compKey)

	initStage := "queued"
	if shouldDispatchNow {
		initStage = "factory"
	}

	item := &componentQueueItem{
		instanceID:  instID,
		parentID:    parentSt.instanceID,
		parentKey:   parentSt.pluginKey,
		compKey:     compKey,
		handle:      handle,
		payload:     cfg.payload,
		deps:        def.Requires,
		stage:       initStage,
		dispatched:  shouldDispatchNow,
		executeDone: make(chan struct{}),
	}

	// 准入时在 e.mu 下登记进 liveItems 私有账本
	e.liveItems[instID] = item

	// 增加父子树计数
	parentSt.inc()
	// 记录真实资源依赖占用
	e.retainDeps(def.Requires)

	if shouldQueue {
		slot.queuedItems = append(slot.queuedItems, item)
		parentSt.mu.Lock()
		parentSt.queuedItemMap[instID] = item
		parentSt.mu.Unlock()
	}

	slot.mu.Unlock()
	e.mu.Unlock()

	// 立即派发执行（在引擎与槽锁外）
	if shouldDispatchNow {
		go e.executePipeline(item, def, fac, slot)
	}

	return handle, nil
}
