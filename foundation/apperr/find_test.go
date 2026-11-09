package apperr_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// asProjectingError 通过标准 As 钩子暴露内部统一错误，但不提供 Unwrap。
type asProjectingError struct {
	inner *apperr.Error
}

// Error 返回包装描述。
func (e *asProjectingError) Error() string {
	return "投影包装"
}

// As 按标准钩子把内部统一错误投影给调用方。
func (e *asProjectingError) As(target any) bool {
	ptr, ok := target.(**apperr.Error)
	if !ok {
		return false
	}
	*ptr = e.inner
	return true
}

// nilProjectingError 的 As 钩子命中 nil 统一错误。
type nilProjectingError struct{}

// Error 返回包装描述。
func (e *nilProjectingError) Error() string {
	return "nil 投影包装"
}

// As 命中但给出 nil 统一错误，用于验证不会遮蔽其它真实原因。
func (e *nilProjectingError) As(target any) bool {
	ptr, ok := target.(**apperr.Error)
	if !ok {
		return false
	}
	*ptr = nil
	return true
}

// cyclicError 是自引用包装，用于验证环检测。
type cyclicError struct {
	next error
}

// Error 返回环包装描述。
func (e *cyclicError) Error() string {
	return "环包装"
}

// Unwrap 返回下一层原因，测试中指向自身形成环。
func (e *cyclicError) Unwrap() error {
	return e.next
}

// asOnlyProjectingError 通过标准 As 钩子暴露统一错误，其 Unwrap 一旦被访问即 panic。
type asOnlyProjectingError struct {
	inner *apperr.Error
}

// Error 返回包装描述。
func (e *asOnlyProjectingError) Error() string {
	return "仅投影包装"
}

// As 按标准钩子把内部统一错误投影给调用方。
func (e *asOnlyProjectingError) As(target any) bool {
	ptr, ok := target.(**apperr.Error)
	if !ok {
		return false
	}
	*ptr = e.inner
	return true
}

// Unwrap 被访问即 panic：命中 As 钩子后不得再展开原因链。
func (e *asOnlyProjectingError) Unwrap() error {
	panic("命中 As 钩子后不得访问 Unwrap")
}

// sliceCauseError 是以切片为载体的原因，本身不可比较。
type sliceCauseError []int

// Error 返回固定描述。
func (s sliceCauseError) Error() string {
	return "切片载体原因"
}

// structCauseError 是值接收器包装，内部持有不可比较的原因。
type structCauseError struct {
	cause error
}

// Error 返回固定描述。
func (s structCauseError) Error() string {
	return "结构体包装原因"
}

// Unwrap 暴露内部原因。
func (s structCauseError) Unwrap() error {
	return s.cause
}

// mapCauseErrorA 与 mapCauseErrorB 是同一底层映射的两个具方法命名类型。
type mapCauseErrorA map[string]error

// Error 返回固定描述。
func (m mapCauseErrorA) Error() string {
	return "映射原因 A"
}

// Unwrap 取出映射登记的内部原因。
func (m mapCauseErrorA) Unwrap() error {
	return m["cause"]
}

// mapCauseErrorB 是同一底层映射的另一个具方法命名类型。
type mapCauseErrorB map[string]error

// Error 返回固定描述。
func (m mapCauseErrorB) Error() string {
	return "映射原因 B"
}

// Unwrap 取出映射登记的内部原因。
func (m mapCauseErrorB) Unwrap() error {
	return m["other"]
}

func TestFind_DirectAndUnknownCauses(t *testing.T) {
	if apperr.Find(nil) != nil {
		t.Fatal("nil 原因不得找到统一错误")
	}
	var typedNil *panicNilError
	if apperr.Find(typedNil) != nil {
		t.Fatal("nil 载体不得找到统一错误")
	}
	if apperr.Find(errors.New("未知原因")) != nil {
		t.Fatal("未知原因不得找到统一错误")
	}

	inner := apperr.New(apperr.FrameworkComponentNotFound, "组件未注册")
	if apperr.Find(inner) != inner {
		t.Fatal("统一错误自身必须被原样找到")
	}
}

func TestFind_FindsWrappedError(t *testing.T) {
	inner := apperr.New(apperr.FrameworkComponentNotFound, "组件未注册")
	wrapper := fmt.Errorf("装配阶段: %w", inner)
	if apperr.Find(wrapper) != inner {
		t.Fatal("包装链中的统一错误必须被找到")
	}
}

func TestFind_FindsProjectedError(t *testing.T) {
	inner := apperr.New(apperr.FrameworkComponentNotFound, "组件未注册")
	wrapper := &asProjectingError{inner: inner}

	var target *apperr.Error
	if !errors.As(wrapper, &target) || target != inner {
		t.Fatal("标准 As 钩子必须能暴露内部统一错误")
	}
	if apperr.Find(wrapper) != inner {
		t.Fatal("通过标准 As 钩子暴露的统一错误必须被找到")
	}
}

func TestFind_AsHitDoesNotTouchUnwrap(t *testing.T) {
	inner := apperr.New(apperr.FrameworkHookPanic, "钩子 panic")
	wrapper := &asOnlyProjectingError{inner: inner}

	var target *apperr.Error
	if !errors.As(wrapper, &target) || target != inner {
		t.Fatal("标准 As 钩子必须能暴露内部统一错误")
	}
	if apperr.Find(wrapper) != inner {
		t.Fatal("命中 As 钩子后必须直接返回，不得访问 Unwrap")
	}

	appErr := apperr.From(apperr.FrameworkInternalError, "外层上下文", wrapper)
	if appErr.Code.Code != 50021 {
		t.Fatalf("命中 As 钩子后不得被重新分类: %+v", appErr.Code)
	}
}

func TestFind_UncomparableCauseDoesNotPanic(t *testing.T) {
	sliceCause := sliceCauseError{1, 2}
	wrapper := structCauseError{cause: sliceCause}

	real := apperr.New(apperr.FrameworkComponentNotFound, "组件未注册", wrapper)
	found := apperr.Find(real)
	if found != real {
		t.Fatalf("真实统一错误必须被找到: %+v", found)
	}
	if found.Code.Code != 50003 {
		t.Fatalf("真实统一错误的错误码不符: %+v", found.Code)
	}

	if apperr.Find(fmt.Errorf("外层包装: %w", wrapper)) != nil {
		t.Fatal("不含统一错误的链不得返回结果")
	}
	if apperr.From(apperr.FrameworkInternalError, "外层上下文", wrapper).Code.Code != 50000 {
		t.Fatal("不含统一错误的原因必须使用调用方显式给出的错误码")
	}
}

func TestFind_SameMapDifferentNamedTypesAreDistinct(t *testing.T) {
	real := apperr.New(apperr.FrameworkComponentNotFound, "组件未注册")
	base := map[string]error{"cause": nil, "other": real}
	joined := errors.Join(mapCauseErrorA(base), mapCauseErrorB(base))

	found := apperr.Find(joined)
	if found != real {
		t.Fatalf("同一底层映射的不同具方法命名类型不得被错误去重: %+v", found)
	}
}

func TestFind_NilProjectionDoesNotShadowRealError(t *testing.T) {
	real := apperr.New(apperr.FrameworkComponentNotFound, "组件未注册")
	joined := errors.Join(&nilProjectingError{}, real)

	if apperr.Find(&nilProjectingError{}) != nil {
		t.Fatal("命中 nil 统一错误的投影不算命中")
	}
	if apperr.Find(joined) != real {
		t.Fatal("命中 nil 统一错误的投影不得遮蔽真实统一错误")
	}
}

func TestFind_DeepChainKeepsClassification(t *testing.T) {
	for _, depth := range []int{80, 100} {
		inner := apperr.New(apperr.FrameworkComponentNotFound, "组件未注册")
		inner.Data = map[string]string{"component": "database"}

		chain := error(inner)
		for i := 0; i < depth; i++ {
			chain = fmt.Errorf("第 %d 层包装: %w", i, chain)
		}

		if apperr.Find(chain) != inner {
			t.Fatalf("%d 层包装链中的统一错误必须被找到", depth)
		}

		appErr := apperr.From(apperr.FrameworkInternalError, "外层上下文", chain)
		if appErr.Code.Code != 50003 {
			t.Fatalf("%d 层包装链中的统一错误不得被重新分类: %+v", depth, appErr.Code)
		}
		if appErr.ErrorMessage != "组件未注册" {
			t.Fatalf("%d 层包装链中的安全描述必须保留: %q", depth, appErr.ErrorMessage)
		}
		if appErr.Data == nil {
			t.Fatalf("%d 层包装链中的业务数据必须保留", depth)
		}
		if appErr.Unwrap() != chain {
			t.Fatalf("%d 层包装链的原始原因必须保留", depth)
		}
		if !errors.Is(appErr, inner) {
			t.Fatalf("%d 层包装链中的统一错误必须可经 errors.Is 识别", depth)
		}
	}
}

func TestFrom_ProjectedErrorKeepsClassificationAndChain(t *testing.T) {
	inner := apperr.New(apperr.FrameworkHookPanic, "钩子 panic")
	inner.Data = map[string]string{"stage": "start"}
	wrapper := &asProjectingError{inner: inner}

	appErr := apperr.From(apperr.FrameworkInternalError, "外层上下文", wrapper)
	if appErr.Code.Code != 50021 {
		t.Fatalf("通过标准 As 钩子暴露的统一错误不得被重新分类: %+v", appErr.Code)
	}
	if appErr.ErrorMessage != "钩子 panic" {
		t.Fatalf("通过标准 As 钩子暴露的安全描述必须保留: %q", appErr.ErrorMessage)
	}
	if appErr.Data == nil {
		t.Fatal("通过标准 As 钩子暴露的业务数据必须保留")
	}
	if appErr.Unwrap() != wrapper {
		t.Fatal("原始包装原因必须保留为根因")
	}
}

func TestFrom_CyclicChainTerminates(t *testing.T) {
	cycle := &cyclicError{}
	cycle.next = cycle

	if apperr.Find(cycle) != nil {
		t.Fatal("环链中不存在统一错误")
	}
	appErr := apperr.From(apperr.FrameworkInternalError, "环链", cycle)
	if appErr == nil {
		t.Fatal("环链原因必须仍被包装为非 nil 错误")
	}
	if appErr.Code.Code != 50000 {
		t.Fatalf("环链原因必须使用调用方显式给出的错误码: %+v", appErr.Code)
	}
}
