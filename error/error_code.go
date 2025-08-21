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

	// NotExist 内容不存在
	//
	// 内容不存在, 用于定义内容不存在信息；该错误码为 40000，用于定义内容不存在信息；
	// 用于返回内容不存在信息。
	//
	// 该错误码为 NOT_EXIST；该错误码为 40000； 该错误信息为 内容不存在。
	NotExist = NewErrorCode("NotExist", 40000, "内容不存在")

	// Existed 内容已存在
	//
	// 内容已存在, 用于定义内容已存在信息；
	// 该错误码为 40001，用于定义内容已存在信息；用于返回内容已存在信息。
	//
	// 该错误码为 EXISTED；该错误码为 40001； 该错误信息为 内容已存在。
	Existed = NewErrorCode("Existed", 40001, "内容已存在")

	// ============================== 4xx 客户端错误 ==============================
	// ----- 400 Bad Request -----

	// BadRequest 错误请求
	//
	// 错误请求, 用于定义错误请求信息；
	// 该错误码为 40024，用于定义错误请求信息；用于返回错误请求信息。
	//
	// 该错误码为 BAD_REQUEST；该错误码为 40024； 该错误信息为 错误请求。
	BadRequest = NewErrorCode("BadRequest", 40024, "错误请求")

	// ParameterError 参数错误
	//
	// 参数错误, 用于定义参数错误信息；
	// 该错误码为 40002，用于定义参数错误信息；用于返回参数错误信息。
	//
	// 该错误码为 PARAMETER_ERROR；该错误码为 40002； 该错误信息为 参数错误。
	ParameterError = NewErrorCode("ParameterError", 40002, "参数错误")

	// ParameterMissing 参数缺失
	//
	// 参数缺失, 用于定义参数缺失信息；
	// 该错误码为 40003，用于定义参数缺失信息；用于返回参数缺失信息。
	//
	// 该错误码为 PARAMETER_MISSING；该错误码为 40003； 该错误信息为 参数缺失。
	ParameterMissing = NewErrorCode("ParameterMissing", 40003, "参数缺失")

	// ParameterInvalid 参数无效
	//
	// 参数无效, 用于定义参数无效信息；
	// 该错误码为 40004，用于定义参数无效信息；用于返回参数无效信息。
	//
	// 该错误码为 PARAMETER_INVALID；该错误码为 40004； 该错误信息为 参数无效。
	ParameterInvalid = NewErrorCode("ParameterInvalid", 40004, "参数无效")

	// ParameterIllegal 参数非法
	//
	// 参数非法, 用于定义参数非法信息；
	// 该错误码为 40005，用于定义参数非法信息；用于返回参数非法信息。
	//
	// 该错误码为 PARAMETER_ILLEGAL；该错误码为 40005； 该错误信息为 参数非法。
	ParameterIllegal = NewErrorCode("ParameterIllegal", 40005, "参数非法")

	// ParameterTypeError 参数类型错误
	//
	// 参数类型错误, 用于定义参数类型错误信息；
	// 该错误码为 40006，用于定义参数类型错误信息；用于返回参数类型错误信息。
	//
	// 该错误码为 PARAMETER_TYPE_ERROR；该错误码为 40006； 该错误信息为 参数类型错误。
	ParameterTypeError = NewErrorCode("ParameterTypeError", 40006, "参数类型错误")

	// BodyError 请求体错误
	//
	// 请求体错误, 用于定义请求体错误信息；
	// 该错误码为 40007，用于定义请求体错误信息；用于返回请求体错误信息。
	//
	// 该错误码为 BODY_ERROR；该错误码为 40007； 该错误信息为 请求体错误。
	BodyError = NewErrorCode("BodyError", 40007, "请求体错误")

	// BodyMissing 请求体缺失
	//
	// 请求体缺失, 用于定义请求体缺失信息；
	// 该错误码为 40008，用于定义请求体缺失信息；用于返回请求体缺失信息。
	//
	// 该错误码为 BODY_MISSING；该错误码为 40008； 该错误信息为 请求体缺失。
	BodyMissing = NewErrorCode("BodyMissing", 40008, "请求体缺失")

	// BodyInvalid 请求体无效
	//
	// 请求体无效, 用于定义请求体无效信息；
	// 该错误码为 40009，用于定义请求体无效信息；用于返回请求体无效信息。
	//
	// 该错误码为 BODY_INVALID；该错误码为 40009； 该错误信息为 请求体无效。
	BodyInvalid = NewErrorCode("BodyInvalid", 40009, "请求体无效")

	// BodyIllegal 请求体非法
	//
	// 请求体非法, 用于定义请求体非法信息；
	// 该错误码为 40010，用于定义请求体非法信息；用于返回请求体非法信息。
	//
	// 该错误码为 BODY_ILLEGAL；该错误码为 40010； 该错误信息为 请求体非法。
	BodyIllegal = NewErrorCode("BodyIllegal", 40010, "请求体非法")

	// BodyTypeError 请求体类型错误
	//
	// 请求体类型错误, 用于定义请求体类型错误信息；
	// 该错误码为 40011，用于定义请求体类型错误信息；用于返回请求体类型错误信息。
	//
	// 该错误码为 BODY_TYPE_ERROR；该错误码为 40011； 该错误信息为 请求体类型错误。
	BodyTypeError = NewErrorCode("BodyTypeError", 40011, "请求体类型错误")

	// HeaderError 请求头错误
	//
	// 请求头错误, 用于定义请求头错误信息；
	// 该错误码为 40012，用于定义请求头错误信息；用于返回请求头错误信息。
	//
	// 该错误码为 HEADER_ERROR；该错误码为 40012； 该错误信息为 请求头错误。
	HeaderError = NewErrorCode("HeaderError", 40012, "请求头错误")

	// HeaderMissing 请求头缺失
	//
	// 请求头缺失, 用于定义请求头缺失信息；
	// 该错误码为 40013，用于定义请求头缺失信息；用于返回请求头缺失信息。
	//
	// 该错误码为 HEADER_MISSING；该错误码为 40013； 该错误信息为 请求头缺失。
	HeaderMissing = NewErrorCode("HeaderMissing", 40013, "请求头缺失")

	// HeaderInvalid 请求头无效
	//
	// 请求头无效, 用于定义请求头无效信息；
	// 该错误码为 40014，用于定义请求头无效信息；用于返回请求头无效信息。
	//
	// 该错误码为 HEADER_INVALID；该错误码为 40014； 该错误信息为 请求头无效。
	HeaderInvalid = NewErrorCode("HeaderInvalid", 40014, "请求头无效")

	// HeaderIllegal 请求头非法
	//
	// 请求头非法, 用于定义请求头非法信息；
	// 该错误码为 40015，用于定义请求头非法信息；用于返回请求头非法信息。
	//
	// 该错误码为 HEADER_ILLEGAL；该错误码为 40015； 该错误信息为 请求头非法。
	HeaderIllegal = NewErrorCode("HeaderIllegal", 40015, "请求头非法")

	// HeaderTypeError 请求头类型错误
	//
	// 请求头类型错误, 用于定义请求头类型错误信息；
	// 该错误码为 40016，用于定义请求头类型错误信息；用于返回请求头类型错误信息。
	//
	// 该错误码为 HEADER_TYPE_ERROR；该错误码为 40016； 该错误信息为 请求头类型错误。
	HeaderTypeError = NewErrorCode("HeaderTypeError", 40016, "请求头类型错误")

	// OperationError 操作错误
	//
	// 操作错误, 用于定义操作错误信息；
	// 该错误码为 40017，用于定义操作错误信息；用于返回操作错误信息。
	//
	// 该错误码为 OPERATION_ERROR；该错误码为 40017； 该错误信息为 操作错误。
	OperationError = NewErrorCode("OperationError", 40017, "操作错误")

	// OperationFailed 操作失败
	//
	// 操作失败, 用于定义操作失败信息；
	// 该错误码为 40018，用于定义操作失败信息；用于返回操作失败信息。
	//
	// 该错误码为 OPERATION_FAILED；该错误码为 40018； 该错误信息为 操作失败。
	OperationFailed = NewErrorCode("OperationFailed", 40018, "操作失败")

	// OperationInvalid 操作无效
	//
	// 操作无效, 用于定义操作无效信息；
	// 该错误码为 40019，用于定义操作无效信息；用于返回操作无效信息。
	//
	// 该错误码为 OPERATION_INVALID；该错误码为 40019； 该错误信息为 操作无效。
	OperationInvalid = NewErrorCode("OperationInvalid", 40019, "操作无效")

	// OperationIllegal 操作非法
	//
	// 操作非法, 用于定义操作非法信息；
	// 该错误码为 40020，用于定义操作非法信息；用于返回操作非法信息。
	//
	// 该错误码为 OPERATION_ILLEGAL；该错误码为 40020； 该错误信息为 操作非法。
	OperationIllegal = NewErrorCode("OperationIllegal", 40020, "操作非法")

	// OperationDenied 操作被拒绝
	//
	// 操作被拒绝, 用于定义操作被拒绝信息；
	// 该错误码为 40021，用于定义操作被拒绝信息；用于返回操作被拒绝信息。
	//
	// 该错误码为 OPERATION_DENIED；该错误码为 40021； 该错误信息为 操作被拒绝。
	OperationDenied = NewErrorCode("OperationDenied", 40021, "操作被拒绝")

	// OperationNotAllowed 操作不允许
	//
	// 操作不允许, 用于定义操作不允许信息；
	// 该错误码为 40022，用于定义操作不允许信息；用于返回操作不允许信息。
	//
	// 该错误码为 OPERATION_NOT_ALLOWED；该错误码为 40022； 该错误信息为 操作不允许。
	OperationNotAllowed = NewErrorCode("OperationNotAllowed", 40022, "操作不允许")

	// OperationNotSupported 操作不支持
	//
	// 操作不支持, 用于定义操作不支持信息；
	// 该错误码为 40023，用于定义操作不支持信息；用于返回操作不支持信息。
	//
	// 该错误码为 OPERATION_NOT_SUPPORTED；该错误码为 40023； 该错误信息为 操作不支持。
	OperationNotSupported = NewErrorCode("OperationNotSupported", 40023, "操作不支持")

	// OperationTypeError 操作类型错误
	//
	// 操作类型错误, 用于定义操作类型错误信息；
	// 该错误码为 40025，用于定义操作类型错误信息；用于返回操作类型错误信息。
	//
	// 该错误码为 OPERATION_TYPE_ERROR；该错误码为 40025； 该错误信息为 操作类型错误。
	OperationTypeError = NewErrorCode("OperationTypeError", 40025, "操作类型错误")

	// Expired 已过期
	//
	// 已过期, 用于定义已过期信息；
	// 该错误码为 40026，用于定义已过期信息；用于返回已过期信息。
	//
	// 该错误码为 EXPIRED；该错误码为 40026； 该错误信息为 内容已过期。
	Expired = NewErrorCode("Expired", 40026, "内容已过期")

	// DeveloperOperateError 开发者操作错误
	//
	// 开发者操作错误, 用于定义开发者操作错误信息；
	// 该错误码为 40027，用于定义开发者操作错误信息；用于返回开发者操作错误信息。
	//
	// 该错误码为 DEVELOPER_OPERATE_ERROR；该错误码为 40027； 该错误信息为 开发者操作错误。
	DeveloperOperateError = NewErrorCode("DeveloperOperateError", 40027, "开发者操作错误")

	// ----- 401 Unauthorized -----

	// Unauthorized 未授权
	//
	// 未授权, 用于定义未授权信息；
	// 该错误码为 40101，用于定义未授权信息；用于返回未授权信息。
	//
	// 该错误码为 UNAUTHORIZED；该错误码为 40101； 该错误信息为 未授权。
	Unauthorized = NewErrorCode("Unauthorized", 40101, "未授权")

	// LoginFailed 登录失败
	//
	// 登录失败, 用于定义登录失败信息；
	// 该错误码为 40102，用于定义登录失败信息；用于返回登录失败信息。
	//
	// 该错误码为 LOGIN_FAILED；该错误码为 40102； 该错误信息为 登录失败。
	LoginFailed = NewErrorCode("LoginFailed", 40102, "登录失败")

	// ----- 403 Forbidden -----

	// Forbidden 禁止访问
	//
	// 禁止访问, 用于定义禁止访问信息；
	// 该错误码为 40301，用于定义禁止访问信息；用于返回禁止访问信息。
	//
	// 该错误码为 FORBIDDEN；该错误码为 40301； 该错误信息为 禁止访问。
	Forbidden = NewErrorCode("Forbidden", 40301, "禁止访问")

	// PermissionDenied 权限拒绝
	//
	// 权限拒绝, 用于定义权限拒绝信息；
	// 该错误码为 40302，用于定义权限拒绝信息；用于返回权限拒绝信息。
	//
	// 该错误码为 PERMISSION_DENIED；该错误码为 40302； 该错误信息为 权限拒绝。
	PermissionDenied = NewErrorCode("PermissionDenied", 40302, "权限拒绝")

	// AccessLimited 访问受限
	//
	// 访问受限, 用于定义访问受限信息；
	// 该错误码为 40303，用于定义访问受限信息；用于返回访问受限信息。
	//
	// 该错误码为 ACCESS_LIMITED；该错误码为 40303； 该错误信息为 访问受限。
	AccessLimited = NewErrorCode("AccessLimited", 40303, "访问受限")

	// ----- 404 Not Found -----

	// PageNotFound 页面未找到
	//
	// 页面未找到, 用于定义页面未找到信息；
	// 该错误码为 40401，用于定义页面未找到信息；用于返回页面未找到信息。
	//
	// 该错误码为 PAGE_NOT_FOUND；该错误码为 40401； 该错误信息为 页面未找到。
	PageNotFound = NewErrorCode("PageNotFound", 40401, "页面未找到")

	// NotFound 未找到
	//
	// 未找到, 用于定义未找到信息；
	// 该错误码为 40402，用于定义未找到信息；用于返回未找到信息。
	//
	// 该错误码为 NOT_FOUND；该错误码为 40402； 该错误信息为 未找到。
	NotFound = NewErrorCode("NotFound", 40402, "未找到")

	// ResourceNotFound 资源未找到
	//
	// 资源未找到, 用于定义资源未找到信息；
	// 该错误码为 40403，用于定义资源未找到信息；用于返回资源未找到信息。
	//
	// 该错误码为 RESOURCE_NOT_FOUND；该错误码为 40403； 该错误信息为 资源未找到。
	ResourceNotFound = NewErrorCode("ResourceNotFound", 40403, "资源未找到")

	// ----- 405 Method Not Allowed -----

	// MethodNotAllowed 方法不允许
	//
	// 方法不允许, 用于定义方法不允许信息；
	// 该错误码为 40501，用于定义方法不允许信息；用于返回方法不允许信息。
	//
	// 该错误码为 METHOD_NOT_ALLOWED；该错误码为 40501； 该错误信息为 方法不允许。
	MethodNotAllowed = NewErrorCode("MethodNotAllowed", 40501, "方法不允许")

	// ----- 406 Not Acceptable -----

	// NotAcceptable 不可接受
	//
	// 不可接受, 用于定义不可接受信息；
	// 该错误码为 40601，用于定义不可接受信息；用于返回不可接受信息。
	//
	// 该错误码为 NOT_ACCEPTABLE；该错误码为 40601； 该错误信息为 不可接受。
	NotAcceptable = NewErrorCode("NotAcceptable", 40601, "不可接受")

	// ----- 408 Request Timeout -----

	// Timeout 请求超时
	//
	// 请求超时, 用于定义请求超时信息；
	// 该错误码为 40801，用于定义请求超时信息；用于返回请求超时信息。
	//
	// 该错误码为 TIMEOUT；该错误码为 40801； 该错误信息为 请求超时。
	Timeout = NewErrorCode("Timeout", 40801, "请求超时")

	// ConnectionTimeout 连接超时
	//
	// 连接超时, 用于定义连接超时信息；
	// 该错误码为 40802，用于定义连接超时信息；用于返回连接超时信息。
	//
	// 该错误码为 CONNECTION_TIMEOUT；该错误码为 40802； 该错误信息为 连接超时。
	ConnectionTimeout = NewErrorCode("ConnectionTimeout", 40802, "连接超时")

	// ReadTimeout 读取超时
	//
	// 读取超时, 用于定义读取超时信息；
	// 该错误码为 40803，用于定义读取超时信息；用于返回读取超时信息。
	//
	// 该错误码为 READ_TIMEOUT；该错误码为 40803； 该错误信息为 读取超时。
	ReadTimeout = NewErrorCode("ReadTimeout", 40803, "读取超时")

	// WriteTimeout 写入超时
	//
	// 写入超时, 用于定义写入超时信息；
	// 该错误码为 40804，用于定义写入超时信息；用于返回写入超时信息。
	//
	// 该错误码为 WRITE_TIMEOUT；该错误码为 40804； 该错误信息为 写入超时。
	WriteTimeout = NewErrorCode("WriteTimeout", 40804, "写入超时")

	// ----- 429 Too Many Requests -----

	// TooManyRequests 请求过多
	//
	// 请求过多, 用于定义请求过多信息；
	// 该错误码为 42901，用于定义请求过多信息；用于返回请求过多信息。
	//
	// 该错误码为 TOO_MANY_REQUESTS；该错误码为 42901； 该错误信息为 请求过多。
	TooManyRequests = NewErrorCode("TooManyRequests", 42901, "请求过多")

	// RequestRateTooHigh 请求频率过高
	//
	// 请求频率过高, 用于定义请求频率过高信息；
	// 该错误码为 42902，用于定义请求频率过高信息；用于返回请求频率过高信息。
	//
	// 该错误码为 REQUEST_RATE_TOO_HIGH；该错误码为 42902； 该错误信息为 请求频率过高。
	RequestRateTooHigh = NewErrorCode("RequestRateTooHigh", 42902, "请求频率过高")

	// ============================== 5xx 服务器错误 ==============================

	// ServerInternalError 服务器内部错误
	//
	// 服务器内部错误, 用于定义服务器内部错误信息；
	// 该错误码为 50001，用于定义服务器内部错误信息；用于返回服务器内部错误信息。
	//
	// 该错误码为 SERVER_INTERNAL_ERROR；该错误码为 50001； 该错误信息为 服务器内部错误。
	ServerInternalError = NewErrorCode("ServerInternalError", 50001, "服务器内部错误")

	// ServiceUnavailable 服务不可用
	//
	// 服务不可用, 用于定义服务不可用信息；
	// 该错误码为 50301，用于定义服务不可用信息；用于返回服务不可用信息。
	//
	// 该错误码为 SERVICE_UNAVAILABLE；该错误码为 50301； 该错误信息为 服务不可用。
	ServiceUnavailable = NewErrorCode("ServiceUnavailable", 50301, "服务不可用")

	// GatewayError 网关错误
	//
	// 网关错误, 用于定义网关错误信息；
	// 该错误码为 50201，用于定义网关错误信息；用于返回网关错误信息。
	//
	// 该错误码为 GATEWAY_ERROR；该错误码为 50201； 该错误信息为 网关错误。
	GatewayError = NewErrorCode("GatewayError", 50201, "网关错误")

	// SystemMaintenance 系统维护
	//
	// 系统维护, 用于定义系统维护信息；
	// 该错误码为 50302，用于定义系统维护信息；用于返回系统维护信息。
	//
	// 该错误码为 SYSTEM_MAINTENANCE；该错误码为 50302； 该错误信息为 系统维护。
	SystemMaintenance = NewErrorCode("SystemMaintenance", 50302, "系统维护")

	// DatabaseError 数据库错误
	//
	// 数据库错误, 用于定义数据库错误信息；
	// 该错误码为 50002，用于定义数据库错误信息；用于返回数据库错误信息。
	//
	// 该错误码为 DATABASE_ERROR；该错误码为 50002； 该错误信息为 数据库错误。
	DatabaseError = NewErrorCode("DatabaseError", 50002, "数据库错误")

	// CacheError 缓存错误
	//
	// 缓存错误, 用于定义缓存错误信息；
	// 该错误码为 50003，用于定义缓存错误信息；用于返回缓存错误信息。
	//
	// 该错误码为 CACHE_ERROR；该错误码为 50003； 该错误信息为 缓存错误。
	CacheError = NewErrorCode("CacheError", 50003, "缓存错误")

	// FileError 文件错误
	//
	// 文件错误, 用于定义文件错误信息；
	// 该错误码为 50004，用于定义文件错误信息；用于返回文件错误信息。
	//
	// 该错误码为 FILE_ERROR；该错误码为 50004； 该错误信息为 文件错误。
	FileError = NewErrorCode("FileError", 50004, "文件错误")

	// StorageError 存储错误
	//
	// 存储错误, 用于定义存储错误信息；
	// 该错误码为 50005，用于定义存储错误信息；用于返回存储错误信息。
	//
	// 该错误码为 STORAGE_ERROR；该错误码为 50005； 该错误信息为 存储错误。
	StorageError = NewErrorCode("StorageError", 50005, "存储错误")

	// RemoteCallError 远程调用错误
	//
	// 远程调用错误, 用于定义远程调用错误信息；
	// 该错误码为 50006，用于定义远程调用错误信息；用于返回远程调用错误信息。
	//
	// 该错误码为 REMOTE_CALL_ERROR；该错误码为 50006； 该错误信息为 远程调用错误。
	RemoteCallError = NewErrorCode("RemoteCallError", 50006, "远程调用错误")

	// ConfigurationError 配置错误
	//
	// 配置错误, 用于定义配置错误信息；
	// 该错误码为 50007，用于定义配置错误信息；用于返回配置错误信息。
	//
	// 该错误码为 CONFIGURATION_ERROR；该错误码为 50007； 该错误信息为 配置错误。
	ConfigurationError = NewErrorCode("ConfigurationError", 50007, "配置错误")

	// ResourceExhausted 资源耗尽
	//
	// 资源耗尽, 用于定义资源耗尽信息；
	// 该错误码为 50008，用于定义资源耗尽信息；用于返回资源耗尽信息。
	//
	// 该错误码为 RESOURCE_EXHAUSTED；该错误码为 50008； 该错误信息为 资源耗尽。
	ResourceExhausted = NewErrorCode("ResourceExhausted", 50008, "资源耗尽")

	// UnknownError 未知错误
	//
	// 未知错误, 用于定义未知错误信息；
	// 该错误码为 50999，用于定义未知错误信息；用于返回未知错误信息。
	//
	// 该错误码为 UNKNOWN_ERROR；该错误码为 50999； 该错误信息为 未知错误。
	UnknownError = NewErrorCode("UnknownError", 50999, "未知错误")
)
