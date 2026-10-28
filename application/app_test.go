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

// mockTool 实现基础 Tool
type mockTool struct {
	startFunc func(ctx context.Context) error
	stopFunc  func(ctx context.Context) error
}

func (m *mockTool) Start(ctx context.Context) error {
	if m.startFunc != nil {
		return m.startFunc(ctx)
	}
	return nil
}

func (m *mockTool) Stop(ctx context.Context) error {
	if m.stopFunc != nil {
		return m.stopFunc(ctx)
	}
	return nil
}

// mockExposingTool 实现带 Exposer 的 Tool
type mockExposingTool struct {
	mockTool
	exposeMap map[ctxkey.Key]any
}

func (m *mockExposingTool) Expose() map[ctxkey.Key]any {
	return m.exposeMap
}

// mockWorker 实现基础 Worker
type mockWorker struct {
	mockTool
	runFunc func(ctx context.Context) error
}

func (m *mockWorker) Run(ctx context.Context) error {
	if m.runFunc != nil {
		return m.runFunc(ctx)
	}
	return nil
}

// 1. 测试双 App 容器与 Context 完全隔离
func TestApp_DualAppIsolation(t *testing.T) {
	app1 := New()
	app2 := New()

	key := ctxkey.Key("shared_tool")
	val1 := "app1_secret"
	val2 := "app2_secret"

	_ = app1.RegisterTool(PluginInfo{Key: key}, func() Plugin {
		return &mockExposingTool{
			exposeMap: map[ctxkey.Key]any{},
		}
	})
	_ = app2.RegisterTool(PluginInfo{Key: key}, func() Plugin {
		return &mockExposingTool{
			exposeMap: map[ctxkey.Key]any{},
		}
	})

	app1.container.setEntry(key, key, val1, EntryReady)
	app2.container.setEntry(key, key, val2, EntryReady)

	ctx1 := context.WithValue(context.Background(), containerCtxKey, app1.container)
	ctx2 := context.WithValue(context.Background(), containerCtxKey, app2.container)

	res1, err1 := Get[string](ctx1, key)
	if err1 != nil || res1 != val1 {
		t.Fatalf("app1 lookup failed: got %v, err: %v", res1, err1)
	}

	res2, err2 := Get[string](ctx2, key)
	if err2 != nil || res2 != val2 {
		t.Fatalf("app2 lookup failed: got %v, err: %v", res2, err2)
	}
}

// 2. 测试配置快照与重复 Run 立即 ErrAlreadyRun
func TestApp_Run_OnceAndAlreadyRun(t *testing.T) {
	app := New()
	workerKey := ctxkey.Key("test_worker")
	barrier := make(chan struct{})

	_ = app.RegisterWorker(PluginInfo{Key: workerKey}, func() RunnablePlugin {
		return &mockWorker{
			runFunc: func(ctx context.Context) error {
				close(barrier)
				<-ctx.Done()
				return nil
			},
		}
	})

	errChan := make(chan error, 1)
	go func() {
		errChan <- app.Run()
	}()

	<-barrier
	// 第二次调用 Run 必须立即返回 ErrAlreadyRun
	err2 := app.Run()
	if !errors.Is(err2, ErrAlreadyRun) {
		t.Fatalf("expected ErrAlreadyRun, got %v", err2)
	}

	// 注册也必须关闭
	errReg := app.RegisterTool(PluginInfo{Key: "late_tool"}, func() Plugin { return &mockTool{} })
	if !errors.Is(errReg, ErrRegistrationClosed) {
		t.Fatalf("expected ErrRegistrationClosed, got %v", errReg)
	}

	_ = app.Shutdown()
	err1 := <-errChan
	if err1 != nil {
		t.Fatalf("expected clean run, got %v", err1)
	}
}

// 3. 测试 Run 之前 Shutdown 封存空 App，随后 Run 返回 ErrAppClosed
func TestApp_ShutdownBeforeRun_SealsApp(t *testing.T) {
	app := New()
	if err := app.Shutdown(); err != nil {
		t.Fatalf("shutdown before run should succeed, got %v", err)
	}
	if app.State() != StateStopped {
		t.Fatalf("expected StateStopped, got %v", app.State())
	}

	err := app.Run()
	if !errors.Is(err, ErrAppClosed) {
		t.Fatalf("expected ErrAppClosed, got %v", err)
	}
}

// 4. 测试启动期 Start 失败原地逆序回滚与错误保留
func TestApp_StartFailureRollback(t *testing.T) {
	app := New()
	toolKey1 := ctxkey.Key("db_tool")
	toolKey2 := ctxkey.Key("cache_tool")

	var stop1Called atomic.Bool
	var stop2Called atomic.Bool

	_ = app.RegisterTool(PluginInfo{Key: toolKey1}, func() Plugin {
		return &mockTool{
			startFunc: func(ctx context.Context) error { return nil },
			stopFunc: func(ctx context.Context) error {
				stop1Called.Store(true)
				return nil
			},
		}
	})

	startErr2 := errors.New("start failed on tool 2")
	_ = app.RegisterTool(PluginInfo{Key: toolKey2, Requires: []ctxkey.Key{toolKey1}}, func() Plugin {
		return &mockTool{
			startFunc: func(ctx context.Context) error {
				return startErr2
			},
			stopFunc: func(ctx context.Context) error {
				stop2Called.Store(true)
				return nil
			},
		}
	})

	err := app.Run()
	if !errors.Is(err, startErr2) {
		t.Fatalf("expected startErr2 preserved, got %v", err)
	}
	if !stop2Called.Load() {
		t.Fatalf("failed tool2 should have its Stop called")
	}
	if !stop1Called.Load() {
		t.Fatalf("previously started tool1 should be stopped in rollback")
	}
}

// 5. 测试 Worker 早退完成事件触发 ErrWorkerUnexpectedExit
func TestApp_WorkerUnexpectedExit(t *testing.T) {
	app := New()
	workerKey := ctxkey.Key("early_worker")

	_ = app.RegisterWorker(PluginInfo{Key: workerKey}, func() RunnablePlugin {
		return &mockWorker{
			runFunc: func(ctx context.Context) error {
				// 立即早退返回 nil
				return nil
			},
		}
	})

	err := app.Run()
	if !errors.Is(err, ErrWorkerUnexpectedExit) {
		t.Fatalf("expected ErrWorkerUnexpectedExit, got %v", err)
	}
}

// 6. 测试 Expose 原子校验与防半发布
func TestApp_ExposeValidationFailure(t *testing.T) {
	app := New()
	k := ctxkey.Key("exposing_plugin")
	res := ctxkey.Key("res_key")

	var stopCalled atomic.Bool

	_ = app.RegisterTool(PluginInfo{Key: k, Provides: []ctxkey.Key{res}}, func() Plugin {
		return &mockExposingTool{
			mockTool: mockTool{
				stopFunc: func(ctx context.Context) error {
					stopCalled.Store(true)
					return nil
				},
			},
			// 漏交付 res_key
			exposeMap: map[ctxkey.Key]any{},
		}
	})

	err := app.Run()
	if !errors.Is(err, ErrExposedKeysMismatch) {
		t.Fatalf("expected ErrExposedKeysMismatch, got %v", err)
	}
	if !stopCalled.Load() {
		t.Fatalf("tool should be stopped upon expose mismatch")
	}
	// 校验未在容器中半发布条目
	_, found := app.container.getEntry(res)
	if found {
		t.Fatalf("res_key should not be published in container")
	}
}

// 7. 测试 Stop panic：不重试、不退役、条目标记 EntryUnavailable 并报告 StuckInstance
func TestApp_StopPanic_RetainsDependency(t *testing.T) {
	app := New(
		WithGraceTimeout(200*time.Millisecond),
		WithCleanupTimeout(200*time.Millisecond),
		WithShutdownTimeout(1*time.Second),
	)
	dbKey := ctxkey.Key("db_provider")
	toolKey := ctxkey.Key("panicking_tool")
	workerKey := ctxkey.Key("controlling_worker")

	var dbStopCount atomic.Int64
	dbTool := &mockTool{
		stopFunc: func(ctx context.Context) error {
			dbStopCount.Add(1)
			return nil
		},
	}

	_ = app.RegisterTool(PluginInfo{Key: dbKey}, func() Plugin {
		return dbTool
	})

	_ = app.RegisterTool(PluginInfo{
		Key:      toolKey,
		Requires: []ctxkey.Key{dbKey},
	}, func() Plugin {
		return &mockTool{
			stopFunc: func(ctx context.Context) error {
				panic("stop critical hardware failure")
			},
		}
	})

	ready := make(chan struct{})
	stopRun := make(chan struct{})
	runDone := make(chan struct{})
	var stopOnce sync.Once

	worker := &mockWorker{
		runFunc: func(ctx context.Context) error {
			defer close(runDone)
			close(ready)
			select {
			case <-stopRun:
			case <-ctx.Done():
			}
			return nil
		},
	}
	worker.stopFunc = func(ctx context.Context) error {
		stopOnce.Do(func() {
			close(stopRun)
		})
		select {
		case <-runDone:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	_ = app.RegisterWorker(PluginInfo{Key: workerKey}, func() RunnablePlugin {
		return worker
	})

	errChan := make(chan error, 1)
	go func() {
		errChan <- app.Run()
	}()

	select {
	case <-ready:
	case err := <-errChan:
		t.Fatalf("app exited before worker ready: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for worker ready")
	}

	if shutdownErr := app.Shutdown(); shutdownErr != nil {
		// Stop panic 也会反映在关机错误中
		t.Logf("shutdown returned error: %v", shutdownErr)
	}

	var err error
	select {
	case err = <-errChan:
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout waiting for app.Run to finish after shutdown")
	}

	if err == nil {
		t.Fatalf("expected error from stop panic, got nil")
	}

	var incompleteErr *StopIncompleteError
	if !errors.As(err, &incompleteErr) {
		t.Fatalf("expected StopIncompleteError in run return, got %v", err)
	}
	if len(incompleteErr.StuckInstances) != 1 {
		t.Fatalf("expected 1 stuck instance, got %d", len(incompleteErr.StuckInstances))
	}
	if incompleteErr.StuckInstances[0].Stage != "stop-panic" {
		t.Fatalf("expected stage 'stop-panic', got %s", incompleteErr.StuckInstances[0].Stage)
	}
	if len(incompleteErr.StuckInstances[0].RetainedDeps) != 1 || incompleteErr.StuckInstances[0].RetainedDeps[0] != dbKey {
		t.Fatalf("expected retained deps [%v], got %v", dbKey, incompleteErr.StuckInstances[0].RetainedDeps)
	}

	// 被依赖的 provider 此时不应被执行 Stop
	if count := dbStopCount.Load(); count != 0 {
		t.Fatalf("expected db_provider Stop count to be 0 (retained), got %d", count)
	}

	// 容器中条目应为 EntryUnavailable 而非 EntryStopped
	entry, ok := app.container.getEntry(toolKey)
	if !ok || EntryState(entry.state) != EntryUnavailable {
		t.Fatalf("expected EntryUnavailable, got entry state %v", entry.state)
	}

	// 依赖 provider 仍保留在容器中且状态为 Ready，仍可供读取
	dbEntry, dbOk := app.container.getEntry(dbKey)
	if !dbOk || EntryState(dbEntry.state) != EntryReady {
		t.Fatalf("expected db_provider entry state EntryReady, got %v", dbEntry.state)
	}

	// 再次读取该组件应返回 ErrComponentUnavailable
	ctx := context.WithValue(context.Background(), containerCtxKey, app.container)
	_, getErr := Get[Plugin](ctx, toolKey)
	if !errors.Is(getErr, ErrComponentUnavailable) {
		t.Fatalf("expected ErrComponentUnavailable, got %v", getErr)
	}

	gotDB, getDBErr := Get[Plugin](ctx, dbKey)
	if getDBErr != nil {
		t.Fatalf("expected Get db_provider success, got %v", getDBErr)
	}
	if gotDB != dbTool {
		t.Fatalf("expected gotDB == dbTool instance")
	}
	if !errors.Is(getErr, ErrComponentUnavailable) {
		t.Fatalf("expected ErrComponentUnavailable, got %v", getErr)
	}
}

// 8. 测试 Context 白名单视图在 Start 阶段越权拦截
func TestApp_Context_RequiresStartScopeWhitelist(t *testing.T) {
	app := New()
	k1 := ctxkey.Key("db")
	k2 := ctxkey.Key("service")

	_ = app.RegisterTool(PluginInfo{Key: k1}, func() Plugin {
		return &mockTool{}
	})

	var undeclaredErr error
	_ = app.RegisterTool(PluginInfo{Key: k2}, func() Plugin { // 未在 Requires 声明 k1
		return &mockTool{
			startFunc: func(ctx context.Context) error {
				_, appErr := Get[Plugin](ctx, k1)
				if appErr != nil {
					undeclaredErr = appErr.Err
					return appErr.Err
				}
				return nil
			},
		}
	})

	err := app.Run()
	if !errors.Is(err, ErrUndeclaredDependency) {
		t.Fatalf("expected ErrUndeclaredDependency, got %v (inner err: %v)", err, undeclaredErr)
	}
}

// 9. 测试 BridgeRequest 与 Detach
func TestApp_BridgeRequestAndDetach(t *testing.T) {
	app := New()
	toolKey := ctxkey.Key("app_cache")
	toolVal := "redis_handle"

	app.container.setEntry(toolKey, toolKey, toolVal, EntryReady)

	workerLifetimeCtx := context.WithValue(context.Background(), containerCtxKey, app.container)
	workerLifetimeCtx = context.WithValue(workerLifetimeCtx, metaCtxKey, InstanceMeta{
		InstanceID: "worker_1",
		PluginKey:  "my_worker",
		TraceID:    "trace_w1",
	})

	// BridgeRequest
	type testContextKey string
	const tokenKey testContextKey = "user_token"
	reqCtx := context.WithValue(context.Background(), tokenKey, "abc-123")
	bridged := BridgeRequest(reqCtx, workerLifetimeCtx, "req_999")

	// 1. 可以读原生 reqCtx
	if bridged.Value(tokenKey) != "abc-123" {
		t.Fatalf("failed to read reqCtx value from bridged")
	}
	// 2. 可以读 RequestKey
	if bridged.Value(ctxkey.RequestKey) != "req_999" {
		t.Fatalf("failed to read requestID from bridged")
	}
	// 3. 可以读共享容器组件
	val, appErr := Get[string](bridged, toolKey)
	if appErr != nil || val != toolVal {
		t.Fatalf("failed to Get from bridged: %v, %v", val, appErr)
	}

	// Detach
	detached, cancel := Detach(bridged)
	defer cancel()

	// 1. 保留容器与追踪 ID
	valDetached, appErr := Get[string](detached, toolKey)
	if appErr != nil || valDetached != toolVal {
		t.Fatalf("failed to Get from detached: %v, %v", valDetached, appErr)
	}
	// 2. 原生 reqCtx 的 user_token 不保留
	if detached.Value("user_token") != nil {
		t.Fatalf("detached context should not retain user_token")
	}
}

// 10. 测试 GraceContext 和 CleanupContext 预算控制
func TestApp_BudgetCalculations(t *testing.T) {
	totalDeadline := time.Now().Add(500 * time.Millisecond)
	graceDeadline := time.Now().Add(100 * time.Millisecond)
	budget := &budgetMetadata{
		graceDeadline: graceDeadline,
		totalDeadline: totalDeadline,
		cleanupLimit:  50 * time.Millisecond,
	}

	stopCtx := context.WithValue(context.Background(), budgetCtxKey, budget)

	// GraceContext
	graceCtx, graceCancel := GraceContext(stopCtx)
	defer graceCancel()

	dl, ok := graceCtx.Deadline()
	if !ok || dl != graceDeadline {
		t.Fatalf("unexpected grace deadline: %v, ok: %v", dl, ok)
	}

	// CleanupContext
	cleanupCtx, cleanupCancel := CleanupContext(stopCtx)
	defer cleanupCancel()

	cleanDl, ok := cleanupCtx.Deadline()
	if !ok {
		t.Fatalf("cleanupCtx should have deadline")
	}
	if cleanDl.After(totalDeadline) {
		t.Fatalf("cleanupCtx deadline %v should not exceed totalDeadline %v", cleanDl, totalDeadline)
	}
}
