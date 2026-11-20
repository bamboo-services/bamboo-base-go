package toolkit

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// 错误码由 spec/0002 统一分配；新增定义必须先登记词表，禁止重复编号或标识。
var (
	PasswordFailed = &apperr.ErrorCode{Code: 50032, Output: "TOOLKIT_PASSWORD_FAILED", Message: "密码加密或校验失败"} // 密码加密或校验失败。
)
