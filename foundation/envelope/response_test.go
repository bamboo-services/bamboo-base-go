package envelope_test

import (
	"encoding/json"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/envelope"
)

func TestEnvelopeResponseJSON(t *testing.T) {
	badRequest := apperr.ErrorCode{Code: 40010, Output: "BAD_REQUEST", Message: "错误请求"}
	resp := envelope.Response{
		Context:      "req-123456",
		Code:         badRequest.Code,
		Output:       badRequest.Output,
		Message:      badRequest.Message,
		ErrorMessage: "field 'username' is invalid",
		Overhead:     12,
		Data:         map[string]any{"field": "username"},
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(bytes, &m); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if m["context"] != "req-123456" {
		t.Errorf("context mismatch: got %v", m["context"])
	}
	if float64(badRequest.Code) != m["code"] {
		t.Errorf("code mismatch: got %v", m["code"])
	}
	if string(badRequest.Output) != m["output"] {
		t.Errorf("output mismatch: got %v", m["output"])
	}
	if string(badRequest.Message) != m["message"] {
		t.Errorf("message mismatch: got %v", m["message"])
	}
	if m["error_message"] != "field 'username' is invalid" {
		t.Errorf("error_message mismatch: got %v", m["error_message"])
	}
	if m["overhead"] != float64(12) {
		t.Errorf("overhead mismatch: got %v", m["overhead"])
	}

	// 验证 omitempty
	minimalResp := envelope.Response{
		Code:    apperr.FrameworkInternalError.Code,
		Output:  apperr.FrameworkInternalError.Output,
		Message: apperr.FrameworkInternalError.Message,
	}

	minBytes, err := json.Marshal(minimalResp)
	if err != nil {
		t.Fatalf("marshal minimalResp failed: %v", err)
	}

	var minMap map[string]any
	if err := json.Unmarshal(minBytes, &minMap); err != nil {
		t.Fatalf("unmarshal minimalResp failed: %v", err)
	}

	for _, omittedKey := range []string{"context", "error_message", "overhead", "data"} {
		if _, ok := minMap[omittedKey]; ok {
			t.Errorf("expected key %s to be omitted, but present", omittedKey)
		}
	}
	for _, requiredKey := range []string{"code", "output", "message"} {
		if _, ok := minMap[requiredKey]; !ok {
			t.Errorf("expected key %s to be present, but missing", requiredKey)
		}
	}
}
