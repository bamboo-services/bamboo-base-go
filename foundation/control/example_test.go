package control_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
)

// ExampleClassify 演示取消原因被归一为 499 分组的控制错误。
func ExampleClassify() {
	err := control.Classify(context.Canceled, control.PhaseOperation)
	fmt.Println(err.Code.GetCode(), err.Code.GetOutput())
	fmt.Println(errors.Is(err, context.Canceled))

	// Output:
	// 49900 CONTROL_CANCELED
	// true
}

// ExampleClassify_requestPhase 演示请求接收阶段的超时归一为 408。
func ExampleClassify_requestPhase() {
	err := control.Classify(context.DeadlineExceeded, control.PhaseRequestRead)
	fmt.Println(err.Code.GetCode(), err.Code.GetOutput())

	// Output:
	// 40802 CONTROL_REQUEST_TIMEOUT
}

// ExampleClassify_unknownCause 演示未识别的普通原因必须由所属域显式兜底。
func ExampleClassify_unknownCause() {
	cause := errors.New("数据库连接失败")

	err := control.Classify(cause, control.PhaseOperation)
	fmt.Println(err == nil)

	// 返回 nil 只表示不是控制原因，调用方必须在自己的所属域兜底并保留原因链。
	fallback := apperr.From(apperr.FrameworkInternalError, "读取配置失败", cause)
	fmt.Println(fallback.Code.GetCode())
	fmt.Println(errors.Is(fallback, cause))

	// Output:
	// true
	// 50000
	// true
}
