package validator_test

import (
	"fmt"

	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/validator"
)

// ExampleUser 用于示例演示的结构体定义。
type ExampleUser struct {
	Name  string `json:"name" binding:"required,min=2"`
	Role  string `json:"role" binding:"required,enum_string=admin user guest"`
	Score int    `json:"score" binding:"min=0,max=100"`
}

// 本示例演示构造独立验证引擎并执行结构体验证。
func ExampleEngine_Validate() {
	engine, err := validator.New()
	if err != nil {
		panic(err)
	}

	validUser := ExampleUser{
		Name:  "konata",
		Role:  "admin",
		Score: 98,
	}
	fmt.Printf("合法用户校验: %t\n", engine.Validate(&validUser) == nil)

	invalidUser := ExampleUser{
		Name:  "k",
		Role:  "superadmin",
		Score: 120,
	}
	fmt.Printf("非法用户校验失败: %t\n", engine.Validate(&invalidUser) != nil)

	// Output:
	// 合法用户校验: true
	// 非法用户校验失败: true
}

// 本示例演示将结构体校验错误翻译为本地化中文提示。
func ExampleEngine_TranslateError() {
	engine, err := validator.New()
	if err != nil {
		panic(err)
	}

	invalidUser := ExampleUser{
		Name:  "",
		Role:  "unknown",
		Score: -5,
	}
	err = engine.Validate(&invalidUser)
	if err != nil {
		errMap := engine.TranslateError(err)
		fmt.Printf("name 错误包含必填: %t\n", errMap["name"] != "")
		fmt.Printf("role 错误非空: %t\n", errMap["role"] != "")
	}

	// Output:
	// name 错误包含必填: true
	// role 错误非空: true
}
