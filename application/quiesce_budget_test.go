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

// -----------------------------------------------------------------------------
// 1. Quiesce 超时与依赖保留测试
// -----------------------------------------------------------------------------

type quiesceTestProviderTool struct {
	resKey    ctxkey.Key
	stopCalls atomic.Int32
}

func (p *quiesceTestProviderTool) Start(ctx context.Context) error {
	return nil
}

func (p *quiesceTestProviderTool) Stop(ctx context.Context) error {
	p.stopCalls.Add(1)
	return nil
}

func (p *quiesceTestProviderTool) Expose() map[ctxkey.Key]any {
	return map[ctxkey.Key]any{
		p.resKey: "provider_resource",
	}
}

type quiesceBlockingWorker struct {
	startDone      chan struct{}
	releaseQuiesce chan struct{}
	stopCalls      atomic.Int32
}

func (q *quiesceBlockingWorker) Start(ctx context.Context) error {
	return nil
}

func (q *quiesceBlockingWorker) Run(ctx context.Context) error {
	select {
	case <-q.startDone:
	default:
		close(q.startDone)
	}
	<-ctx.Done()
	return nil
}

func (q *quiesceBlockingWorker) Quiesce(ctx context.Context) error {
	// 忽略 ctx，持续阻塞直至测试显式释放屏障
	<-q.releaseQuiesce
	return nil
}

func (q *quiesceBlockingWorker) Stop(ctx context.Context) error {
	q.stopCalls.Add(1)
	return nil
}

func TestQuiesceBudget_RetainedDepsAndStuckInstance(t *testing.T) {
	pKey := ctxkey.Key("test_quiesce_p_tool")
	qKey := ctxkey.Key("test_quiesce_q_worker")
	resKey := ctxkey.Key("test_quiesce_res_p")

	provider := &quiesceTestProviderTool{resKey: resKey}

	startDone := make(chan struct{})
	releaseQuiesce := make(chan struct{})
	var releaseOnce sync.Once
	cleanupFn := func() {
		releaseOnce.Do(func() {
			close(releaseQuiesce)
		})
	}
	t.Cleanup(cleanupFn)

	worker := &quiesceBlockingWorker{
		startDone:      startDone,
		releaseQuiesce: releaseQuiesce,
	}

	app := New(
		WithGraceTimeout(20*time.Millisecond),
		WithCleanupTimeout(20*time.Millisecond),
		WithShutdownTimeout(80*time.Millisecond),
	)

	if err := app.RegisterTool(PluginInfo{
		Key:      pKey,
		Provides: []ctxkey.Key{resKey},
		Mode:     ModeTool,
	}, func() Plugin {
		return provider
	}); err != nil {
		t.Fatalf("failed to register provider: %v", err)
	}

	if err := app.RegisterWorker(PluginInfo{
		Key:      qKey,
		Requires: []ctxkey.Key{resKey},
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

	// 真实 Run 屏障
	select {
	case <-startDone:
	case <-time.After(1 * time.Second):
		t.Fatal("worker did not start within timeout")
	}

	// 触发关机，预算内必须返回
	startShutdown := time.Now()
	shutErr := app.Shutdown()
	shutdownDuration := time.Since(startShutdown)
	if shutdownDuration > 500*time.Millisecond {
		t.Fatalf("shutdown exceeded total budget: %v", shutdownDuration)
	}

	if shutErr == nil {
		t.Fatal("expected Shutdown to return error due to quiesce timeout, got nil")
	}

	var incompleteErr *StopIncompleteError
	if !errors.As(shutErr, &incompleteErr) {
		t.Fatalf("expected StopIncompleteError, got: %v", shutErr)
	}

	// 校验 StuckInstances 快照
	var foundQ bool
	for _, stuck := range incompleteErr.StuckInstances {
		if stuck.PluginKey == qKey {
			foundQ = true
			if stuck.Stage != "quiesce" {
				t.Fatalf("expected stuck stage 'quiesce', got '%s'", stuck.Stage)
			}
			if stuck.InstanceID == "" {
				t.Fatal("expected real instance ID for stuck instance, got empty")
			}
			var hasPRes bool
			for _, dep := range stuck.RetainedDeps {
				if dep == resKey {
					hasPRes = true
					break
				}
			}
			if !hasPRes {
				t.Fatalf("expected RetainedDeps to include provider resource key %v, got %v", resKey, stuck.RetainedDeps)
			}
		}
	}
	if !foundQ {
		t.Fatalf("stuck instances did not contain Q (%v): %+v", qKey, incompleteErr.StuckInstances)
	}

	// Q.Stop 与 P.Stop 严禁提前调用
	if qStops := worker.stopCalls.Load(); qStops != 0 {
		t.Fatalf("expected Q.Stop == 0 while quiesce is stuck, got %d", qStops)
	}
	if pStops := provider.stopCalls.Load(); pStops != 0 {
		t.Fatalf("expected P.Stop == 0 due to retained dependency, got %d", pStops)
	}

	// 重复 Shutdown 必须共享冻结结果
	dupErr := app.Shutdown()
	if !errors.Is(dupErr, shutErr) && dupErr.Error() != shutErr.Error() {
		t.Fatalf("expected duplicate Shutdown to return shared result %v, got %v", shutErr, dupErr)
	}

	// 释放 Quiesce 阻塞屏障，清理自有 goroutine
	cleanupFn()

	// 释放后验证 App 状态依旧保持 StateIncomplete，不可被假改写为 StateStopped
	if st := app.State(); st != StateIncomplete {
		t.Fatalf("expected app state to remain StateIncomplete, got %v", st)
	}

	// 等待 Run 协程返回
	select {
	case <-runErrCh:
	case <-time.After(1 * time.Second):
		t.Fatal("app.Run did not return after release")
	}
}

// -----------------------------------------------------------------------------
// 2. 消费者启动失败与清理测试
// -----------------------------------------------------------------------------

var (
	errSentinelStart = errors.New("sentinel_start_failure")
	errSentinelStop  = errors.New("sentinel_stop_failure")
)

type startFailProviderTool struct {
	resKey     ctxkey.Key
	startCalls atomic.Int32
	stopCalls  atomic.Int32
}

func (p *startFailProviderTool) Start(ctx context.Context) error {
	p.startCalls.Add(1)
	return nil
}

func (p *startFailProviderTool) Stop(ctx context.Context) error {
	p.stopCalls.Add(1)
	return nil
}

func (p *startFailProviderTool) Expose() map[ctxkey.Key]any {
	return map[ctxkey.Key]any{
		p.resKey: "provider_val",
	}
}

type startFailConsumerTool struct {
	startCalls atomic.Int32
	stopCalls  atomic.Int32
}

func (c *startFailConsumerTool) Start(ctx context.Context) error {
	c.startCalls.Add(1)
	if ctx == nil {
		return errors.New("ctx cannot be nil")
	}
	return errSentinelStart
}

func (c *startFailConsumerTool) Stop(ctx context.Context) error {
	c.stopCalls.Add(1)
	if ctx == nil {
		return errors.New("ctx cannot be nil")
	}
	return errSentinelStop
}

func TestStartFailureCleanup_ErrorsJoinAndReverseCleanup(t *testing.T) {
	pKey := ctxkey.Key("test_start_fail_provider")
	cKey := ctxkey.Key("test_start_fail_consumer")
	resKey := ctxkey.Key("test_start_fail_res")

	provider := &startFailProviderTool{resKey: resKey}
	consumer := &startFailConsumerTool{}

	app := New(
		WithStartTimeout(50*time.Millisecond),
		WithGraceTimeout(20*time.Millisecond),
		WithCleanupTimeout(20*time.Millisecond),
		WithShutdownTimeout(80*time.Millisecond),
	)

	if err := app.RegisterTool(PluginInfo{
		Key:      pKey,
		Provides: []ctxkey.Key{resKey},
		Mode:     ModeTool,
	}, func() Plugin {
		return provider
	}); err != nil {
		t.Fatalf("failed to register provider: %v", err)
	}

	if err := app.RegisterTool(PluginInfo{
		Key:      cKey,
		Requires: []ctxkey.Key{resKey},
		Mode:     ModeTool,
	}, func() Plugin {
		return consumer
	}); err != nil {
		t.Fatalf("failed to register consumer: %v", err)
	}

	runErr := app.Run()
	if runErr == nil {
		t.Fatal("expected App.Run to return error, got nil")
	}

	// 真实 errors.Is 验证两个 sentinel 错误均被保留成立
	if !errors.Is(runErr, errSentinelStart) {
		t.Fatalf("expected errors.Is(runErr, errSentinelStart) to be true, got %v", runErr)
	}
	if !errors.Is(runErr, errSentinelStop) {
		t.Fatalf("expected errors.Is(runErr, errSentinelStop) to be true, got %v", runErr)
	}

	// 消费者 Stop 必须且仅被调用一次
	if cStops := consumer.stopCalls.Load(); cStops != 1 {
		t.Fatalf("expected consumer.Stop to be called exactly once, got %d", cStops)
	}

	// 上层已启动成功的 provider 必须在回滚时被逆序清理且 Stop 一次
	if pStarts := provider.startCalls.Load(); pStarts != 1 {
		t.Fatalf("expected provider.Start to be called once, got %d", pStarts)
	}
	if pStops := provider.stopCalls.Load(); pStops != 1 {
		t.Fatalf("expected provider.Stop to be called exactly once during rollback, got %d", pStops)
	}
}
