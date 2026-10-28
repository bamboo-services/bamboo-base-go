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

type activateRecord struct {
	handle application.RunHandle
	err    error
	time   time.Time
}

type recordingProxy struct {
	inner    application.Activator
	mu       sync.Mutex
	records  []activateRecord
	recordCh chan activateRecord
}

func newRecordingProxy(inner application.Activator) *recordingProxy {
	return &recordingProxy{
		inner:    inner,
		recordCh: make(chan activateRecord, 32),
	}
}

func (p *recordingProxy) Activate(ctx context.Context, compKey ctxkey.Key, opts ...application.ActivateOption) (application.RunHandle, error) {
	h, err := p.inner.Activate(ctx, compKey, opts...)
	rec := activateRecord{
		handle: h,
		err:    err,
		time:   time.Now(),
	}
	p.mu.Lock()
	p.records = append(p.records, rec)
	p.mu.Unlock()

	select {
	case p.recordCh <- rec:
	default:
	}
	return h, err
}

type realAppTestComponent struct {
	runFunc  func(ctx context.Context) error
	stopFunc func(ctx context.Context) error
}

func (c *realAppTestComponent) Start(_ context.Context) error {
	return nil
}

func (c *realAppTestComponent) Run(ctx context.Context) error {
	if c.runFunc != nil {
		return c.runFunc(ctx)
	}
	return nil
}

func (c *realAppTestComponent) Quiesce(_ context.Context) error {
	return nil
}

func (c *realAppTestComponent) Stop(ctx context.Context) error {
	if c.stopFunc != nil {
		return c.stopFunc(ctx)
	}
	return nil
}

func TestRealApp_CronSkip_RealComponentLifecycle(t *testing.T) {
	app := application.New(
		application.WithGraceTimeout(20*time.Millisecond),
		application.WithCleanupTimeout(200*time.Millisecond),
		application.WithShutdownTimeout(1*time.Second),
	)

	compKey := ctxkey.Key("real_cron_skip_comp")
	var factoryCalls int32
	var stopCalls int32

	run1Entered := make(chan struct{})
	releaseRun1 := make(chan struct{})
	var releaseRun1Once sync.Once
	defer releaseRun1Once.Do(func() { close(releaseRun1) })

	stop1Entered := make(chan struct{})
	releaseStop1 := make(chan struct{})
	var releaseStop1Once sync.Once
	defer releaseStop1Once.Do(func() { close(releaseStop1) })

	run2Entered := make(chan struct{})

	var firstMeta application.InstanceMeta
	var firstPayload time.Time
	var secondMeta application.InstanceMeta
	var secondPayload time.Time

	err := app.RegisterComponent(application.PluginInfo{
		Key:     compKey,
		Overlap: application.OverlapPolicy{Kind: application.OverlapSkip},
	}, func() application.RunnablePlugin {
		callNum := atomic.AddInt32(&factoryCalls, 1)
		return &realAppTestComponent{
			runFunc: func(ctx context.Context) error {
				if callNum == 1 {
					if meta, ok := application.InstanceMetaFrom(ctx); ok {
						firstMeta = meta
					}
					if payload, ok := application.PayloadFrom[time.Time](ctx); ok {
						firstPayload = payload
					}
					close(run1Entered)
					<-releaseRun1
					return nil
				}
				if callNum == 2 {
					if meta, ok := application.InstanceMetaFrom(ctx); ok {
						secondMeta = meta
					}
					if payload, ok := application.PayloadFrom[time.Time](ctx); ok {
						secondPayload = payload
					}
					close(run2Entered)
					return nil
				}
				return nil
			},
			stopFunc: func(ctx context.Context) error {
				atomic.AddInt32(&stopCalls, 1)
				if callNum == 1 {
					close(stop1Entered)
					<-releaseStop1
				}
				return nil
			},
		}
	})
	if err != nil {
		t.Fatalf("RegisterComponent failed: %v", err)
	}

	proxy := newRecordingProxy(app.Activator())
	cronInfo, cronFac := New(
		WithActivator(proxy),
		WithIntervalJob(50*time.Millisecond, compKey),
	)

	if err := app.RegisterWorker(cronInfo, cronFac); err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	t.Cleanup(func() {
		releaseRun1Once.Do(func() { close(releaseRun1) })
		releaseStop1Once.Do(func() { close(releaseStop1) })

		_ = app.Shutdown()

		select {
		case <-runErrCh:
		case <-time.After(2 * time.Second):
			t.Errorf("failfast: timed out waiting for app.Run to terminate")
		}
	})

	// 1. 等待首次 Run 屏障阻塞
	select {
	case <-run1Entered:
	case <-time.After(2 * time.Second):
		t.Fatal("failfast: timed out waiting for first run to enter")
	}

	// 2. 等待第二次真实 tick 到达并被重叠策略拒绝（返回 ErrComponentSkipped，且 handle 为 nil）
	var skippedRec activateRecord
	foundSkipped := false
	for !foundSkipped {
		select {
		case rec := <-proxy.recordCh:
			if errors.Is(rec.err, application.ErrComponentSkipped) {
				skippedRec = rec
				foundSkipped = true
			}
		case <-time.After(2 * time.Second):
			t.Fatal("failfast: timed out waiting for second tick overlap rejection")
		}
	}

	if skippedRec.handle != nil {
		t.Errorf("expected nil handle on skip rejection, got: %v", skippedRec.handle)
	}

	// 证明首 run 阻塞期间没有调用第二次 factory
	if calls := atomic.LoadInt32(&factoryCalls); calls != 1 {
		t.Fatalf("expected exactly 1 factory call while first run is blocked, got: %d", calls)
	}

	// 3. 释放首个 Run，等待 Stop 真退
	releaseRun1Once.Do(func() { close(releaseRun1) })

	select {
	case <-stop1Entered:
	case <-time.After(2 * time.Second):
		t.Fatal("failfast: timed out waiting for first stop to enter")
	}

	releaseStop1Once.Do(func() { close(releaseStop1) })

	// 4. 等待下一个 tick 产生新 InstanceID 并执行
	select {
	case <-run2Entered:
	case <-time.After(2 * time.Second):
		t.Fatal("failfast: timed out waiting for second instance run after stop")
	}

	if calls := atomic.LoadInt32(&factoryCalls); calls != 2 {
		t.Fatalf("expected 2 factory calls after first instance fully stopped, got: %d", calls)
	}

	if firstMeta.InstanceID == "" || secondMeta.InstanceID == "" {
		t.Fatalf("expected non-empty instance IDs, got first=%q, second=%q", firstMeta.InstanceID, secondMeta.InstanceID)
	}
	if firstMeta.InstanceID == secondMeta.InstanceID {
		t.Fatalf("expected distinct instance IDs across ticks, got same: %s", firstMeta.InstanceID)
	}

	// 5. 校验 InstanceMeta 与 payload
	if firstMeta.ParentKey != cronInfo.Key {
		t.Errorf("expected parent key %v, got %v", cronInfo.Key, firstMeta.ParentKey)
	}
	if firstMeta.ParentInstanceID == "" {
		t.Errorf("expected non-empty parent instance ID for first instance")
	}
	if secondMeta.ParentKey != cronInfo.Key {
		t.Errorf("expected second parent key %v, got %v", cronInfo.Key, secondMeta.ParentKey)
	}
	if secondMeta.ParentInstanceID != firstMeta.ParentInstanceID {
		t.Errorf("expected same parent worker instance ID, got first=%q, second=%q", firstMeta.ParentInstanceID, secondMeta.ParentInstanceID)
	}

	if firstPayload.IsZero() {
		t.Errorf("expected valid non-zero time.Time payload for first instance")
	}
	if secondPayload.IsZero() {
		t.Errorf("expected valid non-zero time.Time payload for second instance")
	}
}
