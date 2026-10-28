package cron

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

type fakeRunHandle struct {
	id string
}

func (h *fakeRunHandle) InstanceID() string {
	return h.id
}

func (h *fakeRunHandle) Wait(_ context.Context) (application.RunResult, error) {
	return application.RunResult{
		InstanceID: h.id,
		Outcome:    application.OutcomeSuccess,
	}, nil
}

type fakeActivator struct {
	mu           sync.Mutex
	calls        []time.Time
	targets      []ctxkey.Key
	notifyCh     chan time.Time
	activateFunc func(ctx context.Context, compKey ctxkey.Key, opts ...application.ActivateOption) (application.RunHandle, error)
}

func newFakeActivator(buffer int) *fakeActivator {
	return &fakeActivator{
		notifyCh: make(chan time.Time, buffer),
	}
}

func (f *fakeActivator) Activate(ctx context.Context, compKey ctxkey.Key, opts ...application.ActivateOption) (application.RunHandle, error) {
	if f.activateFunc != nil {
		return f.activateFunc(ctx, compKey, opts...)
	}

	f.mu.Lock()
	now := time.Now()
	f.calls = append(f.calls, now)
	f.targets = append(f.targets, compKey)
	f.mu.Unlock()

	select {
	case f.notifyCh <- now:
	default:
	}

	return &fakeRunHandle{id: "fake-instance"}, nil
}

func TestPlugin_NoActivatorFailsStart(t *testing.T) {
	info, factory := New(WithCronJob("0 * * * *", "target_comp"))
	if info.Mode != application.ModeWorker {
		t.Fatalf("expected ModeWorker, got %v", info.Mode)
	}
	if len(info.Requires) != 0 {
		t.Fatalf("targetKey must not be in Requires: %v", info.Requires)
	}

	p := factory()
	err := p.Start(context.Background())
	if !errors.Is(err, ErrNoActivator) {
		t.Fatalf("expected ErrNoActivator, got: %v", err)
	}
}

func TestPlugin_InvalidSpecFailsStart(t *testing.T) {
	act := newFakeActivator(10)
	_, factory := New(
		WithActivator(act),
		WithCronJob("not a valid cron spec", "target_comp"),
	)
	p := factory()
	err := p.Start(context.Background())
	if !errors.Is(err, ErrInvalidJobSchedule) {
		t.Fatalf("expected ErrInvalidJobSchedule, got: %v", err)
	}
}

func TestPlugin_FactoryHasNoBackgroundResources(t *testing.T) {
	act := newFakeActivator(10)
	_, factory := New(
		WithActivator(act),
		WithIntervalJob(10*time.Millisecond, "target_comp"),
	)
	_ = factory()
	// 未调用 Start/Run，调度器不得有任何执行或激活
	time.Sleep(30 * time.Millisecond)
	if len(act.calls) != 0 {
		t.Fatalf("factory must not start background scheduler, calls: %d", len(act.calls))
	}
}

func TestPlugin_IntervalJobTriggerAndCleanStop(t *testing.T) {
	act := newFakeActivator(10)
	var activeCount int64

	act.activateFunc = func(ctx context.Context, compKey ctxkey.Key, opts ...application.ActivateOption) (application.RunHandle, error) {
		atomic.AddInt64(&activeCount, 1)
		defer atomic.AddInt64(&activeCount, -1)

		act.mu.Lock()
		act.calls = append(act.calls, time.Now())
		act.targets = append(act.targets, compKey)
		act.mu.Unlock()

		select {
		case act.notifyCh <- time.Now():
		default:
		}
		return &fakeRunHandle{id: "fake-inst"}, nil
	}

	const targetKey ctxkey.Key = "test_comp_key"
	info, factory := New(
		WithActivator(act),
		WithIntervalJob(50*time.Millisecond, targetKey),
	)

	if len(info.Requires) != 0 {
		t.Fatalf("component target must not be in Requires: %v", info.Requires)
	}

	p := factory()
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	runCtx, cancelRun := context.WithCancel(context.Background())
	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- p.Run(runCtx)
	}()

	// 真实等待通知，不猜 sleep 顺序
	select {
	case <-act.notifyCh:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for first interval tick")
	}

	// 触发正常收到，停止调度器
	stopCtx, cancelStop := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelStop()

	if err := p.Stop(stopCtx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	cancelRun()

	select {
	case err := <-runErrCh:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for Run to return")
	}

	// 验证所有激活回调均已退出，无遗留调用
	if cur := atomic.LoadInt64(&activeCount); cur != 0 {
		t.Fatalf("expected 0 active trigger handlers, got %d", cur)
	}

	// 验证重复 Stop 幂等共享结果
	if err := p.Stop(context.Background()); err != nil {
		t.Fatalf("subsequent Stop returned error: %v", err)
	}

	// 停止后不再产生新的 tick
	act.mu.Lock()
	countAtStop := len(act.calls)
	act.mu.Unlock()

	time.Sleep(120 * time.Millisecond)

	act.mu.Lock()
	countAfterWait := len(act.calls)
	act.mu.Unlock()

	if countAfterWait != countAtStop {
		t.Fatalf("received ticks after stop: before=%d, after=%d", countAtStop, countAfterWait)
	}
}
