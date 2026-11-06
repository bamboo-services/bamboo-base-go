package apperr_test

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

type customTestError struct {
	msg string
}

func (c *customTestError) Error() string {
	return c.msg
}

func TestErrorCodesInventorySnapshot(t *testing.T) {
	data, err := os.ReadFile("testdata/error_codes.json")
	if err != nil {
		t.Fatalf("read error code snapshot: %v", err)
	}
	var expected []apperr.ErrorCode
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatalf("decode error code snapshot: %v", err)
	}
	if len(allExportedCodes) != 8 || len(expected) != len(allExportedCodes) {
		t.Fatalf("expected 8 framework error codes, got inventory=%d snapshot=%d", len(allExportedCodes), len(expected))
	}

	codeSet := make(map[apperr.Code]apperr.ErrorCode)
	outputSet := make(map[apperr.Output]bool)
	for _, ec := range allExportedCodes {
		if ec == nil {
			t.Fatal("found nil ErrorCode pointer")
		}
		if ec.Code/100 != 500 || !strings.HasPrefix(string(ec.Output), "FRAMEWORK_") {
			t.Fatalf("foundation must contain only framework 500-series codes: %+v", ec)
		}
		if _, exists := codeSet[ec.Code]; exists {
			t.Fatalf("duplicate code detected: %d (%s)", ec.Code, ec.Output)
		}
		if outputSet[ec.Output] {
			t.Fatalf("duplicate output detected: %s", ec.Output)
		}
		codeSet[ec.Code] = apperr.ErrorCode{Code: ec.GetCode(), Output: ec.GetOutput(), Message: ec.GetMessage()}
		outputSet[ec.Output] = true
	}
	for _, want := range expected {
		got, exists := codeSet[want.Code]
		if !exists || got != want {
			t.Errorf("error code snapshot mismatch: want %+v, got %+v", want, got)
		}
		delete(codeSet, want.Code)
	}
	if len(codeSet) != 0 {
		t.Errorf("error codes missing from snapshot: %v", codeSet)
	}
}

func TestErrorUnwrapAndIsAs(t *testing.T) {
	rootCause := &customTestError{msg: "connection reset by peer"}
	wrapped := &apperr.Error{
		Code:         *apperr.FrameworkInternalError,
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
		Code: *apperr.FrameworkInternalError,
		Err:  io.EOF,
	}
	if !errors.Is(eofErr, io.EOF) {
		t.Errorf("errors.Is should match io.EOF")
	}
}

func TestErrorJSONSerializationNoLeak(t *testing.T) {
	rootCause := errors.New("super secret database credential error")
	appErr := &apperr.Error{
		Code:         *apperr.FrameworkInternalError,
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
		Code: *apperr.FrameworkContainerNotMounted,
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
