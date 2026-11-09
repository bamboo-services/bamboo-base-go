package valueerr_test

import (
	"errors"
	"fmt"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/valueerr"
)

// ExampleFrom 演示输入值格式非法归 400 组值错误。
func ExampleFrom() {
	cause := errors.New("strconv.ParseInt: invalid syntax")

	err := valueerr.From(valueerr.KindInvalidFormat, "端口号格式非法", cause)
	fmt.Println(err.Code.GetCode(), err.Code.GetOutput())
	fmt.Println(errors.Is(err, cause))

	// Output:
	// 40013 VALUE_INVALID_FORMAT
	// true
}

// ExampleFrom_unsupportedTarget 演示转换目标不受支持归 500 组契约错误。
func ExampleFrom_unsupportedTarget() {
	cause := errors.New("目标类型没有对应的转换通道")

	err := valueerr.From(valueerr.KindUnsupportedTarget, "不支持的转换目标", cause)
	fmt.Println(err.Code.GetCode(), err.Code.GetOutput())

	// Output:
	// 50018 VALUE_UNSUPPORTED_TARGET
}

// ExampleFrom_existingStructured 演示已有结构化错误不会被改判为值错误。
func ExampleFrom_existingStructured() {
	existing := apperr.New(apperr.FrameworkInternalError, "插件初始化失败", errors.New("端口被占用"))

	err := valueerr.From(valueerr.KindInvalidFormat, "端口号格式非法", existing)
	fmt.Println(err.Code.GetCode(), err.Code.GetOutput())

	// Output:
	// 50000 FRAMEWORK_INTERNAL_ERROR
}

// ExampleFrom_configurationResponsibility 演示配置责任边界由所属域重新包装。
func ExampleFrom_configurationResponsibility() {
	cause := errors.New("配置文件端口号不是数字")

	valueErr := valueerr.From(valueerr.KindInvalidFormat, "端口号格式非法", cause)
	// 读取服务自身配置的责任边界：用所属域的 500 码显式重新包装，
	// 并把本包产出的值错误保留为原因。
	wrapped := apperr.New(apperr.FrameworkInternalError, "读取服务配置失败", valueErr)
	fmt.Println(wrapped.Code.GetCode())
	fmt.Println(errors.Is(wrapped, valueErr))

	// Output:
	// 50000
	// true
}
