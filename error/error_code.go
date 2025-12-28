package xError

// ErrorCode 错误信息类型
//
// 用于定义系统中的错误相关信息，包括错误代码及错误信息。
// 提供统一的结构化方式管理和访问错误数据。
type ErrorCode struct {
	Code    uint   // 错误码
	Message string // 错误信息（中文）
}

// GetCode 获取错误码
func (e *ErrorCode) GetCode() uint {
	return e.Code
}

// GetMessage 获取错误信息
func (e *ErrorCode) GetMessage() string {
	return e.Message
}

// GetOutput 根据错误码获取 HTTP 状态输出标识
func (e *ErrorCode) GetOutput() string {
	switch e.Code / 100 {
	case 200:
		return "Success"
	case 400:
		return "BadRequest"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "NotFound"
	case 405:
		return "MethodNotAllowed"
	case 406:
		return "NotAcceptable"
	case 408:
		return "RequestTimeout"
	case 429:
		return "TooManyRequests"
	case 500:
		return "InternalServerError"
	case 502:
		return "BadGateway"
	case 503:
		return "ServiceUnavailable"
	default:
		return "Error"
	}
}

// ============================== 通用错误码 (400xx) ==============================

var (
	NotExist = &ErrorCode{40000, "内容不存在"}
	Existed  = &ErrorCode{40001, "内容已存在"}
	Expired  = &ErrorCode{40002, "内容已过期"}
)

// ============================== 400 Bad Request (400xx) ==============================

var (
	BadRequest     = &ErrorCode{40010, "错误请求"}
	ParameterError = &ErrorCode{40011, "参数错误"}
	ParameterEmpty = &ErrorCode{40012, "参数缺失"}
	ParameterType  = &ErrorCode{40013, "参数类型错误"}
	BodyError      = &ErrorCode{40014, "请求体错误"}
	BodyEmpty      = &ErrorCode{40015, "请求体缺失"}
	BodyType       = &ErrorCode{40016, "请求体类型错误"}
	HeaderError    = &ErrorCode{40017, "请求头错误"}
	HeaderEmpty    = &ErrorCode{40018, "请求头缺失"}
	HeaderType     = &ErrorCode{40019, "请求头类型错误"}
)

// ============================== 操作错误 (400xx) ==============================

var (
	OperationError   = &ErrorCode{40020, "操作错误"}
	OperationFailed  = &ErrorCode{40021, "操作失败"}
	OperationDenied  = &ErrorCode{40022, "操作被拒绝"}
	OperationInvalid = &ErrorCode{40023, "操作无效"}
	DeveloperError   = &ErrorCode{40024, "开发者操作错误"}
)

// ============================== 401 Unauthorized (401xx) ==============================

var (
	Unauthorized = &ErrorCode{40100, "未授权"}
	LoginFailed  = &ErrorCode{40101, "登录失败"}
	TokenInvalid = &ErrorCode{40102, "令牌无效"}
	TokenExpired = &ErrorCode{40103, "令牌过期"}
)

// ============================== 403 Forbidden (403xx) ==============================

var (
	Forbidden        = &ErrorCode{40300, "禁止访问"}
	PermissionDenied = &ErrorCode{40301, "权限不足"}
	AccessLimited    = &ErrorCode{40302, "访问受限"}
)

// ============================== 404 Not Found (404xx) ==============================

var (
	NotFound         = &ErrorCode{40400, "未找到"}
	PageNotFound     = &ErrorCode{40401, "页面未找到"}
	ResourceNotFound = &ErrorCode{40402, "资源未找到"}
)

// ============================== 405 Method Not Allowed (405xx) ==============================

var (
	MethodNotAllowed = &ErrorCode{40500, "方法不允许"}
)

// ============================== 406 Not Acceptable (406xx) ==============================

var (
	NotAcceptable = &ErrorCode{40600, "不可接受"}
)

// ============================== 408 Request Timeout (408xx) ==============================

var (
	Timeout        = &ErrorCode{40800, "请求超时"}
	ConnectTimeout = &ErrorCode{40801, "连接超时"}
	ReadTimeout    = &ErrorCode{40802, "读取超时"}
	WriteTimeout   = &ErrorCode{40803, "写入超时"}
)

// ============================== 429 Too Many Requests (429xx) ==============================

var (
	TooManyRequests = &ErrorCode{42900, "请求过多"}
	RateLimited     = &ErrorCode{42901, "请求频率过高"}
)

// ============================== 500 Internal Server Error (500xx) ==============================

var (
	ServerError   = &ErrorCode{50000, "服务器内部错误"}
	DatabaseError = &ErrorCode{50001, "数据库错误"}
	CacheError    = &ErrorCode{50002, "缓存错误"}
	FileError     = &ErrorCode{50003, "文件错误"}
	StorageError  = &ErrorCode{50004, "存储错误"}
	RemoteError   = &ErrorCode{50005, "远程调用错误"}
	ConfigError   = &ErrorCode{50006, "配置错误"}
	NetworkError  = &ErrorCode{50007, "网络错误"}
	UnknownError  = &ErrorCode{50099, "未知错误"}
)

// ============================== 502 Bad Gateway (502xx) ==============================

var (
	GatewayError = &ErrorCode{50200, "网关错误"}
)

// ============================== 503 Service Unavailable (503xx) ==============================

var (
	ServiceUnavailable = &ErrorCode{50300, "服务不可用"}
	SystemMaintenance  = &ErrorCode{50301, "系统维护中"}
	ResourceExhausted  = &ErrorCode{50302, "资源耗尽"}
)
