package envelope_test

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/envelope"
)

// ExampleResponse 展示一个完整的错误信封如何序列化为标准 JSON：
// 可选字段非零时全部输出。
func ExampleResponse() {
	resp := envelope.Response{
		Context:      "req-123456",
		Code:         apperr.BadRequest.Code,
		Output:       apperr.BadRequest.Output,
		Message:      apperr.BadRequest.Message,
		ErrorMessage: "field 'username' is invalid",
		Overhead:     12,
		Data:         map[string]any{"field": "username"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
	// Output:
	// {"context":"req-123456","code":40010,"output":"BAD_REQUEST","message":"错误请求","error_message":"field 'username' is invalid","overhead":12,"data":{"field":"username"}}
}

// ExampleResponse_minimal 展示最小信封的序列化结果：
// Context、ErrorMessage、Overhead 与 Data 为零值时被省略，
// Code、Output 与 Message 恒定输出。
func ExampleResponse_minimal() {
	resp := envelope.Response{
		Code:    apperr.ServerInternalError.Code,
		Output:  apperr.ServerInternalError.Output,
		Message: apperr.ServerInternalError.Message,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
	// Output:
	// {"code":50000,"output":"SERVER_INTERNAL_ERROR","message":"服务器内部错误"}
}
