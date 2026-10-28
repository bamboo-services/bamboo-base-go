package generate_test

import (
	"fmt"

	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/generate"
)

// ExampleBaseResponse 演示 BaseResponse 基础响应结构的使用。
func ExampleBaseResponse() {
	resp := &generate.BaseResponse{
		Context: "req-123456",
		Output:  "Success",
		Code:    200,
		Message: "操作成功",
	}

	fmt.Println(resp.GetCode())
	fmt.Println(resp.GetOutput())
	fmt.Println(resp.GetMessage())
	// Output:
	// 200
	// Success
	// 操作成功
}
