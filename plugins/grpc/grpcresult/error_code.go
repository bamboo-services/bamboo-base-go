package grpcresult

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// 错误码由 spec/0002 统一分配；新增定义必须先登记词表，禁止重复编号或标识。
var (
	ParameterError     = &apperr.ErrorCode{Code: 40008, Output: "GRPC_PARAMETER_ERROR", Message: "参数错误"}      // 参数错误。
	ParameterEmpty     = &apperr.ErrorCode{Code: 40009, Output: "GRPC_PARAMETER_EMPTY", Message: "参数缺失"}      // 参数缺失。
	ParameterType      = &apperr.ErrorCode{Code: 40010, Output: "GRPC_PARAMETER_TYPE", Message: "参数类型错误"}     // 参数类型错误。
	Disabled           = &apperr.ErrorCode{Code: 40011, Output: "GRPC_DISABLED", Message: "内容已禁用"}            // 内容已禁用。
	MetadataMissing    = &apperr.ErrorCode{Code: 40012, Output: "GRPC_METADATA_MISSING", Message: "请求元数据缺失"}  // 请求元数据缺失。
	Unauthorized       = &apperr.ErrorCode{Code: 40101, Output: "GRPC_UNAUTHORIZED", Message: "未授权"}          // 未授权。
	Forbidden          = &apperr.ErrorCode{Code: 40303, Output: "GRPC_FORBIDDEN", Message: "禁止访问"}            // 禁止访问。
	PermissionDenied   = &apperr.ErrorCode{Code: 40304, Output: "GRPC_PERMISSION_DENIED", Message: "权限不足"}    // 权限不足。
	OperationDenied    = &apperr.ErrorCode{Code: 40305, Output: "GRPC_OPERATION_DENIED", Message: "操作被拒绝"}    // 操作被拒绝。
	NotFound           = &apperr.ErrorCode{Code: 40401, Output: "GRPC_NOT_FOUND", Message: "未找到"}             // 未找到。
	Timeout            = &apperr.ErrorCode{Code: 40801, Output: "GRPC_TIMEOUT", Message: "请求超时"}              // 请求超时。
	Conflict           = &apperr.ErrorCode{Code: 40901, Output: "GRPC_CONFLICT", Message: "冲突"}               // 冲突。
	InvalidState       = &apperr.ErrorCode{Code: 41200, Output: "GRPC_INVALID_STATE", Message: "无效状态"}        // 无效状态。
	TooManyRequests    = &apperr.ErrorCode{Code: 42901, Output: "GRPC_TOO_MANY_REQUESTS", Message: "请求过多"}    // 请求过多。
	InternalError      = &apperr.ErrorCode{Code: 50011, Output: "GRPC_INTERNAL_ERROR", Message: "服务器内部错误"}    // 服务器内部错误。
	ResponseMissing    = &apperr.ErrorCode{Code: 50012, Output: "GRPC_RESPONSE_MISSING", Message: "处理器未返回响应"} // 处理器未返回响应。
	ContextMissing     = &apperr.ErrorCode{Code: 50013, Output: "GRPC_CONTEXT_MISSING", Message: "RPC上下文缺失"}  // RPC上下文缺失。
	ServiceUnavailable = &apperr.ErrorCode{Code: 50302, Output: "GRPC_SERVICE_UNAVAILABLE", Message: "服务不可用"} // 服务不可用。
)
