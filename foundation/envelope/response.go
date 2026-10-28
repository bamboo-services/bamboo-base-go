package envelope

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// Response 是与 apperr 并列的标准响应信封契约定义（原 xBase.BaseResponse）。
// 它只描述跨协议的标准返回字段，不绑定 HTTP 或 gRPC 写出行为；
// 序列化与字段省略规则见字段注释及包文档。
type Response struct {
	// Context 是请求唯一标识，由协议入口生成并透传；
	// 零值时 JSON 输出省略该字段。
	Context string `json:"context,omitempty"`

	// Code 是业务状态码，错误场景通常取自 apperr 预定义错误码的
	// Code 字段；JSON 输出恒包含该字段。
	Code apperr.Code `json:"code"`

	// Output 是业务输出类型标识，错误场景通常取自 apperr 预定义
	// 错误码的 Output 字段；JSON 输出恒包含该字段。
	Output apperr.Output `json:"output"`

	// Message 是面向调用方的状态描述，错误场景通常取自 apperr
	// 预定义错误码的 Message 字段；JSON 输出恒包含该字段。
	Message apperr.Message `json:"message"`

	// ErrorMessage 是错误详情，仅错误场景填写；
	// 零值时 JSON 输出省略该字段。
	ErrorMessage apperr.ErrMessage `json:"error_message,omitempty"`

	// Overhead 是服务端处理耗时，单位为微秒；
	// 零值时 JSON 输出省略该字段。
	Overhead int64 `json:"overhead,omitempty"`

	// Data 是业务数据负载，具体类型由接口约定；
	// 零值时 JSON 输出省略该字段。
	Data any `json:"data,omitempty"`
}
