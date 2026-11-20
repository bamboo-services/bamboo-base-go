package validator

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// 错误码由 spec/0002 统一分配；新增定义必须先登记词表，禁止重复编号或标识。
var (
	BodyError       = &apperr.ErrorCode{Code: 40006, Output: "HTTP_VALIDATOR_BODY_ERROR", Message: "请求体错误"}      // 请求体错误。
	ValidationError = &apperr.ErrorCode{Code: 40007, Output: "HTTP_VALIDATOR_VALIDATION_ERROR", Message: "验证错误"} // 验证错误。
)
