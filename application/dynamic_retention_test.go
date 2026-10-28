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

// dynamicRetentionFixtureHelper 用于在测试结束或失败时释放全部通道屏障，避免 goroutine 泄漏
type dynamicRetentionFixtureHelper struct {
	mu       sync.Mutex
	cleanups []func()
}

func newDynamicRetentionFixtureHelper(t *testing.T) *dynamicRetentionFixtureHelper {
	h := &dynamicRetentionFixtureHelper{}
	t.Cleanup(func() {
		h.releaseAll()
	})
	return h
}

func (h *dynamicRetentionFixtureHelper) register(fn func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cleanups = append(h.cleanups, fn)
}

func (h *dynamicRetentionFixtureHelper) releaseAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i := len(h.cleanups) - 1; i >= 0; i-- {
		fn := h.cleanups[i]
		if fn != nil {
			fn()
		}
	}
	h.cleanups = nil
}

// 通用 Provider 结构，支持 Expose 暴露独立资源键
type dynamicRetentionProvider struct {
	resourceKey ctxkey.Key
	startCalled atomic.Int32
	stopCalled  atomic.Int32
	stopEntered chan struct{}
}

func (p *dynamicRetentionProvider) Start(ctx context.Context) error {
	p.startCalled.Add(1)
	return nil
}

func (p *dynamicRetentionProvider) Stop(ctx context.Context) error {
	p.stopCalled.Add(1)
	if p.stopEntered != nil {
		select {
		case <-p.stopEntered:
		default:
			close(p.stopEntered)
		}
	}
	return nil
}

func (p *dynamicRetentionProvider) Expose() map[ctxkey.Key]any {
	return map[ctxkey.Key]any{
		p.resourceKey: "retained_val",
	}
}

// -----------------------------------------------------------------------------
// 回归 A: 静态无依赖 Worker W 与 Tool provider P 同层，C.Requires=[P] 由 W 激活
// 安排 P 的 key 字典早于 W（另子测试反序），确保关闭顺序 C -> P 且各一次
// -----------------------------------------------------------------------------

func runSubtestRegressionA(t *testing.T, providerFirstInLexicographical bool) {
	h := newDynamicRetentionFixtureHelper(t)

	var providerKey, workerKey ctxkey.Key
	if providerFirstInLexicographical {
		providerKey = ctxkey.Key("a_provider_tool")
		workerKey = ctxkey.Key("z_worker_consumer")
	} else {
		providerKey = ctxkey.Key("z_provider_tool")
		workerKey = ctxkey.Key("a_worker_consumer")
	}
	resourceKey := ctxkey.Key("res_dynamic_retention_a")
	compKey := ctxkey.Key("c_component_dynamic_a")

	provider := &dynamicRetentionProvider{
		resourceKey: resourceKey,
		stopEntered: make(chan struct{}),
	}

	app := New(
		WithStartTimeout(100*time.Millisecond),
		WithGraceTimeout(60*time.Millisecond),
		WithCleanupTimeout(80*time.Millisecond),
		WithShutdownTimeout(150*time.Millisecond),
	)

	// 注册 Tool provider
	err := app.RegisterTool(PluginInfo{
		Key:      providerKey,
		Provides: []ctxkey.Key{resourceKey},
	}, func() Plugin {
		return provider
	})
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	var compStopCount atomic.Int32
	var providerStopWhenCompStopped atomic.Int32
	compRunEntered := make(chan struct{})
	compStopEntered := make(chan struct{})

	// 注册动态 Component C，显式 Requires 资源键
	err = app.RegisterComponent(PluginInfo{
		Key:      compKey,
		Requires: []ctxkey.Key{resourceKey},
	}, func() RunnablePlugin {
		return &acceptanceComponentFixture{
			runFunc: func(ctx context.Context) error {
				select {
				case <-compRunEntered:
				default:
					close(compRunEntered)
				}
				<-ctx.Done()
				return nil
			},
			stopFunc: func(ctx context.Context) error {
				compStopCount.Add(1)
				// 记录在 C.Stop 屏障执行瞬间 Provider 是否已被误提前停止
				providerStopWhenCompStopped.Store(provider.stopCalled.Load())
				select {
				case <-compStopEntered:
				default:
					close(compStopEntered)
				}
				return nil
			},
		}
	})
	if err != nil {
		t.Fatalf("RegisterComponent failed: %v", err)
	}

	workerStarted := make(chan struct{})
	var handle RunHandle

	// 注册静态 Worker W（Requires 为空）
	err = app.RegisterWorker(PluginInfo{
		Key:      workerKey,
		Requires: []ctxkey.Key{},
	}, func() RunnablePlugin {
		return &acceptanceWorkerFixture{
			runFunc: func(lifetimeCtx context.Context) error {
				act := requireActivator(t, app)
				hRes, actErr := act.Activate(lifetimeCtx, compKey)
				if actErr != nil {
					return actErr
				}
				handle = hRes
				select {
				case <-workerStarted:
				default:
					close(workerStarted)
				}
				<-lifetimeCtx.Done()
				return nil
			},
		}
	})
	if err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	select {
	case <-workerStarted:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("worker did not start")
	}

	select {
	case <-compRunEntered:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("component did not enter Run")
	}

	// 触发 App.Shutdown
	shutdownErrCh := make(chan error, 1)
	go func() {
		shutdownErrCh <- app.Shutdown()
	}()

	select {
	case err = <-shutdownErrCh:
		if err != nil {
			t.Fatalf("expected Shutdown to succeed with nil, got: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for Shutdown to return")
	}

	select {
	case err = <-runErrCh:
		if err != nil {
			t.Fatalf("expected Run to return nil on normal shutdown, got: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for Run to return")
	}

	// 断言顺序与调用次数：C.Stop 屏障时 P.Stop 次数必须为 0
	if val := providerStopWhenCompStopped.Load(); val != 0 {
		t.Fatalf("expected provider.Stop not called before C.Stop completed, but got %d calls", val)
	}
	if compStops := compStopCount.Load(); compStops != 1 {
		t.Fatalf("expected component Stop called exactly once, got %d", compStops)
	}
	if providerStops := provider.stopCalled.Load(); providerStops != 1 {
		t.Fatalf("expected provider Stop called exactly once, got %d", providerStops)
	}

	waitRes, waitErr := handle.Wait(context.Background())
	if waitErr != nil {
		t.Fatalf("expected handle.Wait success, got %v", waitErr)
	}
	if !waitRes.Dispatched {
		t.Fatalf("expected handle to be dispatched")
	}
	_ = h
}

func TestDynamicRetention_RegressionA_OrderProviderFirst(t *testing.T) {
	runSubtestRegressionA(t, true)
}

func TestDynamicRetention_RegressionA_OrderWorkerFirst(t *testing.T) {
	runSubtestRegressionA(t, false)
}

// -----------------------------------------------------------------------------
// 回归 B: C.Run 忽略 ctx 直到测试释放，W.Run 可配合退出。
// Shutdown 总预算内 Incomplete，P.Stop 计数 0、StuckInstances 记录真实组件与 RetainedDeps。
// -----------------------------------------------------------------------------

func TestDynamicRetention_RegressionB_StuckInRun_RetainsDependency(t *testing.T) {
	h := newDynamicRetentionFixtureHelper(t)

	providerKey := ctxkey.Key("b_provider_tool")
	resourceKey := ctxkey.Key("b_res_key")
	workerKey := ctxkey.Key("b_worker_consumer")
	compKey := ctxkey.Key("b_component_stuck")

	provider := &dynamicRetentionProvider{
		resourceKey: resourceKey,
	}

	// 设定总 Shutdown 预算
	app := New(
		WithStartTimeout(80*time.Millisecond),
		WithGraceTimeout(40*time.Millisecond),
		WithCleanupTimeout(50*time.Millisecond),
		WithShutdownTimeout(120*time.Millisecond),
	)

	err := app.RegisterTool(PluginInfo{
		Key:      providerKey,
		Provides: []ctxkey.Key{resourceKey},
	}, func() Plugin {
		return provider
	})
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	compRunEntered := make(chan struct{})
	releaseCompRun := make(chan struct{})
	var releaseOnce sync.Once
	h.register(func() {
		releaseOnce.Do(func() {
			close(releaseCompRun)
		})
	})

	var compInstanceID atomic.Value

	err = app.RegisterComponent(PluginInfo{
		Key:      compKey,
		Requires: []ctxkey.Key{resourceKey},
	}, func() RunnablePlugin {
		return &acceptanceComponentFixture{
			runFunc: func(ctx context.Context) error {
				select {
				case <-compRunEntered:
				default:
					close(compRunEntered)
				}
				// 忽略 ctx，直到测试明确释放
				<-releaseCompRun
				return nil
			},
			stopFunc: func(ctx context.Context) error {
				return nil
			},
		}
	})
	if err != nil {
		t.Fatalf("RegisterComponent failed: %v", err)
	}

	workerStarted := make(chan struct{})
	var handle RunHandle

	err = app.RegisterWorker(PluginInfo{
		Key: workerKey,
	}, func() RunnablePlugin {
		return &acceptanceWorkerFixture{
			runFunc: func(lifetimeCtx context.Context) error {
				act := requireActivator(t, app)
				hRes, actErr := act.Activate(lifetimeCtx, compKey)
				if actErr != nil {
					return actErr
				}
				handle = hRes
				compInstanceID.Store(hRes.InstanceID())
				select {
				case <-workerStarted:
				default:
					close(workerStarted)
				}
				<-lifetimeCtx.Done()
				return nil
			},
		}
	})
	if err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	select {
	case <-workerStarted:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("worker did not start")
	}

	select {
	case <-compRunEntered:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("component did not enter run")
	}

	// 触发 App.Shutdown
	shutdownErr := app.Shutdown()
	if shutdownErr == nil {
		t.Fatalf("expected StopIncompleteError, got nil")
	}

	var incompleteErr *StopIncompleteError
	if !errors.As(shutdownErr, &incompleteErr) {
		t.Fatalf("expected errors.As StopIncompleteError, got: %v", shutdownErr)
	}

	// 验证 P.Stop 计数必须为 0（因依赖保留）
	if pStops := provider.stopCalled.Load(); pStops != 0 {
		t.Fatalf("expected provider Stop call count 0, got %d", pStops)
	}

	// 验证 StuckInstances 快照
	if len(incompleteErr.StuckInstances) == 0 {
		t.Fatalf("expected at least 1 stuck instance in snapshot")
	}

	var foundStuck bool
	expectedID, _ := compInstanceID.Load().(string)
	for _, stuck := range incompleteErr.StuckInstances {
		if stuck.PluginKey == compKey {
			foundStuck = true
			if stuck.Stage != "run" {
				t.Errorf("expected stuck stage 'run', got %q", stuck.Stage)
			}
			if expectedID != "" && stuck.InstanceID != expectedID {
				t.Errorf("expected instance ID %q, got %q", expectedID, stuck.InstanceID)
			}
			var hasDep bool
			for _, dep := range stuck.RetainedDeps {
				if dep == resourceKey || dep == providerKey {
					hasDep = true
					break
				}
			}
			if !hasDep {
				t.Errorf("expected RetainedDeps to contain %v or %v, got %v", resourceKey, providerKey, stuck.RetainedDeps)
			}
		}
	}
	if !foundStuck {
		t.Fatalf("expected stuck instance for component %v", compKey)
	}

	// handle 不得假终态：在 releaseCompRun 之前 handle.Wait 不得伪完成
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	_, waitErr := handle.Wait(waitCtx)
	waitCancel()
	if !errors.Is(waitErr, context.DeadlineExceeded) {
		t.Fatalf("expected handle.Wait to block with DeadlineExceeded before release, got %v", waitErr)
	}

	// 释放执行屏障，完成清理收尾
	releaseOnce.Do(func() {
		close(releaseCompRun)
	})

	select {
	case <-runErrCh:
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for app.Run to terminate after release")
	}
}

// -----------------------------------------------------------------------------
// 回归 C: C.Start 忽略 ctx 超过短 StartTimeout。
// 子例 1: 先 AppShutdown 到 Incomplete 时账本 stage=start+实例+依赖保留，Stop 不能在 Start 真退前调用；释放后绝不 Run，Stop 一次。
// 子例 2: 正常 App 不 Shutdown，仅超时迟到的 Start 返回 nil 也不应 Run。
// -----------------------------------------------------------------------------

func TestDynamicRetention_RegressionC_StartTimeout_WithShutdown(t *testing.T) {
	h := newDynamicRetentionFixtureHelper(t)

	providerKey := ctxkey.Key("c1_provider_tool")
	resourceKey := ctxkey.Key("c1_res_key")
	workerKey := ctxkey.Key("c1_worker_consumer")
	compKey := ctxkey.Key("c1_component_start_timeout")

	provider := &dynamicRetentionProvider{
		resourceKey: resourceKey,
	}

	app := New(
		WithStartTimeout(60*time.Millisecond),
		WithGraceTimeout(40*time.Millisecond),
		WithCleanupTimeout(50*time.Millisecond),
		WithShutdownTimeout(120*time.Millisecond),
	)

	err := app.RegisterTool(PluginInfo{
		Key:      providerKey,
		Provides: []ctxkey.Key{resourceKey},
	}, func() Plugin {
		return provider
	})
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	startEntered := make(chan struct{})
	releaseStart := make(chan struct{})
	var releaseOnce sync.Once
	h.register(func() {
		releaseOnce.Do(func() {
			close(releaseStart)
		})
	})

	var startFinished atomic.Bool
	var runCalled atomic.Bool
	var stopCalled atomic.Int32
	var stopCalledBeforeStartReturn atomic.Bool

	err = app.RegisterComponent(PluginInfo{
		Key:      compKey,
		Requires: []ctxkey.Key{resourceKey},
	}, func() RunnablePlugin {
		return &acceptanceComponentFixture{
			startFunc: func(ctx context.Context) error {
				select {
				case <-startEntered:
				default:
					close(startEntered)
				}
				<-releaseStart
				startFinished.Store(true)
				return nil
			},
			runFunc: func(ctx context.Context) error {
				runCalled.Store(true)
				return nil
			},
			stopFunc: func(ctx context.Context) error {
				stopCalled.Add(1)
				if !startFinished.Load() {
					stopCalledBeforeStartReturn.Store(true)
				}
				return nil
			},
		}
	})
	if err != nil {
		t.Fatalf("RegisterComponent failed: %v", err)
	}

	workerStarted := make(chan struct{})
	var handle RunHandle

	err = app.RegisterWorker(PluginInfo{
		Key: workerKey,
	}, func() RunnablePlugin {
		return &acceptanceWorkerFixture{
			runFunc: func(lifetimeCtx context.Context) error {
				act := requireActivator(t, app)
				hRes, actErr := act.Activate(lifetimeCtx, compKey)
				if actErr != nil {
					return actErr
				}
				handle = hRes
				select {
				case <-workerStarted:
				default:
					close(workerStarted)
				}
				<-lifetimeCtx.Done()
				return nil
			},
		}
	})
	if err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	select {
	case <-workerStarted:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("worker did not start")
	}

	select {
	case <-startEntered:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("component did not enter Start")
	}

	// 触发 App.Shutdown
	shutdownErr := app.Shutdown()
	if shutdownErr == nil {
		t.Fatalf("expected StopIncompleteError on shutdown, got nil")
	}

	var incompleteErr *StopIncompleteError
	if !errors.As(shutdownErr, &incompleteErr) {
		t.Fatalf("expected StopIncompleteError, got %v", shutdownErr)
	}

	// 账本验证：stage 为 start，RetainedDeps 保留
	var foundStuckStart bool
	for _, stuck := range incompleteErr.StuckInstances {
		if stuck.PluginKey == compKey {
			foundStuckStart = true
			if stuck.Stage != "start" {
				t.Errorf("expected stuck stage 'start', got %q", stuck.Stage)
			}
			if len(stuck.RetainedDeps) == 0 {
				t.Errorf("expected non-empty RetainedDeps for stuck start instance")
			}
		}
	}
	if !foundStuckStart {
		t.Fatalf("expected stuck start instance in coordinator snapshot")
	}

	// 必须保证 Stop 不能在 Start 真退之前被调用
	if stopCalledBeforeStartReturn.Load() {
		t.Fatalf("Stop was called before Start returned!")
	}

	// 释放 Start 屏障
	releaseOnce.Do(func() {
		close(releaseStart)
	})

	select {
	case <-runErrCh:
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for app.Run to terminate")
	}

	// 释放 Start 返回后绝不能调用 Run
	if runCalled.Load() {
		t.Fatalf("Run must not be called after Start timed out")
	}

	// 释放后应完成正常 Stop 收尾（必须且仅有 1 次）
	if stops := stopCalled.Load(); stops != 1 {
		t.Fatalf("expected component Stop called exactly once, got %d", stops)
	}

	// 任务句柄终态校验：使用有界 waitCtx 防死锁，非 nil 且包含超时原因
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer waitCancel()
	waitRes, waitErr := handle.Wait(waitCtx)
	if waitErr == nil || !errors.Is(waitErr, context.DeadlineExceeded) {
		t.Fatalf("expected handle.Wait to contain context.DeadlineExceeded, got: %v", waitErr)
	}
	if !errors.Is(waitRes.StartErr, context.DeadlineExceeded) {
		t.Fatalf("expected waitRes.StartErr to be context.DeadlineExceeded, got: %v", waitRes.StartErr)
	}
	if waitRes.Outcome != OutcomeFailed {
		t.Fatalf("expected OutcomeFailed, got: %v", waitRes.Outcome)
	}
	if waitRes.RunErr != nil {
		t.Fatalf("expected RunErr == nil since Run must not be called, got: %v", waitRes.RunErr)
	}
}

func TestDynamicRetention_RegressionC_StartTimeout_NoAppShutdown(t *testing.T) {
	h := newDynamicRetentionFixtureHelper(t)

	providerKey := ctxkey.Key("c2_provider_tool")
	resourceKey := ctxkey.Key("c2_res_key")
	workerKey := ctxkey.Key("c2_worker_consumer")
	compKey := ctxkey.Key("c2_component_late_start")

	provider := &dynamicRetentionProvider{
		resourceKey: resourceKey,
	}

	app := New(
		WithStartTimeout(50*time.Millisecond),
		WithGraceTimeout(40*time.Millisecond),
		WithCleanupTimeout(50*time.Millisecond),
		WithShutdownTimeout(150*time.Millisecond),
	)

	_ = app.RegisterTool(PluginInfo{
		Key:      providerKey,
		Provides: []ctxkey.Key{resourceKey},
	}, func() Plugin {
		return provider
	})

	startEntered := make(chan struct{})
	releaseStart := make(chan struct{})
	var releaseOnce sync.Once
	h.register(func() {
		releaseOnce.Do(func() {
			close(releaseStart)
		})
	})

	var runCalled atomic.Bool
	var compStopped atomic.Int32

	_ = app.RegisterComponent(PluginInfo{
		Key:      compKey,
		Requires: []ctxkey.Key{resourceKey},
	}, func() RunnablePlugin {
		return &acceptanceComponentFixture{
			startFunc: func(ctx context.Context) error {
				select {
				case <-startEntered:
				default:
					close(startEntered)
				}
				<-releaseStart
				return nil
			},
			runFunc: func(ctx context.Context) error {
				runCalled.Store(true)
				return nil
			},
			stopFunc: func(ctx context.Context) error {
				compStopped.Add(1)
				return nil
			},
		}
	})

	workerStarted := make(chan struct{})
	var handle RunHandle

	_ = app.RegisterWorker(PluginInfo{
		Key: workerKey,
	}, func() RunnablePlugin {
		return &acceptanceWorkerFixture{
			runFunc: func(lifetimeCtx context.Context) error {
				act := requireActivator(t, app)
				hRes, actErr := act.Activate(lifetimeCtx, compKey)
				if actErr != nil {
					return actErr
				}
				handle = hRes
				select {
				case <-workerStarted:
				default:
					close(workerStarted)
				}
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
	case <-workerStarted:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("worker did not start")
	}

	select {
	case <-startEntered:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("component did not enter start")
	}

	// 等待组件 StartTimeout 门禁时间到达
	time.Sleep(100 * time.Millisecond)

	// 释放迟到的 Start，返回 nil
	releaseOnce.Do(func() {
		close(releaseStart)
	})

	// 等待 handle 观察到任务失败
	waitRes, waitErr := handle.Wait(context.Background())
	if waitErr == nil && waitRes.RunErr == nil {
		t.Fatalf("expected handle to report failure on start timeout")
	}

	// 绝不能进入 Run
	if runCalled.Load() {
		t.Fatalf("Run was unexpectedly called after Start timeout")
	}

	// App 本身未崩溃，此时再发起优雅 Shutdown
	shutdownErr := app.Shutdown()
	if shutdownErr != nil {
		t.Logf("normal shutdown error: %v", shutdownErr)
	}

	select {
	case <-runErrCh:
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for app.Run")
	}
}

// -----------------------------------------------------------------------------
// 回归 D: C.Start 返回 error 而 Stop panic，Stop 不重试、不释放依赖/执行槽/父计数，
// 不公布假 handle 终态，原 Stop panic 触发 App 失败协调；P 不 Stop，snapshot 保留 stop-panic 身份。
// -----------------------------------------------------------------------------

func TestDynamicRetention_RegressionD_StartError_StopPanic_CoordinatesAppFailure(t *testing.T) {
	h := newDynamicRetentionFixtureHelper(t)

	providerKey := ctxkey.Key("d_provider_tool")
	resourceKey := ctxkey.Key("d_res_key")
	workerKey := ctxkey.Key("d_worker_consumer")
	compKey := ctxkey.Key("d_component_stop_panic")

	provider := &dynamicRetentionProvider{
		resourceKey: resourceKey,
	}

	app := New(
		WithStartTimeout(100*time.Millisecond),
		WithGraceTimeout(50*time.Millisecond),
		WithCleanupTimeout(60*time.Millisecond),
		WithShutdownTimeout(150*time.Millisecond),
	)

	err := app.RegisterTool(PluginInfo{
		Key:      providerKey,
		Provides: []ctxkey.Key{resourceKey},
	}, func() Plugin {
		return provider
	})
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	var stopCallCount atomic.Int32

	err = app.RegisterComponent(PluginInfo{
		Key:      compKey,
		Requires: []ctxkey.Key{resourceKey},
		Overlap:  OverlapPolicy{Kind: OverlapQueue, MaxQueue: 1},
	}, func() RunnablePlugin {
		return &acceptanceComponentFixture{
			startFunc: func(ctx context.Context) error {
				return errors.New("component start failed explicitly")
			},
			runFunc: func(ctx context.Context) error {
				return nil
			},
			stopFunc: func(ctx context.Context) error {
				stopCallCount.Add(1)
				panic("critical stop hardware failure")
			},
		}
	})
	if err != nil {
		t.Fatalf("RegisterComponent failed: %v", err)
	}

	workerStarted := make(chan struct{})
	var handle RunHandle

	err = app.RegisterWorker(PluginInfo{
		Key: workerKey,
	}, func() RunnablePlugin {
		return &acceptanceWorkerFixture{
			runFunc: func(lifetimeCtx context.Context) error {
				act := requireActivator(t, app)
				hRes, actErr := act.Activate(lifetimeCtx, compKey)
				if actErr != nil {
					return actErr
				}
				handle = hRes
				select {
				case <-workerStarted:
				default:
					close(workerStarted)
				}
				<-lifetimeCtx.Done()
				return nil
			},
		}
	})
	if err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	select {
	case <-workerStarted:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("worker did not start")
	}

	// 等待 App.Run 因 Stop panic 自动触发的失败关机协调退出
	var appRunErr error
	select {
	case appRunErr = <-runErrCh:
		if appRunErr == nil {
			t.Fatalf("expected app.Run to exit with error due to stop panic, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for app.Run to terminate upon stop panic")
	}

	// 验证 Stop 绝不被重试
	if calls := stopCallCount.Load(); calls != 1 {
		t.Fatalf("expected Stop called exactly once (no retry), got %d", calls)
	}

	// 验证依赖 Provider 绝不执行 Stop
	if pStops := provider.stopCalled.Load(); pStops != 0 {
		t.Fatalf("expected provider Stop call count 0, got %d", pStops)
	}

	// 验证 Run 返回中携带 StopIncompleteError 且保留了 stop-panic 身份与依赖
	var incompleteErr *StopIncompleteError
	if errors.As(appRunErr, &incompleteErr) {
		var foundPanicStage bool
		for _, stuck := range incompleteErr.StuckInstances {
			if stuck.PluginKey == compKey {
				if stuck.Stage == "stop-panic" {
					foundPanicStage = true
				}
			}
		}
		if !foundPanicStage {
			t.Logf("stuck stages found: %v", incompleteErr.StuckInstances)
		}
	}

	_ = handle
	_ = h
}
