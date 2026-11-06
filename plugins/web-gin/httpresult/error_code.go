package httpresult

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// 错误码由 spec/0002 统一分配；新增定义必须先登记词表，禁止重复编号或标识。
var (
	BadRequest          = &apperr.ErrorCode{Code: 40000, Output: "HTTP_BAD_REQUEST", Message: "错误请求"}             // 错误请求。
	Disabled            = &apperr.ErrorCode{Code: 40001, Output: "HTTP_DISABLED", Message: "内容已禁用"}               // 内容已禁用。
	InvalidState        = &apperr.ErrorCode{Code: 40002, Output: "HTTP_INVALID_STATE", Message: "无效状态"}           // 无效状态。
	ParameterEmpty      = &apperr.ErrorCode{Code: 40003, Output: "HTTP_PARAMETER_EMPTY", Message: "参数缺失"}         // 参数缺失。
	ParameterType       = &apperr.ErrorCode{Code: 40004, Output: "HTTP_PARAMETER_TYPE", Message: "参数类型错误"}        // 参数类型错误。
	Unauthorized        = &apperr.ErrorCode{Code: 40100, Output: "HTTP_UNAUTHORIZED", Message: "未授权"}             // 未授权。
	Forbidden           = &apperr.ErrorCode{Code: 40300, Output: "HTTP_FORBIDDEN", Message: "禁止访问"}               // 禁止访问。
	PermissionDenied    = &apperr.ErrorCode{Code: 40301, Output: "HTTP_PERMISSION_DENIED", Message: "权限不足"}       // 权限不足。
	OperationDenied     = &apperr.ErrorCode{Code: 40302, Output: "HTTP_OPERATION_DENIED", Message: "操作被拒绝"}       // 操作被拒绝。
	NotFound            = &apperr.ErrorCode{Code: 40400, Output: "HTTP_NOT_FOUND", Message: "未找到"}                // 未找到。
	MethodNotAllowed    = &apperr.ErrorCode{Code: 40500, Output: "HTTP_METHOD_NOT_ALLOWED", Message: "方法不允许"}     // 方法不允许。
	NotAcceptable       = &apperr.ErrorCode{Code: 40600, Output: "HTTP_NOT_ACCEPTABLE", Message: "不可接受"}          // 不可接受。
	Timeout             = &apperr.ErrorCode{Code: 40800, Output: "HTTP_TIMEOUT", Message: "请求超时"}                 // 请求超时。
	Conflict            = &apperr.ErrorCode{Code: 40900, Output: "HTTP_CONFLICT", Message: "冲突"}                  // 冲突。
	Gone                = &apperr.ErrorCode{Code: 41000, Output: "HTTP_GONE", Message: "资源已删除"}                   // 资源已删除。
	PayloadTooLarge     = &apperr.ErrorCode{Code: 41300, Output: "HTTP_PAYLOAD_TOO_LARGE", Message: "请求体过大"}      // 请求体过大。
	UnsupportedMedia    = &apperr.ErrorCode{Code: 41500, Output: "HTTP_UNSUPPORTED_MEDIA", Message: "不支持的媒体类型"}   // 不支持的媒体类型。
	UnprocessableEntity = &apperr.ErrorCode{Code: 42200, Output: "HTTP_UNPROCESSABLE_ENTITY", Message: "无法处理的实体"} // 无法处理的实体。
	TooManyRequests     = &apperr.ErrorCode{Code: 42900, Output: "HTTP_TOO_MANY_REQUESTS", Message: "请求过多"}       // 请求过多。
	InternalError       = &apperr.ErrorCode{Code: 50008, Output: "HTTP_INTERNAL_ERROR", Message: "服务器内部错误"}       // 服务器内部错误。
	ResponseMissing     = &apperr.ErrorCode{Code: 50009, Output: "HTTP_RESPONSE_MISSING", Message: "处理器未输出响应"}    // 处理器未输出响应。
	UnknownError        = &apperr.ErrorCode{Code: 50010, Output: "HTTP_UNKNOWN_ERROR", Message: "未知错误"}           // 未知错误。
	GatewayError        = &apperr.ErrorCode{Code: 50200, Output: "HTTP_GATEWAY_ERROR", Message: "网关错误"}           // 网关错误。
	ServiceUnavailable  = &apperr.ErrorCode{Code: 50300, Output: "HTTP_SERVICE_UNAVAILABLE", Message: "服务不可用"}    // 服务不可用。
	ResourceExhausted   = &apperr.ErrorCode{Code: 50301, Output: "HTTP_RESOURCE_EXHAUSTED", Message: "资源耗尽"}      // 资源耗尽。
	GatewayTimeout      = &apperr.ErrorCode{Code: 50400, Output: "HTTP_GATEWAY_TIMEOUT", Message: "网关超时"}         // 网关超时。
)
