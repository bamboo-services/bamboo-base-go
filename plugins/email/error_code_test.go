package email_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/plugins/email/v2"
)

func TestEmailErrorCode(t *testing.T) {
	want := apperr.ErrorCode{Code: 50017, Output: "EMAIL_ERROR", Message: "邮件服务错误"}
	if email.EmailError == nil || *email.EmailError != want {
		t.Fatalf("plugin error code changed: got %+v, want %+v", email.EmailError, want)
	}
	cause := errors.New("operation failed")
	err := apperr.New(email.EmailError, "操作失败", cause)
	if err.Code != want || !errors.Is(err, cause) {
		t.Fatalf("plugin error code or cause was lost: %+v", err)
	}
}

// ExampleEmailError 演示调用方按需包装插件错误，保留原始根因。
func ExampleEmailError() {
	cause := errors.New("operation failed")
	err := apperr.New(email.EmailError, "操作失败", cause)
	fmt.Println(err.Code.GetCode(), err.Code.GetOutput())
	fmt.Println(errors.Is(err, cause))
	// Output:
	// 50017 EMAIL_ERROR
	// true
}
