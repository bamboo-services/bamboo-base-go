package control

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// 错误码由 spec/0002 统一分配；新增定义必须先登记词表，禁止重复编号或标识。
//
// 取消使用 499 分组（非标准 HTTP 扩展），请求接收/读取超时使用 408 分组，
// 应用内部操作超时使用 504 分组；三者不得互相替代。
var (
	Canceled         = &apperr.ErrorCode{Code: 49900, Output: "CONTROL_CANCELED", Message: "操作已取消"}             // 操作已取消。
	RequestTimeout   = &apperr.ErrorCode{Code: 40802, Output: "CONTROL_REQUEST_TIMEOUT", Message: "请求接收超时"}     // 请求接收超时。
	DeadlineExceeded = &apperr.ErrorCode{Code: 50401, Output: "CONTROL_DEADLINE_EXCEEDED", Message: "操作超过截止时间"} // 操作超过截止时间。
)
