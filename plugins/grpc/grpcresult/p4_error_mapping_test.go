package grpcresult_test

import (
	"context"
	"errors"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/generate"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/grpcresult"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"testing"
)

// 五项协议输入夹具逐字对应受控词表；不引入跨插件运行依赖。
func TestP4DatabaseCodesOverGRPC(t *testing.T) {
	for _, code := range []apperr.ErrorCode{
		{Code: 50033, Output: "DATABASE_DIALECT_DRIVER_INVALID", Message: "数据库驱动声明非法"},
		{Code: 50034, Output: "DATABASE_DIALECT_DRIVER_NOT_LOADED", Message: "数据库驱动未加载"},
		{Code: 50035, Output: "DATABASE_DIALECT_ADAPTER_INVALID", Message: "数据库适配器无效"},
		{Code: 50036, Output: "DATABASE_DIALECT_ADAPTER_CONFIG_INVALID", Message: "数据库适配器配置非法"},
		{Code: 50037, Output: "DATABASE_MODEL_INVALID", Message: "数据库模型契约无效"},
	} {
		t.Run(string(code.Output), func(t *testing.T) {
			e := apperr.New(&code, "数据库操作失败", errors.New("dsn-secret"))
			st := status.Convert(grpcresult.ToStatusError(context.Background(), e))
			if st.Code() != codes.Internal || st.Message() != string(code.Message) {
				t.Fatalf("incorrect status: %v", st)
			}
			details := st.Details()
			if len(details) != 1 {
				t.Fatalf("details count %d", len(details))
			}
			response, ok := details[0].(*generate.BaseResponse)
			if !ok || response.Code != uint64(code.Code) || response.Output != string(code.Output) || response.Message != string(code.Message) {
				t.Fatalf("incorrect details: %+v", details)
			}
			if strings.Contains(st.Err().Error(), "dsn-secret") {
				t.Fatal("native cause leaked")
			}
		})
	}
}
