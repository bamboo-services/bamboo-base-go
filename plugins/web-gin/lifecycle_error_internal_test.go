package webgin

import (
	"context"
	"errors"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/httpresult"
)

// TestNormalizeLifecycleErr 验证生命周期内部失败的归一顺序与内核保持一致：
// 空原因不产生假失败、既有统一错误沿用分类、取消与超时归入控制类、
// 其余原因按模块所属错误码兜底且根因不丢失。
func TestNormalizeLifecycleErr(t *testing.T) {
	t.Run("空原因不产生假失败", func(t *testing.T) {
		if got := normalizeLifecycleErr(httpresult.InternalError, "初始化失败", nil); got != nil {
			t.Fatalf("空原因必须返回真正的 nil 指针, 得到: %+v", got)
		}
	})

	t.Run("普通原因按模块错误码兜底并保留根因", func(t *testing.T) {
		cause := errors.New("底层失败")
		got := normalizeLifecycleErr(httpresult.InternalError, "初始化失败", cause)
		if got == nil {
			t.Fatal("普通原因必须产生失败结果")
		}
		if got.Code.Code != httpresult.InternalError.Code {
			t.Fatalf("错误码错误: 期望 %d, 得到 %d", httpresult.InternalError.Code, got.Code.Code)
		}
		if !errors.Is(got, cause) {
			t.Fatalf("底层根因丢失: %v", got)
		}
	})

	t.Run("取消归入控制类错误码", func(t *testing.T) {
		got := normalizeLifecycleErr(httpresult.InternalError, "停机失败", context.Canceled)
		if got == nil {
			t.Fatal("取消必须产生失败结果")
		}
		if got.Code.Code != control.Canceled.Code {
			t.Fatalf("错误码错误: 期望 %d, 得到 %d", control.Canceled.Code, got.Code.Code)
		}
		if !errors.Is(got, context.Canceled) {
			t.Fatalf("原生 context.Canceled 根因丢失: %v", got)
		}
	})

	t.Run("超时归入控制类错误码", func(t *testing.T) {
		got := normalizeLifecycleErr(httpresult.InternalError, "停机失败", context.DeadlineExceeded)
		if got == nil {
			t.Fatal("超时必须产生失败结果")
		}
		if got.Code.Code != control.DeadlineExceeded.Code {
			t.Fatalf("错误码错误: 期望 %d, 得到 %d", control.DeadlineExceeded.Code, got.Code.Code)
		}
		if !errors.Is(got, context.DeadlineExceeded) {
			t.Fatalf("原生 context.DeadlineExceeded 根因丢失: %v", got)
		}
	})

	t.Run("链内已有统一错误时沿用其分类", func(t *testing.T) {
		inner := apperr.New(httpresult.BadRequest, "已有分类")
		cause := errors.New("底层失败")
		got := normalizeLifecycleErr(httpresult.InternalError, "外层上下文", errors.Join(inner, cause))
		if got == nil {
			t.Fatal("已有统一错误必须产生失败结果")
		}
		if got.Code.Code != httpresult.BadRequest.Code {
			t.Fatalf("错误码错误: 期望 %d, 得到 %d", httpresult.BadRequest.Code, got.Code.Code)
		}
		if !errors.Is(got, cause) {
			t.Fatalf("整条原因链必须保留: %v", got)
		}
	})
}

// TestJoinLifecycleErrs 验证聚合语义：无有效成员返回真正的 nil 指针，
// 单成员原样返回，多成员保留全部子失败。
func TestJoinLifecycleErrs(t *testing.T) {
	if got := joinLifecycleErrs(); got != nil {
		t.Fatalf("空聚合必须返回真正的 nil 指针, 得到: %+v", got)
	}
	if got := joinLifecycleErrs(nil, nil); got != nil {
		t.Fatalf("全空聚合必须返回真正的 nil 指针, 得到: %+v", got)
	}

	first := apperr.New(httpresult.InternalError, "首个失败")
	if got := joinLifecycleErrs(first); got != first {
		t.Fatalf("单成员聚合必须原样返回, 得到: %+v", got)
	}

	second := apperr.New(httpresult.InvalidState, "次个失败")
	got := joinLifecycleErrs(first, second)
	if got == nil {
		t.Fatal("多成员聚合必须产生失败结果")
	}
	if got.Code.Code != apperr.FrameworkMultipleFailures.Code {
		t.Fatalf("多成员聚合错误码应为 FrameworkMultipleFailures: 期望 %d, 得到 %d", apperr.FrameworkMultipleFailures.Code, got.Code.Code)
	}
	if !errors.Is(got, first) || !errors.Is(got, second) {
		t.Fatalf("聚合必须保留全部子失败: %v", got)
	}
}

type customDoneContext struct {
	context.Context
	done chan struct{}
	err  error
}

func (c *customDoneContext) Done() <-chan struct{} { return c.done }
func (c *customDoneContext) Err() error            { return c.err }

// TestRequestGate_WaitDrainedCustomContextErrorFallback 验证 waitDrained 在遇到非标准 ctx.Err() 时
// 绝不因 control.Classify 返回 nil 而假成功放行，而是兜底返回模块内部错误并保留原原因。
func TestRequestGate_WaitDrainedCustomContextErrorFallback(t *testing.T) {
	gate := newRequestGate()
	if !gate.enter() {
		t.Fatal("预期准入成功")
	}
	defer gate.leave()

	ch := make(chan struct{})
	close(ch)
	customErr := errors.New("custom drain abort")
	ctx := &customDoneContext{Context: context.Background(), done: ch, err: customErr}

	got := gate.waitDrained(ctx)
	if got == nil {
		t.Fatal("非标准 ctx.Err() 不得返回 nil 假成功")
	}
	if got.Code.Code != httpresult.InternalError.Code {
		t.Fatalf("错误码错误: 期望 %d, 得到 %d", httpresult.InternalError.Code, got.Code.Code)
	}
	if !errors.Is(got, customErr) {
		t.Fatalf("期望保留原始上下文错误, 得到: %v", got)
	}
}

// TestIsSameStopContextErr 验证同源控制错误不重复聚合，异源关闭错误仍正常进入多错聚合。
func TestIsSameStopContextErr(t *testing.T) {
	cancelStop := normalizeLifecycleErr(httpresult.InternalError, "停机等待被取消或超时", context.Canceled)
	cancelShut := normalizeLifecycleErr(httpresult.InternalError, "HTTP 服务优雅关闭失败", context.Canceled)
	if !isSameStopContextErr(cancelStop, cancelShut, context.Canceled) {
		t.Fatal("同源 context.Canceled 应识别为同一次停机取消")
	}

	listenerErr := normalizeLifecycleErr(httpresult.InternalError, "HTTP 服务优雅关闭失败", errors.New("close tcp failed"))
	if isSameStopContextErr(cancelStop, listenerErr, context.Canceled) {
		t.Fatal("异源监听器关闭错误不得被当作同源取消忽略")
	}
}
