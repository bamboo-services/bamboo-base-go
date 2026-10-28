package application

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ctxkey "github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

type witnessWorker struct {
	ready chan struct{}
}

func (w *witnessWorker) Start(ctx context.Context) error {
	return nil
}

func (w *witnessWorker) Run(ctx context.Context) error {
	close(w.ready)
	<-ctx.Done()
	return nil
}

func (w *witnessWorker) Stop(ctx context.Context) error {
	return nil
}

type blockingStopPlugin struct {
	stopCalled  atomic.Int32
	stopUnblock chan struct{}
}

func (p *blockingStopPlugin) Start(ctx context.Context) error {
	return nil
}

func (p *blockingStopPlugin) Stop(ctx context.Context) error {
	p.stopCalled.Add(1)
	<-p.stopUnblock
	return nil
}

type dummyProviderPlugin struct {
	resourceKey ctxkey.Key
	started     chan struct{}
	stopCalled  atomic.Int32
}

func (p *dummyProviderPlugin) Start(ctx context.Context) error {
	close(p.started)
	return nil
}

func (p *dummyProviderPlugin) Stop(ctx context.Context) error {
	p.stopCalled.Add(1)
	return nil
}

func (p *dummyProviderPlugin) Expose() map[ctxkey.Key]any {
	return map[ctxkey.Key]any{p.resourceKey: "dummy"}
}

func TestApp_StopBudget_TimeoutRetainsDepsAndUnblocks(t *testing.T) {
	providerKey := ctxkey.Key("db_provider")
	consumerKey := ctxkey.Key("service_consumer")
	resourceKey := ctxkey.Key("db_resource")
	witnessKey := ctxkey.Key("witness_worker")

	provider := &dummyProviderPlugin{
		resourceKey: resourceKey,
		started:     make(chan struct{}),
	}
	unblockChan := make(chan struct{})
	var unblockOnce sync.Once
	cleanupUnblock := func() {
		unblockOnce.Do(func() {
			close(unblockChan)
		})
	}
	t.Cleanup(cleanupUnblock)

	consumer := &blockingStopPlugin{
		stopUnblock: unblockChan,
	}

	witness := &witnessWorker{
		ready: make(chan struct{}),
	}

	app := New(
		WithGraceTimeout(20*time.Millisecond),
		WithShutdownTimeout(60*time.Millisecond),
		WithCleanupTimeout(20*time.Millisecond),
	)

	if err := app.RegisterTool(PluginInfo{
		Key:      providerKey,
		Provides: []ctxkey.Key{resourceKey},
		Mode:     ModeTool,
	}, func() Plugin { return provider }); err != nil {
		t.Fatalf("failed to register provider tool: %v", err)
	}

	if err := app.RegisterTool(PluginInfo{
		Key:      consumerKey,
		Requires: []ctxkey.Key{resourceKey},
		Mode:     ModeTool,
	}, func() Plugin { return consumer }); err != nil {
		t.Fatalf("failed to register consumer tool: %v", err)
	}

	if err := app.RegisterWorker(PluginInfo{
		Key:      witnessKey,
		Requires: []ctxkey.Key{consumerKey},
		Mode:     ModeWorker,
	}, func() RunnablePlugin { return witness }); err != nil {
		t.Fatalf("failed to register witness worker: %v", err)
	}

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	// 等待末尾依赖 worker 进入 Run，确保整图拓扑启动完毕并进入 Running 状态
	<-witness.ready

	// 等待并触发 Shutdown：consumer Stop 同步阻塞，超出短总上限返回 Incomplete
	shutdownErr := app.Shutdown()
	if shutdownErr == nil {
		t.Fatalf("expected Incomplete shutdown error due to timeout, got nil")
	}

	runErr := <-runErrCh
	if runErr != nil && !errors.Is(runErr, shutdownErr) && runErr.Error() != shutdownErr.Error() {
		t.Fatalf("expected run err to reflect shutdown result, got %v", runErr)
	}

	var incErr *StopIncompleteError
	if !errors.As(shutdownErr, &incErr) {
		t.Fatalf("expected StopIncompleteError, got %v", shutdownErr)
	}

	// 验证消费者处于 stage stop 残留，底层 provider 未被调用 Stop(stopCalled == 0)
	if provider.stopCalled.Load() != 0 {
		t.Fatalf("expected provider Stop not called (retained), got %d", provider.stopCalled.Load())
	}

	// 2. 解除阻塞，消费者实际 Stop 完成
	cleanupUnblock()

	// 等待消费者 goroutine 退出并确认计数
	recConsumer := app.staticInstances[consumerKey]
	recConsumer.stopMu.Lock()
	consumerStopDone := recConsumer.stopDone
	recConsumer.stopMu.Unlock()
	if consumerStopDone != nil {
		<-consumerStopDone
	}
	if consumer.stopCalled.Load() != 1 {
		t.Fatalf("expected consumer Stop called exactly 1 time, got %d", consumer.stopCalled.Load())
	}

	// 3. 重复调用 Shutdown 共享旧协调结果，快照不随之改写
	secondErr := app.Shutdown()
	if !errors.Is(secondErr, shutdownErr) && secondErr.Error() != shutdownErr.Error() {
		t.Fatalf("expected shared shutdown result, got %v vs %v", secondErr, shutdownErr)
	}

	app.stateMu.RLock()
	finalState := app.state
	app.stateMu.RUnlock()
	if finalState != StateIncomplete {
		t.Fatalf("expected app state to remain Incomplete, got %v", finalState)
	}
}
