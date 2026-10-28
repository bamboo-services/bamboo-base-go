package apperr_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}

func TestNew_CodeSnapshot(t *testing.T) {
	origCode := &apperr.ErrorCode{
		Code:    40001,
		Output:  "ORIGINAL_CODE",
		Message: "原始信息",
	}

	err := apperr.New(origCode, "测试快照")
	if err.Code.Code != 40001 || err.Code.Output != "ORIGINAL_CODE" || err.Code.Message != "原始信息" {
		t.Fatalf("初始错误码复制不正确: %+v", err.Code)
	}

	// 修改 origCode，验证快照不受调用方后改影响
	origCode.Code = 49999
	origCode.Output = "MODIFIED_CODE"
	origCode.Message = "已修改信息"

	if err.Code.Code != 40001 || err.Code.Output != "ORIGINAL_CODE" || err.Code.Message != "原始信息" {
		t.Fatalf("错误码被外部修改影响: %+v", err.Code)
	}
}

func TestNew_NilCodeDefault(t *testing.T) {
	err := apperr.New(nil, "缺省测试")
	if err.Code != *apperr.ServerInternalError {
		t.Fatalf("nil 代码未回退为 ServerInternalError: %+v", err.Code)
	}
	if err.Code.Code != 50000 || err.Code.Output != "SERVER_INTERNAL_ERROR" {
		t.Fatalf("默认错误码字段不符: %+v", err.Code)
	}
}

func TestNew_CauseHandlingAndUnwrap(t *testing.T) {
	t.Run("无根因或全nil", func(t *testing.T) {
		err1 := apperr.New(apperr.ServerInternalError, "无根因")
		if err1.Err != nil || err1.Unwrap() != nil {
			t.Fatalf("期望 Err 和 Unwrap 为 nil: %v", err1.Err)
		}

		err2 := apperr.New(apperr.ServerInternalError, "全nil根因", nil, nil)
		if err2.Err != nil || err2.Unwrap() != nil {
			t.Fatalf("全 nil 根因期望 Err 和 Unwrap 为 nil: %v", err2.Err)
		}
	})

	t.Run("单根因 errors.Is 与 errors.As", func(t *testing.T) {
		target := errors.New("底层根因")
		custom := &customError{msg: "自定义错误"}

		err := apperr.New(apperr.ServerInternalError, "包装错误", target)
		if !errors.Is(err, target) {
			t.Fatalf("errors.Is 判定失败")
		}

		errWithCustom := apperr.New(apperr.ServerInternalError, "自定义包装", custom)
		var extracted *customError
		if !errors.As(errWithCustom, &extracted) || extracted.msg != "自定义错误" {
			t.Fatalf("errors.As 提取失败")
		}
	})

	t.Run("多根因 errors.Join 判定", func(t *testing.T) {
		root1 := errors.New("root 1")
		root2 := &customError{msg: "root 2"}

		err := apperr.New(apperr.ServerInternalError, "多错误", nil, root1, nil, root2)
		if !errors.Is(err, root1) {
			t.Fatalf("多根因中未找到 root1")
		}

		var targetCustom *customError
		if !errors.As(err, &targetCustom) || targetCustom.msg != "root 2" {
			t.Fatalf("多根因中未能通过 errors.As 提取 targetCustom")
		}
	})
}

func TestNew_JSONSerializationDoesNotLeakErr(t *testing.T) {
	root := errors.New("sensitive db connection password string")
	appErr := apperr.New(apperr.ServerInternalError, "查询失败", root)
	appErr.Data = map[string]string{"key": "value"}

	bytes, err := json.Marshal(appErr)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	jsonStr := string(bytes)
	if strings.Contains(jsonStr, "sensitive db connection password string") {
		t.Fatalf("底层 Err 泄漏到了 JSON 中: %s", jsonStr)
	}

	if !strings.Contains(jsonStr, `"Code":50000`) || !strings.Contains(jsonStr, `"key":"value"`) {
		t.Fatalf("JSON 缺失 Code 或 Data 字段: %s", jsonStr)
	}
}

func TestNew_DataAssignment(t *testing.T) {
	err := apperr.New(apperr.ServerInternalError, "带数据错误")
	if err.Data != nil {
		t.Fatalf("初始 Data 应为 nil")
	}

	type payload struct {
		UserID int64 `json:"user_id"`
	}
	err.Data = payload{UserID: 12345}

	p, ok := err.Data.(payload)
	if !ok || p.UserID != 12345 {
		t.Fatalf("Data 字段赋值读取异常: %+v", err.Data)
	}
}

func TestNew_NoLoggingSideEffect(t *testing.T) {
	err := apperr.New(apperr.ServerInternalError, "无副作用测试", fmt.Errorf("纯内存运算"))
	if err == nil {
		t.Fatal("返回了 nil 错误")
	}
}
