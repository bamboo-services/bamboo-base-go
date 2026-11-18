package cron

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// 编译期契约断言：Cron 生命周期出口必须返回强类型错误指针。
var (
	_ func(*Plugin, context.Context) *apperr.Error = (*Plugin).Start
	_ func(*Plugin, context.Context) *apperr.Error = (*Plugin).Run
	_ func(*Plugin, context.Context) *apperr.Error = (*Plugin).Quiesce
	_ func(*Plugin, context.Context) *apperr.Error = (*Plugin).Stop
)

// TestSchedulerErrorCode_Frozen 断言专属错误码三元组已冻结且不再依赖框架内部错误码。
func TestSchedulerErrorCode_Frozen(t *testing.T) {
	want := apperr.ErrorCode{Code: 50040, Output: "CRON_SCHEDULER_ERROR", Message: "定时调度器错误"}
	if *SchedulerError != want {
		t.Fatalf("错误码三元组漂移: got %+v, want %+v", *SchedulerError, want)
	}
	if SchedulerError.Code == apperr.FrameworkInternalError.Code {
		t.Fatal("专属错误码不得复用框架内部错误码 50000")
	}
}

// TestLifecycleErr_UsesSchedulerError 断言生命周期兜底归一使用专属错误码并保留原始根因。
func TestLifecycleErr_UsesSchedulerError(t *testing.T) {
	cause := errors.New("boom")
	err := lifecycleErr("cron: 测试失败", cause)
	if err == nil {
		t.Fatal("期望 lifecycleErr 返回失败结果，实际得到 nil")
	}
	if err.Code.Code != 50040 {
		t.Fatalf("错误码不匹配: got %d, want %d", err.Code.Code, 50040)
	}
	if err.Code.Output != "CRON_SCHEDULER_ERROR" {
		t.Fatalf("错误输出标识不匹配: got %s, want %s", err.Code.Output, "CRON_SCHEDULER_ERROR")
	}
	if err.Code.Code == apperr.FrameworkInternalError.Code {
		t.Fatal("兜底归一不得回退框架内部错误码 50000")
	}
	if !errors.Is(err, cause) {
		t.Fatalf("期望原始根因保留，实际: %v", err)
	}
}

// TestStartFailure_UsesSchedulerError 断言缺少激活器时使用专属错误码并保留哨兵根因。
func TestStartFailure_UsesSchedulerError(t *testing.T) {
	_, factory := New(WithCronJob("0 * * * *", "target_comp"))
	p := factory()

	startErr := p.Start(context.Background())
	if startErr == nil {
		t.Fatal("期望 Start 因缺少激活器而失败，实际得到 nil")
	}
	if startErr.Code.Code != 50040 {
		t.Fatalf("错误码不匹配: got %d, want %d", startErr.Code.Code, 50040)
	}
	if !errors.Is(startErr, ErrNoActivator) {
		t.Fatalf("期望 ErrNoActivator 作为根因保留，实际: %v", startErr)
	}
}

// TestInvalidJobTarget_UsesSchedulerError 断言任务目标非法时使用专属错误码并保留哨兵根因。
func TestInvalidJobTarget_UsesSchedulerError(t *testing.T) {
	act := newFakeActivator(1)
	_, factory := New(
		WithActivator(act),
		WithCronJob("0 * * * *", ""),
	)
	p := factory()

	startErr := p.Start(context.Background())
	if startErr == nil {
		t.Fatal("期望 Start 因任务目标非法而失败，实际得到 nil")
	}
	if startErr.Code.Code != 50040 {
		t.Fatalf("错误码不匹配: got %d, want %d", startErr.Code.Code, 50040)
	}
	if !errors.Is(startErr, ErrInvalidJobTarget) {
		t.Fatalf("期望 ErrInvalidJobTarget 作为根因保留，实际: %v", startErr)
	}
}

// TestRunBeforeStart_UsesSchedulerError 断言未 Start 就 Run 时使用专属错误码并保留哨兵根因。
func TestRunBeforeStart_UsesSchedulerError(t *testing.T) {
	act := newFakeActivator(1)
	_, factory := New(
		WithActivator(act),
		WithIntervalJob(50*time.Millisecond, "target_comp"),
	)
	p := factory()

	runErr := p.Run(context.Background())
	if runErr == nil {
		t.Fatal("期望 Run 因调度器未就绪而失败，实际得到 nil")
	}
	if runErr.Code.Code != 50040 {
		t.Fatalf("错误码不匹配: got %d, want %d", runErr.Code.Code, 50040)
	}
	if !errors.Is(runErr, errSchedulerNotReady) {
		t.Fatalf("期望 errSchedulerNotReady 作为根因保留，实际: %v", runErr)
	}
}

// TestStopWithCancelledContext_UsesControlCode 断言关机上下文取消仍归一为控制类错误码 49900。
func TestStopWithCancelledContext_UsesControlCode(t *testing.T) {
	act := newFakeActivator(1)
	_, factory := New(
		WithActivator(act),
		WithIntervalJob(50*time.Millisecond, "target_comp"),
	)
	p := factory()

	if startErr := p.Start(context.Background()); startErr != nil {
		t.Fatalf("Start 失败: %v", startErr)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stopErr := p.Stop(ctx)
	if stopErr == nil {
		t.Fatal("期望 Stop 因上下文取消而失败，实际得到 nil")
	}
	if stopErr.Code.Code != 49900 {
		t.Fatalf("错误码不匹配: got %d, want %d", stopErr.Code.Code, 49900)
	}
	if !errors.Is(stopErr, context.Canceled) {
		t.Fatalf("期望 context.Canceled 作为根因保留，实际: %v", stopErr)
	}
}

// TestStopWithDeadlineExceeded_UsesControlCode 断言关机上下文超时仍归一为控制类错误码 50401。
func TestStopWithDeadlineExceeded_UsesControlCode(t *testing.T) {
	act := newFakeActivator(1)
	_, factory := New(
		WithActivator(act),
		WithIntervalJob(50*time.Millisecond, "target_comp"),
	)
	p := factory()

	if startErr := p.Start(context.Background()); startErr != nil {
		t.Fatalf("Start 失败: %v", startErr)
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	stopErr := p.Stop(ctx)
	if stopErr == nil {
		t.Fatal("期望 Stop 因上下文超时而失败，实际得到 nil")
	}
	if stopErr.Code.Code != 50401 {
		t.Fatalf("错误码不匹配: got %d, want %d", stopErr.Code.Code, 50401)
	}
	if !errors.Is(stopErr, context.DeadlineExceeded) {
		t.Fatalf("期望 context.DeadlineExceeded 作为根因保留，实际: %v", stopErr)
	}
}
