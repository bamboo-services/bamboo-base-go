package apperr_test

import (
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

type customTestError struct {
	msg string
}

func (c *customTestError) Error() string {
	return c.msg
}

func TestErrorCodesInventoryAndRepresentativeValues(t *testing.T) {
	// 代表值检验
	tests := []struct {
		ec      *apperr.ErrorCode
		code    apperr.Code
		output  apperr.Output
		message apperr.Message
	}{
		{apperr.NotExist, 40000, "NOT_EXIST", "内容不存在"},
		{apperr.BadRequest, 40010, "BAD_REQUEST", "错误请求"},
		{apperr.DeveloperError, 40024, "DEVELOPER_ERROR", "开发者操作错误"},
		{apperr.Unauthorized, 40100, "UNAUTHORIZED", "未授权"},
		{apperr.Forbidden, 40300, "FORBIDDEN", "禁止访问"},
		{apperr.NotFound, 40400, "NOT_FOUND", "未找到"},
		{apperr.MethodNotAllowed, 40500, "METHOD_NOT_ALLOWED", "方法不允许"},
		{apperr.NotAcceptable, 40600, "NOT_ACCEPTABLE", "不可接受"},
		{apperr.Timeout, 40800, "TIMEOUT", "请求超时"},
		{apperr.Conflict, 40900, "CONFLICT", "冲突"},
		{apperr.Gone, 41000, "GONE", "资源已删除"},
		{apperr.PayloadTooLarge, 41300, "PAYLOAD_TOO_LARGE", "请求体过大"},
		{apperr.UnsupportedMedia, 41500, "UNSUPPORTED_MEDIA", "不支持的媒体类型"},
		{apperr.UnprocessableEntity, 42200, "UNPROCESSABLE_ENTITY", "无法处理的实体"},
		{apperr.TooManyRequests, 42900, "TOO_MANY_REQUESTS", "请求过多"},
		{apperr.ServerInternalError, 50000, "SERVER_INTERNAL_ERROR", "服务器内部错误"},
		{apperr.UnknownError, 50099, "UNKNOWN_ERROR", "未知错误"},
		{apperr.GatewayError, 50200, "GATEWAY_ERROR", "网关错误"},
		{apperr.ServiceUnavailable, 50300, "SERVICE_UNAVAILABLE", "服务不可用"},
		{apperr.GatewayTimeout, 50400, "GATEWAY_TIMEOUT", "网关超时"},
	}

	for _, tc := range tests {
		if tc.ec.GetCode() != tc.code {
			t.Errorf("expected code %d, got %d", tc.code, tc.ec.GetCode())
		}
		if tc.ec.GetOutput() != tc.output {
			t.Errorf("expected output %s, got %s", tc.output, tc.ec.GetOutput())
		}
		if tc.ec.GetMessage() != tc.message {
			t.Errorf("expected message %s, got %s", tc.message, tc.ec.GetMessage())
		}
	}

	// 统计并确保错误码总数与旧库 189 个完全一致
	codeSet := make(map[apperr.Code]string)
	for _, ec := range allExportedCodes {
		if ec == nil {
			t.Fatal("found nil ErrorCode pointer")
		}
		if _, exists := codeSet[ec.Code]; exists {
			t.Fatalf("duplicate code detected: %d (%s)", ec.Code, ec.Output)
		}
		codeSet[ec.Code] = string(ec.Output)
	}

	if len(allExportedCodes) != 189 {
		t.Fatalf("expected total 189 error codes, got %d", len(allExportedCodes))
	}
}

func TestErrorUnwrapAndIsAs(t *testing.T) {
	rootCause := &customTestError{msg: "connection reset by peer"}
	wrapped := &apperr.Error{
		Code:         *apperr.ServerInternalError,
		ErrorMessage: "failed to connect upstream",
		Err:          rootCause,
	}

	if !errors.Is(wrapped, rootCause) {
		t.Errorf("errors.Is should match underlying rootCause")
	}

	var target *customTestError
	if !errors.As(wrapped, &target) {
		t.Errorf("errors.As should match target customTestError")
	} else if target.msg != "connection reset by peer" {
		t.Errorf("target msg mismatch: got %s", target.msg)
	}

	// 验证 io.EOF 等标准 error
	eofErr := &apperr.Error{
		Code: *apperr.IoError,
		Err:  io.EOF,
	}
	if !errors.Is(eofErr, io.EOF) {
		t.Errorf("errors.Is should match io.EOF")
	}
}

func TestErrorJSONSerializationNoLeak(t *testing.T) {
	rootCause := errors.New("super secret database credential error")
	appErr := &apperr.Error{
		Code:         *apperr.DatabaseError,
		ErrorMessage: "query failed",
		Data:         map[string]any{"table": "users"},
		Err:          rootCause,
	}

	data, err := json.Marshal(appErr)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	jsonStr := string(data)
	if containsSubstr(jsonStr, "super secret") {
		t.Fatalf("root cause leaked into JSON: %s", jsonStr)
	}
	if containsSubstr(jsonStr, "Err") || containsSubstr(jsonStr, "err") {
		// 校验确实没有 err/Err 字段
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatalf("unmarshal back failed: %v", err)
		}
		if _, ok := m["err"]; ok {
			t.Fatalf("err field found in JSON: %s", jsonStr)
		}
		if _, ok := m["Err"]; ok {
			t.Fatalf("Err field found in JSON: %s", jsonStr)
		}
	}

	// 测试 omitempty
	emptyErr := &apperr.Error{
		Code: *apperr.BadRequest,
	}
	emptyData, err := json.Marshal(emptyErr)
	if err != nil {
		t.Fatalf("json.Marshal emptyErr failed: %v", err)
	}
	var emptyMap map[string]any
	if err := json.Unmarshal(emptyData, &emptyMap); err != nil {
		t.Fatalf("unmarshal emptyErr failed: %v", err)
	}
	if _, ok := emptyMap["error_message"]; ok {
		t.Errorf("error_message should be omitted when empty")
	}
	if _, ok := emptyMap["data"]; ok {
		t.Errorf("data should be omitted when empty")
	}
	if _, ok := emptyMap["code"]; !ok {
		t.Errorf("code field must be present")
	}
}

func containsSubstr(s, substr string) bool {
	return reflect.ValueOf(s).Len() >= len(substr) && (s == substr || (len(substr) > 0 && searchSubstr(s, substr)))
}

func searchSubstr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
