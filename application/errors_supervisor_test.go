package application

import (
	"context"
	"errors"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// TestAggregateErrs_UsesFrameworkMultipleFailures 断言多错误聚合遵循 R04：
// 空集返回 nil，单错原样返回，多错使用 FrameworkMultipleFailures (50020) 且保留全部子错误可遍历。
func TestAggregateErrs_UsesFrameworkMultipleFailures(t *testing.T) {
	if got := aggregateErrs(); got != nil {
		t.Fatalf("空聚合期望 nil，实际 %+v", got)
	}
	if got := aggregateErrs(nil, nil); got != nil {
		t.Fatalf("全 nil 聚合期望 nil，实际 %+v", got)
	}

	first := apperr.New(apperr.FrameworkComponentInvalidState, "启动失败")
	if got := aggregateErrs(first); got != first {
		t.Fatalf("单成员聚合期望原样返回，实际 %+v", got)
	}

	second := apperr.New(apperr.FrameworkInternalError, "关机失败")
	joined := aggregateErrs(first, nil, second)
	if joined == nil {
		t.Fatal("多成员聚合期望非 nil")
	}
	if joined.Code.Code != apperr.FrameworkMultipleFailures.Code {
		t.Fatalf("多成员聚合错误码 = %d，期望 %d (FrameworkMultipleFailures)",
			joined.Code.Code, apperr.FrameworkMultipleFailures.Code)
	}
	if !errors.Is(joined, first) || !errors.Is(joined, second) {
		t.Fatalf("多成员聚合必须保留全部子错误: %v", joined)
	}
}

// TestComputeWaitError_DispatchedPreRunCancellationPreservesCancelErr 断言已派发但在 Start/Run 前
// 被取消的 RunResult 不会在 computeWaitError 中静默丢弃 CancelErr。
func TestComputeWaitError_DispatchedPreRunCancellationPreservesCancelErr(t *testing.T) {
	cancelErr := apperr.New(apperr.FrameworkComponentInvalidState, "父级组件正在关闭", ErrParentClosing)
	stopErr := apperr.New(apperr.FrameworkInternalError, "停止失败")

	t.Run("已派发且Start前取消保留CancelErr", func(t *testing.T) {
		res := RunResult{
			Dispatched:      true,
			Outcome:         OutcomeCancelled,
			CancelRequested: true,
			CancelErr:       cancelErr,
		}
		got := res.computeWaitError()
		if got != cancelErr {
			t.Fatalf("期望返回 cancelErr (%v)，实际 %v", cancelErr, got)
		}
	})

	t.Run("已派发Run前取消且Stop成功保留CancelErr", func(t *testing.T) {
		res := RunResult{
			Dispatched:      true,
			Outcome:         OutcomeCancelled,
			CancelRequested: true,
			CancelErr:       cancelErr,
		}
		got := res.computeWaitError()
		if got != cancelErr {
			t.Fatalf("期望返回 cancelErr (%v)，实际 %v", cancelErr, got)
		}
	})

	t.Run("已派发Run前取消且Stop失败聚合两项错误", func(t *testing.T) {
		res := RunResult{
			Dispatched:      true,
			Outcome:         OutcomeFailed,
			CancelRequested: true,
			CancelErr:       cancelErr,
			StopErr:         stopErr,
		}
		got := res.computeWaitError()
		if got == nil {
			t.Fatal("期望返回非 nil 聚合错误")
		}
		if got.Code.Code != apperr.FrameworkMultipleFailures.Code {
			t.Fatalf("期望 FrameworkMultipleFailures (%d)，实际 %d", apperr.FrameworkMultipleFailures.Code, got.Code.Code)
		}
		if !errors.Is(got, cancelErr) || !errors.Is(got, stopErr) {
			t.Fatalf("聚合错误必须同时保留 cancelErr 与 stopErr: %v", got)
		}
	})

	t.Run("正常执行Run成功时晚到CancelErr不污染成功结果", func(t *testing.T) {
		res := RunResult{
			Dispatched:      true,
			Outcome:         OutcomeSuccess,
			CancelRequested: true,
			CancelErr:       cancelErr,
		}
		if got := res.computeWaitError(); got != nil {
			t.Fatalf("OutcomeSuccess 不得返回错误，实际 %v", got)
		}
	})

	t.Run("正常执行Run响应取消时保留RunErr不重复混入CancelErr", func(t *testing.T) {
		runErr := normalizeErr(apperr.FrameworkInternalError, "运行被取消", context.Canceled)
		res := RunResult{
			Dispatched:      true,
			Outcome:         OutcomeCancelled,
			CancelRequested: true,
			CancelErr:       cancelErr,
			RunErr:          runErr,
		}
		if got := res.computeWaitError(); got != runErr {
			t.Fatalf("期望返回 runErr (%v)，实际 %v", runErr, got)
		}
	})
}
