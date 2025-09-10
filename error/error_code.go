package xError

// ErrorCode 错误信息类型
//
// 用于定义系统中的错误相关信息，包括输出内容、错误代码及错误信息。
// 提供统一的结构化方式管理和访问错误数据。
//
// 注意: 请确保错误代码和信息符合系统统一规范，便于调试与问题定位。
type ErrorCode struct {
	Output  string
	Code    uint
	Message string
}

// NewErrorCode 创建错误代码实例
func NewErrorCode(output string, code uint, message string) *ErrorCode {
	return &ErrorCode{
		Output:  output,
		Code:    code,
		Message: message,
	}
}

// GetOutput 获取输出
//
// 该输出为英文输出, 用于输出错误信息的英文信息。
//
// @return string 输出(英文)
func (e *ErrorCode) GetOutput() string {
	return e.Output
}

// GetCode 获取错误码
//
// 该数字由 5 位数字组成, 用于定义系统中的错误码信息, 用于统一管理系统中的错误码信息。
//
// @return int 错误码(00000)
func (e *ErrorCode) GetCode() uint {
	return e.Code
}

// GetMessage 获取错误信息
//
// 该信息为中文信息, 用于定义系统中的错误信息, 用于统一管理系统中的错误信息。
//
// @return string 错误信息(中文)
func (e *ErrorCode) GetMessage() string {
	return e.Message
}

var (
	// ============================== 通用错误码 ==============================
	NotExist = NewErrorCode("NotExist", 40000, "内容不存在") // 40000: 内容不存在
	Existed  = NewErrorCode("Existed", 40001, "内容已存在")  // 40001: 内容已存在

	// ============================== 4xx 客户端错误 ==============================
	// ----- 400 Bad Request -----
	BadRequest            = NewErrorCode("BadRequest", 40024, "错误请求")               // 40024: 错误请求
	ParameterError        = NewErrorCode("ParameterError", 40002, "参数错误")           // 40002: 参数错误
	ParameterMissing      = NewErrorCode("ParameterMissing", 40003, "参数缺失")         // 40003: 参数缺失
	ParameterInvalid      = NewErrorCode("ParameterInvalid", 40004, "参数无效")         // 40004: 参数无效
	ParameterIllegal      = NewErrorCode("ParameterIllegal", 40005, "参数非法")         // 40005: 参数非法
	ParameterTypeError    = NewErrorCode("ParameterTypeError", 40006, "参数类型错误")     // 40006: 参数类型错误
	BodyError             = NewErrorCode("BodyError", 40007, "请求体错误")               // 40007: 请求体错误
	BodyMissing           = NewErrorCode("BodyMissing", 40008, "请求体缺失")             // 40008: 请求体缺失
	BodyInvalid           = NewErrorCode("BodyInvalid", 40009, "请求体无效")             // 40009: 请求体无效
	BodyIllegal           = NewErrorCode("BodyIllegal", 40010, "请求体非法")             // 40010: 请求体非法
	BodyTypeError         = NewErrorCode("BodyTypeError", 40011, "请求体类型错误")         // 40011: 请求体类型错误
	HeaderError           = NewErrorCode("HeaderError", 40012, "请求头错误")             // 40012: 请求头错误
	HeaderMissing         = NewErrorCode("HeaderMissing", 40013, "请求头缺失")           // 40013: 请求头缺失
	HeaderInvalid         = NewErrorCode("HeaderInvalid", 40014, "请求头无效")           // 40014: 请求头无效
	HeaderIllegal         = NewErrorCode("HeaderIllegal", 40015, "请求头非法")           // 40015: 请求头非法
	HeaderTypeError       = NewErrorCode("HeaderTypeError", 40016, "请求头类型错误")       // 40016: 请求头类型错误
	OperationError        = NewErrorCode("OperationError", 40017, "操作错误")           // 40017: 操作错误
	OperationFailed       = NewErrorCode("OperationFailed", 40018, "操作失败")          // 40018: 操作失败
	OperationInvalid      = NewErrorCode("OperationInvalid", 40019, "操作无效")         // 40019: 操作无效
	OperationIllegal      = NewErrorCode("OperationIllegal", 40020, "操作非法")         // 40020: 操作非法
	OperationDenied       = NewErrorCode("OperationDenied", 40021, "操作被拒绝")         // 40021: 操作被拒绝
	OperationNotAllowed   = NewErrorCode("OperationNotAllowed", 40022, "操作不允许")     // 40022: 操作不允许
	OperationNotSupported = NewErrorCode("OperationNotSupported", 40023, "操作不支持")   // 40023: 操作不支持
	OperationTypeError    = NewErrorCode("OperationTypeError", 40025, "操作类型错误")     // 40025: 操作类型错误
	Expired               = NewErrorCode("Expired", 40026, "内容已过期")                 // 40026: 内容已过期
	DeveloperOperateError = NewErrorCode("DeveloperOperateError", 40027, "开发者操作错误") // 40027: 开发者操作错误

	// ----- 401 Unauthorized -----
	Unauthorized = NewErrorCode("Unauthorized", 40101, "未授权") // 40101: 未授权
	LoginFailed  = NewErrorCode("LoginFailed", 40102, "登录失败") // 40102: 登录失败

	// ----- 403 Forbidden -----
	Forbidden        = NewErrorCode("Forbidden", 40301, "禁止访问")        // 40301: 禁止访问
	PermissionDenied = NewErrorCode("PermissionDenied", 40302, "权限拒绝") // 40302: 权限拒绝
	AccessLimited    = NewErrorCode("AccessLimited", 40303, "访问受限")    // 40303: 访问受限

	// ----- 404 Not Found -----
	PageNotFound     = NewErrorCode("PageNotFound", 40401, "页面未找到")     // 40401: 页面未找到
	NotFound         = NewErrorCode("NotFound", 40402, "未找到")           // 40402: 未找到
	ResourceNotFound = NewErrorCode("ResourceNotFound", 40403, "资源未找到") // 40403: 资源未找到

	// ----- 405 Method Not Allowed -----
	MethodNotAllowed = NewErrorCode("MethodNotAllowed", 40501, "方法不允许") // 40501: 方法不允许

	// ----- 406 Not Acceptable -----
	NotAcceptable = NewErrorCode("NotAcceptable", 40601, "不可接受") // 40601: 不可接受

	// ----- 408 Request Timeout -----
	Timeout           = NewErrorCode("Timeout", 40801, "请求超时")           // 40801: 请求超时
	ConnectionTimeout = NewErrorCode("ConnectionTimeout", 40802, "连接超时") // 40802: 连接超时
	ReadTimeout       = NewErrorCode("ReadTimeout", 40803, "读取超时")       // 40803: 读取超时
	WriteTimeout      = NewErrorCode("WriteTimeout", 40804, "写入超时")      // 40804: 写入超时

	// ----- 429 Too Many Requests -----
	TooManyRequests    = NewErrorCode("TooManyRequests", 42901, "请求过多")      // 42901: 请求过多
	RequestRateTooHigh = NewErrorCode("RequestRateTooHigh", 42902, "请求频率过高") // 42902: 请求频率过高

	// ============================== 5xx 服务器错误 ==============================
	ServerInternalError = NewErrorCode("ServerInternalError", 50001, "服务器内部错误") // 50001: 服务器内部错误
	ServiceUnavailable  = NewErrorCode("ServiceUnavailable", 50301, "服务不可用")    // 50301: 服务不可用
	GatewayError        = NewErrorCode("GatewayError", 50201, "网关错误")           // 50201: 网关错误
	SystemMaintenance   = NewErrorCode("SystemMaintenance", 50302, "系统维护")      // 50302: 系统维护
	DatabaseError       = NewErrorCode("DatabaseError", 50002, "数据库错误")         // 50002: 数据库错误
	CacheError          = NewErrorCode("CacheError", 50003, "缓存错误")             // 50003: 缓存错误
	FileError           = NewErrorCode("FileError", 50004, "文件错误")              // 50004: 文件错误
	StorageError        = NewErrorCode("StorageError", 50005, "存储错误")           // 50005: 存储错误
	RemoteCallError     = NewErrorCode("RemoteCallError", 50006, "远程调用错误")      // 50006: 远程调用错误
	ConfigurationError  = NewErrorCode("ConfigurationError", 50007, "配置错误")     // 50007: 配置错误
	NetworkError        = NewErrorCode("NetworkError", 50009, "网络错误")           // 50009: 网络错误
	ResourceExhausted   = NewErrorCode("ResourceExhausted", 50008, "资源耗尽")      // 50008: 资源耗尽
	UnknownError        = NewErrorCode("UnknownError", 50999, "未知错误")           // 50999: 未知错误
)
