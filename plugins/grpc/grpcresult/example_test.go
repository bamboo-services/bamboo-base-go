package grpcresult_test

import (
	"context"
	"fmt"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/grpcresult"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/grpcresult/internaltest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ExampleSuccess 演示构造标准成功 BaseResponse 信封。
func ExampleSuccess() {
	ctx := context.Background()
	resp := grpcresult.Success(ctx, "操作成功")

	fmt.Println(resp.GetCode())
	fmt.Println(resp.GetOutput())
	fmt.Println(resp.GetMessage())
	// Output:
	// 200
	// Success
	// 操作成功
}

// ExampleSuccessWith 演示使用泛型方法 SuccessWith 构造业务 Protobuf 响应并自动注入 BaseResponse 信封。
func ExampleSuccessWith() {
	ctx := context.Background()

	// ValidAlphaResponse 顶层包含唯一类型为 BaseResponse 的 header 字段
	resp := grpcresult.SuccessWith[*internaltest.ValidAlphaResponse](ctx, "数据加载成功")
	resp.Payload = "user_data_payload"

	fmt.Println(resp.GetHeader().GetCode())
	fmt.Println(resp.GetHeader().GetMessage())
	fmt.Println(resp.GetPayload())
	// Output:
	// 200
	// 数据加载成功
	// user_data_payload
}

// ExampleToStatusError 演示将统一业务错误 apperr.Error 转换为标准 gRPC status 错误。
func ExampleToStatusError() {
	ctx := context.Background()
	bizErr := apperr.New(apperr.ParameterError, "请求参数无效")

	stErr := grpcresult.ToStatusError(ctx, bizErr)
	st, ok := status.FromError(stErr)

	fmt.Println(ok)
	fmt.Println(st.Code() == codes.InvalidArgument)
	fmt.Println(st.Message())
	// Output:
	// true
	// true
	// 参数错误
}
