package bind

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// 错误码由 spec/0002 统一分配；新增定义必须先登记词表，禁止重复编号或标识。
var (
	ParameterError = &apperr.ErrorCode{Code: 40005, Output: "HTTP_BIND_PARAMETER_ERROR", Message: "参数错误"} // 参数错误。
)
