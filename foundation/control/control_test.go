package control_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
)

// Classify 的公开签名必须与跨分工约定一致。
var _ func(error, control.Phase) *apperr.Error = control.Classify

// nilCarrier 是测试用的第三方错误类型，用于构造装入 error 接口的 nil 指针。
type nilCarrier struct{}

// Error 实现 error 接口。
func (*nilCarrier) Error() string { return "nil 载体" }

// asCarrier 是自定义 As 的错误类型，用于验证结构化原因查找尊重调用方自定义的 As。
type asCarrier struct {
	inner *apperr.Error
}

// Error 实现 error 接口。
func (c *asCarrier) Error() string { return "自定义 As 载体" }

// As 把内部结构化错误暴露给 errors.As 与结构化原因查找。
func (c *asCarrier) As(target any) bool {
	appErr, ok := target.(**apperr.Error)
	if !ok {
		return false
	}
	*appErr = c.inner
	return true
}

// TestClassifyNilCauseReturnsNil 验证 nil 与 nil 载体都不产生假失败。
func TestClassifyNilCauseReturnsNil(t *testing.T) {
	var typedNil *apperr.Error
	var typedNilCarrier *nilCarrier

	cases := []struct {
		name  string
		cause error
	}{
		{name: "真正的nil", cause: nil},
		{name: "装入error接口的nil错误指针", cause: typedNil},
		{name: "装入error接口的nil第三方指针", cause: typedNilCarrier},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := control.Classify(testCase.cause, control.PhaseOperation); got != nil {
				t.Fatalf("空原因必须返回 nil: %+v", got)
			}
			if got := control.Classify(testCase.cause, control.PhaseRequestRead); got != nil {
				t.Fatalf("空原因必须返回 nil: %+v", got)
			}
		})
	}
}

// TestClassifyControlCauses 按表驱动断言取消与超时的错误码归属。
func TestClassifyControlCauses(t *testing.T) {
	cases := []struct {
		name       string
		cause      error
		phase      control.Phase
		wantCode   apperr.Code
		wantOutput apperr.Output
	}{
		{
			name:       "取消与阶段无关",
			cause:      context.Canceled,
			phase:      control.PhaseOperation,
			wantCode:   49900,
			wantOutput: "CONTROL_CANCELED",
		},
		{
			name:       "请求阶段取消仍为取消",
			cause:      context.Canceled,
			phase:      control.PhaseRequestRead,
			wantCode:   49900,
			wantOutput: "CONTROL_CANCELED",
		},
		{
			name:       "包装后的取消",
			cause:      fmt.Errorf("查询失败: %w", context.Canceled),
			phase:      control.PhaseOperation,
			wantCode:   49900,
			wantOutput: "CONTROL_CANCELED",
		},
		{
			name:       "内部操作超时",
			cause:      context.DeadlineExceeded,
			phase:      control.PhaseOperation,
			wantCode:   50401,
			wantOutput: "CONTROL_DEADLINE_EXCEEDED",
		},
		{
			name:       "请求接收读取超时",
			cause:      context.DeadlineExceeded,
			phase:      control.PhaseRequestRead,
			wantCode:   40802,
			wantOutput: "CONTROL_REQUEST_TIMEOUT",
		},
		{
			name:       "包装后的请求超时",
			cause:      fmt.Errorf("读取请求体失败: %w", context.DeadlineExceeded),
			phase:      control.PhaseRequestRead,
			wantCode:   40802,
			wantOutput: "CONTROL_REQUEST_TIMEOUT",
		},
		{
			name:       "未登记阶段按内部操作处理",
			cause:      context.DeadlineExceeded,
			phase:      control.Phase(9),
			wantCode:   50401,
			wantOutput: "CONTROL_DEADLINE_EXCEEDED",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := control.Classify(testCase.cause, testCase.phase)
			if got == nil {
				t.Fatal("控制原因必须归一为统一错误")
			}
			if got.Code.Code != testCase.wantCode || got.Code.Output != testCase.wantOutput {
				t.Fatalf("控制错误码不匹配: got %d/%s, want %d/%s",
					got.Code.Code, got.Code.Output, testCase.wantCode, testCase.wantOutput)
			}
			if !errors.Is(got, testCase.cause) {
				t.Fatalf("原始原因必须可经 errors.Is 遍历: %+v", got)
			}
			if got.Err != testCase.cause {
				t.Fatalf("根因必须是原始原因本身，不得伪造: got %v, want %v", got.Err, testCase.cause)
			}
		})
	}
}

// TestClassifyKeepsHTTPGroup 断言控制码的 HTTP 分组不会漂移。
func TestClassifyKeepsHTTPGroup(t *testing.T) {
	cases := []struct {
		name string
		err  *apperr.Error
		want int
	}{
		{name: "取消", err: control.Classify(context.Canceled, control.PhaseOperation), want: 499},
		{name: "请求超时", err: control.Classify(context.DeadlineExceeded, control.PhaseRequestRead), want: 408},
		{name: "内部超时", err: control.Classify(context.DeadlineExceeded, control.PhaseOperation), want: 504},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.err == nil {
				t.Fatal("控制原因必须归一为统一错误")
			}
			if group := int(testCase.err.Code.Code / 100); group != testCase.want {
				t.Fatalf("HTTP 分组漂移: got %d, want %d", group, testCase.want)
			}
		})
	}
}

// TestClassifyKeepsExistingStructured 验证已有结构化错误不会被深层控制原因重分类。
func TestClassifyKeepsExistingStructured(t *testing.T) {
	existing := apperr.New(apperr.FrameworkInternalError, "数据库操作失败", context.Canceled)
	sibling := errors.New("连接池关闭失败")

	cases := []struct {
		name  string
		cause error
	}{
		{name: "结构化错误本身", cause: existing},
		{name: "结构化错误被普通包装", cause: fmt.Errorf("查询用户失败: %w", existing)},
		{name: "结构化错误与兄弟原因聚合", cause: errors.Join(existing, sibling)},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := control.Classify(testCase.cause, control.PhaseOperation)
			if got == nil {
				t.Fatal("已有结构化错误必须被保留")
			}
			if got.Code.Code != 50000 || got.Code.Output != "FRAMEWORK_INTERNAL_ERROR" {
				t.Fatalf("已有结构化错误不得被重分类: got %d/%s", got.Code.Code, got.Code.Output)
			}
			if got.ErrorMessage != existing.ErrorMessage {
				t.Fatalf("已有安全描述必须保留: got %q, want %q", got.ErrorMessage, existing.ErrorMessage)
			}
			if !errors.Is(got, context.Canceled) {
				t.Fatal("原始原因链必须完整保留")
			}
			var extracted *apperr.Error
			if !errors.As(got, &extracted) {
				t.Fatal("统一错误必须可经 errors.As 提取")
			}
			if extracted.Code.Code != 50000 {
				t.Fatalf("errors.As 提取到的必须是原有分类: got %d", extracted.Code.Code)
			}
			if testCase.name == "结构化错误与兄弟原因聚合" && !errors.Is(got, sibling) {
				t.Fatal("聚合中的兄弟原因必须可遍历")
			}
		})
	}
}

// TestClassifyKeepsDataOfExistingStructured 验证既有结构化错误的业务数据不被丢弃。
func TestClassifyKeepsDataOfExistingStructured(t *testing.T) {
	existing := apperr.New(apperr.FrameworkInternalError, "组件未就绪", context.DeadlineExceeded)
	existing.Data = map[string]string{"component": "cache"}

	got := control.Classify(existing, control.PhaseRequestRead)
	if got == nil {
		t.Fatal("已有结构化错误必须被保留")
	}
	if got.Code.Code != 50000 {
		t.Fatalf("已有结构化错误不得被重分类: got %d", got.Code.Code)
	}
	data, ok := got.Data.(map[string]string)
	if !ok || data["component"] != "cache" {
		t.Fatalf("业务数据必须保留: %+v", got.Data)
	}
}

// TestClassifyUnknownCauseReturnsNil 验证普通原因不被猜测成控制错误。
func TestClassifyUnknownCauseReturnsNil(t *testing.T) {
	cases := []struct {
		name  string
		cause error
	}{
		{name: "普通错误", cause: errors.New("数据库连接失败")},
		{name: "文本形似取消", cause: errors.New("rpc status: Canceled")},
		{name: "文本形似超时", cause: errors.New("context deadline exceeded")},
		{name: "文本形似请求超时", cause: errors.New("请求接收超时")},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := control.Classify(testCase.cause, control.PhaseOperation); got != nil {
				t.Fatalf("未知普通原因必须返回 nil，不得猜测分类: %+v", got)
			}
			if got := control.Classify(testCase.cause, control.PhaseRequestRead); got != nil {
				t.Fatalf("未知普通原因必须返回 nil，不得猜测分类: %+v", got)
			}
		})
	}
}

// TestClassifyNilDoesNotSwallowUnknownFailure 验证未识别时调用方仍可显式兜底。
func TestClassifyNilDoesNotSwallowUnknownFailure(t *testing.T) {
	cause := errors.New("数据库连接失败")

	if classified := control.Classify(cause, control.PhaseOperation); classified != nil {
		t.Fatalf("未识别的普通原因不应被归一: %+v", classified)
	}

	fallback := apperr.From(apperr.FrameworkInternalError, "读取配置失败", cause)
	if fallback == nil {
		t.Fatal("调用方必须能显式兜底")
	}
	if !errors.Is(fallback, cause) {
		t.Fatal("兜底必须保留原始原因链")
	}
	if fallback.Code.Code != 50000 {
		t.Fatalf("兜底错误码不匹配: got %d", fallback.Code.Code)
	}
}

// TestClassifyKeepsDeepStructured 验证深层结构化错误仍被沿用且不重分类。
func TestClassifyKeepsDeepStructured(t *testing.T) {
	existing := apperr.New(apperr.FrameworkInternalError, "数据库操作失败", context.Canceled)
	cause := fmt.Errorf("外层包装: %w", fmt.Errorf("中层包装: %w", fmt.Errorf("内层包装: %w", existing)))

	got := control.Classify(cause, control.PhaseRequestRead)
	if got == nil {
		t.Fatal("深层结构化错误必须被沿用")
	}
	if got.Code.Code != 50000 {
		t.Fatalf("深层结构化错误不得被重分类: got %d", got.Code.Code)
	}
	if !errors.Is(got, context.Canceled) {
		t.Fatal("深层原始原因链必须完整保留")
	}
}

// TestClassifyRespectsCustomAs 验证自定义 As 暴露的结构化原因会被沿用。
func TestClassifyRespectsCustomAs(t *testing.T) {
	inner := apperr.New(apperr.FrameworkInternalError, "数据库操作失败", context.Canceled)
	cause := &asCarrier{inner: inner}

	got := control.Classify(cause, control.PhaseOperation)
	if got == nil {
		t.Fatal("自定义 As 暴露的结构化原因必须被沿用")
	}
	if got.Code.Code != 50000 {
		t.Fatalf("自定义 As 暴露的结构化原因不得被改分类: got %d", got.Code.Code)
	}
}

// TestClassifyTypedNilDoesNotShadowStructured 验证 nil 载体不会遮蔽链中真实的结构化错误。
func TestClassifyTypedNilDoesNotShadowStructured(t *testing.T) {
	var typedNil *apperr.Error
	existing := apperr.New(apperr.FrameworkInternalError, "数据库操作失败", context.Canceled)

	got := control.Classify(errors.Join(typedNil, existing), control.PhaseOperation)
	if got == nil {
		t.Fatal("真实结构化错误必须被保留")
	}
	if got.Code.Code != 50000 {
		t.Fatalf("nil 载体不得遮蔽真实结构化错误: got %d", got.Code.Code)
	}
	if !errors.Is(got, context.Canceled) {
		t.Fatal("原始原因链必须完整保留")
	}
}

// TestClassifyDoesNotFabricateControlCause 验证不会为凑结果伪造控制哨兵。
func TestClassifyDoesNotFabricateControlCause(t *testing.T) {
	cause := errors.New("rpc status: Canceled")

	got := control.Classify(cause, control.PhaseOperation)
	if got != nil {
		t.Fatalf("文本形似取消的普通错误不得被归一: %+v", got)
	}

	wrapped := fmt.Errorf("调用失败: %w", context.DeadlineExceeded)
	classified := control.Classify(wrapped, control.PhaseRequestRead)
	if classified == nil {
		t.Fatal("真实超时原因必须被归一")
	}
	if classified.Err != wrapped {
		t.Fatalf("根因必须是调用方给出的真实原因: got %v", classified.Err)
	}
}
