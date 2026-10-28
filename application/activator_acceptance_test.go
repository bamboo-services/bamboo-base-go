package application

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

// 本文件自有的测试 fixture，避免与其它测试 mock 命名冲突。
type acceptanceWorkerFixture struct {
	startFunc func(ctx context.Context) error
	runFunc   func(ctx context.Context) error
	stopFunc  func(ctx context.Context) error
}

func (w *acceptanceWorkerFixture) Start(ctx context.Context) error {
	if w.startFunc != nil {
		return w.startFunc(ctx)
	}
	return nil
}

func (w *acceptanceWorkerFixture) Run(ctx context.Context) error {
	if w.runFunc != nil {
		return w.runFunc(ctx)
	}
	return nil
}

func (w *acceptanceWorkerFixture) Stop(ctx context.Context) error {
	if w.stopFunc != nil {
		return w.stopFunc(ctx)
	}
	return nil
}

type acceptanceComponentFixture struct {
	startFunc func(ctx context.Context) error
	runFunc   func(ctx context.Context) error
	stopFunc  func(ctx context.Context) error
}

func (c *acceptanceComponentFixture) Start(ctx context.Context) error {
	if c.startFunc != nil {
		return c.startFunc(ctx)
	}
	return nil
}

func (c *acceptanceComponentFixture) Run(ctx context.Context) error {
	if c.runFunc != nil {
		return c.runFunc(ctx)
	}
	return nil
}

func (c *acceptanceComponentFixture) Stop(ctx context.Context) error {
	if c.stopFunc != nil {
		return c.stopFunc(ctx)
	}
	return nil
}

// 辅助函数：断言 Activator() 必须非 nil，目前未实现时明确报错，不使用 skip 假造通过。
func requireActivator(t *testing.T, app *App) Activator {
	t.Helper()
	act := app.Activator()
	if act == nil {
		t.Errorf("app.Activator() returned nil: Activator is not implemented yet")
		// 主动触发 Shutdown，避免 Worker goroutine 与 App 永久阻塞
		_ = app.Shutdown()
		t.FailNow()
	}
	return act
}

// 场景 1: OverlapSkip 上一实例 Stop 未返回时拒绝新激活并返回 ErrComponentSkipped，且无句柄
func TestActivatorAcceptance_OverlapSkip_RejectsUntilStopReturned(t *testing.T) {
	app := New(WithGraceTimeout(10*time.Millisecond), WithShutdownTimeout(1*time.Second), WithCleanupTimeout(100*time.Millisecond))
	workerKey := ctxkey.Key("acc_worker_skip")
	compKey := ctxkey.Key("acc_comp_skip")

	var compFactoryCalls int32
	compRunEntered := make(chan struct{})
	releaseCompRun := make(chan struct{})
	compStopEntered := make(chan struct{})
	releaseCompStop := make(chan struct{})

	_ = app.RegisterComponent(PluginInfo{
		Key:     compKey,
		Overlap: OverlapPolicy{Kind: OverlapSkip},
	}, func() RunnablePlugin {
		atomic.AddInt32(&compFactoryCalls, 1)
		return &acceptanceComponentFixture{
			runFunc: func(ctx context.Context) error {
				close(compRunEntered)
				<-releaseCompRun
				return nil
			},
			stopFunc: func(ctx context.Context) error {
				close(compStopEntered)
				<-releaseCompStop
				return nil
			},
		}
	})

	workerStarted := make(chan struct{})
	testDone := make(chan struct{})

	_ = app.RegisterWorker(PluginInfo{Key: workerKey}, func() RunnablePlugin {
		return &acceptanceWorkerFixture{
			runFunc: func(lifetimeCtx context.Context) error {
				close(workerStarted)

				act := requireActivator(t, app)

				// 第一次激活成功
				h1, err1 := act.Activate(lifetimeCtx, compKey)
				if err1 != nil || h1 == nil {
					t.Errorf("expected first activate success, got h1=%v, err=%v", h1, err1)
					return err1
				}

				// 等待第一次激活进入 Run
				<-compRunEntered

				// 在 Run 执行期间尝试激活：应当拒绝并返回 ErrComponentSkipped，句柄为 nil
				h2, err2 := act.Activate(lifetimeCtx, compKey)
				if !errors.Is(err2, ErrComponentSkipped) || h2 != nil {
					t.Errorf("expected ErrComponentSkipped and nil handle while running, got h2=%v, err=%v", h2, err2)
				}

				// 释放 Run，进入 Stop 阶段
				close(releaseCompRun)
				<-compStopEntered

				// 在 Stop 尚未返回时再次激活：必须仍被拒绝，且句柄为 nil
				h3, err3 := act.Activate(lifetimeCtx, compKey)
				if !errors.Is(err3, ErrComponentSkipped) || h3 != nil {
					t.Errorf("expected ErrComponentSkipped and nil handle while Stop has not returned, got h3=%v, err=%v", h3, err3)
				}

				// 释放 Stop 完成真实收尾
				close(releaseCompStop)

				res1, waitErr1 := h1.Wait(lifetimeCtx)
				if waitErr1 != nil || res1.Outcome != OutcomeSuccess {
					t.Errorf("expected h1 success, got outcome=%v, waitErr=%v", res1.Outcome, waitErr1)
				}

				// Stop 真正返回后，再次激活应可被接纳
				h4, err4 := act.Activate(lifetimeCtx, compKey)
				if err4 != nil || h4 == nil {
					t.Errorf("expected activate after Stop returned to succeed, got h4=%v, err=%v", h4, err4)
				}

				close(testDone)
				<-lifetimeCtx.Done()
				return nil
			},
		}
	})

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	select {
	case <-testDone:
	case <-time.After(5 * time.Second):
		t.Fatal("test timeout waiting for skip test completion")
	}

	_ = app.Shutdown()
	<-runErrCh
}

// 场景 2: OverlapAllow 上限控制，超限立即返回 ErrConcurrencyLimitExceeded 且无句柄
func TestActivatorAcceptance_OverlapAllow_LimitExceeded(t *testing.T) {
	app := New(WithGraceTimeout(10*time.Millisecond), WithShutdownTimeout(1*time.Second), WithCleanupTimeout(100*time.Millisecond))
	workerKey := ctxkey.Key("acc_worker_allow")
	compKey := ctxkey.Key("acc_comp_allow")

	maxConcurrency := 2
	runEntered := make(chan struct{}, maxConcurrency)
	releaseRun := make(chan struct{})

	_ = app.RegisterComponent(PluginInfo{
		Key: compKey,
		Overlap: OverlapPolicy{
			Kind:           OverlapAllow,
			MaxConcurrency: maxConcurrency,
		},
	}, func() RunnablePlugin {
		return &acceptanceComponentFixture{
			runFunc: func(ctx context.Context) error {
				runEntered <- struct{}{}
				<-releaseRun
				return nil
			},
		}
	})

	testDone := make(chan struct{})

	_ = app.RegisterWorker(PluginInfo{Key: workerKey}, func() RunnablePlugin {
		return &acceptanceWorkerFixture{
			runFunc: func(lifetimeCtx context.Context) error {
				act := requireActivator(t, app)

				// 激活达到上限并发 (2)
				h1, err1 := act.Activate(lifetimeCtx, compKey)
				if err1 != nil || h1 == nil {
					t.Errorf("expected h1 success, got %v, err=%v", h1, err1)
				}
				h2, err2 := act.Activate(lifetimeCtx, compKey)
				if err2 != nil || h2 == nil {
					t.Errorf("expected h2 success, got %v, err=%v", h2, err2)
				}

				<-runEntered
				<-runEntered

				// 超出上限激活，必须返回 ErrConcurrencyLimitExceeded，且句柄为 nil
				h3, err3 := act.Activate(lifetimeCtx, compKey)
				if !errors.Is(err3, ErrConcurrencyLimitExceeded) || h3 != nil {
					t.Errorf("expected ErrConcurrencyLimitExceeded and nil handle, got h3=%v, err=%v", h3, err3)
				}

				close(releaseRun)
				_, _ = h1.Wait(lifetimeCtx)
				_, _ = h2.Wait(lifetimeCtx)

				close(testDone)
				<-lifetimeCtx.Done()
				return nil
			},
		}
	})

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	select {
	case <-testDone:
	case <-time.After(5 * time.Second):
		t.Fatal("test timeout waiting for allow test completion")
	}

	_ = app.Shutdown()
	<-runErrCh
}

// 场景 3: OverlapQueue 跨父 Worker 共享串行 FIFO 且 MaxQueue 仅数等待项，接纳即交付句柄（排队时工厂调用数为 0），
// 原请求 ctx 取消不撤销已接纳项，Wait 短超时不杀任务。
func TestActivatorAcceptance_OverlapQueue_SharedFifo_MaxQueueWaitingOnly_ContextIndependence(t *testing.T) {
	app := New(WithGraceTimeout(10*time.Millisecond), WithShutdownTimeout(1*time.Second), WithCleanupTimeout(100*time.Millisecond))
	workerAKey := ctxkey.Key("acc_worker_qa")
	workerBKey := ctxkey.Key("acc_worker_qb")
	compKey := ctxkey.Key("acc_comp_queue")

	var factoryCalls int32
	var dispatchOrder []string
	var mu sync.Mutex

	recordDispatch := func(id string) {
		mu.Lock()
		dispatchOrder = append(dispatchOrder, id)
		mu.Unlock()
	}

	releaseCurrentTask := make(chan struct{})
	taskStarted := make(chan string, 10)

	_ = app.RegisterComponent(PluginInfo{
		Key: compKey,
		Overlap: OverlapPolicy{
			Kind:     OverlapQueue,
			MaxQueue: 2, // 1 执行槽 + 最多 2 个等待项
		},
	}, func() RunnablePlugin {
		calls := atomic.AddInt32(&factoryCalls, 1)
		id := string(rune('A' - 1 + calls))
		recordDispatch(id)
		return &acceptanceComponentFixture{
			runFunc: func(ctx context.Context) error {
				taskStarted <- id
				<-releaseCurrentTask
				return nil
			},
		}
	})

	var (
		workerALifetime context.Context
		workerBLifetime context.Context
		barrierWorkerA  = make(chan struct{})
		barrierWorkerB  = make(chan struct{})
	)

	_ = app.RegisterWorker(PluginInfo{Key: workerAKey}, func() RunnablePlugin {
		return &acceptanceWorkerFixture{
			runFunc: func(lifetimeCtx context.Context) error {
				workerALifetime = lifetimeCtx
				close(barrierWorkerA)
				<-lifetimeCtx.Done()
				return nil
			},
		}
	})

	_ = app.RegisterWorker(PluginInfo{Key: workerBKey}, func() RunnablePlugin {
		return &acceptanceWorkerFixture{
			runFunc: func(lifetimeCtx context.Context) error {
				workerBLifetime = lifetimeCtx
				close(barrierWorkerB)
				<-lifetimeCtx.Done()
				return nil
			},
		}
	})

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	<-barrierWorkerA
	<-barrierWorkerB

	act := requireActivator(t, app)

	// 1. Worker A 激发第一个任务 T1，占据唯一的执行槽
	h1, err1 := act.Activate(workerALifetime, compKey)
	if err1 != nil || h1 == nil {
		t.Fatalf("h1 activate failed: %v", err1)
	}
	<-taskStarted

	// 此时工厂调用数应为 1
	if calls := atomic.LoadInt32(&factoryCalls); calls != 1 {
		t.Fatalf("expected factory calls=1, got %d", calls)
	}

	// 2. 使用一个带有短超时的 BridgeRequest ctx 激发第二个任务 T2（进入等待队列）
	shortCtx, shortCancel := context.WithTimeout(workerBLifetime, 20*time.Millisecond)
	h2, err2 := act.Activate(shortCtx, compKey)
	if err2 != nil || h2 == nil {
		t.Fatalf("h2 activate failed: %v", err2)
	}
	// 接纳即交付句柄：此时排队中，工厂尚未被调用
	if calls := atomic.LoadInt32(&factoryCalls); calls != 1 {
		t.Fatalf("expected factory calls still 1 while T2 queued, got %d", calls)
	}

	// 原激活 ctx 取消，不撤销已接纳项
	shortCancel()

	// 3. Worker A 激发第三个任务 T3（进入等待队列，队满等待项达 MaxQueue=2）
	h3, err3 := act.Activate(workerALifetime, compKey)
	if err3 != nil || h3 == nil {
		t.Fatalf("h3 activate failed: %v", err3)
	}

	// 4. 再次激活应被 ErrQueueFull 拒绝，且无句柄（因为 MaxQueue 只数等待项，已有 T2, T3 等待）
	h4, err4 := act.Activate(workerBLifetime, compKey)
	if !errors.Is(err4, ErrQueueFull) || h4 != nil {
		t.Fatalf("expected ErrQueueFull with nil handle, got h4=%v, err=%v", h4, err4)
	}

	// 5. 对排队中的 h2 执行 Wait，使用短超时 context，Wait 超时后不杀任务
	waitTimeoutCtx, waitTimeoutCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer waitTimeoutCancel()
	_, waitErr := h2.Wait(waitTimeoutCtx)
	if !errors.Is(waitErr, context.DeadlineExceeded) {
		t.Fatalf("expected wait timeout DeadlineExceeded, got %v", waitErr)
	}

	// 6. 释放 T1，T2 应当按 FIFO 被派发执行
	releaseCurrentTask <- struct{}{}
	id2 := <-taskStarted
	if id2 != "B" {
		t.Fatalf("expected T2 ('B') dispatched next, got %s", id2)
	}

	// 释放 T2，等待 T2 完成，验证 T2 执行成功且没有被短超时取消
	releaseCurrentTask <- struct{}{}
	res2, errRes2 := h2.Wait(context.Background())
	if errRes2 != nil || res2.Outcome != OutcomeSuccess {
		t.Fatalf("expected T2 to complete successfully, got outcome=%v, err=%v", res2.Outcome, errRes2)
	}

	// 释放 T3
	<-taskStarted
	releaseCurrentTask <- struct{}{}
	res3, _ := h3.Wait(context.Background())
	if res3.Outcome != OutcomeSuccess {
		t.Fatalf("expected T3 outcome success, got %v", res3.Outcome)
	}

	_ = app.Shutdown()
	<-runErrCh
}

// 场景 4: 父 Worker 关闭或 App 关闭取消未派发项：
// Dispatched=false、CancelRequested=true、CancelErr 匹配 context.Canceled 且 RunErr 为 nil、
// 工厂与所有 Hook 次数为 0、占用退还一次。
func TestActivatorAcceptance_UnDispatchedCancelledOnAppOrParentShutdown(t *testing.T) {
	app := New(WithGraceTimeout(10*time.Millisecond), WithShutdownTimeout(1*time.Second), WithCleanupTimeout(100*time.Millisecond))
	workerKey := ctxkey.Key("acc_worker_parent_close")
	compKey := ctxkey.Key("acc_comp_parent_close")

	var factoryCalls int32
	var startCalls int32
	var stopCalls int32

	blockFirstTask := make(chan struct{})
	firstTaskStarted := make(chan struct{})

	_ = app.RegisterComponent(PluginInfo{
		Key: compKey,
		Overlap: OverlapPolicy{
			Kind:     OverlapQueue,
			MaxQueue: 5,
		},
	}, func() RunnablePlugin {
		atomic.AddInt32(&factoryCalls, 1)
		return &acceptanceComponentFixture{
			startFunc: func(ctx context.Context) error {
				atomic.AddInt32(&startCalls, 1)
				return nil
			},
			runFunc: func(ctx context.Context) error {
				close(firstTaskStarted)
				<-blockFirstTask
				return nil
			},
			stopFunc: func(ctx context.Context) error {
				atomic.AddInt32(&stopCalls, 1)
				return nil
			},
		}
	})

	workerStarted := make(chan struct{})
	queuedAccepted := make(chan struct{})
	var queuedHandle RunHandle

	_ = app.RegisterWorker(PluginInfo{Key: workerKey}, func() RunnablePlugin {
		return &acceptanceWorkerFixture{
			runFunc: func(lifetimeCtx context.Context) error {
				close(workerStarted)
				act := requireActivator(t, app)

				// 占住执行槽
				h1, err1 := act.Activate(lifetimeCtx, compKey)
				if err1 != nil || h1 == nil {
					t.Errorf("h1 activate failed: %v", err1)
					return err1
				}

				<-firstTaskStarted

				// 激活第二个任务排队
				h2, err2 := act.Activate(lifetimeCtx, compKey)
				if err2 != nil || h2 == nil {
					t.Errorf("h2 activate failed: %v", err2)
					return err2
				}
				queuedHandle = h2
				close(queuedAccepted)

				// Worker 在未释放 h1 前直接退出，触发关停
				return nil
			},
		}
	})

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	<-workerStarted
	<-firstTaskStarted
	<-queuedAccepted

	// 此时 Worker 已退出，h2 应当被父级关停取消（Dispatched=false）
	// 先等 h2 真实未派发取消结果（有界 Wait 防挂），禁止先释放 blockFirstTask 让队列可能派发
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer waitCancel()
	res, waitErr := queuedHandle.Wait(waitCtx)

	// 关闭 App 并释放首个任务
	close(blockFirstTask)
	_ = app.Shutdown()
	<-runErrCh
	if res.Dispatched != false {
		t.Errorf("expected Dispatched=false, got true")
	}
	if !res.CancelRequested {
		t.Errorf("expected CancelRequested=true, got false")
	}
	if res.RunErr != nil {
		t.Errorf("expected RunErr=nil, got %v", res.RunErr)
	}
	if res.FactoryErr != nil || res.StartErr != nil || res.StopErr != nil {
		t.Errorf("expected all stage hook errs nil, got factory=%v, start=%v, stop=%v", res.FactoryErr, res.StartErr, res.StopErr)
	}
	if res.Outcome != OutcomeCancelled {
		t.Errorf("expected OutcomeCancelled, got %v", res.Outcome)
	}
	if !errors.Is(res.CancelErr, context.Canceled) {
		t.Errorf("expected CancelErr to match context.Canceled, got %v", res.CancelErr)
	}
	if !errors.Is(waitErr, context.Canceled) {
		t.Errorf("expected Wait() error to match context.Canceled, got %v", waitErr)
	}

	// 未派发项绝不调用工厂与生命周期钩子，总次数应只有首个执行任务调用的 1 次
	if fCalls := atomic.LoadInt32(&factoryCalls); fCalls != 1 {
		t.Errorf("expected factoryCalls=1 (only dispatched task), got %d", fCalls)
	}
	if sCalls := atomic.LoadInt32(&startCalls); sCalls != 1 {
		t.Errorf("expected startCalls=1, got %d", sCalls)
	}
}

// 场景 5: Component 嵌套激活目标 Queue 忙立即 ErrNestedQueueWouldBlock 不建句柄；祖先环拒绝 ErrActivationCycle
func TestActivatorAcceptance_NestedQueue_WouldBlock_And_ActivationCycle(t *testing.T) {
	app := New(WithGraceTimeout(10*time.Millisecond), WithShutdownTimeout(1*time.Second), WithCleanupTimeout(100*time.Millisecond))
	workerKey := ctxkey.Key("acc_worker_nested")
	compParentKey := ctxkey.Key("acc_comp_parent")
	compTargetKey := ctxkey.Key("acc_comp_target")

	blockTargetRun := make(chan struct{})
	targetRunStarted := make(chan struct{})

	_ = app.RegisterComponent(PluginInfo{
		Key: compTargetKey,
		Overlap: OverlapPolicy{
			Kind:     OverlapQueue,
			MaxQueue: 2,
		},
	}, func() RunnablePlugin {
		return &acceptanceComponentFixture{
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

	_ = app.RegisterComponent(PluginInfo{
		Key: compParentKey,
		Overlap: OverlapPolicy{
			Kind: OverlapAllow, MaxConcurrency: 2,
		},
	}, func() RunnablePlugin {
		return &acceptanceComponentFixture{
			runFunc: func(componentLifetimeCtx context.Context) error {
				act := requireActivator(t, app)

				// 1. 尝试祖先环激活：父级为 compParentKey，目标亦为 compParentKey -> 必须返回 ErrActivationCycle
				hCycle, errCycle := act.Activate(componentLifetimeCtx, compParentKey)
				if !errors.Is(errCycle, ErrActivationCycle) || hCycle != nil {
					t.Errorf("expected ErrActivationCycle and nil handle, got hCycle=%v, err=%v", hCycle, errCycle)
				}

				// 2. 目标 compTargetKey 此时处于忙碌（已有实例正在 Run）
				// 作为 Component 父级去激活繁忙的 Queue，必须立即返回 ErrNestedQueueWouldBlock，不建句柄不排队
				hNested, errNested := act.Activate(componentLifetimeCtx, compTargetKey)
				if !errors.Is(errNested, ErrNestedQueueWouldBlock) || hNested != nil {
					t.Errorf("expected ErrNestedQueueWouldBlock and nil handle, got hNested=%v, err=%v", hNested, errNested)
				}

				close(testDone)
				return nil
			},
		}
	})

	_ = app.RegisterWorker(PluginInfo{Key: workerKey}, func() RunnablePlugin {
		return &acceptanceWorkerFixture{
			runFunc: func(workerLifetimeCtx context.Context) error {
				act := requireActivator(t, app)

				// Worker 先激活 compTargetKey，占住执行槽
				hTarget, errTarget := act.Activate(workerLifetimeCtx, compTargetKey)
				if errTarget != nil || hTarget == nil {
					t.Errorf("target activate failed: %v", errTarget)
					return errTarget
				}
				<-targetRunStarted

				// Worker 激活 compParentKey
				hParent, errParent := act.Activate(workerLifetimeCtx, compParentKey)
				if errParent != nil || hParent == nil {
					t.Errorf("parent activate failed: %v", errParent)
					return errParent
				}

				<-testDone
				close(blockTargetRun)

				_, _ = hTarget.Wait(workerLifetimeCtx)
				_, _ = hParent.Wait(workerLifetimeCtx)

				<-workerLifetimeCtx.Done()
				return nil
			},
		}
	})

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	select {
	case <-testDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for nested queue test")
	}

	_ = app.Shutdown()
	<-runErrCh
}
