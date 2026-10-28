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

// realExitTestHelper 提供通道屏障与 cleanup 注册，保证任何 Fatal 都能释放所有阻塞
type realExitTestHelper struct {
	mu       sync.Mutex
	cleanups []func()
}

func newRealExitTestHelper(t *testing.T) *realExitTestHelper {
	h := &realExitTestHelper{}
	t.Cleanup(func() {
		h.releaseAll()
	})
	return h
}

func (h *realExitTestHelper) register(fn func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cleanups = append(h.cleanups, fn)
}

func (h *realExitTestHelper) releaseAll() {
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

// -----------------------------------------------------------------------------
// 回归 A: StartTimeout 无外部 Shutdown 自行返回启动 deadline + StopIncomplete
// -----------------------------------------------------------------------------

type realExitProviderTool struct {
	resourceKey ctxkey.Key
	startCalled atomic.Int32
	stopCalled  atomic.Int32
}

func (p *realExitProviderTool) Start(ctx context.Context) error {
	p.startCalled.Add(1)
	return nil
}

func (p *realExitProviderTool) Stop(ctx context.Context) error {
	p.stopCalled.Add(1)
	return nil
}

func (p *realExitProviderTool) Expose() map[ctxkey.Key]any {
	return map[ctxkey.Key]any{
		p.resourceKey: "real_exit_resource_val",
	}
}

type realExitConsumerTool struct {
	startEntered chan struct{}
	releaseStart chan struct{}
	startCalled  atomic.Int32
	stopCalled   atomic.Int32
}

func (c *realExitConsumerTool) Start(ctx context.Context) error {
	c.startCalled.Add(1)
	select {
	case <-c.startEntered:
	default:
		close(c.startEntered)
	}
	// 忽略 ctx 阻塞，直到测试释放
	<-c.releaseStart
	return nil
}

func (c *realExitConsumerTool) Stop(ctx context.Context) error {
	c.stopCalled.Add(1)
	return nil
}

func TestKernelStartTimeout_NoExternalShutdown_RealExit(t *testing.T) {
	h := newRealExitTestHelper(t)

	providerKey := ctxkey.Key("test_kernel_real_exit_a_provider")
	consumerKey := ctxkey.Key("test_kernel_real_exit_a_consumer")
	resourceKey := ctxkey.Key("test_kernel_real_exit_a_resource")

	provider := &realExitProviderTool{resourceKey: resourceKey}

	releaseStart := make(chan struct{})
	var releaseOnce sync.Once
	h.register(func() {
		releaseOnce.Do(func() {
			close(releaseStart)
		})
	})

	consumer := &realExitConsumerTool{
		startEntered: make(chan struct{}),
		releaseStart: releaseStart,
	}

	// 设定短 StartTimeout 与 Shutdown 预算，Grace < Shutdown
	app := New(
		WithStartTimeout(60*time.Millisecond),
		WithGraceTimeout(40*time.Millisecond),
		WithShutdownTimeout(120*time.Millisecond),
		WithCleanupTimeout(40*time.Millisecond),
	)

	if err := app.RegisterTool(PluginInfo{
		Key:      providerKey,
		Provides: []ctxkey.Key{resourceKey},
		Mode:     ModeTool,
	}, func() Plugin {
		return provider
	}); err != nil {
		t.Fatalf("failed to register provider: %v", err)
	}

	if err := app.RegisterTool(PluginInfo{
		Key:      consumerKey,
		Requires: []ctxkey.Key{resourceKey},
		Mode:     ModeTool,
	}, func() Plugin {
		return consumer
	}); err != nil {
		t.Fatalf("failed to register consumer: %v", err)
	}

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	// 验证 consumer.Start 真实进入
	select {
	case <-consumer.startEntered:
	case <-time.After(1 * time.Second):
		t.Fatal("consumer.Start did not enter within barrier timeout")
	}

	// 在 consumer.Start 仍阻塞期间：
	// 1. consumer.Stop 严禁提前调用
	// 2. provider.Stop 严禁提前调用（依赖反向拓扑必须等待上层）
	if cStops := consumer.stopCalled.Load(); cStops != 0 {
		t.Fatalf("consumer.Stop called prematurely while Start is still blocked: got %d", cStops)
	}
	if pStops := provider.stopCalled.Load(); pStops != 0 {
		t.Fatalf("provider.Stop called prematurely while consumer.Start is still blocked: got %d", pStops)
	}

	// App.Run 必须完全自己触发超时并返回启动 deadline 超时 + StopIncomplete（不借助外界调用 Shutdown）
	select {
	case runErr := <-runErrCh:
		if runErr == nil {
			t.Fatal("expected App.Run to return error due to StartTimeout, got nil")
		}
		var incErr *StopIncompleteError
		if !errors.As(runErr, &incErr) {
			t.Fatalf("expected StopIncompleteError in App.Run return, got: %v", runErr)
		}
		if st := app.State(); st != StateIncomplete {
			t.Fatalf("expected app state StateIncomplete, got %v", st)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("App.Run hung indefinitely without external Shutdown")
	}

	// 释放 consumer Start，让其协程真正退出
	releaseOnce.Do(func() {
		close(releaseStart)
	})

	// 等待协程调度执行 safeCallStop 晚清理
	time.Sleep(50 * time.Millisecond)

	// consumer.Start 真实退出后，consumer 应被晚 Stop 且仅 Stop 一次
	if cStops := consumer.stopCalled.Load(); cStops != 1 {
		t.Fatalf("expected consumer.Stop to be called exactly once after real exit, got %d", cStops)
	}

	// 此时最终状态 StateIncomplete 不得回写为 StateStopped
	if st := app.State(); st != StateIncomplete {
		t.Fatalf("expected app state to remain StateIncomplete, got %v", st)
	}
}

// -----------------------------------------------------------------------------
// 回归 B: Worker 忽略取消、Stop 空返 nil，Shutdown 返回 Incomplete 且共享结果
// -----------------------------------------------------------------------------

type realExitMockDBProvider struct {
	dbKey      ctxkey.Key
	stopCalled atomic.Int32
}

func (p *realExitMockDBProvider) Start(ctx context.Context) error {
	return nil
}

func (p *realExitMockDBProvider) Stop(ctx context.Context) error {
	p.stopCalled.Add(1)
	return nil
}

func (p *realExitMockDBProvider) Expose() map[ctxkey.Key]any {
	return map[ctxkey.Key]any{p.dbKey: "mock_db_handle"}
}

type realExitStubWorker struct {
	runEntered chan struct{}
	releaseRun chan struct{}
	runDone    chan struct{}
	stopCalled chan struct{}
	stopCount  atomic.Int32
}

func (w *realExitStubWorker) Start(ctx context.Context) error {
	return nil
}

func (w *realExitStubWorker) Run(ctx context.Context) error {
	select {
	case <-w.runEntered:
	default:
		close(w.runEntered)
	}
	defer func() {
		select {
		case <-w.runDone:
		default:
			close(w.runDone)
		}
	}()
	// 忽略 lifetimeCtx 取消，直到测试显式释放屏障
	<-w.releaseRun
	return nil
}

func (w *realExitStubWorker) Stop(ctx context.Context) error {
	w.stopCount.Add(1)
	select {
	case <-w.stopCalled:
	default:
		close(w.stopCalled)
	}
	return nil
}

func TestKernelRealExit_WorkerIgnoresCancel_StopReturnsNil_ShutdownIncomplete(t *testing.T) {
	h := newRealExitTestHelper(t)

	dbKey := ctxkey.Key("test_kernel_real_exit_b_db")
	providerKey := ctxkey.Key("test_kernel_real_exit_b_db_provider")
	workerKey := ctxkey.Key("test_kernel_real_exit_b_worker")

	provider := &realExitMockDBProvider{dbKey: dbKey}

	releaseRun := make(chan struct{})
	var releaseRunOnce sync.Once
	h.register(func() {
		releaseRunOnce.Do(func() {
			close(releaseRun)
		})
	})

	worker := &realExitStubWorker{
		runEntered: make(chan struct{}),
		releaseRun: releaseRun,
		runDone:    make(chan struct{}),
		stopCalled: make(chan struct{}),
	}

	app := New(
		WithGraceTimeout(30*time.Millisecond),
		WithShutdownTimeout(100*time.Millisecond),
		WithCleanupTimeout(30*time.Millisecond),
	)

	if err := app.RegisterTool(PluginInfo{
		Key:      providerKey,
		Provides: []ctxkey.Key{dbKey},
		Mode:     ModeTool,
	}, func() Plugin {
		return provider
	}); err != nil {
		t.Fatalf("failed to register db tool: %v", err)
	}

	if err := app.RegisterWorker(PluginInfo{
		Key:      workerKey,
		Requires: []ctxkey.Key{dbKey},
		Mode:     ModeWorker,
	}, func() RunnablePlugin {
		return worker
	}); err != nil {
		t.Fatalf("failed to register worker: %v", err)
	}

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	// 确认 worker.Run 真实进入
	select {
	case err := <-runErrCh:
		t.Fatalf("app.Run exited unexpectedly before worker.Run entered: %v", err)
	case <-worker.runEntered:
	case <-time.After(1 * time.Second):
		t.Fatal("worker.Run did not enter")
	}

	// 主动调用 Shutdown（短预算），Run 仍在阻塞
	shutErrCh := make(chan error, 1)
	go func() {
		shutErrCh <- app.Shutdown()
	}()

	// 验证 HookStop 屏障已到达：Stop 已被调用但 Run 依然被测试阻塞
	select {
	case <-worker.stopCalled:
	case <-time.After(1 * time.Second):
		t.Fatal("worker.Stop was not called during shutdown")
	}

	// 等待 Shutdown 结算：由于 Run 尚未退出，Shutdown 必须返回 Incomplete 错误
	var firstShutErr error
	select {
	case firstShutErr = <-shutErrCh:
		if firstShutErr == nil {
			t.Fatal("expected Shutdown to return error due to blocking Run, got nil")
		}
		var incErr *StopIncompleteError
		if !errors.As(firstShutErr, &incErr) {
			t.Fatalf("expected StopIncompleteError, got %v", firstShutErr)
		}
		if st := app.State(); st != StateIncomplete {
			t.Fatalf("expected app state StateIncomplete, got %v", st)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Shutdown did not complete within budget")
	}

	// 下游 Worker 仍受阻未退役时，其依赖 provider 严禁 Stop
	if pStops := provider.stopCalled.Load(); pStops != 0 {
		t.Fatalf("provider Stop called while dependent Worker was stuck: %d", pStops)
	}

	// 验证重复 Shutdown 必须共享旧结果
	dupShutErr := app.Shutdown()
	if !errors.Is(dupShutErr, firstShutErr) && dupShutErr.Error() != firstShutErr.Error() {
		t.Fatalf("expected duplicate Shutdown to return shared result %v, got %v", firstShutErr, dupShutErr)
	}

	// 释放 Run 阻塞屏障，让 Worker 真正完成收尾
	releaseRunOnce.Do(func() {
		close(releaseRun)
	})

	select {
	case <-worker.runDone:
	case <-time.After(1 * time.Second):
		t.Fatal("worker Run did not exit after release")
	}
}

// -----------------------------------------------------------------------------
// 回归 C: 全局 Stopping 但父 Run 未响应：首 Component 占槽 + 次 Queue 已接纳
// -----------------------------------------------------------------------------

type realExitComponentPlugin struct {
	runEntered chan struct{}
	releaseRun chan struct{}
}

func (c *realExitComponentPlugin) Start(ctx context.Context) error {
	return nil
}

func (c *realExitComponentPlugin) Run(ctx context.Context) error {
	select {
	case <-c.runEntered:
	default:
		close(c.runEntered)
	}
	<-c.releaseRun
	return nil
}

func (c *realExitComponentPlugin) Stop(ctx context.Context) error {
	return nil
}

func TestKernelRealExit_StoppingBeforeParentExit_QueuedCancelledPromptly(t *testing.T) {
	h := newRealExitTestHelper(t)

	compKey := ctxkey.Key("test_kernel_real_exit_c_comp")
	parentWorkerKey := ctxkey.Key("test_kernel_real_exit_c_parent")

	compFactoryCalls := atomic.Int32{}

	releaseFirstRun := make(chan struct{})
	var releaseOnce sync.Once
	h.register(func() {
		releaseOnce.Do(func() {
			close(releaseFirstRun)
		})
	})

	firstEntered := make(chan struct{})

	// 注册 OverlapQueue 的 Component，MaxQueue=5
	app := New(
		WithGraceTimeout(40*time.Millisecond),
		WithShutdownTimeout(150*time.Millisecond),
	)

	err := app.RegisterComponent(PluginInfo{
		Key:     compKey,
		Overlap: OverlapPolicy{Kind: OverlapQueue, MaxQueue: 5},
	}, func() RunnablePlugin {
		call := compFactoryCalls.Add(1)
		if call == 1 {
			return &realExitComponentPlugin{
				runEntered: firstEntered,
				releaseRun: releaseFirstRun,
			}
		}
		return &realExitComponentPlugin{
			runEntered: make(chan struct{}),
			releaseRun: make(chan struct{}),
		}
	})
	if err != nil {
		t.Fatalf("failed to register component: %v", err)
	}

	parentStarted := make(chan struct{})
	parentRelease := make(chan struct{})
	var parentReleaseOnce sync.Once
	h.register(func() {
		parentReleaseOnce.Do(func() {
			close(parentRelease)
		})
	})

	var hQueued RunHandle
	queuedAccepted := make(chan struct{})

	err = app.RegisterWorker(PluginInfo{
		Key:  parentWorkerKey,
		Mode: ModeWorker,
	}, func() RunnablePlugin {
		return &stubWorker{
			runFn: func(ctx context.Context) error {
				close(parentStarted)

				// 1. 激活首个任务，占住唯一的执行槽
				h1, err := app.Activator().Activate(ctx, compKey)
				if err != nil {
					return err
				}
				_ = h1

				select {
				case <-firstEntered:
				case <-time.After(1 * time.Second):
					return errors.New("timeout waiting for first component run")
				}

				// 2. 激活次个任务，进入 OverlapQueue 排队
				h2, err := app.Activator().Activate(ctx, compKey)
				if err != nil {
					return err
				}
				hQueued = h2
				close(queuedAccepted)

				// 父 Worker 持续阻塞，直到测试在 Shutdown 触发后释放
				<-parentRelease
				return nil
			},
		}
	})
	if err != nil {
		t.Fatalf("failed to register parent worker: %v", err)
	}

	go func() {
		_ = app.Run()
	}()

	// 等待父 Worker 启动且次任务排队成功已接纳
	select {
	case <-parentStarted:
	case <-time.After(1 * time.Second):
		t.Fatal("parent worker did not start")
	}
	select {
	case <-queuedAccepted:
	case <-time.After(1 * time.Second):
		t.Fatal("queued task was not accepted")
	}

	// 此时工序：仅首个 Component 实例被创建
	if calls := compFactoryCalls.Load(); calls != 1 {
		t.Fatalf("expected exactly 1 component factory call, got %d", calls)
	}

	// 启动 App.Shutdown，全局进入 Stopping，但父 Run 尚未释放
	go func() {
		_ = app.Shutdown()
	}()

	// 验证：次排队句柄迅速被取消（Dispatched=false, RunErr=nil, CancelErr != nil）
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer waitCancel()
	resQueued, waitErr := hQueued.Wait(waitCtx)
	if waitErr != nil && !errors.Is(waitErr, context.Canceled) {
		t.Fatalf("unexpected waitErr: %v", waitErr)
	}

	if resQueued.Dispatched {
		t.Fatalf("expected queued task Dispatched=false, got true")
	}
	if resQueued.RunErr != nil {
		t.Fatalf("expected queued task RunErr=nil, got %v", resQueued.RunErr)
	}
	if !resQueued.CancelRequested || resQueued.CancelErr == nil {
		t.Fatalf("expected queued task CancelRequested=true and non-nil CancelErr, got CancelRequested=%v, CancelErr=%v",
			resQueued.CancelRequested, resQueued.CancelErr)
	}

	// 工厂调用次数必须依旧仅为 1（排队取消不调用工厂、Start、Run）
	if calls := compFactoryCalls.Load(); calls != 1 {
		t.Fatalf("expected factory calls to remain 1 after queue cancel, got %d", calls)
	}

	// 释放执行屏障，收尾
	releaseOnce.Do(func() {
		close(releaseFirstRun)
	})
	parentReleaseOnce.Do(func() {
		close(parentRelease)
	})
}
