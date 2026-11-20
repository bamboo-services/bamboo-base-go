package apperr_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// joinChildren 取出聚合结果根因中的子原因列表，用于校验顺序与完整性。
func joinChildren(t *testing.T, appErr *apperr.Error) []error {
	t.Helper()
	multi, ok := appErr.Unwrap().(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("聚合结果的根因必须支持多原因解包: %T", appErr.Unwrap())
	}
	return multi.Unwrap()
}

func TestJoin_NoFailureReturnsNilPointer(t *testing.T) {
	if appErr := apperr.Join(apperr.FrameworkMultipleFailures, "无失败"); appErr != nil {
		t.Fatalf("无失败时必须返回 nil 指针: %+v", appErr)
	}
	if appErr := apperr.Join(apperr.FrameworkMultipleFailures, "仅空值", nil, nil); appErr != nil {
		t.Fatalf("仅含 nil 指针时必须返回 nil 指针: %+v", appErr)
	}
}

func TestJoin_SingleFailureReturnsSameObject(t *testing.T) {
	only := apperr.New(apperr.FrameworkComponentNotFound, "组件未注册")
	appErr := apperr.Join(apperr.FrameworkMultipleFailures, "单项失败", nil, only, nil)
	if appErr != only {
		t.Fatal("单项失败必须原样返回，不得额外套壳")
	}
	if appErr.Code.Code != 50003 {
		t.Fatalf("单项失败必须保留自身错误码: %+v", appErr.Code)
	}
}

func TestJoin_MultipleFailuresKeepCausesAndOrder(t *testing.T) {
	first := apperr.New(apperr.FrameworkComponentNotFound, "组件未注册")
	second := apperr.New(apperr.FrameworkComponentUnavailable, "组件不可用")

	appErr := apperr.Join(apperr.FrameworkMultipleFailures, "多个阶段失败", first, second)
	if appErr == nil {
		t.Fatal("多项失败必须返回非 nil 错误")
	}
	if appErr.Code.Code != 50020 || appErr.Code.Output != "FRAMEWORK_MULTIPLE_FAILURES" {
		t.Fatalf("必须使用调用方显式给出的总码: %+v", appErr.Code)
	}
	if appErr.ErrorMessage != "多个阶段失败" {
		t.Fatalf("总描述必须来自调用方: %q", appErr.ErrorMessage)
	}
	if !errors.Is(appErr, first) || !errors.Is(appErr, second) {
		t.Fatal("全部子失败必须可经 errors.Is 遍历")
	}

	children := joinChildren(t, appErr)
	if len(children) != 2 {
		t.Fatalf("子原因数量不符: %d", len(children))
	}
	if children[0] != error(first) || children[1] != error(second) {
		t.Fatalf("子原因顺序必须与入参一致: %v", children)
	}
}

func TestJoin_DoesNotAutoSelectAggregateCode(t *testing.T) {
	first := apperr.New(apperr.FrameworkComponentNotFound, "组件未注册")
	second := apperr.New(apperr.FrameworkComponentUnavailable, "组件不可用")

	appErr := apperr.Join(nil, "多个阶段失败", first, second)
	if appErr.Code.Code != 50000 || appErr.Code.Output != "FRAMEWORK_INTERNAL_ERROR" {
		t.Fatalf("缺省总码必须回退为框架内部错误: %+v", appErr.Code)
	}
}

func TestJoin_KeepsCancellationCause(t *testing.T) {
	cancelled := apperr.New(apperr.FrameworkComponentStopped, "组件已停止", context.Canceled)
	other := apperr.New(apperr.FrameworkComponentUnavailable, "组件不可用")

	appErr := apperr.Join(apperr.FrameworkMultipleFailures, "多个阶段失败", cancelled, other)
	if !errors.Is(appErr, context.Canceled) {
		t.Fatal("取消原因不得被聚合吞掉")
	}
}

func TestJoin_NestedAggregationKeepsOrder(t *testing.T) {
	first := apperr.New(apperr.FrameworkComponentNotFound, "组件未注册")
	second := apperr.New(apperr.FrameworkComponentUnavailable, "组件不可用")
	third := apperr.New(apperr.FrameworkWorkerUnexpectedExit, "后台服务意外退出")

	inner := apperr.Join(apperr.FrameworkMultipleFailures, "内层聚合", first, second)
	outer := apperr.Join(apperr.FrameworkMultipleFailures, "外层聚合", inner, third)

	children := joinChildren(t, outer)
	if len(children) != 2 {
		t.Fatalf("外层子原因数量不符: %d", len(children))
	}
	if children[0] != error(inner) || children[1] != error(third) {
		t.Fatalf("外层子原因顺序必须与入参一致: %v", children)
	}
	if !errors.Is(outer, first) || !errors.Is(outer, second) || !errors.Is(outer, third) {
		t.Fatal("嵌套聚合必须保留全部子失败")
	}
}

func TestJoin_DiagnosticMembersRemainTraversable(t *testing.T) {
	stopRoot := errors.New("底层停止原因")
	stopDiagnostic := &wrappedCause{msg: "停止超时", err: stopRoot}
	exitRoot := errors.New("底层退出原因")

	memberA := apperr.New(apperr.FrameworkShutdownIncomplete, "组件 A 未停止", stopDiagnostic)
	memberB := apperr.New(apperr.FrameworkWorkerUnexpectedExit, "组件 B 意外退出", exitRoot)

	appErr := apperr.Join(apperr.FrameworkShutdownIncomplete, "框架关闭未完成", memberA, memberB)
	if appErr.Code.Code != 50022 {
		t.Fatalf("必须使用调用方显式给出的总码: %+v", appErr.Code)
	}

	var target *wrappedCause
	if !errors.As(appErr, &target) {
		t.Fatal("诊断对象必须可经 errors.As 提取")
	}
	if target.msg != "停止超时" {
		t.Fatalf("诊断对象内容不符: %q", target.msg)
	}
	if !errors.Is(appErr, stopRoot) {
		t.Fatal("成员根因必须可经 errors.Is 识别")
	}
	if !errors.Is(appErr, exitRoot) {
		t.Fatal("全部成员根因必须可经 errors.Is 识别")
	}

	data, err := json.Marshal(appErr)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}
	jsonStr := string(data)
	for _, leaked := range []string{"底层停止原因", "底层退出原因", "停止超时"} {
		if strings.Contains(jsonStr, leaked) {
			t.Fatalf("内部原因泄漏到 JSON: %s", jsonStr)
		}
	}
	if !strings.Contains(jsonStr, "FRAMEWORK_SHUTDOWN_INCOMPLETE") {
		t.Fatalf("JSON 缺少错误码: %s", jsonStr)
	}
}
