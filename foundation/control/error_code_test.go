package control_test

import (
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
)

// TestControlErrorCodes 按表驱动断言控制错误码的三元组与分组。
func TestControlErrorCodes(t *testing.T) {
	cases := []struct {
		name        string
		code        *apperr.ErrorCode
		wantCode    apperr.Code
		wantOutput  apperr.Output
		wantMessage apperr.Message
		wantGroup   int
	}{
		{
			name:        "取消",
			code:        control.Canceled,
			wantCode:    49900,
			wantOutput:  "CONTROL_CANCELED",
			wantMessage: "操作已取消",
			wantGroup:   499,
		},
		{
			name:        "请求接收超时",
			code:        control.RequestTimeout,
			wantCode:    40802,
			wantOutput:  "CONTROL_REQUEST_TIMEOUT",
			wantMessage: "请求接收超时",
			wantGroup:   408,
		},
		{
			name:        "内部操作超时",
			code:        control.DeadlineExceeded,
			wantCode:    50401,
			wantOutput:  "CONTROL_DEADLINE_EXCEEDED",
			wantMessage: "操作超过截止时间",
			wantGroup:   504,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.code == nil {
				t.Fatal("错误码目录变量不得为 nil")
			}
			want := apperr.ErrorCode{
				Code:    testCase.wantCode,
				Output:  testCase.wantOutput,
				Message: testCase.wantMessage,
			}
			if *testCase.code != want {
				t.Fatalf("错误码三元组漂移: got %+v, want %+v", *testCase.code, want)
			}
			if group := int(testCase.code.Code / 100); group != testCase.wantGroup {
				t.Fatalf("HTTP 分组漂移: got %d, want %d", group, testCase.wantGroup)
			}
		})
	}
}

// TestControlErrorCodesAreDistinct 断言三个控制错误码互不重复。
func TestControlErrorCodesAreDistinct(t *testing.T) {
	codes := []struct {
		name string
		code *apperr.ErrorCode
	}{
		{name: "取消", code: control.Canceled},
		{name: "请求接收超时", code: control.RequestTimeout},
		{name: "内部操作超时", code: control.DeadlineExceeded},
	}
	seen := map[apperr.Code]bool{}
	for _, item := range codes {
		if seen[item.code.Code] {
			t.Fatalf("错误码重复: %d", item.code.Code)
		}
		seen[item.code.Code] = true
	}
}
