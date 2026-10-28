package envelope

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// Response 是与 apperr 并列的标准响应信封契约定义（原 xBase.BaseResponse）。
// 它只描述跨协议的标准返回字段，不绑定 HTTP 或 gRPC 写出行为。
type Response struct {
	Context      string            `json:"context,omitempty"`
	Code         apperr.Code       `json:"code"`
	Output       apperr.Output     `json:"output"`
	Message      apperr.Message    `json:"message"`
	ErrorMessage apperr.ErrMessage `json:"error_message,omitempty"`
	Overhead     int64             `json:"overhead,omitempty"`
	Data         any               `json:"data,omitempty"`
}
