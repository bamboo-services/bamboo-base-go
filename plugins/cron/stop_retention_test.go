package cron

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

type blockingActivatorProxy struct {
	mu           sync.Mutex
	enteredCh    chan struct{}
	enteredOnce  sync.Once
	releaseCh    chan struct{}
	callCount    int
	lastTarget   ctxkey.Key
	instID       string
	activateHook func()
}

func newBlockingActivatorProxy(instID string) *blockingActivatorProxy {
	return &blockingActivatorProxy{
		enteredCh: make(chan struct{}),
		releaseCh: make(chan struct{}),
		instID:    instID,
	}
}

func (b *blockingActivatorProxy) Activate(ctx context.Context, compKey ctxkey.Key, _ ...application.ActivateOption) (application.RunHandle, error) {
	b.mu.Lock()
	b.callCount++
	b.lastTarget = compKey
	hook := b.activateHook
	b.mu.Unlock()

	if hook != nil {
		hook()
	}

	b.enteredOnce.Do(func() {
		close(b.enteredCh)
	})

	select {
	case <-b.releaseCh:
		return &fakeRunHandle{id: b.instID}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (b *blockingActivatorProxy) Release() {
	b.mu.Lock()
	defer b.mu.Unlock()
	select {
	case <-b.releaseCh:
	default:
		close(b.releaseCh)
	}
}

func (b *blockingActivatorProxy) Entered() <-chan struct{} {
	return b.enteredCh
}

func (b *blockingActivatorProxy) GetCallCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.callCount
}

func TestStopActiveTriggersRetentionAndDeadline(t *testing.T) {
	const targetKey ctxkey.Key = "job_test_target"
	proxy := newBlockingActivatorProxy("fake-block-inst")
	defer proxy.Release()

	_, factory := New(
		WithActivator(proxy),
		WithIntervalJob(50*time.Millisecond, targetKey),
	)

	pluginInst := factory()
	plugin, ok := pluginInst.(*Plugin)
	if !ok {
		t.Fatalf("factory 返回的不是 *Plugin: %T", pluginInst)
	}

	if err := plugin.Start(context.Background()); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}

	lifetimeCtx, cancelLifetime := context.WithCancel(context.Background())
	defer cancelLifetime()

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- plugin.Run(lifetimeCtx)
	}()

	select {
	case <-proxy.Entered():
	case <-time.After(2 * time.Second):
		t.Fatal("等待 gocron 50ms interval 触发超时，未能进入 Activate")
	}

	stopCtx, cancelStop := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancelStop()

	type stopResult struct {
		err error
	}
	stopResultCh := make(chan stopResult, 1)

	go func() {
		err := plugin.Stop(stopCtx)
		stopResultCh <- stopResult{err: err}
	}()

	select {
	case <-stopCtx.Done():
	case res := <-stopResultCh:
		t.Fatalf("Stop 在 activeTriggers 未释放、stopCtx deadline 到达前提前 return: %v", res.err)
	}

	select {
	case res := <-stopResultCh:
		t.Fatalf("Stop 在 activeTriggers 仍阻塞时提前返回: %v", res.err)
	default:
	}

	if err := plugin.Quiesce(context.Background()); err != nil {
		t.Errorf("Quiesce 返回错误: %v", err)
	}

	plugin.dispatchTick(targetKey)

	callCountBeforeRelease := proxy.GetCallCount()

	proxy.Release()

	var firstStopRes stopResult
	select {
	case firstStopRes = <-stopResultCh:
	case <-time.After(2 * time.Second):
		t.Fatal("释放 Activate 屏障后 Stop 未在预期时间内返回")
	}

	if !errors.Is(firstStopRes.err, context.DeadlineExceeded) {
		t.Fatalf("期望 Stop 返回 errors.Is(deadlineExceeded)，实际返回: %v", firstStopRes.err)
	}

	callCountAfterRelease := proxy.GetCallCount()
	if callCountAfterRelease != callCountBeforeRelease {
		t.Fatalf("Quiesce/Stop 之后不应再接纳新的触发，调用计数变化: %d -> %d", callCountBeforeRelease, callCountAfterRelease)
	}

	secondStopErr := plugin.Stop(context.Background())
	if !errors.Is(secondStopErr, context.DeadlineExceeded) {
		t.Fatalf("重复 Stop 应当共享首次结果，期望 context.DeadlineExceeded，实际返回: %v", secondStopErr)
	}

	plugin.dispatchTick(targetKey)
	if finalCallCount := proxy.GetCallCount(); finalCallCount != callCountBeforeRelease {
		t.Fatalf("Stop 退出后 dispatchTick 不应再进入 Activate，最终调用计数: %d", finalCallCount)
	}

	cancelLifetime()
	select {
	case runErr := <-runErrCh:
		if runErr != nil {
			t.Errorf("Run 退出返回非预期错误: %v", runErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("lifetimeCtx 取消后 Run 未能退出，可能存在泄漏")
	}
}
