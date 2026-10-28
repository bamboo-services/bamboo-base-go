package grpcresult

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/generate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSuccess(t *testing.T) {
	t.Run("nil context and standard message", func(t *testing.T) {
		var nilCtx context.Context
		resp := Success(nilCtx, "Operation succeeded")
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if resp.Code != 200 {
			t.Errorf("expected code 200, got %d", resp.Code)
		}
		if resp.Output != DefaultSuccessOutput {
			t.Errorf("expected output %s, got %s", DefaultSuccessOutput, resp.Output)
		}
		if resp.Message != "Operation succeeded" {
			t.Errorf("expected message 'Operation succeeded', got %s", resp.Message)
		}
		if resp.Context != "" {
			t.Errorf("expected empty context for nil ctx, got %s", resp.Context)
		}
		if resp.Overhead != nil {
			t.Errorf("expected nil overhead without debug/time, got %v", resp.Overhead)
		}
	})

	t.Run("context with request key and debug overhead", func(t *testing.T) {
		_ = os.Setenv(string(env.Debug), "true")
		defer func() { _ = os.Unsetenv(string(env.Debug)) }()

		now := time.Now().Add(-10 * time.Millisecond)
		ctx := context.WithValue(context.Background(), ctxkey.RequestKey, "req-123456")
		ctx = context.WithValue(ctx, ctxkey.UserStartTimeKey, now)

		resp := Success(ctx, "OK")
		if resp.Context != "req-123456" {
			t.Errorf("expected context req-123456, got %s", resp.Context)
		}
		if resp.Overhead == nil || *resp.Overhead <= 0 {
			t.Errorf("expected non-nil positive overhead in debug mode, got %v", resp.Overhead)
		}
	})
}

func TestToStatusError_NilError(t *testing.T) {
	err := ToStatusError(context.Background(), nil)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected status error")
	}
	if st.Code() != codes.Internal {
		t.Errorf("expected codes.Internal for nil error, got %v", st.Code())
	}
	if len(st.Details()) == 0 {
		t.Fatal("expected details in status")
	}
	detail, ok := st.Details()[0].(*generate.BaseResponse)
	if !ok {
		t.Fatalf("expected BaseResponse detail, got %T", st.Details()[0])
	}
	if detail.Code != uint64(apperr.ServerInternalError.Code) {
		t.Errorf("expected code %d, got %d", apperr.ServerInternalError.Code, detail.Code)
	}
}

func TestToStatusError_GroupMapping(t *testing.T) {
	cases := []struct {
		name         string
		httpGroup    uint64
		expectedCode codes.Code
	}{
		{"400 InvalidArgument", 400, codes.InvalidArgument},
		{"401 Unauthenticated", 401, codes.Unauthenticated},
		{"403 PermissionDenied", 403, codes.PermissionDenied},
		{"404 NotFound", 404, codes.NotFound},
		{"408 DeadlineExceeded", 408, codes.DeadlineExceeded},
		{"504 DeadlineExceeded", 504, codes.DeadlineExceeded},
		{"409 AlreadyExists", 409, codes.AlreadyExists},
		{"412 FailedPrecondition", 412, codes.FailedPrecondition},
		{"429 ResourceExhausted", 429, codes.ResourceExhausted},
		{"499 Canceled", 499, codes.Canceled},
		{"500 Internal", 500, codes.Internal},
		{"501 Unimplemented", 501, codes.Unimplemented},
		{"502 Unavailable", 502, codes.Unavailable},
		{"503 Unavailable", 503, codes.Unavailable},
		{"Unknown group 418", 418, codes.Unknown},
		{"Unknown group 999", 999, codes.Unknown},
		{"Illegal group 0", 0, codes.Unknown},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code := apperr.Code(tc.httpGroup*100 + 1)
			appErr := &apperr.Error{
				Code: apperr.ErrorCode{
					Code:    code,
					Output:  "TEST_OUTPUT",
					Message: "Test Message",
				},
			}
			err := ToStatusError(context.Background(), appErr)
			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected grpc status, got %v", err)
			}
			if st.Code() != tc.expectedCode {
				t.Errorf("group %d: expected code %v, got %v", tc.httpGroup, tc.expectedCode, st.Code())
			}
		})
	}
}

func TestToStatusError_SecurityRawErrorNotLeaked(t *testing.T) {
	sensitiveSQL := "SELECT password_hash, token FROM users WHERE id = 1; -- syntax error near ';'"
	rawErr := errors.New(sensitiveSQL)

	appErr := &apperr.Error{
		Code: apperr.ErrorCode{
			Code:    50001,
			Output:  "DATABASE_ERROR",
			Message: "A database error occurred",
		},
		ErrorMessage: "Safe sanitized business error message",
		Err:          rawErr,
	}

	ctx := context.WithValue(context.Background(), ctxkey.RequestKey, "trace-abc-123")

	err := ToStatusError(ctx, appErr)
	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected status error")
	}

	// 1. 验证 status.Message 不包含底层 SQL
	if strings.Contains(st.Message(), "password_hash") || strings.Contains(st.Message(), "syntax error") {
		t.Fatalf("sensitive SQL leaked in status.Message: %s", st.Message())
	}
	if st.Message() != "A database error occurred" {
		t.Errorf("expected safe message 'A database error occurred', got %s", st.Message())
	}

	// 2. 验证 Details 中也不包含底层 SQL
	details := st.Details()
	if len(details) == 0 {
		t.Fatal("expected details containing BaseResponse")
	}
	detail, ok := details[0].(*generate.BaseResponse)
	if !ok {
		t.Fatalf("expected *generate.BaseResponse, got %T", details[0])
	}
	if detail.ErrorMessage == nil || *detail.ErrorMessage != "Safe sanitized business error message" {
		t.Errorf("expected sanitized error message, got %v", detail.ErrorMessage)
	}
	if strings.Contains(detail.String(), "password_hash") {
		t.Fatalf("sensitive SQL leaked in BaseResponse detail: %s", detail.String())
	}
	if detail.Context != "trace-abc-123" {
		t.Errorf("expected context trace-abc-123, got %s", detail.Context)
	}
	if detail.Code != 50001 {
		t.Errorf("expected code 50001, got %d", detail.Code)
	}
}
