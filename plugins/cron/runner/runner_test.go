package xCronRunner

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	xCron "github.com/bamboo-services/bamboo-base-go/plugins/cron"
)

func TestNew_WithSeconds(t *testing.T) {
	var executed atomic.Int32

	runner := New(
		WithSeconds(),
		WithRegister(
			xCron.NewJob("*/1 * * * * *", func(ctx context.Context) {
				executed.Add(1)
			}),
		),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()

	runner(ctx)

	if executed.Load() < 2 {
		t.Errorf("期望至少执行 2 次，实际执行 %d 次", executed.Load())
	}
}

func TestNew_WithRegister(t *testing.T) {
	var executed atomic.Bool

	runner := New(
		WithSeconds(),
		WithRegister(
			xCron.NewJob("* * * * * *", func() {
				executed.Store(true)
			}),
		),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	runner(ctx)

	if !executed.Load() {
		t.Error("定时任务未执行")
	}
}

func TestNew_WithGracefulStopTimeout(t *testing.T) {
	runner := New(
		WithGracefulStopTimeout(100*time.Millisecond),
		WithRegister(
			xCron.NewJob("0 0 * * *", func() {}),
		),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	runner(ctx)
	elapsed := time.Since(start)

	// 应该在超时时间内完成关闭
	if elapsed > 500*time.Millisecond {
		t.Errorf("关闭耗时过长: %v", elapsed)
	}
}

func TestNew_MultipleJobs(t *testing.T) {
	var counter atomic.Int32

	runner := New(
		WithSeconds(),
		WithRegister(
			xCron.NewJob("* * * * * *", func() {
				counter.Add(1)
			}),
			xCron.NewJob("* * * * * *", func(ctx context.Context) {
				counter.Add(10)
			}),
		),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()

	runner(ctx)

	// 两个任务各执行约 2 次（第1秒和第2秒）
	// 任务1: +1, 任务2: +10, 每轮合计 +11
	total := counter.Load()
	if total < 20 {
		t.Errorf("期望至少执行合计 20 (实际: %d)", total)
	}
}

func TestNew_InvalidSpec(t *testing.T) {
	var executed atomic.Bool

	runner := New(
		WithRegister(
			xCron.NewJob("invalid spec", func() {
				executed.Store(true)
			}),
		),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 不应该 panic
	runner(ctx)

	if executed.Load() {
		t.Error("无效的 spec 不应该执行任务")
	}
}

func TestNew_NilJobFunc(t *testing.T) {
	runner := New(
		WithRegister(
			xCron.Job{
				Spec: "@every 1s",
				Func: nil,
			},
		),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// 不应该 panic
	runner(ctx)
}

func TestNew_EmptySpec(t *testing.T) {
	var executed atomic.Bool

	runner := New(
		WithRegister(
			xCron.NewJob("", func() {
				executed.Store(true)
			}),
		),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	runner(ctx)

	if executed.Load() {
		t.Error("空 spec 不应该执行任务")
	}
}

func TestNew_RuntimeOption(t *testing.T) {
	var executed atomic.Bool

	runner := New() // 不注册任务

	// 给足够时间让任务执行
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	// 运行时注册任务
	runner(ctx,
		WithSeconds(),
		WithRegister(
			xCron.NewJob("* * * * * *", func() {
				executed.Store(true)
			}),
		),
	)

	if !executed.Load() {
		t.Error("运行时注册的任务未执行")
	}
}

func TestNew_MultipleRuntimeOptions(t *testing.T) {
	var counter atomic.Int32

	runner := New()

	// 给足够时间让任务执行
	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()

	// 多个运行时选项
	runner(ctx,
		WithSeconds(),
		WithRegister(
			xCron.NewJob("* * * * * *", func() {
				counter.Add(1)
			}),
		),
		WithRegister(
			xCron.NewJob("* * * * * *", func(ctx context.Context) {
				counter.Add(10)
			}),
		),
	)

	// 两任务在 2.5s 内各执行约 2 次，合计应该 > 20
	if counter.Load() < 20 {
		t.Errorf("期望至少执行合计 20 (实际: %d)", counter.Load())
	}
}
