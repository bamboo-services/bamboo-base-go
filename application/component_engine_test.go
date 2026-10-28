package application

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

type testComponentFixture struct {
	startFunc func(ctx context.Context) error
	runFunc   func(ctx context.Context) error
	stopFunc  func(ctx context.Context) error
}

func (f *testComponentFixture) Start(ctx context.Context) error {
	if f.startFunc != nil {
		return f.startFunc(ctx)
	}
	return nil
}

func (f *testComponentFixture) Run(ctx context.Context) error {
	if f.runFunc != nil {
		return f.runFunc(ctx)
	}
	return nil
}

func (f *testComponentFixture) Stop(ctx context.Context) error {
	if f.stopFunc != nil {
		return f.stopFunc(ctx)
	}
	return nil
}

// 辅助函数：构造带有合法 Worker 可信身份的 Context
func newWorkerTestContext(app *App, instanceID string, workerKey ctxkey.Key) context.Context {
	ctx := context.Background()
	if app != nil {
		ctx = context.WithValue(ctx, containerCtxKey, app.container)
		ctx = context.WithValue(ctx, forceCtxKey, app.forceCtx)
	}
	ctx = context.WithValue(ctx, metaCtxKey, InstanceMeta{
		InstanceID: instanceID,
		PluginKey:  workerKey,
		TraceID:    fmt.Sprintf("trace_%s", instanceID),
	})
	appID := "test_app"
	if app != nil {
		appID = app.appID
	}
	ctx = context.WithValue(ctx, trustedAuthKey, trustedAuthToken{
		appID:      appID,
		instanceID: instanceID,
		pluginKey:  workerKey,
	})
	return ctx
}

// 1. Skip 策略：上一实例 Run 结束但 Stop 未真退时，新激活必须拒绝 ErrComponentSkipped，且无句柄；
// Stop 真正返回后，新激活成功被接纳
func TestComponentEngine_Skip_BarrierStopsUntilStopReturned(t *testing.T) {
	app := New()
	// 手动将 app 状态标记为 StateRunning，支持引擎测试
	app.stateMu.Lock()
	app.state = StateRunning
	app.stateMu.Unlock()

	engine := newComponentEngine(app)
	workerKey := ctxkey.Key("test_worker")
	compKey := ctxkey.Key("test_comp_skip")
	workerID := "worker_inst_1"
	engine.registerWorker(workerID, workerKey)

	runEntered1 := make(chan struct{})
	releaseRun1 := make(chan struct{})
	stopEntered1 := make(chan struct{})
	releaseStop1 := make(chan struct{})

	runEntered4 := make(chan struct{})
	releaseRun4 := make(chan struct{})
	stopEntered4 := make(chan struct{})
	releaseStop4 := make(chan struct{})

	var instanceCount int32
	engine.RegisterComponentDef(PluginInfo{
		Key:     compKey,
		Overlap: OverlapPolicy{Kind: OverlapSkip},
	}, func() RunnablePlugin {
		idx := atomic.AddInt32(&instanceCount, 1)
		if idx == 1 {
			return &testComponentFixture{
				runFunc: func(ctx context.Context) error {
					close(runEntered1)
					<-releaseRun1
					return nil
				},
				stopFunc: func(ctx context.Context) error {
					close(stopEntered1)
					<-releaseStop1
					return nil
				},
			}
		}
		return &testComponentFixture{
			runFunc: func(ctx context.Context) error {
				close(runEntered4)
				<-releaseRun4
				return nil
			},
			stopFunc: func(ctx context.Context) error {
				close(stopEntered4)
				<-releaseStop4
				return nil
			},
		}
	})

	workerCtx := newWorkerTestContext(app, workerID, workerKey)

	// 1. 首次激活
	h1, err1 := engine.Activate(workerCtx, compKey)
	if err1 != nil || h1 == nil {
		t.Fatalf("first activate failed: %v", err1)
	}

	<-runEntered1

	// 2. 在 Run 期间激活：拒绝并返回 ErrComponentSkipped，无句柄
	h2, err2 := engine.Activate(workerCtx, compKey)
	if !errors.Is(err2, ErrComponentSkipped) || h2 != nil {
		t.Fatalf("expected ErrComponentSkipped while running, got h2=%v, err=%v", h2, err2)
	}

	// 3. 释放 Run，进入 Stop
	close(releaseRun1)
	<-stopEntered1

	// 4. 在 Stop 期间（尚未真退）激活：必须依然被拒绝 ErrComponentSkipped
	h3, err3 := engine.Activate(workerCtx, compKey)
	if !errors.Is(err3, ErrComponentSkipped) || h3 != nil {
		t.Fatalf("expected ErrComponentSkipped while stopping, got h3=%v, err=%v", h3, err3)
	}

	// 5. 释放 Stop，等待真退完成
	close(releaseStop1)
	res1, waitErr1 := h1.Wait(context.Background())
	if waitErr1 != nil || res1.Outcome != OutcomeSuccess {
		t.Fatalf("expected h1 success, got outcome=%v, waitErr=%v", res1.Outcome, waitErr1)
	}

	// 6. Stop 真退后再次激活：成功接纳
	h4, err4 := engine.Activate(workerCtx, compKey)
	if err4 != nil || h4 == nil {
		t.Fatalf("expected activate after stop returned to succeed, got %v, %v", h4, err4)
	}
	<-runEntered4
	close(releaseRun4)
	<-stopEntered4
	close(releaseStop4)

	res4, waitErr4 := h4.Wait(context.Background())
	if waitErr4 != nil || res4.Outcome != OutcomeSuccess {
		t.Fatalf("expected h4 success, got %v, %v", res4.Outcome, waitErr4)
	}
}

// 2. Allow 策略：并发上限有界控制，达到上限后拒绝 ErrConcurrencyLimitExceeded
func TestComponentEngine_Allow_ConcurrencyLimit(t *testing.T) {
	app := New()
	app.stateMu.Lock()
	app.state = StateRunning
	app.stateMu.Unlock()

	engine := newComponentEngine(app)
	workerKey := ctxkey.Key("test_worker_allow")
	compKey := ctxkey.Key("test_comp_allow")
	workerID := "worker_allow_1"
	engine.registerWorker(workerID, workerKey)

	maxConc := 2
	runEntered := make(chan struct{}, maxConc)
	releaseRun := make(chan struct{})

	engine.RegisterComponentDef(PluginInfo{
		Key: compKey,
		Overlap: OverlapPolicy{
			Kind:           OverlapAllow,
			MaxConcurrency: maxConc,
		},
	}, func() RunnablePlugin {
		return &testComponentFixture{
			runFunc: func(ctx context.Context) error {
				runEntered <- struct{}{}
				<-releaseRun
				return nil
			},
		}
	})

	workerCtx := newWorkerTestContext(app, workerID, workerKey)

	// 激活 2 个
	h1, err1 := engine.Activate(workerCtx, compKey)
	h2, err2 := engine.Activate(workerCtx, compKey)
	if err1 != nil || h1 == nil || err2 != nil || h2 == nil {
		t.Fatalf("expected h1 and h2 success")
	}

	<-runEntered
	<-runEntered

	// 第 3 个超限拒绝
	h3, err3 := engine.Activate(workerCtx, compKey)
	if !errors.Is(err3, ErrConcurrencyLimitExceeded) || h3 != nil {
		t.Fatalf("expected ErrConcurrencyLimitExceeded and nil handle, got h3=%v, err=%v", h3, err3)
	}

	close(releaseRun)
	_, _ = h1.Wait(context.Background())
	_, _ = h2.Wait(context.Background())

	// 释放后再次激活成功
	h4, err4 := engine.Activate(workerCtx, compKey)
	if err4 != nil || h4 == nil {
		t.Fatalf("expected h4 success after release, got %v", err4)
	}
	_, _ = h4.Wait(context.Background())
}

// 3. Queue 策略：跨父级共享 FIFO，MaxQueue 仅数等待项，接纳即交付句柄，队满 ErrQueueFull，
// 原激活 ctx 取消不撤销已接纳项，Wait 短超时不杀任务
func TestComponentEngine_Queue_SharedFIFO_MaxQueue_ContextIndependence(t *testing.T) {
	app := New()
	app.stateMu.Lock()
	app.state = StateRunning
	app.stateMu.Unlock()

	engine := newComponentEngine(app)
	workerAKey := ctxkey.Key("worker_qa")
	workerBKey := ctxkey.Key("worker_qb")
	compKey := ctxkey.Key("comp_shared_queue")
	workerAID := "worker_qa_1"
	workerBID := "worker_qb_1"
	engine.registerWorker(workerAID, workerAKey)
	engine.registerWorker(workerBID, workerBKey)

	var factoryCalls int32
	var dispatchMu sync.Mutex
	var dispatchOrder []string

	releaseRun := make(chan struct{})
	taskStarted := make(chan string, 10)

	engine.RegisterComponentDef(PluginInfo{
		Key: compKey,
		Overlap: OverlapPolicy{
			Kind:     OverlapQueue,
			MaxQueue: 2, // 1 槽 + 2 等待
		},
	}, func() RunnablePlugin {
		call := atomic.AddInt32(&factoryCalls, 1)
		tag := fmt.Sprintf("task_%d", call)
		return &testComponentFixture{
			runFunc: func(ctx context.Context) error {
				dispatchMu.Lock()
				dispatchOrder = append(dispatchOrder, tag)
				dispatchMu.Unlock()
				taskStarted <- tag
				<-releaseRun
				return nil
			},
		}
	})

	workerACtx := newWorkerTestContext(app, workerAID, workerAKey)
	workerBCtx := newWorkerTestContext(app, workerBID, workerBKey)

	// 1. Worker A 激发 T1，占用执行槽
	h1, err1 := engine.Activate(workerACtx, compKey)
	if err1 != nil || h1 == nil {
		t.Fatalf("h1 activate failed: %v", err1)
	}
	tag1 := <-taskStarted
	if tag1 != "task_1" {
		t.Fatalf("expected task_1 started, got %s", tag1)
	}

	// 2. Worker B 携带短超时 ctx 激发 T2（排队项 1/2）
	shortCtx, shortCancel := context.WithTimeout(workerBCtx, 20*time.Millisecond)
	h2, err2 := engine.Activate(shortCtx, compKey)
	if err2 != nil || h2 == nil {
		t.Fatalf("h2 activate failed: %v", err2)
	}
	// 原请求 ctx 取消
	shortCancel()

	// 排队中，此时工厂调用次数仍应为 1
	if calls := atomic.LoadInt32(&factoryCalls); calls != 1 {
		t.Fatalf("expected factory calls=1 while T2 queued, got %d", calls)
	}

	// 3. Worker A 激发 T3（排队项 2/2，队满）
	h3, err3 := engine.Activate(workerACtx, compKey)
	if err3 != nil || h3 == nil {
		t.Fatalf("h3 activate failed: %v", err3)
	}

	// 4. 再次激活：ErrQueueFull
	h4, err4 := engine.Activate(workerBCtx, compKey)
	if !errors.Is(err4, ErrQueueFull) || h4 != nil {
		t.Fatalf("expected ErrQueueFull and nil handle, got h4=%v, err=%v", h4, err4)
	}

	// 5. 对 h2 使用短超时 Wait，验证超时仅观察，不杀任务
	waitTimeoutCtx, waitTimeoutCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer waitTimeoutCancel()
	_, waitErr := h2.Wait(waitTimeoutCtx)
	if !errors.Is(waitErr, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded from wait, got %v", waitErr)
	}

	// 6. 释放 T1，T2 应当按 FIFO 被派发
	releaseRun <- struct{}{}
	tag2 := <-taskStarted
	if tag2 != "task_2" {
		t.Fatalf("expected task_2 dispatched, got %s", tag2)
	}

	// 释放 T2
	releaseRun <- struct{}{}
	res2, errRes2 := h2.Wait(context.Background())
	if errRes2 != nil || res2.Outcome != OutcomeSuccess {
		t.Fatalf("expected T2 success, got outcome=%v, err=%v", res2.Outcome, errRes2)
	}

	// 释放 T3
	tag3 := <-taskStarted
	if tag3 != "task_3" {
		t.Fatalf("expected task_3 dispatched, got %s", tag3)
	}
	releaseRun <- struct{}{}
	res3, _ := h3.Wait(context.Background())
	if res3.Outcome != OutcomeSuccess {
		t.Fatalf("expected T3 outcome success, got %v", res3.Outcome)
	}
}

// 4. 未派发项在父级关闭或 App 停止时取消：Dispatched=false, CancelRequested=true, CancelErr 匹配 context.Canceled, RunErr=nil
func TestComponentEngine_UndispatchedCancelledOnParentClose(t *testing.T) {
	app := New()
	app.stateMu.Lock()
	app.state = StateRunning
	app.stateMu.Unlock()

	engine := newComponentEngine(app)
	workerKey := ctxkey.Key("worker_parent_close")
	compKey := ctxkey.Key("comp_parent_close")
	workerID := "worker_parent_close_1"
	depKey := ctxkey.Key("dep_database")

	engine.registerWorker(workerID, workerKey)

	var factoryCalls int32
	blockFirst := make(chan struct{})
	firstStarted := make(chan struct{})

	engine.RegisterComponentDef(PluginInfo{
		Key:      compKey,
		Requires: []ctxkey.Key{depKey},
		Overlap: OverlapPolicy{
			Kind:     OverlapQueue,
			MaxQueue: 5,
		},
	}, func() RunnablePlugin {
		atomic.AddInt32(&factoryCalls, 1)
		return &testComponentFixture{
			runFunc: func(ctx context.Context) error {
				close(firstStarted)
				<-blockFirst
				return nil
			},
		}
	})

	workerCtx := newWorkerTestContext(app, workerID, workerKey)

	// 1. T1 占住执行槽
	h1, err1 := engine.Activate(workerCtx, compKey)
	if err1 != nil || h1 == nil {
		t.Fatalf("h1 activate failed: %v", err1)
	}
	<-firstStarted

	// 2. T2 排队中
	h2, err2 := engine.Activate(workerCtx, compKey)
	if err2 != nil || h2 == nil {
		t.Fatalf("h2 activate failed: %v", err2)
	}

	// 验证依赖占用：T1 与 T2 均占用 depKey，引用计数应为 2
	depsMap := engine.GetRetainedDependencies()
	if depsMap[depKey] != 2 {
		t.Fatalf("expected depKey count 2, got %d", depsMap[depKey])
	}

	// 3. 关闭 Worker 父级
	engine.closeParent(workerID, ErrParentClosing)

	// 释放 T1
	close(blockFirst)
	_, _ = h1.Wait(context.Background())

	// 检查未派发项 h2 的结果
	res2, waitErr2 := h2.Wait(context.Background())
	if res2.Dispatched != false {
		t.Errorf("expected Dispatched=false, got true")
	}
	if !res2.CancelRequested {
		t.Errorf("expected CancelRequested=true")
	}
	if res2.RunErr != nil {
		t.Errorf("expected RunErr=nil, got %v", res2.RunErr)
	}
	if res2.Outcome != OutcomeCancelled {
		t.Errorf("expected OutcomeCancelled, got %v", res2.Outcome)
	}
	if !errors.Is(res2.CancelErr, context.Canceled) {
		t.Errorf("expected CancelErr to match context.Canceled, got %v", res2.CancelErr)
	}
	if !errors.Is(waitErr2, context.Canceled) {
		t.Errorf("expected Wait error to match context.Canceled, got %v", waitErr2)
	}

	// 未派发项绝不调用工厂：工厂调用次数必须仅为 1（T1）
	if calls := atomic.LoadInt32(&factoryCalls); calls != 1 {
		t.Fatalf("expected factory calls=1, got %d", calls)
	}

	// 占用全部退还：depKey 计数归零
	depsMapAfter := engine.GetRetainedDependencies()
	if count := depsMapAfter[depKey]; count != 0 {
		t.Fatalf("expected depKey count 0 after all done, got %d", count)
	}
}

// 5. Component 嵌套激活目标 Queue 忙立即 ErrNestedQueueWouldBlock 不建句柄；祖先环拒绝 ErrActivationCycle
func TestComponentEngine_NestedQueue_WouldBlock_And_Cycle(t *testing.T) {
	app := New()
	app.stateMu.Lock()
	app.state = StateRunning
	app.stateMu.Unlock()

	engine := newComponentEngine(app)
	workerKey := ctxkey.Key("worker_nested")
	compParentKey := ctxkey.Key("comp_parent")
	compTargetKey := ctxkey.Key("comp_target")
	workerID := "worker_nested_1"
	engine.registerWorker(workerID, workerKey)

	blockTargetRun := make(chan struct{})
	targetRunStarted := make(chan struct{})

	engine.RegisterComponentDef(PluginInfo{
		Key: compTargetKey,
		Overlap: OverlapPolicy{
			Kind:     OverlapQueue,
			MaxQueue: 2,
		},
	}, func() RunnablePlugin {
		return &testComponentFixture{
			runFunc: func(ctx context.Context) error {
				select {
				case targetRunStarted <- struct{}{}:
				default:
				}
				<-blockTargetRun
				return nil
			},
		}
	})

	testDone := make(chan struct{})

	engine.RegisterComponentDef(PluginInfo{
		Key: compParentKey,
		Overlap: OverlapPolicy{
			Kind:           OverlapAllow,
			MaxConcurrency: 2,
		},
	}, func() RunnablePlugin {
		return &testComponentFixture{
			runFunc: func(compCtx context.Context) error {
				// 1. 尝试激活祖先环：父自身为 compParentKey，目标亦为 compParentKey
				hCycle, errCycle := engine.Activate(compCtx, compParentKey)
				if !errors.Is(errCycle, ErrActivationCycle) || hCycle != nil {
					t.Errorf("expected ErrActivationCycle and nil handle, got h=%v, err=%v", hCycle, errCycle)
				}

				// 2. 目标 compTargetKey 此时正处于忙碌
				hNested, errNested := engine.Activate(compCtx, compTargetKey)
				if !errors.Is(errNested, ErrNestedQueueWouldBlock) || hNested != nil {
					t.Errorf("expected ErrNestedQueueWouldBlock and nil handle, got h=%v, err=%v", hNested, errNested)
				}

				close(testDone)
				return nil
			},
		}
	})

	workerCtx := newWorkerTestContext(app, workerID, workerKey)

	// Worker 先激活 compTargetKey，占住执行槽
	hTarget, errTarget := engine.Activate(workerCtx, compTargetKey)
	if errTarget != nil || hTarget == nil {
		t.Fatalf("target activate failed: %v", errTarget)
	}
	<-targetRunStarted

	// Worker 激活 compParentKey
	hParent, errParent := engine.Activate(workerCtx, compParentKey)
	if errParent != nil || hParent == nil {
		t.Fatalf("parent activate failed: %v", errParent)
	}

	select {
	case <-testDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for nested queue test")
	}

	close(blockTargetRun)
	_, _ = hTarget.Wait(context.Background())
	_, _ = hParent.Wait(context.Background())
}

// 6. 身份防伪与状态防御：未登记身份、跨 App、已关闭父级、App 停止拒绝激活
func TestComponentEngine_IdentityAndStateDefense(t *testing.T) {
	app1 := New()
	app1.stateMu.Lock()
	app1.state = StateRunning
	app1.stateMu.Unlock()

	engine := newComponentEngine(app1)
	compKey := ctxkey.Key("comp_defense")
	engine.RegisterComponentDef(PluginInfo{
		Key: compKey,
	}, func() RunnablePlugin {
		return &testComponentFixture{}
	})

	// 1. 裸 Background Context（无任何凭据）
	h1, err1 := engine.Activate(context.Background(), compKey)
	if !errors.Is(err1, ErrInvalidActivationParent) || h1 != nil {
		t.Fatalf("expected ErrInvalidActivationParent for background ctx, got h1=%v, err=%v", h1, err1)
	}

	// 2. 伪造 token（跨 App）
	fakeCtx := context.WithValue(context.Background(), trustedAuthKey, trustedAuthToken{
		appID:      "other_app_id",
		instanceID: "fake_inst",
		pluginKey:  "fake_key",
	})
	h2, err2 := engine.Activate(fakeCtx, compKey)
	if !errors.Is(err2, ErrInvalidActivationParent) || h2 != nil {
		t.Fatalf("expected ErrInvalidActivationParent for cross-app token, got h2=%v, err=%v", h2, err2)
	}

	// 3. 已关闭的父级
	workerKey := ctxkey.Key("worker_defense")
	workerID := "worker_defense_1"
	engine.registerWorker(workerID, workerKey)
	engine.closeParent(workerID, ErrParentClosing)

	workerCtx := newWorkerTestContext(app1, workerID, workerKey)
	h3, err3 := engine.Activate(workerCtx, compKey)
	if !errors.Is(err3, ErrInvalidActivationParent) || h3 != nil {
		t.Fatalf("expected ErrInvalidActivationParent for closed parent, got h3=%v, err=%v", h3, err3)
	}

	// 4. App 停止状态下拒绝新接纳
	app1.stateMu.Lock()
	app1.state = StateStopping
	app1.stateMu.Unlock()

	activeWorkerKey := ctxkey.Key("worker_active")
	activeWorkerID := "worker_active_1"
	engine.registerWorker(activeWorkerID, activeWorkerKey)
	activeWorkerCtx := newWorkerTestContext(app1, activeWorkerID, activeWorkerKey)

	h4, err4 := engine.Activate(activeWorkerCtx, compKey)
	if !errors.Is(err4, ErrAppClosed) || h4 != nil {
		t.Fatalf("expected ErrAppClosed when app stopping, got h4=%v, err=%v", h4, err4)
	}
}

// 7. Component.Stop panic 处理：不重试、不公布伪终态、标记不可用、保留依赖并触发 App 异步关机
func TestComponentEngine_StopPanic_RetainsDepsAndTriggersAppShutdown(t *testing.T) {
	app := New()
	app.stateMu.Lock()
	app.state = StateRunning
	app.stateMu.Unlock()

	engine := newComponentEngine(app)
	workerKey := ctxkey.Key("worker_panic")
	compKey := ctxkey.Key("comp_stop_panic")
	depKey := ctxkey.Key("dep_redis")
	workerID := "worker_panic_1"
	engine.registerWorker(workerID, workerKey)

	engine.RegisterComponentDef(PluginInfo{
		Key:      compKey,
		Requires: []ctxkey.Key{depKey},
	}, func() RunnablePlugin {
		return &testComponentFixture{
			stopFunc: func(ctx context.Context) error {
				panic("boom in component stop")
			},
		}
	})

	workerCtx := newWorkerTestContext(app, workerID, workerKey)
	h, err := engine.Activate(workerCtx, compKey)
	if err != nil || h == nil {
		t.Fatalf("activate failed: %v", err)
	}

	// Wait 短超时观察：因为 Stop panic 不关闭句柄，Wait 会等待直到 ctx 超时
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, waitErr := h.Wait(timeoutCtx)
	if !errors.Is(waitErr, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded because handle is never closed on stop panic, got %v", waitErr)
	}

	// 依赖保留：必须仍记录在 engine.retainedDeps 中，不能被释放
	depsMap := engine.GetRetainedDependencies()
	if count := depsMap[depKey]; count <= 0 {
		t.Fatalf("expected retained dependency count > 0, got %d", count)
	}

	// 协调器记录 StuckInstance
	stuckList := app.coordinator.getStuckInstances()
	if len(stuckList) != 1 {
		t.Fatalf("expected 1 stuck instance, got %d", len(stuckList))
	}
	if stuckList[0].Stage != "stop-panic" {
		t.Fatalf("expected stage 'stop-panic', got %s", stuckList[0].Stage)
	}
}
