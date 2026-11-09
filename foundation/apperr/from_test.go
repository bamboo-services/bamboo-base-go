package apperr_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// wrappedCause 是带解包能力的自定义原因，用于验证原因链遍历与诊断提取。
type wrappedCause struct {
	msg string
	err error
}

// Error 返回自定义原因描述。
func (e *wrappedCause) Error() string {
	return e.msg
}

// Unwrap 暴露下一层原因。
func (e *wrappedCause) Unwrap() error {
	return e.err
}

func TestFrom_NilCauseReturnsNilPointer(t *testing.T) {
	if appErr := apperr.From(apperr.FrameworkInternalError, "无原因", nil); appErr != nil {
		t.Fatalf("nil 原因必须返回 nil 指针: %+v", appErr)
	}

	var typedNil *panicNilError
	if appErr := apperr.From(apperr.FrameworkInternalError, "nil 载体原因", typedNil); appErr != nil {
		t.Fatalf("nil 载体原因必须返回 nil 指针: %+v", appErr)
	}
}

func TestFrom_UnknownCauseUsesCallerCodeAndMessage(t *testing.T) {
	cause := errors.New("底层连接失败")
	appErr := apperr.From(apperr.FrameworkComponentUnavailable, "组件不可用", cause)
	if appErr == nil {
		t.Fatal("未知原因必须返回非 nil 错误")
	}
	if appErr.Code.Code != 50006 || appErr.Code.Output != "FRAMEWORK_COMPONENT_UNAVAILABLE" {
		t.Fatalf("必须使用调用方显式给出的错误码: %+v", appErr.Code)
	}
	if appErr.ErrorMessage != "组件不可用" {
		t.Fatalf("安全描述必须来自调用方: %q", appErr.ErrorMessage)
	}
	if strings.Contains(string(appErr.ErrorMessage), "底层连接失败") {
		t.Fatal("安全描述不得自动复制原因文本")
	}
	if appErr.Unwrap() != cause {
		t.Fatal("原始原因必须原样保留")
	}
	if !errors.Is(appErr, cause) {
		t.Fatal("原始原因必须可经 errors.Is 识别")
	}
}

func TestFrom_NilCodeFallsBackToInternalError(t *testing.T) {
	appErr := apperr.From(nil, "缺省错误码", errors.New("原因"))
	if appErr.Code.Code != 50000 || appErr.Code.Output != "FRAMEWORK_INTERNAL_ERROR" {
		t.Fatalf("缺省错误码必须回退为框架内部错误: %+v", appErr.Code)
	}
}

func TestFrom_ExistingErrorKeepsClassification(t *testing.T) {
	inner := apperr.New(apperr.FrameworkComponentNotFound, "组件 database 未注册")
	inner.Data = map[string]string{"component": "database"}

	appErr := apperr.From(apperr.FrameworkInternalError, "外层上下文", inner)
	if appErr != inner {
		t.Fatal("已有统一错误必须原样返回，不得被重新分类")
	}
	if appErr.Code.Code != 50003 {
		t.Fatalf("已有统一错误的错误码必须保留: %+v", appErr.Code)
	}
	if appErr.ErrorMessage != "组件 database 未注册" {
		t.Fatalf("已有统一错误的安全描述必须保留: %q", appErr.ErrorMessage)
	}
	if appErr.Data == nil {
		t.Fatal("已有统一错误的业务数据必须保留")
	}
}

func TestFrom_WrappedExistingErrorKeepsChainAndClassification(t *testing.T) {
	inner := apperr.New(apperr.FrameworkComponentNotFound, "组件 database 未注册")
	inner.Data = map[string]string{"component": "database"}
	wrapper := fmt.Errorf("启动阶段失败: %w", inner)

	appErr := apperr.From(apperr.FrameworkInternalError, "外层上下文", wrapper)
	if appErr.Code.Code != 50003 {
		t.Fatalf("包装链中的统一错误不得被重新分类: %+v", appErr.Code)
	}
	if appErr.ErrorMessage != "组件 database 未注册" {
		t.Fatalf("包装链中的安全描述必须保留: %q", appErr.ErrorMessage)
	}
	if appErr.Data == nil {
		t.Fatal("包装链中的业务数据必须保留")
	}
	if appErr.Unwrap() != wrapper {
		t.Fatal("原包装原因必须保留为根因")
	}
	if !errors.Is(appErr, inner) {
		t.Fatal("内层统一错误必须仍可经 errors.Is 识别")
	}
}

func TestFrom_KeepsDiagnosticObjectInChain(t *testing.T) {
	sentinel := errors.New("诊断根因")
	diagnostic := &wrappedCause{msg: "钩子异常", err: sentinel}
	inner := apperr.New(apperr.FrameworkHookPanic, "钩子 panic", diagnostic)

	appErr := apperr.From(apperr.FrameworkInternalError, "外层上下文", inner)
	if appErr.Code.Code != 50021 {
		t.Fatalf("已有统一错误不得被重新分类: %+v", appErr.Code)
	}

	var target *wrappedCause
	if !errors.As(appErr, &target) {
		t.Fatal("诊断对象必须可经 errors.As 提取")
	}
	if target.msg != "钩子异常" {
		t.Fatalf("诊断对象内容不符: %q", target.msg)
	}
	if !errors.Is(appErr, sentinel) {
		t.Fatal("诊断根因必须可经 errors.Is 识别")
	}
}

func TestFrom_JoinedCausesKeepAllSiblings(t *testing.T) {
	inner := apperr.New(apperr.FrameworkComponentNotFound, "组件未注册")
	sibling := errors.New("兄弟原因")
	joined := errors.Join(inner, sibling)

	appErr := apperr.From(apperr.FrameworkInternalError, "外层上下文", joined)
	if appErr.Unwrap() != joined {
		t.Fatal("原聚合原因必须保留为根因")
	}
	if appErr.Code.Code != 50003 {
		t.Fatalf("包装链中的统一错误不得被重新分类: %+v", appErr.Code)
	}
	if !errors.Is(appErr, inner) {
		t.Fatal("聚合链中的统一错误必须可经 errors.Is 识别")
	}
	if !errors.Is(appErr, sibling) {
		t.Fatal("聚合链中的兄弟原因不得丢失")
	}
}

func TestFrom_NilEntryDoesNotShadowStructuredError(t *testing.T) {
	var typedNil *panicNilError
	inner := apperr.New(apperr.FrameworkComponentNotFound, "组件未注册")
	joined := errors.Join(typedNil, inner)

	appErr := apperr.From(apperr.FrameworkInternalError, "外层上下文", joined)
	if appErr.Code.Code != 50003 {
		t.Fatalf("nil 条目不得遮蔽后续真实统一错误: %+v", appErr.Code)
	}
	if appErr.Unwrap() != joined {
		t.Fatal("原聚合原因必须保留为根因")
	}
}

func TestFrom_DoesNotMutateExistingError(t *testing.T) {
	inner := apperr.New(apperr.FrameworkComponentNotFound, "组件未注册")
	wrapper := fmt.Errorf("外层包装: %w", inner)

	appErr := apperr.From(apperr.FrameworkInternalError, "外层上下文", wrapper)
	if appErr == inner {
		t.Fatal("包装链场景必须返回新的统一错误")
	}
	if inner.Code.Code != 50003 || inner.ErrorMessage != "组件未注册" {
		t.Fatalf("既有错误不得被改写: %+v", inner)
	}
	if inner.Unwrap() != nil {
		t.Fatal("既有错误的根因不得被改写")
	}
}
