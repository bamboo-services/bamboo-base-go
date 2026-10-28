package apperr_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// ExampleNew 演示以预置错误码构造业务错误并读取其文本。
func ExampleNew() {
	err := apperr.New(apperr.RecordNotFound, "记录 42 不存在")
	fmt.Println(err.Code.GetCode(), err.Code.GetOutput())
	fmt.Println(err.Error())
	// Output:
	// 40406 RECORD_NOT_FOUND
	// 记录未找到: 记录 42 不存在
}

// ExampleNew_withCause 演示携带底层根因时通过 errors.Is 识别根因。
func ExampleNew_withCause() {
	err := apperr.New(apperr.DatabaseError, "查询用户失败", io.EOF)
	fmt.Println(errors.Is(err, io.EOF))
	fmt.Println(err.Error())
	// Output:
	// true
	// 数据库错误: 查询用户失败: EOF
}

// ExampleNew_nilCode 演示 code 为 nil 时回退为服务器内部错误。
func ExampleNew_nilCode() {
	err := apperr.New(nil, "")
	fmt.Println(err.Code.GetCode(), err.Code.GetOutput())
	fmt.Println(err.Error())
	// Output:
	// 50000 SERVER_INTERNAL_ERROR
	// 服务器内部错误
}

// ExampleError 演示 JSON 序列化只输出对外字段，底层根因不会泄漏。
func ExampleError() {
	root := errors.New("password=hunter2")
	err := apperr.New(apperr.DatabaseError, "查询用户失败", root)

	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		fmt.Println(marshalErr)
		return
	}
	fmt.Println(string(data))
	// Output:
	// {"code":{"Code":50001,"Output":"DATABASE_ERROR","Message":"数据库错误"},"error_message":"查询用户失败"}
}
