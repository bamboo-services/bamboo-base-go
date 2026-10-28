package cron_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	cron "github.com/bamboo-services/bamboo-base-go/plugins/cron/v2"
)

type recordEvent struct {
	Index   int
	Handle  application.RunHandle
	Err     error
	Payload any
}

type recordingActivator struct {
	inner application.Activator
	mu    sync.Mutex
	recs  []recordEvent
	ch    chan recordEvent
}

func (r *recordingActivator) Activate(ctx context.Context, key ctxkey.Key, opts ...application.ActivateOption) (application.RunHandle, error) {
	h, err := r.inner.Activate(ctx, key, opts...)
	r.mu.Lock()
	rec := recordEvent{
		Index:  len(r.recs) + 1,
		Handle: h,
		Err:    err,
	}
	r.recs = append(r.recs, rec)
	r.mu.Unlock()
	select {
	case r.ch <- rec:
	default:
	}
	return h, err
}

type barrierComponent struct {
	id          string
	runEntered  chan struct{}
	runRelease  chan struct{}
	stopEntered chan struct{}
	stopRelease chan struct{}
}

func (c *barrierComponent) Start(context.Context) error { return nil }

func (c *barrierComponent) Run(ctx context.Context) error {
	select {
	case c.runEntered <- struct{}{}:
	default:
	}
	select {
	case <-ctx.Done():
		return nil
	case <-c.runRelease:
		return nil
	}
}

func (c *barrierComponent) Stop(context.Context) error {
	select {
	case c.stopEntered <- struct{}{}:
	default:
	}
	<-c.stopRelease
	return nil
}

func TestCron_RealApp_QueueFIFOAndFullBehavior(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	app := application.New(
		application.WithGraceTimeout(20*time.Millisecond),
		application.WithCleanupTimeout(100*time.Millisecond),
		application.WithShutdownTimeout(1*time.Second),
	)

	runDone := make(chan struct{})
	var runErr error

	runEntered1 := make(chan struct{}, 1)
	runRelease1 := make(chan struct{})
	stopEntered1 := make(chan struct{}, 1)
	stopRelease1 := make(chan struct{})

	runEntered2 := make(chan struct{}, 1)
	runRelease2 := make(chan struct{})
	stopEntered2 := make(chan struct{}, 1)
	stopRelease2 := make(chan struct{})

	t.Cleanup(func() {
		select {
		case <-runRelease1:
		default:
			close(runRelease1)
		}
		select {
		case <-stopRelease1:
		default:
			close(stopRelease1)
		}
		select {
		case <-runRelease2:
		default:
			close(runRelease2)
		}
		select {
		case <-stopRelease2:
		default:
			close(stopRelease2)
		}
		_ = app.Shutdown()
		select {
		case <-runDone:
		case <-time.After(1 * time.Second):
		}
	})
	const targetKey ctxkey.Key = "comp_queue_target"
	proxy := &recordingActivator{
		inner: app.Activator(),
		ch:    make(chan recordEvent, 16),
	}

	var factoryCount int32
	var createdIDs []string
	var facMu sync.Mutex

	compInfo := application.PluginInfo{
		Key:  targetKey,
		Name: "QueueTarget",
		Mode: application.ModeComponent,
		Overlap: application.OverlapPolicy{
			Kind:     application.OverlapQueue,
			MaxQueue: 2,
		},
	}

	err := app.RegisterComponent(compInfo, func() application.RunnablePlugin {
		n := atomic.AddInt32(&factoryCount, 1)
		facMu.Lock()
		defer facMu.Unlock()
		switch n {
		case 1:
			createdIDs = append(createdIDs, "instance-1")
			return &barrierComponent{
				id:          "instance-1",
				runEntered:  runEntered1,
				runRelease:  runRelease1,
				stopEntered: stopEntered1,
				stopRelease: stopRelease1,
			}
		case 2:
			createdIDs = append(createdIDs, "instance-2")
			return &barrierComponent{
				id:          "instance-2",
				runEntered:  runEntered2,
				runRelease:  runRelease2,
				stopEntered: stopEntered2,
				stopRelease: stopRelease2,
			}
		default:
			createdIDs = append(createdIDs, "unexpected-instance")
			return &barrierComponent{
				runEntered:  make(chan struct{}, 1),
				runRelease:  runRelease2,
				stopEntered: make(chan struct{}, 1),
				stopRelease: stopRelease2,
			}
		}
	})
	if err != nil {
		t.Fatalf("RegisterComponent failed: %v", err)
	}

	cronInfo, cronFac := cron.New(
		cron.WithIntervalJob(50*time.Millisecond, targetKey),
		cron.WithActivator(proxy),
	)
	if err := app.RegisterWorker(cronInfo, cronFac); err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	go func() {
		runErr = app.Run()
		close(runDone)
	}()

	// 1. 等待首个实例启动并进入 Run 屏障
	select {
	case <-runEntered1:
	case <-ctx.Done():
		t.Fatalf("timeout waiting for instance 1 to enter Run")
	}

	// 2. 依次观察 4 次来自 gocron 的调度激活事件
	var observed []recordEvent
	for len(observed) < 4 {
		select {
		case ev := <-proxy.ch:
			observed = append(observed, ev)
		case <-ctx.Done():
			t.Fatalf("timeout waiting for 4 activation events, got %d", len(observed))
		}
	}

	// 验证事件契约
	// Tick 1: 成功激活并取得句柄
	if observed[0].Err != nil || observed[0].Handle == nil {
		t.Fatalf("tick 1 expected handle, got err=%v", observed[0].Err)
	}
	id1 := observed[0].Handle.InstanceID()

	// Tick 2: 成功入队，取得句柄，属于独立唯一 ID
	if observed[1].Err != nil || observed[1].Handle == nil {
		t.Fatalf("tick 2 expected handle, got err=%v", observed[1].Err)
	}
	id2 := observed[1].Handle.InstanceID()
	if id2 == id1 {
		t.Fatalf("tick 2 must have distinct instance ID, got %s == %s", id2, id1)
	}

	// Tick 3: 成功入队满额(MaxQueue:2)，取得第 2 个等待句柄
	if observed[2].Err != nil || observed[2].Handle == nil {
		t.Fatalf("tick 3 expected handle, got err=%v", observed[2].Err)
	}
	id3 := observed[2].Handle.InstanceID()
	if id3 == id1 || id3 == id2 {
		t.Fatalf("tick 3 must have distinct instance ID, got %s", id3)
	}

	// Tick 4: 队列已满(已达 MaxQueue 限制)，真实返回 ErrQueueFull
	if !errors.Is(observed[3].Err, application.ErrQueueFull) {
		t.Fatalf("tick 4 expected ErrQueueFull, got err=%v", observed[3].Err)
	}

	// 验证在此期间仅调用过一次 factory（仅首实例派发构造，排队项尚未构造）
	if cnt := atomic.LoadInt32(&factoryCount); cnt != 1 {
		t.Fatalf("expected 1 factory call while queued, got %d", cnt)
	}

	// 3. 放行首个实例的 Run，但 Stop 处于屏障中
	close(runRelease1)

	select {
	case <-stopEntered1:
	case <-ctx.Done():
		t.Fatalf("timeout waiting for instance 1 to enter Stop")
	}

	// 证明：在 Stop 真实返回之前，即使 Run 已结束，仍不进入第二个实例的 factory
	if cnt := atomic.LoadInt32(&factoryCount); cnt != 1 {
		t.Fatalf("instance 2 must not be constructed before instance 1 Stop completes, got %d", cnt)
	}

	// 4. 放行 Stop，此时进入第 2 个实例
	close(stopRelease1)

	select {
	case <-runEntered2:
	case <-ctx.Done():
		t.Fatalf("timeout waiting for instance 2 to enter Run")
	}

	// 此时第 2 个实例已派发并构造，调用了第 2 次 factory；但第 3 个实例仍在排队
	if cnt := atomic.LoadInt32(&factoryCount); cnt != 2 {
		t.Fatalf("expected 2 factory calls, got %d", cnt)
	}

	// 5. 启动 App.Shutdown 关闸
	shutdownErrCh := make(chan error, 1)
	go func() {
		shutdownErrCh <- app.Shutdown()
	}()

	// 此时第 2 个实例收到全局取消退出 Run 并进入 Stop，等待进入 stopEntered2
	select {
	case <-stopEntered2:
	case <-ctx.Done():
		t.Fatalf("timeout waiting for instance 2 to enter Stop")
	}

	// 放行第 2 个实例的 Stop，使关机流程顺利收尾
	close(stopRelease2)

	// 等待 App.Shutdown 完成并验证无错误
	select {
	case err := <-shutdownErrCh:
		if err != nil {
			t.Fatalf("app.Shutdown failed: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("timeout waiting for app.Shutdown")
	}

	// 等待 App.Run 退出
	select {
	case <-runDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("app.Run did not exit in time after Shutdown")
	}
	if runErr != nil {
		t.Fatalf("app.Run failed: %v", runErr)
	}

	// 6. 验证已排队且未派发的第 3 个实例的状态
	h3 := observed[2].Handle
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer waitCancel()
	res3, waitErr := h3.Wait(waitCtx)
	if res3.Dispatched {
		t.Errorf("instance 3 must not be dispatched, got true")
	}
	if res3.CancelErr == nil {
		t.Errorf("instance 3 CancelErr must not be nil")
	}
	if !errors.Is(waitErr, res3.CancelErr) {
		t.Errorf("expected waitErr to be res3.CancelErr (%v), got: %v", res3.CancelErr, waitErr)
	}
	if res3.RunErr != nil {
		t.Errorf("instance 3 RunErr must be nil, got: %v", res3.RunErr)
	}
	if res3.Outcome != application.OutcomeCancelled {
		t.Errorf("instance 3 Outcome must be Cancelled, got %v", res3.Outcome)
	}

	// 验证未派发的第 3 个实例绝未调用过 factory
	if cnt := atomic.LoadInt32(&factoryCount); cnt != 2 {
		t.Errorf("factory must never be called for undispatched instance 3, total factoryCount=%d", cnt)
	}
}
