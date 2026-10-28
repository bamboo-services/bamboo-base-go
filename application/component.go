package application

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

// componentInstance 是 Component 运行实例在引擎内部的记录
type componentInstance struct {
	id           string
	compKey      ctxkey.Key
	parentID     string
	parentKey    ctxkey.Key
	rawPlugin    RunnablePlugin
	handle       *runHandleImpl
	stopDone     chan struct{}
	stopped      atomic.Bool
	stopPanicked atomic.Bool
	stopErr      error
	retainedDeps []ctxkey.Key
}

func newComponentInstance(id string, compKey ctxkey.Key, parentID string, parentKey ctxkey.Key, deps []ctxkey.Key, handle *runHandleImpl) *componentInstance {
	return &componentInstance{
		id:           id,
		compKey:      compKey,
		parentID:     parentID,
		parentKey:    parentKey,
		handle:       handle,
		stopDone:     make(chan struct{}),
		retainedDeps: deps,
	}
}

// executePipeline 封装单一已取得执行槽的 Component 实例完整生命周期
// 阶段：factory -> Start -> Run -> wait children -> Stop -> publishResult
func (e *componentEngine) executePipeline(
	item *componentQueueItem,
	def PluginInfo,
	fac func() RunnablePlugin,
	slot *componentSlotState,
) {
	var slotReleased bool
	var retainSlot bool
	releaseSlotOnce := func() {
		if !slotReleased {
			slotReleased = true
			e.onSlotReleased(item.compKey, slot)
		}
	}
	defer func() {
		if !retainSlot {
			releaseSlotOnce()
		}
	}()

	instID := item.instanceID
	compKey := item.compKey
	parentID := item.parentID
	parentKey := item.parentKey

	res := RunResult{
		InstanceID: instID,
		PluginKey:  compKey,
		Dispatched: true,
	}

	// 0. factory 阶段标记
	e.setTaskStage(item, "factory")

	// 1. 调用工厂函数实例化（不在引擎锁内）
	var rawInst RunnablePlugin
	var facErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				facErr = &HookPanicError{
					PluginKey:  compKey,
					InstanceID: instID,
					Stage:      "factory",
					Value:      r,
					Stack:      debug.Stack(),
				}
			}
		}()
		if fac != nil {
			rawInst = fac()
		}
	}()

	if facErr != nil {
		res.Outcome = OutcomeFailed
		res.FactoryErr = facErr
		e.finishTask(item, res, nil, false)
		e.forgetTask(item)
		return
	}
	if rawInst == nil || isTypedNil(rawInst) {
		res.Outcome = OutcomeFailed
		res.FactoryErr = fmt.Errorf("%w: plugin %s", ErrNilPlugin, compKey)
		e.finishTask(item, res, nil, false)
		e.forgetTask(item)
		return
	}

	compInst := newComponentInstance(instID, compKey, parentID, parentKey, item.deps, item.handle)
	compInst.rawPlugin = rawInst

	// Start claim 与 e.stopped / 父关闸在 e.mu + parent 锁边界检查
	var preStartCancelled bool
	var preStartCancelErr error

	e.mu.Lock()
	if e.stopped {
		preStartCancelled = true
		preStartCancelErr = e.stopErr
		if preStartCancelErr == nil {
			preStartCancelErr = ErrShuttingDown
		}
	} else if p, ok := e.parents[parentID]; ok {
		p.mu.Lock()
		if p.closed {
			preStartCancelled = true
			preStartCancelErr = p.closeCause
			if preStartCancelErr == nil {
				preStartCancelErr = ErrParentClosing
			}
		}
		p.mu.Unlock()
	}

	if !preStartCancelled {
		compParentState := newParentState(instID, compKey, parentID, true)
		e.parents[instID] = compParentState
	}
	e.mu.Unlock()

	if preStartCancelled {
		item.mu.Lock()
		item.cancelled = true
		item.cancelErr = preStartCancelErr
		item.mu.Unlock()

		res.Outcome = OutcomeCancelled
		res.CancelRequested = true
		res.CancelErr = preStartCancelErr
		e.finishTask(item, res, nil, false)
		e.forgetTask(item)
		return
	}

	defer func() {
		// 任务结束后从父级表中注销该 Component
		e.mu.Lock()
		delete(e.parents, instID)
		e.mu.Unlock()
	}()

	// 2. 构造 StartCtx（受限视图 + 独立 StartTimeout）
	startTimeout := 15 * time.Second
	if e.app != nil && e.app.cfg.StartTimeout > 0 {
		startTimeout = e.app.cfg.StartTimeout
	}
	startCtx, startCancel := context.WithTimeout(context.Background(), startTimeout)

	allowedMap := make(map[ctxkey.Key]struct{}, len(def.Requires))
	for _, r := range def.Requires {
		allowedMap[r] = struct{}{}
	}
	var scopeView *startScopeView
	if e.app != nil {
		scopeView = &startScopeView{
			container:   e.app.container,
			ownerKey:    compKey,
			allowedKeys: allowedMap,
		}
	}

	startCtxWithVal := context.WithValue(startCtx, scopeViewCtxKey, scopeView)
	if e.app != nil {
		startCtxWithVal = context.WithValue(startCtxWithVal, containerCtxKey, e.app.container)
		startCtxWithVal = context.WithValue(startCtxWithVal, forceCtxKey, e.app.forceCtx)
	}
	startCtxWithVal = context.WithValue(startCtxWithVal, metaCtxKey, InstanceMeta{
		InstanceID:       instID,
		PluginKey:        compKey,
		ParentKey:        parentKey,
		ParentInstanceID: parentID,
		TraceID:          fmt.Sprintf("trace_%s", instID),
	})

	e.setTaskStage(item, "start")

	var startErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				startErr = &HookPanicError{
					PluginKey:  compKey,
					InstanceID: instID,
					Stage:      "start",
					Value:      r,
					Stack:      debug.Stack(),
				}
			}
		}()
		startErr = rawInst.Start(startCtxWithVal)
	}()

	startCtxExpiredErr := startCtx.Err()
	startCancel()

	// 若迟到 nil 但 ctx 已 deadline 则 res.StartErr = deadline
	if startErr == nil && startCtxExpiredErr != nil {
		startErr = startCtxExpiredErr
	}

	if startErr != nil {
		res.StartErr = startErr
		// Start 失败：先为当前有效实例调用唯一一次 Stop，再以 OutcomeFailed 收尾
		e.setTaskStage(item, "stop")
		stopErr := e.safeCallComponentStop(compInst, def)
		res.StopErr = stopErr

		if compInst.stopPanicked.Load() {
			retainSlot = true
			e.setTaskStage(item, "stop-panic")
			e.handleStopPanic(compInst, def, res.StopErr)
			return
		}

		res.Outcome = OutcomeFailed
		e.finishTask(item, res, compInst, true)
		e.forgetTask(item)
		return
	}

	// 3. Start 真退后在 e.mu + 父锁边界检查关闸/父关闭并 claim Run 及 runningCancels
	runCtx, runCancel := context.WithCancel(context.Background())
	item.mu.Lock()
	item.runCancel = runCancel
	item.mu.Unlock()

	var cancelBeforeRun bool
	var cancelBeforeRunCause error

	e.mu.Lock()
	if e.stopped {
		cancelBeforeRun = true
		cancelBeforeRunCause = e.stopErr
		if cancelBeforeRunCause == nil {
			cancelBeforeRunCause = ErrShuttingDown
		}
	} else if parentSt, ok := e.parents[parentID]; ok {
		parentSt.mu.Lock()
		if parentSt.closed {
			cancelBeforeRun = true
			cancelBeforeRunCause = parentSt.closeCause
			if cancelBeforeRunCause == nil {
				cancelBeforeRunCause = ErrParentClosing
			}
		} else {
			parentSt.runningCancels[instID] = runCancel
		}
		parentSt.mu.Unlock()
	}
	e.mu.Unlock()

	if cancelBeforeRun {
		item.mu.Lock()
		item.cancelled = true
		item.cancelErr = cancelBeforeRunCause
		item.mu.Unlock()

		res.CancelRequested = true
		res.CancelErr = cancelBeforeRunCause

		// 取消路径真实 Stop 后 Cancelled 且保 CancelErr，绝不 Run
		e.setTaskStage(item, "stop")
		stopErr := e.safeCallComponentStop(compInst, def)
		res.StopErr = stopErr

		if compInst.stopPanicked.Load() {
			retainSlot = true
			e.setTaskStage(item, "stop-panic")
			e.handleStopPanic(compInst, def, res.StopErr)
			return
		}

		res.Outcome = OutcomeCancelled
		e.finishTask(item, res, compInst, true)
		e.forgetTask(item)
		return
	}

	var lifetimeCtx = runCtx
	if e.app != nil {
		lifetimeCtx = context.WithValue(lifetimeCtx, containerCtxKey, e.app.container)
		lifetimeCtx = context.WithValue(lifetimeCtx, forceCtxKey, e.app.forceCtx)
	}
	lifetimeCtx = context.WithValue(lifetimeCtx, metaCtxKey, InstanceMeta{
		InstanceID:       instID,
		PluginKey:        compKey,
		ParentKey:        parentKey,
		ParentInstanceID: parentID,
		TraceID:          fmt.Sprintf("trace_%s", instID),
	})
	lifetimeCtx = context.WithValue(lifetimeCtx, payloadCtxKey, item.payload)
	lifetimeCtx = context.WithValue(lifetimeCtx, trustedAuthKey, trustedAuthToken{
		appID:      e.getAppID(),
		instanceID: instID,
		pluginKey:  compKey,
	})

	// 4. 调用 Run(lifetimeCtx)
	e.setTaskStage(item, "run")
	var runErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				runErr = &HookPanicError{
					PluginKey:  compKey,
					InstanceID: instID,
					Stage:      "run",
					Value:      r,
					Stack:      debug.Stack(),
				}
			}
		}()
		runErr = rawInst.Run(lifetimeCtx)
	}()

	res.RunErr = runErr

	// Run 结束后：从父级 runningCancels 移除
	e.mu.Lock()
	if parentSt, ok := e.parents[parentID]; ok {
		parentSt.mu.Lock()
		delete(parentSt.runningCancels, instID)
		parentSt.mu.Unlock()
	}
	e.mu.Unlock()

	// 5. 先等待当前 Component 的子树全部归零，再执行 Stop
	e.closeParent(instID, ErrParentClosing)
	_ = e.waitParent(context.Background(), instID)

	// 6. 执行 Stop 真实收尾
	e.setTaskStage(item, "stop")
	stopErr := e.safeCallComponentStop(compInst, def)
	res.StopErr = stopErr

	// 7. 判定最终归宿 Outcome
	item.mu.Lock()
	cancelRequested := item.cancelled
	if cancelRequested && item.cancelErr != nil {
		res.CancelErr = item.cancelErr
	}
	item.mu.Unlock()
	res.CancelRequested = cancelRequested

	if compInst.stopPanicked.Load() {
		// Stop panic: 无法确认清理完成，不能发布正常终态，不关闭句柄！
		// 依赖保持不释放，触发 App 异步关机
		retainSlot = true
		e.setTaskStage(item, "stop-panic")
		e.handleStopPanic(compInst, def, res.StopErr)
		return
	}

	if res.StartErr != nil || res.RunErr != nil || res.StopErr != nil {
		if cancelRequested && isContextCanceledError(res.RunErr) && res.StopErr == nil {
			res.Outcome = OutcomeCancelled
		} else {
			res.Outcome = OutcomeFailed
		}
	} else {
		res.Outcome = OutcomeSuccess
	}

	e.finishTask(item, res, compInst, true)
	e.forgetTask(item)
}

func isContextCanceledError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func (e *componentEngine) safeCallComponentStop(ci *componentInstance, def PluginInfo) (retErr error) {
	defer func() {
		close(ci.stopDone)
	}()

	defer func() {
		if r := recover(); r != nil {
			ci.stopPanicked.Store(true)
			panicErr := &HookPanicError{
				PluginKey:  ci.compKey,
				InstanceID: ci.id,
				Stage:      "stop",
				Value:      r,
				Stack:      debug.Stack(),
			}
			ci.stopErr = panicErr
			retErr = panicErr
		}
	}()

	cleanupLimit := 10 * time.Second
	if e.app != nil && e.app.cfg.CleanupTimeout > 0 {
		cleanupLimit = e.app.cfg.CleanupTimeout
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), cleanupLimit)
	defer cancel()

	stopErr := ci.rawPlugin.Stop(stopCtx)
	ci.stopErr = stopErr
	if stopErr != nil {
		retErr = stopErr
	}

	ci.stopped.Store(true)
	return retErr
}

func (e *componentEngine) finishTask(item *componentQueueItem, res RunResult, ci *componentInstance, hasCleaned bool) {
	// 先从活动任务中移除，避免释放依赖唤醒协调器后 snapshotLiveTasks 读到假活动窗口
	e.forgetTask(item)

	// 释放父级占用与依赖占用
	e.mu.RLock()
	parentSt, hasParent := e.parents[item.parentID]
	e.mu.RUnlock()
	if hasParent {
		parentSt.dec()
	}
	e.releaseDeps(item.deps)

	// 发布终态给句柄
	item.handle.publishResult(res)
}

func (e *componentEngine) handleStopPanic(ci *componentInstance, def PluginInfo, stopErr error) {
	// 1. 保留依赖占用，不调用 e.releaseDeps(ci.retainedDeps)
	// 2. 将容器对应条目（若有）标记为不可用
	if e.app != nil && e.app.container != nil {
		e.app.container.setOwnerState(ci.compKey, EntryUnavailable)
	}

	// 3. 记录 StuckInstance 到协调器
	if e.app != nil && e.app.coordinator != nil {
		e.app.coordinator.recordStuck(StuckInstance{
			PluginKey:    ci.compKey,
			InstanceID:   ci.id,
			Stage:        "stop-panic",
			RetainedDeps: ci.retainedDeps,
			Err:          stopErr,
		})
	}

	// 4. 触发 App 异步失败关机
	if e.app != nil {
		go func() {
			_ = e.app.Shutdown()
		}()
	}
}

func (e *componentEngine) getAppID() string {
	if e.app != nil {
		return e.app.appID
	}
	return "test_app"
}
