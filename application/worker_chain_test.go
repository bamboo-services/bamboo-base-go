package application

import (
	"context"
	"errors"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

// errWorkerBoom 是 Worker.Run 返回的自定义哨兵错误，
// 用于验证看门狗链路在包装后仍保留原始错误类型与 cause。
var errWorkerBoom = errors.New("worker boom")

// 提取链中最外层的统一错误对象。
func extractAppErr(t *testing.T, err error) *apperr.Error {
	t.Helper()
	var target *apperr.Error
	if !errors.As(err, &target) {
		t.Fatalf("expected *apperr.Error in chain, got %v", err)
	}
	return target
}

// 1. Worker.Run 返回自定义哨兵错误时，App.Run 返回值必须同时保留：
// ErrWorkerUnexpectedExit、原始哨兵错误与 50023 统一错误码。
func TestApp_WorkerUnexpectedExit_PreservesErrorChain(t *testing.T) {
	app := New()
	workerKey := ctxkey.Key("chain_worker")

	if err := app.RegisterWorker(PluginInfo{Key: workerKey}, func() RunnablePlugin {
		return &mockWorker{
			runFunc: func(ctx context.Context) error {
				return errWorkerBoom
			},
		}
	}); err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	err := app.Run()
	if err == nil {
		t.Fatal("expected error from App.Run, got nil")
	}
	if !errors.Is(err, ErrWorkerUnexpectedExit) {
		t.Fatalf("expected ErrWorkerUnexpectedExit in chain, got %v", err)
	}
	if !errors.Is(err, errWorkerBoom) {
		t.Fatalf("expected original sentinel errWorkerBoom in chain, got %v", err)
	}
	appErr := extractAppErr(t, err)
	if appErr.Code.Code != 50023 {
		t.Fatalf("expected outer code 50023, got %d", appErr.Code.Code)
	}
	if appErr.Code.Output != "FRAMEWORK_WORKER_UNEXPECTED_EXIT" {
		t.Fatalf("expected output FRAMEWORK_WORKER_UNEXPECTED_EXIT, got %s", appErr.Code.Output)
	}
}

// 2. Worker.Run 提前返回 nil（Running 阶段非预期停止）仍判定为失败，
// 且错误链中最外层统一错误码为 50023。
func TestApp_WorkerUnexpectedExit_NilErrorStillFails(t *testing.T) {
	app := New()
	workerKey := ctxkey.Key("nil_worker")

	if err := app.RegisterWorker(PluginInfo{Key: workerKey}, func() RunnablePlugin {
		return &mockWorker{
			runFunc: func(ctx context.Context) error {
				return nil
			},
		}
	}); err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	err := app.Run()
	if err == nil {
		t.Fatal("expected error from App.Run, got nil")
	}
	if !errors.Is(err, ErrWorkerUnexpectedExit) {
		t.Fatalf("expected ErrWorkerUnexpectedExit in chain, got %v", err)
	}
	appErr := extractAppErr(t, err)
	if appErr.Code.Code != 50023 {
		t.Fatalf("expected outer code 50023, got %d", appErr.Code.Code)
	}
}

// 3. Worker.Run 返回的原始错误本身是 *apperr.Error 时，
// 外层仍为 50023，且内层统一错误对象在链中依然可达。
func TestApp_WorkerUnexpectedExit_PreservesNestedAppErr(t *testing.T) {
	app := New()
	workerKey := ctxkey.Key("nested_worker")
	inner := apperr.New(apperr.FrameworkComponentNotFound, apperr.ErrMessage("inner failure"))

	if err := app.RegisterWorker(PluginInfo{Key: workerKey}, func() RunnablePlugin {
		return &mockWorker{
			runFunc: func(ctx context.Context) error {
				return inner
			},
		}
	}); err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	err := app.Run()
	if err == nil {
		t.Fatal("expected error from App.Run, got nil")
	}
	if !errors.Is(err, ErrWorkerUnexpectedExit) {
		t.Fatalf("expected ErrWorkerUnexpectedExit in chain, got %v", err)
	}
	appErr := extractAppErr(t, err)
	if appErr.Code.Code != 50023 {
		t.Fatalf("expected outer code 50023, got %d", appErr.Code.Code)
	}
	if !errors.Is(err, inner) {
		t.Fatalf("expected inner *apperr.Error reachable in chain, got %v", err)
	}
}
