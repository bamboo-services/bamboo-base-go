package apperr_test

import (
	"errors"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// writeCore 模拟强类型内部实现：成功返回 nil 指针，失败返回统一错误。
func writeCore(payload []byte) *apperr.Error {
	if len(payload) == 0 {
		return apperr.New(apperr.FrameworkInternalError, "载荷为空")
	}
	return nil
}

// writeThin 演示固定外部签名的薄适配：成功时返回真正的 nil 错误。
func writeThin(payload []byte) (int, error) {
	appErr := writeCore(payload)
	if appErr != nil {
		return 0, appErr
	}
	return len(payload), nil
}

func TestThinAdapter_ReturnsRealNilOnSuccess(t *testing.T) {
	written, err := writeThin([]byte("ok"))
	if err != nil {
		t.Fatalf("成功路径必须返回真正的 nil 错误: %v", err)
	}
	if written != 2 {
		t.Fatalf("成功路径的写入字节数不符: %d", written)
	}
}

func TestThinAdapter_ReturnsStrongErrorOnFailure(t *testing.T) {
	_, err := writeThin(nil)
	if err == nil {
		t.Fatal("失败路径必须返回非 nil 错误")
	}
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		t.Fatal("失败路径必须返回可提取的统一错误")
	}
	if appErr.Code.Code != 50000 {
		t.Fatalf("失败路径的错误码不符: %+v", appErr.Code)
	}
}

func TestFrameworkDiagnosticCodes(t *testing.T) {
	cases := []struct {
		code    *apperr.ErrorCode
		wantNum apperr.Code
		wantOut apperr.Output
		wantMsg apperr.Message
	}{
		{apperr.FrameworkMultipleFailures, 50020, "FRAMEWORK_MULTIPLE_FAILURES", "多个框架阶段失败"},
		{apperr.FrameworkHookPanic, 50021, "FRAMEWORK_HOOK_PANIC", "框架生命周期钩子异常"},
		{apperr.FrameworkShutdownIncomplete, 50022, "FRAMEWORK_SHUTDOWN_INCOMPLETE", "框架关闭未完成"},
		{apperr.FrameworkWorkerUnexpectedExit, 50023, "FRAMEWORK_WORKER_UNEXPECTED_EXIT", "后台服务意外退出"},
	}

	for _, item := range cases {
		if item.code == nil {
			t.Fatal("框架诊断码不得为 nil")
		}
		if item.code.GetCode() != item.wantNum {
			t.Errorf("错误码数值不符: got %d want %d", item.code.GetCode(), item.wantNum)
		}
		if item.code.GetOutput() != item.wantOut {
			t.Errorf("输出标识不符: got %s want %s", item.code.GetOutput(), item.wantOut)
		}
		if item.code.GetMessage() != item.wantMsg {
			t.Errorf("对外文案不符: got %s want %s", item.code.GetMessage(), item.wantMsg)
		}
		if item.code.GetCode()/100 != 500 {
			t.Errorf("框架诊断码必须处于 500 分组: %d", item.code.GetCode())
		}
	}
}
