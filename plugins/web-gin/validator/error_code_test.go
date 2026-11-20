package validator_test

import (
	"errors"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/validator"
)

func TestProtocolErrorCodes(t *testing.T) {
	tests := []struct {
		name string
		code *apperr.ErrorCode
		want apperr.ErrorCode
	}{
		{"BodyError", validator.BodyError, apperr.ErrorCode{Code: 40006, Output: "HTTP_VALIDATOR_BODY_ERROR", Message: "请求体错误"}},
		{"ValidationError", validator.ValidationError, apperr.ErrorCode{Code: 40007, Output: "HTTP_VALIDATOR_VALIDATION_ERROR", Message: "验证错误"}},
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
