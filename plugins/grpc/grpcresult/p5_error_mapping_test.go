package grpcresult_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/generate"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/grpcresult"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// 五项 P5 新增协议输入夹具逐字对应受控词表；不引入跨插件运行依赖。
func TestP5PluginCodesOverGRPC(t *testing.T) {
	tests := []struct {
		code       apperr.ErrorCode
		wantStatus codes.Code
	}{
		{code: apperr.ErrorCode{Code: 42903, Output: "ASYNC_POOL_FULL", Message: "异步任务池已满"}, wantStatus: codes.ResourceExhausted},
		{code: apperr.ErrorCode{Code: 50038, Output: "CACHE_DRIVER_CODEC_FAILED", Message: "缓存编解码失败"}, wantStatus: codes.Internal},
		{code: apperr.ErrorCode{Code: 50039, Output: "CACHE_DRIVER_OPERATION_FAILED", Message: "缓存操作失败"}, wantStatus: codes.Internal},
		{code: apperr.ErrorCode{Code: 50040, Output: "CRON_SCHEDULER_ERROR", Message: "定时调度器错误"}, wantStatus: codes.Internal},
		{code: apperr.ErrorCode{Code: 50041, Output: "ASYNC_POOL_ERROR", Message: "异步任务池错误"}, wantStatus: codes.Internal},
	}
	for _, tc := range tests {
		t.Run(string(tc.code.Output), func(t *testing.T) {
			e := apperr.New(&tc.code, "插件操作失败", errors.New("secret-credential-or-payload"))
			st := status.Convert(grpcresult.ToStatusError(context.Background(), e))
			if st.Code() != tc.wantStatus || st.Message() != string(tc.code.Message) {
				t.Fatalf("incorrect status: %v", st)
			}
			details := st.Details()
			if len(details) != 1 {
				t.Fatalf("details count %d", len(details))
			}
			response, ok := details[0].(*generate.BaseResponse)
			if !ok || response.Code != uint64(tc.code.Code) || response.Output != string(tc.code.Output) || response.Message != string(tc.code.Message) {
				t.Fatalf("incorrect details: %+v", details)
			}
			if strings.Contains(st.Err().Error(), "secret-credential-or-payload") {
				t.Fatal("native cause leaked")
			}
		})
	}
}
