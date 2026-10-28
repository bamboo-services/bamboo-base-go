package async

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

type testWorker struct {
	startedCh chan context.Context
	stopCh    chan struct{}
}

func (w *testWorker) Start(_ context.Context) error {
	return nil
}

func (w *testWorker) Run(lifetimeCtx context.Context) error {
	w.startedCh <- lifetimeCtx
	<-w.stopCh
	return nil
}

func (w *testWorker) Stop(_ context.Context) error {
	close(w.stopCh)
	return nil
}

func setupTestApp(t *testing.T, asyncOpts ...Option) (*application.App, context.Context) {
	t.Helper()
	info, factory := New(asyncOpts...)
	app := application.New()
	if err := app.RegisterTool(info, factory); err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	workerStartedCh := make(chan context.Context, 1)
	workerInfo := application.PluginInfo{
		Key:      "test_worker",
		Requires: []ctxkey.Key{PoolKey},
	}
	workerFactory := func() application.RunnablePlugin {
		return &testWorker{
			startedCh: workerStartedCh,
			stopCh:    make(chan struct{}),
		}
	}
	if err := app.RegisterWorker(workerInfo, workerFactory); err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	appRunErr := make(chan error, 1)
	go func() {
		appRunErr <- app.Run()
	}()

	var lifetimeCtx context.Context
	select {
	case lifetimeCtx = <-workerStartedCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for worker to receive lifetimeCtx")
	}

	reqCtx := context.WithValue(context.Background(), ctxkey.RequestKey, "test-trace-12345")
	bridgeCtx := application.BridgeRequest(reqCtx, lifetimeCtx, "test-trace-12345")

	return app, bridgeCtx
}

func TestPlugin_NewAndInfo(t *testing.T) {
	info, factory := New(
		WithName("custom_async"),
		WithCapacity(50),
		WithRequires(ctxkey.DatabaseKey, ctxkey.RedisClientKey),
	)

	if info.Key != PluginKey {
		t.Errorf("expected PluginKey %s, got %s", PluginKey, info.Key)
	}
	if info.Mode != application.ModeTool {
		t.Errorf("expected ModeTool, got %v", info.Mode)
	}
	if len(info.Provides) != 1 || info.Provides[0] != PoolKey {
		t.Errorf("expected Provides [PoolKey], got %v", info.Provides)
	}
	if len(info.Requires) != 2 {
		t.Errorf("expected 2 Requires, got %v", info.Requires)
	}

	p := factory()
	if p == nil {
		t.Fatal("factory returned nil plugin")
	}
}

func TestPlugin_BasicExecution(t *testing.T) {
	app, ctx := setupTestApp(t)
	defer func() {
		_ = app.Shutdown()
	}()

	executed := make(chan struct{})
	task, err := Submit(ctx, func(taskCtx context.Context) {
		close(executed)
	}, WithTaskName("basic_task"))

	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if task.Name() != "basic_task" {
		t.Errorf("expected task name 'basic_task', got %s", task.Name())
	}

	select {
	case <-executed:
	case <-time.After(1 * time.Second):
		t.Fatal("task was not executed in time")
	}

	if err := task.Wait(); err != nil {
		t.Errorf("task.Wait returned error: %v", err)
	}
	if !task.IsDone() {
		t.Error("task.IsDone should be true")
	}
}

func TestPlugin_ContextDetachedFromParentCancel(t *testing.T) {
	app, baseCtx := setupTestApp(t)
	defer func() {
		_ = app.Shutdown()
	}()

	parentCtx, parentCancel := context.WithCancel(baseCtx)

	taskExecuted := make(chan struct{})
	task, err := Submit(parentCtx, func(taskCtx context.Context) {
		time.Sleep(50 * time.Millisecond)
		// 校验任务的 ctx 未被父 context 的取消所影响
		if taskCtx.Err() != nil {
			t.Errorf("detached context should not be canceled, got: %v", taskCtx.Err())
		}
		close(taskExecuted)
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	// 立即取消原父请求 Context
	parentCancel()

	select {
	case <-taskExecuted:
	case <-time.After(1 * time.Second):
		t.Fatal("task did not complete after parent cancellation")
	}

	if err := task.Wait(); err != nil {
		t.Fatalf("Wait failed: %v", err)
	}
}

func TestPlugin_RequestKeyCopied(t *testing.T) {
	app, ctx := setupTestApp(t)
	defer func() {
		_ = app.Shutdown()
	}()

	gotTraceID := make(chan string, 1)
	_, err := Submit(ctx, func(taskCtx context.Context) {
		if val := taskCtx.Value(ctxkey.RequestKey); val != nil {
			if s, ok := val.(string); ok {
				gotTraceID <- s
				return
			}
		}
		gotTraceID <- ""
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	select {
	case trace := <-gotTraceID:
		if trace == "" {
			t.Errorf("expected non-empty RequestKey from parent context, got empty")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for traceID")
	}
}

func TestPlugin_TaskCancel(t *testing.T) {
	app, ctx := setupTestApp(t)
	defer func() {
		_ = app.Shutdown()
	}()

	canceled := make(chan struct{})
	task, err := Submit(ctx, func(taskCtx context.Context) {
		<-taskCtx.Done()
		close(canceled)
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	task.Cancel()

	select {
	case <-canceled:
	case <-time.After(1 * time.Second):
		t.Fatal("task was not canceled within 1s")
	}
}

func TestPlugin_WaitWithTimeoutDoesNotCancelTask(t *testing.T) {
	app, ctx := setupTestApp(t)
	defer func() {
		_ = app.Shutdown()
	}()

	taskContinued := make(chan struct{})
	task, err := Submit(ctx, func(taskCtx context.Context) {
		time.Sleep(100 * time.Millisecond)
		// 校验 Wait 超时并没有取消 taskCtx
		if taskCtx.Err() == nil {
			close(taskContinued)
		}
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	// 等待 20ms，必触发超时
	waitCtx, cancelWait := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancelWait()

	err = task.Wait(waitCtx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}

	// 任务依然在后台运行且未被取消
	select {
	case <-taskContinued:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("task was killed or did not finish after Wait timeout")
	}

	if err := task.Wait(); err != nil {
		t.Errorf("final Wait failed: %v", err)
	}
}

func TestPlugin_PanicRecovery(t *testing.T) {
	app, ctx := setupTestApp(t)
	defer func() {
		_ = app.Shutdown()
	}()

	task, err := Submit(ctx, func(taskCtx context.Context) {
		panic("boom in task")
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	if err := task.Wait(); err != nil {
		t.Errorf("task.Wait returned error on panic: %v", err)
	}

	if !task.IsDone() {
		t.Error("task should be marked done after panic")
	}
	if !task.Panicked() {
		t.Error("task.Panicked() should be true")
	}
	if val := task.PanicValue(); val != "boom in task" {
		t.Errorf("expected panic value 'boom in task', got: %v", val)
	}
	if len(task.PanicStack()) == 0 {
		t.Error("task.PanicStack() should not be empty")
	}
}

func TestPlugin_PoolFullRejection(t *testing.T) {
	// 容量为 1 的小池
	app, ctx := setupTestApp(t, WithCapacity(1))
	defer func() {
		_ = app.Shutdown()
	}()

	releaseBlock := make(chan struct{})
	started := make(chan struct{})

	// 提交第一个任务占满池
	task1, err := Submit(ctx, func(taskCtx context.Context) {
		close(started)
		<-releaseBlock
	})
	if err != nil {
		t.Fatalf("Submit task1 failed: %v", err)
	}

	<-started

	// 提交第二个任务，由于非阻塞有界池已满，应直接返回 ErrPoolFull
	_, err = Submit(ctx, func(taskCtx context.Context) {})
	if !errors.Is(err, ErrPoolFull) {
		t.Fatalf("expected ErrPoolFull, got: %v", err)
	}

	// 释放第一个任务
	close(releaseBlock)
	_ = task1.Wait()

	// 池空出后，再次提交应恢复正常
	var task2Ok bool
	task2, err := Submit(ctx, func(taskCtx context.Context) {
		task2Ok = true
	})
	if err != nil {
		t.Fatalf("Submit task2 failed after free: %v", err)
	}
	_ = task2.Wait()
	if !task2Ok {
		t.Error("task2 did not run")
	}
}

func TestPlugin_RejectDuringStoppingAndClosed(t *testing.T) {
	app, ctx := setupTestApp(t)

	taskBlock := make(chan struct{})
	started := make(chan struct{})

	task, err := Submit(ctx, func(taskCtx context.Context) {
		close(started)
		<-taskBlock
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	<-started

	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- app.Shutdown()
	}()

	// 等待 App 状态转为 Stopping
	for app.State() != application.StateStopping && app.State() != application.StateStopped {
		time.Sleep(5 * time.Millisecond)
	}

	// 此时提交新任务应被立即拒绝为 ErrPoolClosed
	_, err = Submit(ctx, func(taskCtx context.Context) {})
	if !errors.Is(err, ErrPoolClosed) {
		t.Fatalf("expected ErrPoolClosed during stopping, got: %v", err)
	}

	// 释放正运行的任务，让 Shutdown 正常完成
	close(taskBlock)
	_ = task.Wait()

	if err := <-shutdownDone; err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// 停机后再次提交依然被拒绝
	_, err = Submit(ctx, func(taskCtx context.Context) {})
	if !errors.Is(err, ErrPoolClosed) {
		t.Fatalf("expected ErrPoolClosed after shutdown, got: %v", err)
	}
}

func TestPlugin_ForceCancelReceivedByTask(t *testing.T) {
	app, ctx := setupTestApp(t)
	defer func() {
		_ = app.Shutdown()
	}()

	p, getErr := application.Get[Pool](ctx, PoolKey)
	if getErr != nil {
		t.Fatalf("Get[Pool] failed: %v", getErr)
	}

	forceCanceled := make(chan struct{})
	task, err := p.Submit(ctx, func(taskCtx context.Context) {
		<-taskCtx.Done()
		close(forceCanceled)
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	task.Cancel()

	select {
	case <-forceCanceled:
	case <-time.After(1 * time.Second):
		t.Fatal("task did not receive cancel")
	}

	_ = task.Wait()
}

func TestPlugin_IgnoreCancelTaskMakesStopWait(t *testing.T) {
	app, ctx := setupTestApp(t, WithCapacity(10))

	p, getErr := application.Get[Pool](ctx, PoolKey)
	if getErr != nil {
		t.Fatalf("Get[Pool] failed: %v", getErr)
	}

	taskBlock := make(chan struct{})
	started := make(chan struct{})

	task, err := p.Submit(ctx, func(taskCtx context.Context) {
		close(started)
		<-taskBlock
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	<-started

	stopReturned := make(chan error, 1)
	go func() {
		stopReturned <- p.(*Plugin).Stop(context.Background())
	}()

	select {
	case <-stopReturned:
		t.Fatal("Stop must not return while active task is still running")
	case <-time.After(50 * time.Millisecond):
	}

	close(taskBlock)
	_ = task.Wait()

	select {
	case err := <-stopReturned:
		if err != nil {
			t.Fatalf("Stop failed: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Stop did not return after task finished")
	}

	if err := p.(*Plugin).Stop(context.Background()); err != nil {
		t.Fatalf("subsequent Stop failed: %v", err)
	}
	_ = app.Shutdown()
}

func TestPlugin_SubmitRollbackOnFailure(t *testing.T) {
	app, ctx := setupTestApp(t, WithCapacity(1))
	defer func() {
		_ = app.Shutdown()
	}()

	p, getErr := application.Get[Pool](ctx, PoolKey)
	if getErr != nil {
		t.Fatalf("Get[Pool] failed: %v", getErr)
	}

	// 提交 nil 函数
	_, err := p.Submit(ctx, nil)
	if !errors.Is(err, ErrNilTaskFunc) {
		t.Fatalf("expected ErrNilTaskFunc, got: %v", err)
	}
	if p.Running() != 0 {
		t.Errorf("expected running count 0 after nil func, got %d", p.Running())
	}

	// 占满池后再次提交
	blockCh := make(chan struct{})
	t1, err := p.Submit(ctx, func(c context.Context) {
		<-blockCh
	})
	if err != nil {
		t.Fatalf("t1 submit failed: %v", err)
	}

	_, err = p.Submit(ctx, func(c context.Context) {})
	if !errors.Is(err, ErrPoolFull) {
		t.Fatalf("expected ErrPoolFull, got: %v", err)
	}
	if p.Running() != 1 {
		t.Errorf("expected running count 1 after full reject, got %d", p.Running())
	}

	close(blockCh)
	_ = t1.Wait()
}

func TestPlugin_CrossAppAndBackgroundRejection(t *testing.T) {
	appA, ctxA := setupTestApp(t)
	defer func() {
		_ = appA.Shutdown()
	}()

	appB, ctxB := setupTestApp(t)
	defer func() {
		_ = appB.Shutdown()
	}()

	poolA, errA := application.Get[Pool](ctxA, PoolKey)
	if errA != nil {
		t.Fatalf("Get[Pool] for A failed: %v", errA)
	}
	poolB, errB := application.Get[Pool](ctxB, PoolKey)
	if errB != nil {
		t.Fatalf("Get[Pool] for B failed: %v", errB)
	}

	// 1. AppA 正常通过自己的 context 提交
	aDone := make(chan struct{})
	taskA, err := poolA.Submit(ctxA, func(c context.Context) {
		close(aDone)
	})
	if err != nil {
		t.Fatalf("poolA.Submit with ctxA failed: %v", err)
	}
	<-aDone
	_ = taskA.Wait()

	// 安全等待 worker 归还池中，确保 running 计数安全归零
	deadline := time.Now().Add(500 * time.Millisecond)
	for poolA.Running() != 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if r := poolA.Running(); r != 0 {
		t.Fatalf("poolA.Running() expected 0 after taskA finish, got %d", r)
	}

	// 2. 直持 AppA Pool 但传入 AppB running ctx：必须被拒绝且不增加 A 的计数
	runningBefore := poolA.Running()
	_, errCross := poolA.Submit(ctxB, func(c context.Context) {
		t.Error("cross app task must not execute")
	})
	if !errors.Is(errCross, ErrInvalidContext) {
		t.Fatalf("expected ErrInvalidContext for cross app submit, got: %v", errCross)
	}
	if poolA.Running() != runningBefore || poolA.Running() != 0 {
		t.Errorf("poolA.Running() must not increase on cross app reject, got %d", poolA.Running())
	}

	// 3. 传入 context.Background() (缺容器 / ok=false)：必须拒绝且无任务执行
	_, errBg := poolA.Submit(context.Background(), func(c context.Context) {
		t.Error("background task must not execute")
	})
	if !errors.Is(errBg, ErrInvalidContext) {
		t.Fatalf("expected ErrInvalidContext for background ctx, got: %v", errBg)
	}
	if poolA.Running() != runningBefore || poolA.Running() != 0 {
		t.Errorf("poolA.Running() must not increase on background reject, got %d", poolA.Running())
	}

	// 4. 公开 Meta 伪造无效测试：构造带伪造 RequestKey 或类似 Meta 但未挂载对应 Pool 的 ctx
	fakeCtx := context.WithValue(context.Background(), ctxkey.RequestKey, "fake-meta-trace")
	_, errFake := poolA.Submit(fakeCtx, func(c context.Context) {
		t.Error("fake meta task must not execute")
	})
	if !errors.Is(errFake, ErrInvalidContext) {
		t.Fatalf("expected ErrInvalidContext for forged meta ctx, got: %v", errFake)
	}
	if poolA.Running() != runningBefore || poolA.Running() != 0 {
		t.Errorf("poolA.Running() must not increase on forged meta reject, got %d", poolA.Running())
	}

	_ = poolB
}
