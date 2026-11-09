package valueerr_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/valueerr"
)

// From 的公开签名必须与跨分工约定一致。
var _ func(valueerr.Kind, apperr.ErrMessage, error) *apperr.Error = valueerr.From

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

// TestFromNilCauseReturnsNil 验证 nil 与 nil 载体都不产生假失败。
func TestFromNilCauseReturnsNil(t *testing.T) {
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
			if got := valueerr.From(valueerr.KindInvalidFormat, "值格式非法", testCase.cause); got != nil {
				t.Fatalf("空原因必须返回 nil: %+v", got)
			}
			if got := valueerr.From(valueerr.KindInvalidReceiver, "值接收器无效", testCase.cause); got != nil {
				t.Fatalf("空原因必须返回 nil: %+v", got)
			}
		})
	}
}

// TestFromMapsKindToCode 按表驱动断言分类与错误码的映射关系。
func TestFromMapsKindToCode(t *testing.T) {
	cause := errors.New("解析失败")

	cases := []struct {
		name        string
		kind        valueerr.Kind
		wantCode    apperr.Code
		wantOutput  apperr.Output
		wantGroup   int
		wantMessage string
	}{
		{
			name:        "格式非法",
			kind:        valueerr.KindInvalidFormat,
			wantCode:    40013,
			wantOutput:  "VALUE_INVALID_FORMAT",
			wantGroup:   400,
			wantMessage: "端口号格式非法",
		},
		{
			name:        "超出范围",
			kind:        valueerr.KindOutOfRange,
			wantCode:    40014,
			wantOutput:  "VALUE_OUT_OF_RANGE",
			wantGroup:   400,
			wantMessage: "端口号超出范围",
		},
		{
			name:        "类型非法",
			kind:        valueerr.KindInvalidType,
			wantCode:    40015,
			wantOutput:  "VALUE_INVALID_TYPE",
			wantGroup:   400,
			wantMessage: "端口号类型非法",
		},
		{
			name:        "不支持的转换目标",
			kind:        valueerr.KindUnsupportedTarget,
			wantCode:    50018,
			wantOutput:  "VALUE_UNSUPPORTED_TARGET",
			wantGroup:   500,
			wantMessage: "不支持的转换目标",
		},
		{
			name:        "值接收器无效",
			kind:        valueerr.KindInvalidReceiver,
			wantCode:    50019,
			wantOutput:  "VALUE_INVALID_RECEIVER",
			wantGroup:   500,
			wantMessage: "值接收器无效",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := valueerr.From(testCase.kind, apperr.ErrMessage(testCase.wantMessage), cause)
			if got == nil {
				t.Fatal("非空原因必须产出统一错误")
			}
			if got.Code.Code != testCase.wantCode || got.Code.Output != testCase.wantOutput {
				t.Fatalf("值错误码不匹配: got %d/%s, want %d/%s",
					got.Code.Code, got.Code.Output, testCase.wantCode, testCase.wantOutput)
			}
			if group := int(got.Code.Code / 100); group != testCase.wantGroup {
				t.Fatalf("HTTP 分组漂移: got %d, want %d", group, testCase.wantGroup)
			}
			if got.ErrorMessage != apperr.ErrMessage(testCase.wantMessage) {
				t.Fatalf("补充描述不匹配: got %q, want %q", got.ErrorMessage, testCase.wantMessage)
			}
			if got.Err != cause {
				t.Fatalf("根因必须是原始原因本身: got %v, want %v", got.Err, cause)
			}
			if !errors.Is(got, cause) {
				t.Fatal("原始原因必须可经 errors.Is 遍历")
			}
		})
	}
}

// TestFromKeepsExistingStructured 验证已有结构化错误不会被改判为值错误。
func TestFromKeepsExistingStructured(t *testing.T) {
	existing := apperr.New(apperr.FrameworkInternalError, "插件初始化失败", errors.New("端口被占用"))
	existing.Data = map[string]string{"plugin": "database"}
	sibling := errors.New("连接池关闭失败")

	cases := []struct {
		name  string
		cause error
	}{
		{name: "结构化错误本身", cause: existing},
		{name: "结构化错误被普通包装", cause: fmt.Errorf("启动插件失败: %w", existing)},
		{name: "结构化错误与兄弟原因聚合", cause: errors.Join(existing, sibling)},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := valueerr.From(valueerr.KindInvalidFormat, "值格式非法", testCase.cause)
			if got == nil {
				t.Fatal("已有结构化错误必须被保留")
			}
			if got.Code.Code != 50000 || got.Code.Output != "FRAMEWORK_INTERNAL_ERROR" {
				t.Fatalf("已有结构化错误不得被改判归属: got %d/%s", got.Code.Code, got.Code.Output)
			}
			if got.ErrorMessage != existing.ErrorMessage {
				t.Fatalf("已有安全描述必须保留: got %q, want %q", got.ErrorMessage, existing.ErrorMessage)
			}
			data, ok := got.Data.(map[string]string)
			if !ok || data["plugin"] != "database" {
				t.Fatalf("业务数据必须保留: %+v", got.Data)
			}
			if !errors.Is(got, existing) {
				t.Fatal("原有结构化错误必须仍可经 errors.As / errors.Is 提取")
			}
			if testCase.name == "结构化错误与兄弟原因聚合" && !errors.Is(got, sibling) {
				t.Fatal("聚合中的兄弟原因必须可遍历")
			}
		})
	}
}

// TestFromKeepsDeepStructured 验证深层结构化错误仍被沿用且不改判归属。
func TestFromKeepsDeepStructured(t *testing.T) {
	existing := apperr.New(apperr.FrameworkInternalError, "插件初始化失败", errors.New("端口被占用"))
	cause := fmt.Errorf("外层包装: %w", fmt.Errorf("中层包装: %w", fmt.Errorf("内层包装: %w", existing)))

	got := valueerr.From(valueerr.KindInvalidFormat, "端口号格式非法", cause)
	if got == nil {
		t.Fatal("深层结构化错误必须被沿用")
	}
	if got.Code.Code != 50000 {
		t.Fatalf("深层结构化错误不得被改判归属: got %d", got.Code.Code)
	}
	if !errors.Is(got, existing) {
		t.Fatal("原有结构化错误必须仍可提取")
	}
}

// TestFromRespectsCustomAs 验证自定义 As 暴露的结构化原因会被沿用。
func TestFromRespectsCustomAs(t *testing.T) {
	inner := apperr.New(apperr.FrameworkInternalError, "插件初始化失败", errors.New("端口被占用"))
	cause := &asCarrier{inner: inner}

	got := valueerr.From(valueerr.KindInvalidFormat, "端口号格式非法", cause)
	if got == nil {
		t.Fatal("自定义 As 暴露的结构化原因必须被沿用")
	}
	if got.Code.Code != 50000 {
		t.Fatalf("自定义 As 暴露的结构化原因不得被改判归属: got %d", got.Code.Code)
	}
}

// TestFromTypedNilDoesNotShadowStructured 验证 nil 载体不会遮蔽链中真实的结构化错误。
func TestFromTypedNilDoesNotShadowStructured(t *testing.T) {
	var typedNil *apperr.Error
	existing := apperr.New(apperr.FrameworkInternalError, "插件初始化失败", errors.New("端口被占用"))

	got := valueerr.From(valueerr.KindInvalidFormat, "端口号格式非法", errors.Join(typedNil, existing))
	if got == nil {
		t.Fatal("真实结构化错误必须被保留")
	}
	if got.Code.Code != 50000 {
		t.Fatalf("nil 载体不得遮蔽真实结构化错误: got %d", got.Code.Code)
	}
	if !errors.Is(got, existing) {
		t.Fatal("原有结构化错误必须仍可提取")
	}
}

// TestFromUnknownKindUsesContractError 验证未登记分类按调用契约错误处理。
func TestFromUnknownKindUsesContractError(t *testing.T) {
	cause := errors.New("解析失败")

	got := valueerr.From(valueerr.Kind(42), "未知分类", cause)
	if got == nil {
		t.Fatal("未登记分类必须产出统一错误")
	}
	if got.Code.Code != 50018 || got.Code.Output != "VALUE_UNSUPPORTED_TARGET" {
		t.Fatalf("未登记分类必须归 500 组契约错误: got %d/%s", got.Code.Code, got.Code.Output)
	}
	if group := int(got.Code.Code / 100); group != 500 {
		t.Fatalf("未登记分类的 HTTP 分组漂移: got %d, want 500", group)
	}
	if got.ErrorMessage == "" {
		t.Fatal("未登记分类必须给出可读说明")
	}
	if got.Err != cause {
		t.Fatalf("根因必须是原始原因本身: got %v", got.Err)
	}
}

// TestFromDoesNotGuessByText 验证分类只依据显式 Kind，不解析错误文本。
func TestFromDoesNotGuessByText(t *testing.T) {
	cases := []struct {
		name     string
		cause    error
		kind     valueerr.Kind
		wantCode apperr.Code
	}{
		{
			name:     "文本形似超时仍按格式非法",
			cause:    errors.New("context deadline exceeded"),
			kind:     valueerr.KindInvalidFormat,
			wantCode: 40013,
		},
		{
			name:     "文本形似配置非法仍按格式非法",
			cause:    errors.New("配置文件格式非法"),
			kind:     valueerr.KindInvalidFormat,
			wantCode: 40013,
		},
		{
			name:     "文本形似输入非法仍按契约错误",
			cause:    errors.New("请求参数非法"),
			kind:     valueerr.KindInvalidReceiver,
			wantCode: 50019,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := valueerr.From(testCase.kind, "值校验失败", testCase.cause)
			if got == nil {
				t.Fatal("非空原因必须产出统一错误")
			}
			if got.Code.Code != testCase.wantCode {
				t.Fatalf("分类必须只由 Kind 决定: got %d, want %d", got.Code.Code, testCase.wantCode)
			}
		})
	}
}

// TestFromNoCauseDoesNotForceError 验证无原因时不强行造错。
func TestFromNoCauseDoesNotForceError(t *testing.T) {
	if got := valueerr.From(valueerr.KindInvalidFormat, "值格式非法", nil); got != nil {
		t.Fatalf("无原因必须返回 nil: %+v", got)
	}

	explicit := apperr.New(valueerr.InvalidFormat, "值格式非法")
	if explicit == nil {
		t.Fatal("显式构造必须返回非 nil 错误")
	}
	if explicit.Code.Code != 40013 {
		t.Fatalf("显式构造的错误码不匹配: got %d", explicit.Code.Code)
	}
}

// TestFromErrorTextIsSafe 验证补充描述由调用方提供且不会被原文覆盖。
func TestFromErrorTextIsSafe(t *testing.T) {
	cause := errors.New("password=secret-value 非法")

	got := valueerr.From(valueerr.KindInvalidFormat, "密码格式非法", cause)
	if got == nil {
		t.Fatal("非空原因必须产出统一错误")
	}
	if got.ErrorMessage != "密码格式非法" {
		t.Fatalf("补充描述必须来自调用方的安全描述: got %q", got.ErrorMessage)
	}
}
