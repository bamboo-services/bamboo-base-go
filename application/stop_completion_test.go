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

type blockingMockPlugin struct {
	stopCalls atomic.Int32
	blockCh   chan struct{}
	stopErr   error
	panicVal  any
}

func (m *blockingMockPlugin) Start(ctx context.Context) error {
	return nil
}

func (m *blockingMockPlugin) Stop(ctx context.Context) error {
	m.stopCalls.Add(1)
	if m.blockCh != nil {
		<-m.blockCh
	}
	if m.panicVal != nil {
		panic(m.panicVal)
	}
	return m.stopErr
}

func newTestAppForStop(appID string) *App {
	app := &App{
		appID:           appID,
		container:       newSharedContainer(appID),
		staticInstances: make(map[ctxkey.Key]*staticInstanceRecord),
	}
	app.coordinator = newShutdownCoordinator(app)
	return app
}

func TestStopCompletion_ConcurrentWaitAndSingleExecution(t *testing.T) {
	app := newTestAppForStop("test-stop-wait")
	blockCh := make(chan struct{})

	mock := &blockingMockPlugin{
		blockCh: blockCh,
		stopErr: errors.New("underlying stop error"),
	}

	key := ctxkey.Key("test.plugin.concurrent")
	rec := &staticInstanceRecord{
		info: PluginInfo{
			Key: key,
		},
		instanceID: "inst-concurrent-1",
		plugin:     mock,
	}

	var wg sync.WaitGroup
	var err1, err2 error
	done1 := make(chan struct{})
	done2 := make(chan struct{})

	wg.Add(2)
	go func() {
		defer wg.Done()
		err1 = app.safeCallStop(rec, Config{})
		close(done1)
	}()
	go func() {
		defer wg.Done()
		err2 = app.safeCallStop(rec, Config{})
		close(done2)
	}()

	// 验证在 blockCh 解除前，两个 safeCallStop 均被阻塞未返回
	select {
	case <-done1:
		t.Fatal("call 1 returned before unblocking")
	case <-done2:
		t.Fatal("call 2 returned before unblocking")
	case <-time.After(50 * time.Millisecond):
		// 正常阻塞中
	}

	// 解除阻塞
	close(blockCh)
	wg.Wait()

	if calls := mock.stopCalls.Load(); calls != 1 {
		t.Fatalf("expected 1 Stop call, got %d", calls)
	}
	if !rec.stopped.Load() {
		t.Fatal("expected rec.stopped to be true after normal return")
	}
	if err1 == nil || err2 == nil {
		t.Fatalf("expected non-nil error, got err1=%v, err2=%v", err1, err2)
	}
	if err1 != err2 {
		t.Fatalf("expected both callers to receive the same error instance, got %v vs %v", err1, err2)
	}
}

func TestStopCompletion_PanicHandlingAndSecondCall(t *testing.T) {
	app := newTestAppForStop("test-stop-panic")

	mock := &blockingMockPlugin{
		panicVal: "stop explode",
	}

	key := ctxkey.Key("test.plugin.panic")
	rec := &staticInstanceRecord{
		info: PluginInfo{
			Key:      key,
			Requires: []ctxkey.Key{"dep.a"},
		},
		instanceID: "inst-panic-1",
		plugin:     mock,
	}
	app.container.setEntry(key, key, mock, EntryReady)

	err1 := app.safeCallStop(rec, Config{})
	if err1 == nil {
		t.Fatal("expected panic error from call 1, got nil")
	}
	var panicErr *HookPanicError
	if !errors.As(err1, &panicErr) {
		t.Fatalf("expected HookPanicError, got %T: %v", err1, err1)
	}

	// panic 实例 stopped 必须为 false
	if rec.stopped.Load() {
		t.Fatal("rec.stopped must be false when Stop panics")
	}
	if !rec.stopPanicked.Load() {
		t.Fatal("rec.stopPanicked must be true")
	}

	// 容器状态必须标记为 EntryUnavailable
	entry, ok := app.container.getEntry(key)
	if !ok || EntryState(entry.state) != EntryUnavailable {
		t.Fatalf("expected EntryUnavailable, got %v", entry.state)
	}

	// 第二次调用不执行 Stop 且返回同错
	err2 := app.safeCallStop(rec, Config{})
	if calls := mock.stopCalls.Load(); calls != 1 {
		t.Fatalf("Stop must not be executed again, got calls=%d", calls)
	}
	if err2 == nil {
		t.Fatal("expected non-nil error from second call")
	}
	if err1 != err2 {
		t.Fatalf("expected same error for second call, got %v vs %v", err1, err2)
	}
	if rec.stopped.Load() {
		t.Fatal("rec.stopped must still be false after second call")
	}
}
