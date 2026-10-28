package application

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

type blockStartPlugin struct {
	startEntered chan struct{}
	releaseStart chan struct{}
	stopped      atomic.Bool
	stopCalled   chan struct{}
}

func (p *blockStartPlugin) Start(ctx context.Context) error {
	close(p.startEntered)
	select {
	case <-p.releaseStart:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *blockStartPlugin) Stop(ctx context.Context) error {
	p.stopped.Store(true)
	select {
	case <-p.stopCalled:
	default:
		close(p.stopCalled)
	}
	return nil
}

// TestF2_ShutdownDuringStartClaim verifies F2:
// Shutdown between Start claim and hook completion does not leak Stop.
func TestF2_ShutdownDuringStartClaim(t *testing.T) {
	key := ctxkey.Key("test_f2_tool")
	p := &blockStartPlugin{
		startEntered: make(chan struct{}),
		releaseStart: make(chan struct{}),
		stopCalled:   make(chan struct{}),
	}

	app := New(WithShutdownTimeout(3*time.Second), WithGraceTimeout(1*time.Second))
	err := app.RegisterTool(PluginInfo{Key: key}, func() Plugin {
		return p
	})
	if err != nil {
		t.Fatalf("register tool failed: %v", err)
	}

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	// Wait until Start is claimed and entered
	select {
	case <-p.startEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Start to enter")
	}

	// Trigger concurrent Shutdown while Start is blocked
	shutErrCh := make(chan error, 1)
	go func() {
		shutErrCh <- app.Shutdown()
	}()

	// Give Shutdown time to observe Stopping and wait for Start
	time.Sleep(100 * time.Millisecond)

	// Release Start
	close(p.releaseStart)

	select {
	case err := <-runErrCh:
		if !errors.Is(err, ErrStartupInterrupted) {
			t.Fatalf("expected ErrStartupInterrupted from Run, got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for Run to return")
	}

	select {
	case err := <-shutErrCh:
		if err != nil {
			t.Fatalf("expected nil shutdown error, got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for Shutdown to return")
	}

	if !p.stopped.Load() {
		t.Fatal("expected Stop to be called for claimed plugin during shutdown")
	}
}

type ignoreCtxWorker struct {
	started    chan struct{}
	stopCalled chan struct{}
	releaseRun chan struct{}
}

func (w *ignoreCtxWorker) Start(ctx context.Context) error { return nil }
func (w *ignoreCtxWorker) Stop(ctx context.Context) error {
	select {
	case <-w.stopCalled:
	default:
		close(w.stopCalled)
	}
	// Stop returns nil immediately
	return nil
}

func (w *ignoreCtxWorker) Run(ctx context.Context) error {
	close(w.started)
	// Ignore ctx.Done() completely until releaseRun
	<-w.releaseRun
	return nil
}

// TestF4_StopNilWhileRunBlocks verifies F4:
// When Run ignores ctx and Stop returns nil immediately,
// Shutdown does not declare stopped early, but waits and marks Incomplete if total deadline exceeds.
func TestF4_StopNilWhileRunBlocks(t *testing.T) {
	key := ctxkey.Key("test_f4_worker")
	w := &ignoreCtxWorker{
		started:    make(chan struct{}),
		stopCalled: make(chan struct{}),
		releaseRun: make(chan struct{}),
	}
	defer func() {
		select {
		case <-w.releaseRun:
		default:
			close(w.releaseRun)
		}
	}()

	// Short budget: total 400ms, grace 100ms
	app := New(WithShutdownTimeout(400*time.Millisecond), WithGraceTimeout(100*time.Millisecond))
	err := app.RegisterWorker(PluginInfo{Key: key, Mode: ModeWorker}, func() RunnablePlugin {
		return w
	})
	if err != nil {
		t.Fatalf("register worker failed: %v", err)
	}

	go func() {
		_ = app.Run()
	}()

	select {
	case <-w.started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for worker Run to start")
	}

	shutErrCh := make(chan error, 1)
	startShut := time.Now()
	go func() {
		shutErrCh <- app.Shutdown()
	}()

	select {
	case err := <-shutErrCh:
		elapsed := time.Since(startShut)
		if elapsed < 350*time.Millisecond {
			t.Fatalf("Shutdown returned too early (%v), did not wait for RunDone guard", elapsed)
		}
		var incompleteErr *StopIncompleteError
		if !errors.As(err, &incompleteErr) {
			t.Fatalf("expected StopIncompleteError when Run blocks beyond total deadline, got: %v", err)
		}
		if app.State() != StateIncomplete {
			t.Fatalf("expected state Incomplete, got: %v", app.State())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown hung indefinitely")
	}
}

// TestF4_StopWinsBeforeRun verifies F4:
// When Stop wins the claim, Run is skipped.
func TestF4_StopWinsBeforeRun(t *testing.T) {
	key := ctxkey.Key("test_f4_stopwins")
	entered := make(chan struct{}, 1)
	w := &stubWorker{
		runFn: func(ctx context.Context) error {
			entered <- struct{}{}
			return nil
		},
	}

	rec := &staticInstanceRecord{
		info:       PluginInfo{Key: key, Mode: ModeWorker},
		instanceID: "inst1",
		plugin:     w,
		runnable:   w,
	}

	app := New()
	cfg := Config{CleanupTimeout: 1 * time.Second}

	// safeCallStop claims stop first
	err := app.safeCallStop(rec, cfg)
	if err != nil {
		t.Fatalf("safeCallStop failed: %v", err)
	}

	app.staticMu.Lock()
	app.staticInstances[key] = rec
	app.staticMu.Unlock()

	app.stateMu.Lock()
	app.state = StateRunning
	app.stateMu.Unlock()

	watchdogErrChan := make(chan error, 1)
	app.launchWorkers(watchdogErrChan)

	select {
	case <-entered:
		t.Fatal("expected Run to be skipped when stop won first")
	case <-time.After(30 * time.Millisecond):
	}
}

type stubWorker struct {
	runFn func(ctx context.Context) error
}

func (s *stubWorker) Start(ctx context.Context) error { return nil }
func (s *stubWorker) Stop(ctx context.Context) error  { return nil }
func (s *stubWorker) Run(ctx context.Context) error {
	if s.runFn != nil {
		return s.runFn(ctx)
	}
	return nil
}

// TestF3_AppStoppingCancelsQueueImmediately verifies F3:
// When App enters Stopping, queued items are cancelled immediately without waiting for parent Worker to exit.
// Active tasks are not cancelled until Grace/Force.
func TestF3_AppStoppingCancelsQueueImmediately(t *testing.T) {
	workerKey := ctxkey.Key("test_f3_worker")
	compKey := ctxkey.Key("test_f3_comp")

	activeRunning := make(chan struct{})
	releaseActive := make(chan struct{})
	factoryCalls := atomic.Int32{}

	app := New(WithShutdownTimeout(3*time.Second), WithGraceTimeout(1*time.Second))

	// Register Component with OverlapQueue, maxQueue=5
	if err := app.RegisterComponent(PluginInfo{
		Key:     compKey,
		Overlap: OverlapPolicy{Kind: OverlapQueue, MaxQueue: 5},
	}, func() RunnablePlugin {
		factoryCalls.Add(1)
		return &stubRunnable{
			runFn: func(ctx context.Context) error {
				close(activeRunning)
				select {
				case <-releaseActive:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		}
	}); err != nil {
		t.Fatalf("register component failed: %v", err)
	}

	parentBlock := make(chan struct{})
	defer close(parentBlock)

	if err := app.RegisterWorker(PluginInfo{Key: workerKey, Mode: ModeWorker}, func() RunnablePlugin {
		return &stubRunnable{
			runFn: func(ctx context.Context) error {
				// 1. Activate active task
				h1, err := app.componentEngine.Activate(ctx, compKey)
				if err != nil {
					return err
				}
				_ = h1

				<-activeRunning

				// 2. Activate queued task
				h2, err := app.componentEngine.Activate(ctx, compKey)
				if err != nil {
					return err
				}

				// Trigger Shutdown in background while worker Run is still alive and blocked
				go func() {
					_ = app.Shutdown()
				}()

				// h2 (queued) should be cancelled promptly
				res2, _ := h2.Wait(ctx)
				if res2.Dispatched {
					return errors.New("expected queued item not dispatched")
				}
				if !res2.CancelRequested || res2.CancelErr == nil {
					return errors.New("expected queued item CancelRequested with CancelErr")
				}

				// active task should NOT be cancelled yet (grace not elapsed)
				select {
				case <-ctx.Done():
					return errors.New("worker ctx cancelled prematurely")
				default:
				}

				close(releaseActive)
				return nil
			},
		}
	}); err != nil {
		t.Fatalf("register worker failed: %v", err)
	}

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	select {
	case err := <-runErrCh:
		if err != nil && !errors.Is(err, ErrParentClosing) && !errors.Is(err, context.Canceled) {
			t.Logf("app run returned: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for app.Run")
	}

	// Queued item was cancelled without factory called
	if calls := factoryCalls.Load(); calls != 1 {
		t.Fatalf("expected factory to be called exactly 1 time (active only), got: %d", calls)
	}
}

type stubRunnable struct {
	runFn func(ctx context.Context) error
}

func (s *stubRunnable) Start(ctx context.Context) error { return nil }
func (s *stubRunnable) Stop(ctx context.Context) error  { return nil }
func (s *stubRunnable) Run(ctx context.Context) error {
	if s.runFn != nil {
		return s.runFn(ctx)
	}
	return nil
}
