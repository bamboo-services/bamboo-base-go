package application

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// TestShutdown_ConfigErr_BoundedChannelRepro 验证 New 配置非法时，直接 Shutdown 或 Run 失败后 Shutdown 的协调闭环。
func TestShutdown_ConfigErr_BoundedChannelRepro(t *testing.T) {
	// 构造非法配置：GraceTimeout >= ShutdownTimeout
	opts := []Option{
		WithGraceTimeout(20 * time.Second),
		WithShutdownTimeout(10 * time.Second),
	}

	app := New(opts...)
	if app.cfgErr == nil {
		t.Fatal("expected app.cfgErr to be non-nil for invalid timeout config")
	}

	// 1. New 直接 Shutdown，必须及时返回且闭环协调器
	errDone := make(chan error, 1)
	go func() {
		errDone <- app.Shutdown()
	}()

	select {
	case err := <-errDone:
		if !errors.Is(err, app.cfgErr) {
			t.Fatalf("expected err to be cfgErr %v, got %v", app.cfgErr, err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("first Shutdown timed out on cfgErr")
	}

	// 重复 Shutdown 必须复用相同结果且不挂起
	repeatDone := make(chan error, 1)
	go func() {
		repeatDone <- app.Shutdown()
	}()

	select {
	case err := <-repeatDone:
		if !errors.Is(err, app.cfgErr) {
			t.Fatalf("expected repeated Shutdown to return same cfgErr, got %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("repeated Shutdown timed out on cfgErr")
	}

	// 并发 Shutdown 必须返回相同错误且不挂起
	const concurrentCount = 5
	var wg sync.WaitGroup
	errs := make([]error, concurrentCount)
	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			errs[idx] = app.Shutdown()
		}()
	}
	wg.Wait()
	for _, e := range errs {
		if !errors.Is(e, app.cfgErr) {
			t.Fatalf("expected concurrent Shutdown to return cfgErr, got %v", e)
		}
	}

	// 验证最终状态为 StateStopped
	if st := app.State(); st != StateStopped {
		t.Fatalf("expected state to be StateStopped, got %v", st)
	}
}

// TestShutdown_ConfigErr_AfterRunRepro 验证非法配置 Run 失败后，再调用 Shutdown 的行为。
func TestShutdown_ConfigErr_AfterRunRepro(t *testing.T) {
	opts := []Option{
		WithGraceTimeout(20 * time.Second),
		WithShutdownTimeout(10 * time.Second),
	}

	app := New(opts...)
	if app.cfgErr == nil {
		t.Fatal("expected app.cfgErr to be non-nil")
	}

	runErr := app.Run()
	if !errors.Is(runErr, app.cfgErr) {
		t.Fatalf("expected Run to return cfgErr, got %v", runErr)
	}

	shutDone := make(chan error, 1)
	go func() {
		shutDone <- app.Shutdown()
	}()

	select {
	case err := <-shutDone:
		if !errors.Is(err, app.cfgErr) {
			t.Fatalf("expected Shutdown after Run cfgErr to return cfgErr, got %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Shutdown after Run cfgErr timed out")
	}

	// 重复调用
	if err := app.Shutdown(); !errors.Is(err, app.cfgErr) {
		t.Fatalf("expected repeat Shutdown after Run to return cfgErr, got %v", err)
	}

	if st := app.State(); st != StateStopped {
		t.Fatalf("expected state to be StateStopped, got %v", st)
	}
}
