package xBase

// BaseResponse 表示通用的响应结构体，用于封装 API 的返回结果。
//
// 该结构体设计用于标准化 API 响应，其中包含上下文、输出、状态码、消息和数据等字段。
//
// 注意: `ErrorMessage` 和 `Data` 字段是可选的，可能在某些情况下为空。
type BaseResponse struct {
	Context      string      `json:"context"`
	Output       string      `json:"output"`
	Code         uint        `json:"code"`
	Message      string      `json:"message"`
	ErrorMessage string      `json:"error_message,omitempty"`
	Overhead     *int64      `json:"overhead,omitempty"`
	Data         interface{} `json:"data,omitempty"`
}
