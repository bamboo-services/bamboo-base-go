package bind_test

import (
	"errors"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/bind"
)

func TestProtocolErrorCodes(t *testing.T) {
	tests := []struct {
		name string
		code *apperr.ErrorCode
		want apperr.ErrorCode
	}{
		{"ParameterError", bind.ParameterError, apperr.ErrorCode{Code: 40005, Output: "HTTP_BIND_PARAMETER_ERROR", Message: "参数错误"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.code == nil || *tc.code != tc.want {
				t.Fatalf("protocol error code changed: got %+v, want %+v", tc.code, tc.want)
			}
			cause := errors.New("private underlying error")
			appErr := apperr.New(tc.code, "请求失败", cause)
			if appErr.Code != tc.want || !errors.Is(appErr, cause) {
				t.Fatalf("error code or cause was lost: %+v", appErr)
			}
		})
	}
}
