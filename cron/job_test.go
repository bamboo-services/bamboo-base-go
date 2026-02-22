package xCron

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewJob(t *testing.T) {
	var called atomic.Bool
	job := NewJob("@every 1m", func() {
		called.Store(true)
	})

	if job.Spec != "@every 1m" {
		t.Errorf("期望 spec 为 @every 1m，实际为 %s", job.Spec)
	}
	if job.Func == nil {
		t.Error("Func 不应为 nil")
	}

	// 验证函数可以正常适配
	jobFn, err := AdaptJob(job.Func)
	if err != nil {
		t.Fatalf("AdaptJob 失败: %v", err)
	}
	jobFn(context.Background())
	if !called.Load() {
		t.Error("函数未被调用")
	}
}

func TestNewJob_WithContext(t *testing.T) {
	var receivedCtx atomic.Pointer[context.Context]
	job := NewJob("*/5 * * * *", func(ctx context.Context) {
		receivedCtx.Store(&ctx)
	})

	if job.Spec != "*/5 * * * *" {
		t.Errorf("期望 spec 为 */5 * * * *，实际为 %s", job.Spec)
	}

	jobFn, err := AdaptJob(job.Func)
	if err != nil {
		t.Fatalf("AdaptJob 失败: %v", err)
	}

	ctx := context.Background()
	jobFn(ctx)

	if receivedCtx.Load() == nil {
		t.Error("上下文未被传递")
	}
}

func TestAdaptJob_NilFunc(t *testing.T) {
	_, err := AdaptJob(nil)
	if err == nil {
		t.Error("期望返回错误，但返回 nil")
	}
}

func TestAdaptJob_NonFunc(t *testing.T) {
	_, err := AdaptJob("not a function")
	if err == nil {
		t.Error("期望返回错误，但返回 nil")
	}
}

func TestAdaptJob_FuncNoArgs(t *testing.T) {
	var called atomic.Bool
	fn := func() {
		called.Store(true)
	}

	jobFn, err := AdaptJob(fn)
	if err != nil {
		t.Fatalf("AdaptJob 失败: %v", err)
	}

	ctx := context.Background()
	jobFn(ctx)

	if !called.Load() {
		t.Error("函数未被调用")
	}
}

func TestAdaptJob_FuncWithContext(t *testing.T) {
	var receivedCtx atomic.Pointer[context.Context]
	fn := func(ctx context.Context) {
		receivedCtx.Store(&ctx)
	}

	jobFn, err := AdaptJob(fn)
	if err != nil {
		t.Fatalf("AdaptJob 失败: %v", err)
	}

	ctx := context.Background()
	jobFn(ctx)

	if receivedCtx.Load() == nil {
		t.Error("上下文未被传递")
	}
}

func TestAdaptJob_FuncWithWrongArgType(t *testing.T) {
	fn := func(s string) {}

	_, err := AdaptJob(fn)
	if err == nil {
		t.Error("期望返回错误，但返回 nil")
	}
}

func TestAdaptJob_FuncWithTooManyArgs(t *testing.T) {
	fn := func(ctx context.Context, s string) {}

	_, err := AdaptJob(fn)
	if err == nil {
		t.Error("期望返回错误，但返回 nil")
	}
}

func TestAdaptJob_ContextCancellation(t *testing.T) {
	var cancelled atomic.Bool
	fn := func(ctx context.Context) {
		<-ctx.Done()
		cancelled.Store(true)
	}

	jobFn, err := AdaptJob(fn)
	if err != nil {
		t.Fatalf("AdaptJob 失败: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		jobFn(ctx)
		close(done)
	}()

	// 等待一段时间后取消
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
		if !cancelled.Load() {
			t.Error("上下文取消未被正确处理")
		}
	case <-time.After(1 * time.Second):
		t.Error("函数执行超时")
	}
}
