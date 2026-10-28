package async

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

type asyncLifecycleHelper struct {
	mu       sync.Mutex
	cleanups []func()
}

func newAsyncLifecycleHelper(t *testing.T) *asyncLifecycleHelper {
	h := &asyncLifecycleHelper{}
	t.Cleanup(func() {
		h.releaseAll()
	})
	return h
}

func (h *asyncLifecycleHelper) register(fn func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cleanups = append(h.cleanups, fn)
}

func (h *asyncLifecycleHelper) releaseAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i := len(h.cleanups) - 1; i >= 0; i-- {
		if fn := h.cleanups[i]; fn != nil {
			fn()
		}
	}
	h.cleanups = nil
}

type testResourceProviderTool struct {
	resourceKey ctxkey.Key
	startCalled atomic.Int32
	stopCalled  atomic.Int32
}

func (p *testResourceProviderTool) Start(_ context.Context) error {
	p.startCalled.Add(1)
	return nil
}

func (p *testResourceProviderTool) Stop(_ context.Context) error {
	p.stopCalled.Add(1)
	return nil
}

func (p *testResourceProviderTool) Expose() map[ctxkey.Key]any {
	return map[ctxkey.Key]any{
		p.resourceKey: "test_resource_val",
	}
}

type testLifecycleWorker struct {
	runFn func(ctx context.Context) error
}

func (w *testLifecycleWorker) Start(_ context.Context) error {
	return nil
}

func (w *testLifecycleWorker) Run(ctx context.Context) error {
	if w.runFn != nil {
		return w.runFn(ctx)
	}
	<-ctx.Done()
	return nil
}

func (w *testLifecycleWorker) Stop(_ context.Context) error {
	return nil
}

func TestAsyncDependencyLifecycle_CaseA_AppForceCancelsTask_OrderlyShutdown(t *testing.T) {
	newAsyncLifecycleHelper(t)

	resKey := ctxkey.Key("test_async_res_a")
	resProviderKey := ctxkey.Key("test_async_provider_a")
	workerKey := ctxkey.Key("test_async_worker_a")

	provider := &testResourceProviderTool{resourceKey: resKey}

	orderCh := make(chan string, 10)
	taskStarted := make(chan struct{})

	asyncInfo, asyncFactory := New(
		WithRequires(resKey),
	)

	app := application.New(
		application.WithGraceTimeout(20*time.Millisecond),
		application.WithCleanupTimeout(100*time.Millisecond),
		application.WithShutdownTimeout(150*time.Millisecond),
	)
	t.Cleanup(func() {
		_ = app.Shutdown()
	})

	if err := app.RegisterTool(application.PluginInfo{
		Key:      resProviderKey,
		Provides: []ctxkey.Key{resKey},
		Mode:     application.ModeTool,
	}, func() application.Plugin {
		return provider
	}); err != nil {
		t.Fatalf("RegisterTool provider failed: %v", err)
	}

	if err := app.RegisterTool(asyncInfo, asyncFactory); err != nil {
		t.Fatalf("RegisterTool async failed: %v", err)
	}

	worker := &testLifecycleWorker{
		runFn: func(lifetimeCtx context.Context) error {
			task, err := Submit(lifetimeCtx, func(taskCtx context.Context) {
				close(taskStarted)
				<-taskCtx.Done()
				orderCh <- "task_done"
			})
			if err != nil {
				return err
			}
			return task.Wait()
		},
	}

	if err := app.RegisterWorker(application.PluginInfo{
		Key:      workerKey,
		Requires: []ctxkey.Key{PoolKey},
		Mode:     application.ModeWorker,
	}, func() application.RunnablePlugin {
		return worker
	}); err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	select {
	case <-taskStarted:
	case <-time.After(1 * time.Second):
		t.Fatal("task did not start in time")
	}

	if err := app.Shutdown(); err != nil {
		t.Fatalf("expected clean Shutdown, got: %v", err)
	}

	if err := <-runErrCh; err != nil {
		t.Fatalf("app.Run returned error: %v", err)
	}

	var firstEvent string
	select {
	case firstEvent = <-orderCh:
	default:
		t.Fatal("expected task_done event recorded in chan")
	}

	if firstEvent != "task_done" {
		t.Fatalf("expected first event task_done, got %s", firstEvent)
	}

	if pStops := provider.stopCalled.Load(); pStops != 1 {
		t.Fatalf("expected provider Stop exactly once, got %d", pStops)
	}
	if st := app.State(); st != application.StateStopped {
		t.Fatalf("expected StateStopped, got %v", st)
	}
}

func TestAsyncDependencyLifecycle_CaseB_TaskIgnoresCancel_RetainsResource_Incomplete(t *testing.T) {
	h := newAsyncLifecycleHelper(t)

	resKey := ctxkey.Key("test_async_res_b")
	resProviderKey := ctxkey.Key("test_async_provider_b")
	workerKey := ctxkey.Key("test_async_worker_b")

	provider := &testResourceProviderTool{resourceKey: resKey}

	releaseTask := make(chan struct{})
	var releaseOnce sync.Once
	h.register(func() {
		releaseOnce.Do(func() {
			close(releaseTask)
		})
	})

	taskStarted := make(chan struct{})
	taskCh := make(chan *Task, 1)

	asyncInfo, asyncFactory := New(
		WithRequires(resKey),
	)

	app := application.New(
		application.WithGraceTimeout(20*time.Millisecond),
		application.WithCleanupTimeout(100*time.Millisecond),
		application.WithShutdownTimeout(150*time.Millisecond),
	)
	runErrCh := make(chan error, 1)
	t.Cleanup(func() {
		releaseOnce.Do(func() {
			close(releaseTask)
		})
		_ = app.Shutdown()
		<-runErrCh
	})

	if err := app.RegisterTool(application.PluginInfo{
		Key:      resProviderKey,
		Provides: []ctxkey.Key{resKey},
		Mode:     application.ModeTool,
	}, func() application.Plugin {
		return provider
	}); err != nil {
		t.Fatalf("RegisterTool provider failed: %v", err)
	}

	if err := app.RegisterTool(asyncInfo, asyncFactory); err != nil {
		t.Fatalf("RegisterTool async failed: %v", err)
	}

	worker := &testLifecycleWorker{
		runFn: func(lifetimeCtx context.Context) error {
			resVal, resErr := application.Get[string](lifetimeCtx, resKey)
			if resErr != nil {
				return resErr
			}
			tSub, submitErr := Submit(lifetimeCtx, func(taskCtx context.Context) {
				close(taskStarted)
				<-releaseTask
				_ = resVal
			})
			if submitErr != nil {
				return submitErr
			}
			taskCh <- tSub
			<-lifetimeCtx.Done()
			return nil
		},
	}

	if err := app.RegisterWorker(application.PluginInfo{
		Key:      workerKey,
		Requires: []ctxkey.Key{PoolKey, resKey},
		Mode:     application.ModeWorker,
	}, func() application.RunnablePlugin {
		return worker
	}); err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	go func() {
		runErrCh <- app.Run()
	}()

	select {
	case <-taskStarted:
	case <-time.After(1 * time.Second):
		t.Fatal("task did not start in time")
	}

	var taskRef *Task
	select {
	case taskRef = <-taskCh:
	case <-time.After(1 * time.Second):
		t.Fatal("did not receive taskRef in time")
	}

	waitCtx, waitCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer waitCancel()
	if err := taskRef.Wait(waitCtx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected Wait timeout DeadlineExceeded, got: %v", err)
	}
	if taskRef.IsDone() {
		t.Fatal("task must not be killed by Wait timeout")
	}

	shutErr := app.Shutdown()
	if shutErr == nil {
		t.Fatal("expected Shutdown to return StopIncompleteError, got nil")
	}
	var incErr *application.StopIncompleteError
	if !errors.As(shutErr, &incErr) {
		t.Fatalf("expected StopIncompleteError, got: %v", shutErr)
	}

	var foundAsyncStuck bool
	for _, stuck := range incErr.StuckInstances {
		if stuck.PluginKey == PluginKey {
			foundAsyncStuck = true
			var hasResKey bool
			for _, dep := range stuck.RetainedDeps {
				if dep == resKey {
					hasResKey = true
					break
				}
			}
			if !hasResKey {
				t.Fatalf("expected retained deps to contain %v, got %v", resKey, stuck.RetainedDeps)
			}
		}
	}
	if !foundAsyncStuck {
		t.Fatalf("expected StuckInstances to contain async plugin, got %v", incErr.StuckInstances)
	}

	if pStops := provider.stopCalled.Load(); pStops != 0 {
		t.Fatalf("provider Stop called while task still holding resource: %d", pStops)
	}

	dupShutErr := app.Shutdown()
	if !errors.Is(dupShutErr, shutErr) && dupShutErr.Error() != shutErr.Error() {
		t.Fatalf("expected duplicate Shutdown to return frozen result, got %v", dupShutErr)
	}
	if st := app.State(); st != application.StateIncomplete {
		t.Fatalf("expected StateIncomplete, got %v", st)
	}

	releaseOnce.Do(func() {
		close(releaseTask)
	})

	if err := taskRef.Wait(); err != nil {
		t.Fatalf("task.Wait failed after release: %v", err)
	}
	if !taskRef.IsDone() {
		t.Fatal("expected task to be done after release")
	}
}

func TestAsyncDependencyLifecycle_CaseC_CtxDoneBeforeTaskDone_OriginCancelIsolated(t *testing.T) {
	newAsyncLifecycleHelper(t)

	resKey := ctxkey.Key("test_async_res_c")
	resProviderKey := ctxkey.Key("test_async_provider_c")
	workerKey := ctxkey.Key("test_async_worker_c")

	provider := &testResourceProviderTool{resourceKey: resKey}

	type workerReadyPayload struct {
		pool        Pool
		lifetimeCtx context.Context
	}
	readyCh := make(chan workerReadyPayload, 1)

	asyncInfo, asyncFactory := New(
		WithRequires(resKey),
	)

	app := application.New(
		application.WithGraceTimeout(20*time.Millisecond),
		application.WithCleanupTimeout(100*time.Millisecond),
		application.WithShutdownTimeout(150*time.Millisecond),
	)
	runErrCh := make(chan error, 1)
	t.Cleanup(func() {
		_ = app.Shutdown()
		<-runErrCh
	})

	if err := app.RegisterTool(application.PluginInfo{
		Key:      resProviderKey,
		Provides: []ctxkey.Key{resKey},
		Mode:     application.ModeTool,
	}, func() application.Plugin {
		return provider
	}); err != nil {
		t.Fatalf("RegisterTool provider failed: %v", err)
	}

	if err := app.RegisterTool(asyncInfo, asyncFactory); err != nil {
		t.Fatalf("RegisterTool async failed: %v", err)
	}

	worker := &testLifecycleWorker{
		runFn: func(lifetimeCtx context.Context) error {
			p, err := application.Get[Pool](lifetimeCtx, PoolKey)
			if err != nil {
				return err
			}
			readyCh <- workerReadyPayload{
				pool:        p,
				lifetimeCtx: lifetimeCtx,
			}
			<-lifetimeCtx.Done()
			return nil
		},
	}

	if err := app.RegisterWorker(application.PluginInfo{
		Key:      workerKey,
		Requires: []ctxkey.Key{PoolKey},
		Mode:     application.ModeWorker,
	}, func() application.RunnablePlugin {
		return worker
	}); err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	go func() {
		runErrCh <- app.Run()
	}()

	var ready workerReadyPayload
	select {
	case ready = <-readyCh:
	case <-time.After(1 * time.Second):
		t.Fatal("worker did not get ready payload in time")
	}

	poolRef := ready.pool
	workerCtx := ready.lifetimeCtx

	// 1. 原请求 cancel 仍不误杀任务
	reqCtx, reqCancel := context.WithCancel(context.Background())
	bridgeCtx := application.BridgeRequest(reqCtx, workerCtx, "trace_c_1")

	cancelDone := make(chan struct{})
	taskRunDone := make(chan struct{})
	task1, err := poolRef.Submit(bridgeCtx, func(taskCtx context.Context) {
		<-cancelDone
		if taskCtx.Err() != nil {
			t.Errorf("detached context should not be canceled by reqCancel: %v", taskCtx.Err())
		}
		close(taskRunDone)
	})
	if err != nil {
		t.Fatalf("Submit task1 failed: %v", err)
	}
	reqCancel()
	close(cancelDone)

	select {
	case <-taskRunDone:
	case <-time.After(1 * time.Second):
		t.Fatal("task1 was erroneously killed or timed out")
	}
	if err := task1.Wait(); err != nil {
		t.Fatalf("task1.Wait failed: %v", err)
	}

	// 2. 正常返回下 Ctx().Done 先于 Task.Done
	task2, err := poolRef.Submit(bridgeCtx, func(taskCtx context.Context) {
		// normal return
	})
	if err != nil {
		t.Fatalf("Submit task2 failed: %v", err)
	}
	select {
	case <-task2.Done():
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for task2.Done")
	}
	if err := task2.Wait(); err != nil {
		t.Fatalf("task2.Wait failed: %v", err)
	}
	select {
	case <-task2.Ctx().Done():
		if task2.Ctx().Err() == nil {
			t.Fatal("expected task2.Ctx().Err() != nil after Ctx().Done")
		}
	default:
		t.Fatal("expected task2.Ctx().Done() to already be closed after task2.Done()")
	}

	// 3. panic 恢复下 Ctx().Done 同样先于 Task.Done
	task3, err := poolRef.Submit(bridgeCtx, func(taskCtx context.Context) {
		panic("panic_test_c")
	})
	if err != nil {
		t.Fatalf("Submit task3 failed: %v", err)
	}
	select {
	case <-task3.Done():
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for task3.Done")
	}
	_ = task3.Wait()
	select {
	case <-task3.Ctx().Done():
		if task3.Ctx().Err() == nil {
			t.Fatal("expected task3.Ctx().Err() != nil after Ctx().Done")
		}
	default:
		t.Fatal("expected task3.Ctx().Done() to already be closed after task3.Done()")
	}
	if !task3.Panicked() {
		t.Fatal("expected task3.Panicked() to be true")
	}
	if len(task3.PanicStack()) == 0 {
		t.Fatal("expected task3.PanicStack() to not be empty")
	}
}
