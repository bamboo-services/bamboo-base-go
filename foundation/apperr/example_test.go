package apperr_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// ExampleNew 演示以预置错误码构造框架内部错误并读取其文本。
func ExampleNew() {
	err := apperr.New(apperr.FrameworkComponentNotFound, "组件 database 未注册")
	fmt.Println(err.Code.GetCode(), err.Code.GetOutput())
	fmt.Println(err.Error())
	// Output:
	// 50003 FRAMEWORK_COMPONENT_NOT_FOUND
	// 组件未找到: 组件 database 未注册
}

// ExampleNew_withCause 演示携带底层根因时通过 errors.Is 识别根因。
func ExampleNew_withCause() {
	err := apperr.New(apperr.FrameworkInternalError, "读取组件失败", io.EOF)
	fmt.Println(errors.Is(err, io.EOF))
	fmt.Println(err.Error())
	// Output:
	// true
	// 框架内部错误: 读取组件失败: EOF
}

// ExampleNew_customCode 演示业务自行定义错误码，无需扩充基础目录。
func ExampleNew_customCode() {
	orderClosed := &apperr.ErrorCode{
		Code:    40950,
		Output:  "SHOP_ORDER_CLOSED",
		Message: "订单已关闭",
	}
	err := apperr.New(orderClosed, "不能重复支付")
	fmt.Println(err.Code.GetCode(), err.Code.GetOutput())
	fmt.Println(err.Error())
	// Output:
	// 40950 SHOP_ORDER_CLOSED
	// 订单已关闭: 不能重复支付
}

// ExampleNew_nilCode 演示 code 为 nil 时回退为框架内部错误。
func ExampleNew_nilCode() {
	err := apperr.New(nil, "")
	fmt.Println(err.Code.GetCode(), err.Code.GetOutput())
	fmt.Println(err.Error())
	// Output:
	// 50000 FRAMEWORK_INTERNAL_ERROR
	// 框架内部错误
}

// ExampleError 演示 JSON 序列化只输出对外字段，底层根因不会泄漏。
func ExampleError() {
	root := errors.New("password=hunter2")
	err := apperr.New(apperr.FrameworkInternalError, "读取组件失败", root)

	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		fmt.Println(marshalErr)
		return
	}
	fmt.Println(string(data))
	// Output:
	// {"code":{"Code":50000,"Output":"FRAMEWORK_INTERNAL_ERROR","Message":"框架内部错误"},"error_message":"读取组件失败"}
}

// ExampleFrom 演示转接原生原因时沿用既有统一错误的分类，并保留原始原因链。
func ExampleFrom() {
	inner := apperr.New(apperr.FrameworkComponentNotFound, "组件 database 未注册")
	appErr := apperr.From(apperr.FrameworkInternalError, "启动失败", fmt.Errorf("装配阶段: %w", inner))

	fmt.Println(appErr.Code.GetCode(), appErr.Code.GetOutput())
	fmt.Println(errors.Is(appErr, inner))
	// Output:
	// 50003 FRAMEWORK_COMPONENT_NOT_FOUND
	// true
}

// ExampleJoin 演示多个失败聚合时由调用方显式给出总码，子失败全部保留。
func ExampleJoin() {
	first := apperr.New(apperr.FrameworkComponentNotFound, "组件未注册")
	second := apperr.New(apperr.FrameworkComponentUnavailable, "组件不可用")
	appErr := apperr.Join(apperr.FrameworkMultipleFailures, "多个阶段失败", first, second)

	fmt.Println(appErr.Code.GetCode(), appErr.Code.GetOutput())
	fmt.Println(errors.Is(appErr, first), errors.Is(appErr, second))
	// Output:
	// 50020 FRAMEWORK_MULTIPLE_FAILURES
	// true true
}
