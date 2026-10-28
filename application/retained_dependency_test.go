package application

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

// mockRetainedPlugin 辅助测试桩
type mockRetainedPlugin struct {
	startFunc func(ctx context.Context) error
	stopFunc  func(ctx context.Context) error
}

func (m *mockRetainedPlugin) Start(ctx context.Context) error {
	if m.startFunc != nil {
		return m.startFunc(ctx)
	}
	return nil
}

func (m *mockRetainedPlugin) Stop(ctx context.Context) error {
	if m.stopFunc != nil {
		return m.stopFunc(ctx)
	}
	return nil
}

type mockRetainedExposer struct {
	mockRetainedPlugin
	providedKey ctxkey.Key
}

func (m *mockRetainedExposer) Expose() map[ctxkey.Key]any {
	return map[ctxkey.Key]any{
		m.providedKey: "dummy_resource",
	}
}

// 测试链 Consumer -> 资源键 Provider -> Leaf，旁支独立清理；panic 一次 / Unavailable / 依赖保留 EntryReady；重复 Shutdown 共享结果
func TestApp_RetainedDependency_PropagationAndClosure(t *testing.T) {
	app := New(
		WithGraceTimeout(100*time.Millisecond),
		WithCleanupTimeout(100*time.Millisecond),
		WithShutdownTimeout(500*time.Millisecond),
	)

	leafKey := ctxkey.Key("leaf_provider")
	dbProviderKey := ctxkey.Key("db_provider")
	dbResourceKey := ctxkey.Key("db_resource")
	consumerKey := ctxkey.Key("consumer_plugin")
	siblingKey := ctxkey.Key("sibling_plugin")
	workerKey := ctxkey.Key("trigger_worker")

	var leafStopCount atomic.Int64
	var dbStopCount atomic.Int64
	var siblingStopCount atomic.Int64
	var consumerPanicCount atomic.Int64

	// 1. 叶子节点 (Leaf)
	_ = app.RegisterTool(PluginInfo{Key: leafKey}, func() Plugin {
		return &mockRetainedPlugin{
			stopFunc: func(ctx context.Context) error {
				leafStopCount.Add(1)
				return nil
			},
		}
	})

	// 2. 资源键 Provider: 暴露 db_resource，且依赖 leaf_provider
	_ = app.RegisterTool(PluginInfo{
		Key:      dbProviderKey,
		Provides: []ctxkey.Key{dbResourceKey},
		Requires: []ctxkey.Key{leafKey},
	}, func() Plugin {
		return &mockRetainedExposer{
			mockRetainedPlugin: mockRetainedPlugin{
				stopFunc: func(ctx context.Context) error {
					dbStopCount.Add(1)
					return nil
				},
			},
			providedKey: dbResourceKey,
		}
	})

	// 3. 消费者: 依赖资源键 db_resource，Stop 时发生 panic
	_ = app.RegisterTool(PluginInfo{
		Key:      consumerKey,
		Requires: []ctxkey.Key{dbResourceKey},
	}, func() Plugin {
		return &mockRetainedPlugin{
			stopFunc: func(ctx context.Context) error {
				consumerPanicCount.Add(1)
				panic("consumer stop panic")
			},
		}
	})

	// 4. 无关旁支: 独立插件，应正常执行 Stop
	_ = app.RegisterTool(PluginInfo{Key: siblingKey}, func() Plugin {
		return &mockRetainedPlugin{
			stopFunc: func(ctx context.Context) error {
				siblingStopCount.Add(1)
				return nil
			},
		}
	})

	// 5. 驱动 Worker: 提供启动屏障与正常 Run
	ready := make(chan struct{})
	workerDone := make(chan struct{})
	_ = app.RegisterWorker(PluginInfo{Key: workerKey}, func() RunnablePlugin {
		return &mockWorker{
			runFunc: func(ctx context.Context) error {
				close(ready)
				<-ctx.Done()
				close(workerDone)
				return nil
			},
		}
	})

	errChan := make(chan error, 1)
	go func() {
		errChan <- app.Run()
	}()

	select {
	case <-ready:
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for worker to be ready")
	}

	// 触发首次 Shutdown
	firstErr := app.Shutdown()
	if firstErr == nil {
		t.Fatalf("expected error from shutdown with stop-panic, got nil")
	}

	// 验证重复调用 Shutdown 共享完全一致的结果（无重复 panic / 状态推进）
	secondErr := app.Shutdown()
	if !errors.Is(secondErr, firstErr) && secondErr.Error() != firstErr.Error() {
		t.Fatalf("expected identical shutdown result on repeated call, got %v vs %v", secondErr, firstErr)
	}

	select {
	case runErr := <-errChan:
		var incompleteErr *StopIncompleteError
		if !errors.As(runErr, &incompleteErr) {
			t.Fatalf("expected StopIncompleteError, got %v", runErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for app.Run to terminate")
	}

	// 验证消费者只 panic 了一次
	if count := consumerPanicCount.Load(); count != 1 {
		t.Fatalf("expected consumer panic count 1, got %d", count)
	}

	// 验证受保护的资源提供者与其传递依赖均被保留，未调用 Stop
	if count := dbStopCount.Load(); count != 0 {
		t.Fatalf("expected db_provider stop count 0, got %d", count)
	}
	if count := leafStopCount.Load(); count != 0 {
		t.Fatalf("expected leaf_provider stop count 0, got %d", count)
	}

	// 验证无关旁支正常退役并调用了 Stop
	if count := siblingStopCount.Load(); count != 1 {
		t.Fatalf("expected sibling stop count 1, got %d", count)
	}

	// 验证容器状态：故障消费者 EntryUnavailable，传递依赖项依然保留 EntryReady
	entryConsumer, ok := app.container.getEntry(consumerKey)
	if !ok || EntryState(entryConsumer.state) != EntryUnavailable {
		t.Fatalf("expected consumer state EntryUnavailable, got %v", entryConsumer.state)
	}

	entryDB, ok := app.container.getEntry(dbResourceKey)
	if !ok || EntryState(entryDB.state) != EntryReady {
		t.Fatalf("expected db resource state EntryReady, got %v", entryDB.state)
	}

	entryLeaf, ok := app.container.getEntry(leafKey)
	if !ok || EntryState(entryLeaf.state) != EntryReady {
		t.Fatalf("expected leaf provider state EntryReady, got %v", entryLeaf.state)
	}
}
