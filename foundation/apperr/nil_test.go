package apperr_test

import (
	"errors"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// panicNilError 是 nil 载体的第三方原因模拟：一旦调用 Error 方法就会 panic。
type panicNilError struct{}

// Error 在 nil 载体上被调用时直接 panic，用于验证判定过程不会触发它。
func (e *panicNilError) Error() string {
	panic("nil 载体的 Error 方法不得被调用")
}

// mapError 是以 map 为载体的原因模拟，用于覆盖非指针类 nil 载体。
type mapError map[string]string

// Error 返回固定的载体描述。
func (m mapError) Error() string {
	return "map 载体原因"
}

func TestIsNil_TrueNilAndNilCarriers(t *testing.T) {
	if !apperr.IsNil(nil) {
		t.Fatal("真正的 nil 必须判定为空")
	}

	var typedNil *panicNilError
	if !apperr.IsNil(typedNil) {
		t.Fatal("装入 error 接口的 nil 指针必须判定为空")
	}

	var nilAppErr *apperr.Error
	if !apperr.IsNil(nilAppErr) {
		t.Fatal("nil 的统一错误指针必须判定为空")
	}

	var nilMap mapError
	if !apperr.IsNil(nilMap) {
		t.Fatal("nil 的 map 载体必须判定为空")
	}
}

func TestIsNil_KeepsRealErrors(t *testing.T) {
	if apperr.IsNil(errors.New("真实原因")) {
		t.Fatal("真实原因不得被判定为空")
	}
	if apperr.IsNil(&panicNilError{}) {
		t.Fatal("非 nil 的指针原因不得被判定为空")
	}
	if apperr.IsNil(mapError{"key": "value"}) {
		t.Fatal("非 nil 的 map 原因不得被判定为空")
	}
	if apperr.IsNil(apperr.New(apperr.FrameworkInternalError, "构造失败")) {
		t.Fatal("统一错误不得被判定为空")
	}
}

func TestIsNil_DoesNotCallErrorMethod(t *testing.T) {
	var typedNil *panicNilError
	// IsNil 只做类型与 nil 判定；一旦调用 Error 方法就会 panic 并使本测试失败。
	if !apperr.IsNil(typedNil) {
		t.Fatal("nil 指针必须判定为空")
	}
}

func TestNew_FiltersTypedNilCause(t *testing.T) {
	var typedNil *panicNilError
	appErr := apperr.New(apperr.FrameworkInternalError, "含 nil 载体", typedNil)
	if appErr == nil {
		t.Fatal("New 必须始终返回非 nil 错误")
	}
	if appErr.Err != nil || appErr.Unwrap() != nil {
		t.Fatalf("nil 载体原因必须被过滤: %v", appErr.Err)
	}
}

func TestNew_KeepsNonNilSiblingOfTypedNil(t *testing.T) {
	var typedNil *panicNilError
	real := errors.New("真实根因")
	appErr := apperr.New(apperr.FrameworkInternalError, "混合根因", typedNil, real)
	if appErr.Err == nil {
		t.Fatal("真实根因不得被 nil 载体遮蔽")
	}
	if !errors.Is(appErr, real) {
		t.Fatal("真实根因必须可经 errors.Is 识别")
	}
}
