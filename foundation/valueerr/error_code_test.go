package valueerr_test

import (
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/valueerr"
)

// TestValueErrorCodes 按表驱动断言值错误码的三元组与分组。
func TestValueErrorCodes(t *testing.T) {
	cases := []struct {
		name        string
		code        *apperr.ErrorCode
		wantCode    apperr.Code
		wantOutput  apperr.Output
		wantMessage apperr.Message
		wantGroup   int
	}{
		{
			name:        "格式非法",
			code:        valueerr.InvalidFormat,
			wantCode:    40013,
			wantOutput:  "VALUE_INVALID_FORMAT",
			wantMessage: "值格式非法",
			wantGroup:   400,
		},
		{
			name:        "超出范围",
			code:        valueerr.OutOfRange,
			wantCode:    40014,
			wantOutput:  "VALUE_OUT_OF_RANGE",
			wantMessage: "值超出范围",
			wantGroup:   400,
		},
		{
			name:        "类型非法",
			code:        valueerr.InvalidType,
			wantCode:    40015,
			wantOutput:  "VALUE_INVALID_TYPE",
			wantMessage: "值类型非法",
			wantGroup:   400,
		},
		{
			name:        "不支持的转换目标",
			code:        valueerr.UnsupportedTarget,
			wantCode:    50018,
			wantOutput:  "VALUE_UNSUPPORTED_TARGET",
			wantMessage: "不支持的转换目标",
			wantGroup:   500,
		},
		{
			name:        "值接收器无效",
			code:        valueerr.InvalidReceiver,
			wantCode:    50019,
			wantOutput:  "VALUE_INVALID_RECEIVER",
			wantMessage: "值接收器无效",
			wantGroup:   500,
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

// TestValueErrorCodesAreDistinct 断言五个值错误码互不重复。
func TestValueErrorCodesAreDistinct(t *testing.T) {
	codes := []struct {
		name string
		code *apperr.ErrorCode
	}{
		{name: "格式非法", code: valueerr.InvalidFormat},
		{name: "超出范围", code: valueerr.OutOfRange},
		{name: "类型非法", code: valueerr.InvalidType},
		{name: "不支持的转换目标", code: valueerr.UnsupportedTarget},
		{name: "值接收器无效", code: valueerr.InvalidReceiver},
	}
	seen := map[apperr.Code]bool{}
	for _, item := range codes {
		if seen[item.code.Code] {
			t.Fatalf("错误码重复: %d", item.code.Code)
		}
		seen[item.code.Code] = true
	}
}
