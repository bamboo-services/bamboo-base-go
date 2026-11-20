package email

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// 错误码由 spec/0002 统一分配；新增定义必须先登记词表，禁止重复编号或标识。
var (
	EmailError = &apperr.ErrorCode{Code: 50017, Output: "EMAIL_ERROR", Message: "邮件服务错误"} // 邮件服务错误。
)
