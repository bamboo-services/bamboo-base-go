package async

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/valueerr"
	"github.com/panjf2000/ants/v2"
)

// 编译期契约断言：异步提交与等待出口必须是强类型错误指针。
var (
	_ func(*Plugin, context.Context, func(context.Context), ...TaskOption) (*Task, *apperr.Error) = (*Plugin).Submit
	_ func(context.Context, func(context.Context), ...TaskOption) (*Task, *apperr.Error)          = Submit
	_ func(Pool, context.Context, func(context.Context), ...TaskOption) (*Task, *apperr.Error)    = Pool.Submit
	_ func(*Task, ...context.Context) *apperr.Error                                               = (*Task).Wait
	_ func(*Task, time.Duration) *apperr.Error                                                    = (*Task).WaitFor
)

// TestAsyncErrorCodes_Frozen 断言专属错误码三元组已冻结且不再依赖框架内部错误码。
func TestAsyncErrorCodes_Frozen(t *testing.T) {
	cases := []struct {
		name string
		code *apperr.ErrorCode
		want apperr.ErrorCode
	}{
		{
			name: "池已满",
			code: PoolFull,
			want: apperr.ErrorCode{Code: 42903, Output: "ASYNC_POOL_FULL", Message: "异步任务池已满"},
		},
		{
			name: "池错误",
			code: PoolError,
			want: apperr.ErrorCode{Code: 50041, Output: "ASYNC_POOL_ERROR", Message: "异步任务池错误"},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if *testCase.code != testCase.want {
				t.Fatalf("错误码三元组漂移: got %+v, want %+v", *testCase.code, testCase.want)
			}
			if testCase.code.Code == apperr.FrameworkInternalError.Code {
				t.Fatal("专属错误码不得复用框架内部错误码 50000")
			}
		})
	}
}

// TestPool_SubmitNilTaskFunc_UsesPoolError 断言提交空任务函数归一为池错误并保留哨兵根因。
func TestPool_SubmitNilTaskFunc_UsesPoolError(t *testing.T) {
	app, ctx := setupTestApp(t)
	defer func() {
		_ = app.Shutdown()
	}()

	pool, getErr := application.Get[Pool](ctx, PoolKey)
	if getErr != nil {
		t.Fatalf("Get[Pool] 失败: %v", getErr)
	}

	_, submitErr := pool.Submit(ctx, nil)
	if submitErr == nil {
		t.Fatal("期望提交空任务函数失败，实际得到 nil")
	}
	if submitErr.Code.Code != 50041 {
		t.Fatalf("错误码不匹配: got %d, want %d", submitErr.Code.Code, 50041)
	}
	if submitErr.Code.Output != "ASYNC_POOL_ERROR" {
		t.Fatalf("错误输出标识不匹配: got %s, want %s", submitErr.Code.Output, "ASYNC_POOL_ERROR")
	}
	if !errors.Is(submitErr, ErrNilTaskFunc) {
		t.Fatalf("期望 ErrNilTaskFunc 作为根因保留，实际: %v", submitErr)
	}
}

// TestPool_SubmitPoolFull_UsesPoolFullCode 断言池满归一为 42903 并保留哨兵根因。
func TestPool_SubmitPoolFull_UsesPoolFullCode(t *testing.T) {
	app, ctx := setupTestApp(t, WithCapacity(1))
	defer func() {
		_ = app.Shutdown()
	}()

	pool, getErr := application.Get[Pool](ctx, PoolKey)
	if getErr != nil {
		t.Fatalf("Get[Pool] 失败: %v", getErr)
	}

	started := make(chan struct{})
	release := make(chan struct{})
	first, submitErr := pool.Submit(ctx, func(context.Context) {
		close(started)
		<-release
	})
	if submitErr != nil {
		t.Fatalf("首次提交失败: %v", submitErr)
	}
	<-started

	_, fullErr := pool.Submit(ctx, func(context.Context) {})
	if fullErr == nil {
		t.Fatal("期望池满时提交失败，实际得到 nil")
	}
	if fullErr.Code.Code != 42903 {
		t.Fatalf("错误码不匹配: got %d, want %d", fullErr.Code.Code, 42903)
	}
	if fullErr.Code.Output != "ASYNC_POOL_FULL" {
		t.Fatalf("错误输出标识不匹配: got %s, want %s", fullErr.Code.Output, "ASYNC_POOL_FULL")
	}
	if !errors.Is(fullErr, ErrPoolFull) {
		t.Fatalf("期望 ErrPoolFull 作为根因保留，实际: %v", fullErr)
	}
	if !errors.Is(fullErr, ants.ErrPoolOverload) {
		t.Fatalf("期望 ants.ErrPoolOverload 作为根因保留，实际: %v", fullErr)
	}

	close(release)
	if waitErr := first.Wait(); waitErr != nil {
		t.Fatalf("首个任务等待失败: %v", waitErr)
	}
}

// TestPool_SubmitClosed_UsesPoolError 断言池关闭后提交归一为池错误并保留关闭哨兵根因。
func TestPool_SubmitClosed_UsesPoolError(t *testing.T) {
	app, ctx := setupTestApp(t)
	defer func() {
		_ = app.Shutdown()
	}()

	pool, getErr := application.Get[Pool](ctx, PoolKey)
	if getErr != nil {
		t.Fatalf("Get[Pool] 失败: %v", getErr)
	}
	plugin, ok := pool.(*Plugin)
	if !ok {
		t.Fatal("容器中的池不是 *async.Plugin")
	}

	if stopErr := plugin.Stop(context.Background()); stopErr != nil {
		t.Fatalf("Stop 失败: %v", stopErr)
	}

	_, submitErr := pool.Submit(ctx, func(context.Context) {})
	if submitErr == nil {
		t.Fatal("期望池关闭后提交失败，实际得到 nil")
	}
	if submitErr.Code.Code != 50041 {
		t.Fatalf("错误码不匹配: got %d, want %d", submitErr.Code.Code, 50041)
	}
	if !errors.Is(submitErr, ErrPoolClosed) {
		t.Fatalf("期望 ErrPoolClosed 作为根因保留，实际: %v", submitErr)
	}
}

// TestSubmit_ContainerNotMounted_PreservesFrameworkCode 断言容器未挂载时保留应用层原错误码。
func TestSubmit_ContainerNotMounted_PreservesFrameworkCode(t *testing.T) {
	_, submitErr := Submit(context.Background(), func(context.Context) {})
	if submitErr == nil {
		t.Fatal("期望未挂载容器时提交失败，实际得到 nil")
	}
	if submitErr.Code.Code != apperr.FrameworkContainerNotMounted.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", submitErr.Code.Code, apperr.FrameworkContainerNotMounted.Code)
	}
	if submitErr.Code.Code == PoolError.Code {
		t.Fatal("容器未挂载不得被改判为池错误码")
	}
}

// TestPoolUnavailableError_PreservesAppCodeAndPoolClosed 断言组件停用时保留应用层原码并可判定池已关闭。
func TestPoolUnavailableError_PreservesAppCodeAndPoolClosed(t *testing.T) {
	cases := []struct {
		name     string
		appErr   *apperr.Error
		wantCode apperr.Code
	}{
		{
			name:     "组件已停止",
			appErr:   apperr.New(apperr.FrameworkComponentStopped, "组件已停止", application.ErrComponentStopped),
			wantCode: apperr.FrameworkComponentStopped.Code,
		},
		{
			name:     "组件不可用",
			appErr:   apperr.New(apperr.FrameworkComponentUnavailable, "组件不可用", application.ErrComponentUnavailable),
			wantCode: apperr.FrameworkComponentUnavailable.Code,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := poolUnavailableError(testCase.appErr)
			if err == nil {
				t.Fatal("期望归一失败结果，实际得到 nil")
			}
			if err.Code.Code != testCase.wantCode {
				t.Fatalf("错误码不匹配: got %d, want %d", err.Code.Code, testCase.wantCode)
			}
			if !errors.Is(err, ErrPoolClosed) {
				t.Fatalf("期望 ErrPoolClosed 可判定，实际: %v", err)
			}
			if !errors.Is(err, testCase.appErr) {
				t.Fatalf("期望应用层原错误可判定，实际: %v", err)
			}
		})
	}
}

// TestPoolUnavailableError_KeepsOtherAppCodes 断言其它容器提取失败保持应用层原错误码。
func TestPoolUnavailableError_KeepsOtherAppCodes(t *testing.T) {
	appErr := apperr.New(apperr.FrameworkContainerNotMounted, "容器未挂载", application.ErrContainerNotMounted)

	err := poolUnavailableError(appErr)
	if err == nil {
		t.Fatal("期望归一失败结果，实际得到 nil")
	}
	if err.Code.Code != apperr.FrameworkContainerNotMounted.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", err.Code.Code, apperr.FrameworkContainerNotMounted.Code)
	}
	if errors.Is(err, ErrPoolClosed) {
		t.Fatalf("非停用类失败不得被改判为池已关闭，实际: %v", err)
	}
}

// TestTask_WaitNilReceiver_UsesInvalidReceiver 断言 nil 任务等待返回接收器非法错误。
func TestTask_WaitNilReceiver_UsesInvalidReceiver(t *testing.T) {
	var task *Task

	waitErr := task.Wait()
	if waitErr == nil {
		t.Fatal("期望 nil 任务等待失败，实际得到 nil")
	}
	if waitErr.Code.Code != valueerr.InvalidReceiver.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", waitErr.Code.Code, valueerr.InvalidReceiver.Code)
	}
	if !errors.Is(waitErr, ErrNotInitialized) {
		t.Fatalf("期望 ErrNotInitialized 作为根因保留，实际: %v", waitErr)
	}

	forErr := task.WaitFor(time.Millisecond)
	if forErr == nil {
		t.Fatal("期望 nil 任务限时等待失败，实际得到 nil")
	}
	if forErr.Code.Code != valueerr.InvalidReceiver.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", forErr.Code.Code, valueerr.InvalidReceiver.Code)
	}
}

// TestTask_WaitCancelledContext_UsesControlCode 断言等待期间上下文取消归一为控制类错误码 49900。
func TestTask_WaitCancelledContext_UsesControlCode(t *testing.T) {
	app, ctx := setupTestApp(t)
	defer func() {
		_ = app.Shutdown()
	}()

	release := make(chan struct{})
	started := make(chan struct{})
	task, submitErr := Submit(ctx, func(context.Context) {
		close(started)
		<-release
	})
	if submitErr != nil {
		t.Fatalf("提交失败: %v", submitErr)
	}
	<-started

	waitCtx, cancelWait := context.WithCancel(context.Background())
	cancelWait()

	waitErr := task.Wait(waitCtx)
	if waitErr == nil {
		t.Fatal("期望等待因上下文取消而失败，实际得到 nil")
	}
	if waitErr.Code.Code != 49900 {
		t.Fatalf("错误码不匹配: got %d, want %d", waitErr.Code.Code, 49900)
	}
	if waitErr.Code.Code == PoolError.Code {
		t.Fatal("控制类失败不得被改判为池错误码")
	}
	if !errors.Is(waitErr, context.Canceled) {
		t.Fatalf("期望 context.Canceled 作为根因保留，实际: %v", waitErr)
	}

	close(release)
	if finalErr := task.Wait(); finalErr != nil {
		t.Fatalf("最终等待失败: %v", finalErr)
	}
}

// TestTask_WaitForTimeout_UsesDeadlineCode 断言限时等待超时归一为控制类错误码 50401。
func TestTask_WaitForTimeout_UsesDeadlineCode(t *testing.T) {
	app, ctx := setupTestApp(t)
	defer func() {
		_ = app.Shutdown()
	}()

	release := make(chan struct{})
	started := make(chan struct{})
	task, submitErr := Submit(ctx, func(context.Context) {
		close(started)
		<-release
	})
	if submitErr != nil {
		t.Fatalf("提交失败: %v", submitErr)
	}
	<-started

	waitErr := task.WaitFor(20 * time.Millisecond)
	if waitErr == nil {
		t.Fatal("期望限时等待超时失败，实际得到 nil")
	}
	if waitErr.Code.Code != 50401 {
		t.Fatalf("错误码不匹配: got %d, want %d", waitErr.Code.Code, 50401)
	}
	if waitErr.Code.Code == PoolError.Code {
		t.Fatal("控制类失败不得被改判为池错误码")
	}
	if !errors.Is(waitErr, context.DeadlineExceeded) {
		t.Fatalf("期望 context.DeadlineExceeded 作为根因保留，实际: %v", waitErr)
	}

	close(release)
	if finalErr := task.Wait(); finalErr != nil {
		t.Fatalf("最终等待失败: %v", finalErr)
	}
}
