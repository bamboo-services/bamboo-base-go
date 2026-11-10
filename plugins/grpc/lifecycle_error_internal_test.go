package xGrpc

import (
	"context"
	"errors"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/grpcresult"
)

// TestNormalizeLifecycleErr 验证生命周期内部失败的归一顺序与内核保持一致：
// 空原因不产生假失败、既有统一错误沿用分类、取消与超时归入控制类、
// 其余原因按模块所属错误码兜底且根因不丢失。
func TestNormalizeLifecycleErr(t *testing.T) {
	t.Run("空原因不产生假失败", func(t *testing.T) {
		if got := normalizeLifecycleErr(grpcresult.InternalError, "启动失败", nil); got != nil {
			t.Fatalf("空原因必须返回真正的 nil 指针, 得到: %+v", got)
		}
	})

	t.Run("普通原因按模块错误码兜底并保留根因", func(t *testing.T) {
		cause := errors.New("底层失败")
		got := normalizeLifecycleErr(grpcresult.InternalError, "启动失败", cause)
		if got == nil {
			t.Fatal("普通原因必须产生失败结果")
		}
		if got.Code.Code != grpcresult.InternalError.Code {
			t.Fatalf("错误码错误: 期望 %d, 得到 %d", grpcresult.InternalError.Code, got.Code.Code)
		}
		if !errors.Is(got, cause) {
			t.Fatalf("底层根因丢失: %v", got)
		}
	})

	t.Run("取消归入控制类错误码", func(t *testing.T) {
		got := normalizeLifecycleErr(grpcresult.InternalError, "停机失败", context.Canceled)
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
		got := normalizeLifecycleErr(grpcresult.InternalError, "停机失败", context.DeadlineExceeded)
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
		inner := apperr.New(grpcresult.ParameterError, "已有分类")
		cause := errors.New("底层失败")
		got := normalizeLifecycleErr(grpcresult.InternalError, "外层上下文", errors.Join(inner, cause))
		if got == nil {
			t.Fatal("已有统一错误必须产生失败结果")
		}
		if got.Code.Code != grpcresult.ParameterError.Code {
			t.Fatalf("错误码错误: 期望 %d, 得到 %d", grpcresult.ParameterError.Code, got.Code.Code)
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

	first := apperr.New(grpcresult.InternalError, "首个失败")
	if got := joinLifecycleErrs(first); got != first {
		t.Fatalf("单成员聚合必须原样返回, 得到: %+v", got)
	}

	second := apperr.New(grpcresult.InvalidState, "次个失败")
	got := joinLifecycleErrs(first, second)
	if got == nil {
		t.Fatal("多成员聚合必须产生失败结果")
	}
	if got.Code.Code != grpcresult.InternalError.Code {
		t.Fatalf("聚合错误码应沿用首个成员: 期望 %d, 得到 %d", grpcresult.InternalError.Code, got.Code.Code)
	}
	if !errors.Is(got, first) || !errors.Is(got, second) {
		t.Fatalf("聚合必须保留全部子失败: %v", got)
	}
}
