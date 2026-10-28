package httpresult

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/envelope"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestResolveHTTPStatus_FallbackAndMapping(t *testing.T) {
	tests := []struct {
		name     string
		code     apperr.Code
		expected int
	}{
		{"negative", -100, http.StatusInternalServerError},
		{"zero", 0, http.StatusInternalServerError},
		{"1xx", 10000, http.StatusInternalServerError},
		{"2xx", 20000, http.StatusInternalServerError},
		{"3xx", 30100, http.StatusInternalServerError},
		{"6xx", 60000, http.StatusInternalServerError},
		{"40014->400", 40014, http.StatusBadRequest},
		{"50000->500", 50000, http.StatusInternalServerError},
		{"40400->404", 40400, http.StatusNotFound},
		{"50301->503", 50301, http.StatusServiceUnavailable},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveHTTPStatus(tc.code)
			if got != tc.expected {
				t.Fatalf("resolveHTTPStatus(%d) = %d; want %d", tc.code, got, tc.expected)
			}
		})
	}
}

func TestError_FallbackAndEnvelope(t *testing.T) {
	cases := []struct {
		name       string
		err        *apperr.Error
		wantStatus int
		wantCode   int
	}{
		{
			name:       "nil error fallback 500",
			err:        nil,
			wantStatus: http.StatusInternalServerError,
			wantCode:   int(apperr.ServerInternalError.Code),
		},
		{
			name: "20000 custom error fallback 500",
			err: apperr.New(&apperr.ErrorCode{
				Code:    20000,
				Output:  "CustomOk",
				Message: "Fake OK Error",
			}, ""),
			wantStatus: http.StatusInternalServerError,
			wantCode:   20000,
		},
		{
			name: "30100 custom redirect error fallback 500",
			err: apperr.New(&apperr.ErrorCode{
				Code:    30100,
				Output:  "CustomRedirect",
				Message: "Fake Redirect",
			}, ""),
			wantStatus: http.StatusInternalServerError,
			wantCode:   30100,
		},
		{
			name: "10000 fallback 500",
			err: apperr.New(&apperr.ErrorCode{
				Code:    10000,
				Output:  "ContinueErr",
				Message: "Fake Continue",
			}, ""),
			wantStatus: http.StatusInternalServerError,
			wantCode:   10000,
		},
		{
			name: "60000 fallback 500",
			err: apperr.New(&apperr.ErrorCode{
				Code:    60000,
				Output:  "OutRange",
				Message: "Fake 600",
			}, ""),
			wantStatus: http.StatusInternalServerError,
			wantCode:   60000,
		},
		{
			name: "40014 valid 400",
			err: apperr.New(&apperr.ErrorCode{
				Code:    40014,
				Output:  "InvalidParam",
				Message: "Bad Request",
			}, ""),
			wantStatus: http.StatusBadRequest,
			wantCode:   40014,
		},
		{
			name: "50000 valid 500",
			err: apperr.New(&apperr.ErrorCode{
				Code:    50000,
				Output:  "ServerErr",
				Message: "Internal Error",
			}, ""),
			wantStatus: http.StatusInternalServerError,
			wantCode:   50000,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			c.Request = req

			Error(c, tc.err)

			if w.Code != tc.wantStatus {
				t.Fatalf("expected HTTP status %d, got %d", tc.wantStatus, w.Code)
			}

			var resp envelope.Response
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal JSON response: %v", err)
			}

			if int(resp.Code) != tc.wantCode {
				t.Fatalf("expected envelope code %d, got %d", tc.wantCode, int(resp.Code))
			}
		})
	}
}

func TestError_NoUnderlyingErrLeakEvenInDebug(t *testing.T) {
	gin.SetMode(gin.DebugMode)
	defer gin.SetMode(gin.TestMode)

	secretMsg := "top-secret-internal-db-password-123456"
	appErr := apperr.New(apperr.ServerInternalError, "safe public message", errors.New(secretMsg))
	appErr.Data = map[string]any{"field": "val"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/leak-check", nil)
	c.Request = req

	Error(c, appErr)

	body := w.Body.String()
	if strings.Contains(body, secretMsg) {
		t.Fatalf("internal error leaked in response: %s", body)
	}

	var resp envelope.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.ErrorMessage != "safe public message" {
		t.Fatalf("unexpected ErrorMessage: %q", resp.ErrorMessage)
	}
	if resp.Data == nil {
		t.Fatalf("Data shape lost, want non-nil")
	}
}

func TestAbortError_StopsSubsequentHandlers(t *testing.T) {
	r := gin.New()
	executedNext := false

	r.GET("/abort-test", func(c *gin.Context) {
		err := apperr.New(&apperr.ErrorCode{
			Code:    40014,
			Output:  "BadRequest",
			Message: "Must abort",
		}, "")
		AbortError(c, err)
	}, func(c *gin.Context) {
		executedNext = true
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/abort-test", nil)
	r.ServeHTTP(w, req)

	if executedNext {
		t.Fatal("subsequent handler should not have executed after AbortError")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected HTTP status 400, got %d", w.Code)
	}
}

func TestSuccessAndSuccessWithData_Preserved200(t *testing.T) {
	// Success
	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	req1, _ := http.NewRequest(http.MethodGet, "/succ", nil)
	c1.Request = req1

	Success(c1, "all good")
	if w1.Code != http.StatusOK {
		t.Fatalf("Success status want 200, got %d", w1.Code)
	}

	// SuccessWithData
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	req2, _ := http.NewRequest(http.MethodGet, "/succ-data", nil)
	c2.Request = req2

	SuccessWithData(c2, "with payload", map[string]string{"foo": "bar"})
	if w2.Code != http.StatusOK {
		t.Fatalf("SuccessWithData status want 200, got %d", w2.Code)
	}
}
