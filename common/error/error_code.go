package xError

// ErrorCode 错误信息类型
//
// 用于定义系统中的错误相关信息，包括错误代码及错误信息。
// 提供统一的结构化方式管理和访问错误数据。
type ErrorCode struct {
	Code    uint   // 错误码
	Output  string // 输出标识（大写下划线格式）
	Message string // 错误信息（中文）
}

// GetCode 获取错误码
func (e *ErrorCode) GetCode() uint {
	return e.Code
}

// GetOutput 获取输出标识
func (e *ErrorCode) GetOutput() string {
	return e.Output
}

// GetMessage 获取错误信息
func (e *ErrorCode) GetMessage() string {
	return e.Message
}

// ============================== 通用错误码 (400xx) ==============================

var (
	NotExist = &ErrorCode{40000, "NOT_EXIST", "内容不存在"}
	Existed  = &ErrorCode{40001, "EXISTED", "内容已存在"}
	Expired  = &ErrorCode{40002, "EXPIRED", "内容已过期"}
	Disabled = &ErrorCode{40003, "DISABLED", "内容已禁用"}
	Locked   = &ErrorCode{40004, "LOCKED", "内容已锁定"}
	Pending  = &ErrorCode{40005, "PENDING", "内容待处理"}
	Rejected = &ErrorCode{40006, "REJECTED", "内容已拒绝"}
	Canceled = &ErrorCode{40007, "CANCELED", "内容已取消"}
)

// ============================== 400 Bad Request (400xx) ==============================

var (
	BadRequest     = &ErrorCode{40010, "BAD_REQUEST", "错误请求"}
	ParameterError = &ErrorCode{40011, "PARAMETER_ERROR", "参数错误"}
	ParameterEmpty = &ErrorCode{40012, "PARAMETER_EMPTY", "参数缺失"}
	ParameterType  = &ErrorCode{40013, "PARAMETER_TYPE", "参数类型错误"}
	BodyError      = &ErrorCode{40014, "BODY_ERROR", "请求体错误"}
	BodyEmpty      = &ErrorCode{40015, "BODY_EMPTY", "请求体缺失"}
	BodyType       = &ErrorCode{40016, "BODY_TYPE", "请求体类型错误"}
	HeaderError    = &ErrorCode{40017, "HEADER_ERROR", "请求头错误"}
	HeaderEmpty    = &ErrorCode{40018, "HEADER_EMPTY", "请求头缺失"}
	HeaderType     = &ErrorCode{40019, "HEADER_TYPE", "请求头类型错误"}
)

// ============================== 操作错误 (400xx) ==============================

var (
	OperationError   = &ErrorCode{40020, "OPERATION_ERROR", "操作错误"}
	OperationFailed  = &ErrorCode{40021, "OPERATION_FAILED", "操作失败"}
	OperationDenied  = &ErrorCode{40022, "OPERATION_DENIED", "操作被拒绝"}
	OperationInvalid = &ErrorCode{40023, "OPERATION_INVALID", "操作无效"}
	DeveloperError   = &ErrorCode{40024, "DEVELOPER_ERROR", "开发者操作错误"}
	RepeatOperation  = &ErrorCode{40025, "REPEAT_OPERATION", "重复操作"}
	UnsupportedOp    = &ErrorCode{40026, "UNSUPPORTED_OPERATION", "不支持的操作"}
)

// ============================== 验证错误 (400xx) ==============================

var (
	ValidationError = &ErrorCode{40030, "VALIDATION_ERROR", "验证错误"}
	FormatError     = &ErrorCode{40031, "FORMAT_ERROR", "格式错误"}
	LengthError     = &ErrorCode{40032, "LENGTH_ERROR", "长度错误"}
	RangeError      = &ErrorCode{40033, "RANGE_ERROR", "范围错误"}
	PatternError    = &ErrorCode{40034, "PATTERN_ERROR", "格式不匹配"}
	TypeMismatch    = &ErrorCode{40035, "TYPE_MISMATCH", "类型不匹配"}
	InvalidValue    = &ErrorCode{40036, "INVALID_VALUE", "无效值"}
	InvalidFormat   = &ErrorCode{40037, "INVALID_FORMAT", "无效格式"}
	InvalidState    = &ErrorCode{40038, "INVALID_STATE", "无效状态"}
	InvalidInput    = &ErrorCode{40039, "INVALID_INPUT", "无效输入"}
)

// ============================== 数据错误 (400xx) ==============================

var (
	DataError      = &ErrorCode{40040, "DATA_ERROR", "数据错误"}
	DataInvalid    = &ErrorCode{40041, "DATA_INVALID", "数据无效"}
	DataConflict   = &ErrorCode{40042, "DATA_CONFLICT", "数据冲突"}
	DataDuplicate  = &ErrorCode{40043, "DATA_DUPLICATE", "数据重复"}
	DataNotMatch   = &ErrorCode{40044, "DATA_NOT_MATCH", "数据不匹配"}
	DataIncomplete = &ErrorCode{40045, "DATA_INCOMPLETE", "数据不完整"}
	DataCorrupted  = &ErrorCode{40046, "DATA_CORRUPTED", "数据损坏"}
	DataOutOfRange = &ErrorCode{40047, "DATA_OUT_OF_RANGE", "数据超出范围"}
	DataTooLarge   = &ErrorCode{40048, "DATA_TOO_LARGE", "数据过大"}
	DataTooSmall   = &ErrorCode{40049, "DATA_TOO_SMALL", "数据过小"}
)

// ============================== 文件错误 (400xx) ==============================

var (
	FileUploadError    = &ErrorCode{40050, "FILE_UPLOAD_ERROR", "文件上传错误"}
	FileDownloadError  = &ErrorCode{40051, "FILE_DOWNLOAD_ERROR", "文件下载错误"}
	FileSizeExceeded   = &ErrorCode{40052, "FILE_SIZE_EXCEEDED", "文件大小超限"}
	FileTypeNotAllowed = &ErrorCode{40053, "FILE_TYPE_NOT_ALLOWED", "文件类型不允许"}
	FileNotFound       = &ErrorCode{40054, "FILE_NOT_FOUND", "文件未找到"}
	FileReadError      = &ErrorCode{40055, "FILE_READ_ERROR", "文件读取错误"}
	FileWriteError     = &ErrorCode{40056, "FILE_WRITE_ERROR", "文件写入错误"}
	FileDeleteError    = &ErrorCode{40057, "FILE_DELETE_ERROR", "文件删除错误"}
	FileFormatError    = &ErrorCode{40058, "FILE_FORMAT_ERROR", "文件格式错误"}
	FileEmpty          = &ErrorCode{40059, "FILE_EMPTY", "文件为空"}
)

// ============================== 业务错误 (400xx) ==============================

var (
	BusinessError     = &ErrorCode{40060, "BUSINESS_ERROR", "业务错误"}
	TransactionFailed = &ErrorCode{40061, "TRANSACTION_FAILED", "事务失败"}
	StateError        = &ErrorCode{40062, "STATE_ERROR", "状态错误"}
	FlowError         = &ErrorCode{40063, "FLOW_ERROR", "流程错误"}
	RuleViolation     = &ErrorCode{40064, "RULE_VIOLATION", "规则违反"}
	QuotaExceeded     = &ErrorCode{40065, "QUOTA_EXCEEDED", "配额超限"}
	LimitExceeded     = &ErrorCode{40066, "LIMIT_EXCEEDED", "限制超出"}
	BalanceInsuff     = &ErrorCode{40067, "BALANCE_INSUFFICIENT", "余额不足"}
	StockInsuff       = &ErrorCode{40068, "STOCK_INSUFFICIENT", "库存不足"}
	ConditionNotMet   = &ErrorCode{40069, "CONDITION_NOT_MET", "条件不满足"}
)

// ============================== 401 Unauthorized (401xx) ==============================

var (
	Unauthorized     = &ErrorCode{40100, "UNAUTHORIZED", "未授权"}
	LoginFailed      = &ErrorCode{40101, "LOGIN_FAILED", "登录失败"}
	TokenInvalid     = &ErrorCode{40102, "TOKEN_INVALID", "令牌无效"}
	TokenExpired     = &ErrorCode{40103, "TOKEN_EXPIRED", "令牌过期"}
	TokenMissing     = &ErrorCode{40104, "TOKEN_MISSING", "令牌缺失"}
	SessionExpired   = &ErrorCode{40105, "SESSION_EXPIRED", "会话过期"}
	SessionInvalid   = &ErrorCode{40106, "SESSION_INVALID", "会话无效"}
	CredentialError  = &ErrorCode{40107, "CREDENTIAL_ERROR", "凭证错误"}
	SignatureInvalid = &ErrorCode{40108, "SIGNATURE_INVALID", "签名无效"}
	SignatureExpired = &ErrorCode{40109, "SIGNATURE_EXPIRED", "签名过期"}
)

// ============================== 用户错误 (401xx) ==============================

var (
	UserNotFound    = &ErrorCode{40110, "USER_NOT_FOUND", "用户不存在"}
	UserDisabled    = &ErrorCode{40111, "USER_DISABLED", "用户已禁用"}
	UserLocked      = &ErrorCode{40112, "USER_LOCKED", "用户已锁定"}
	UserExpired     = &ErrorCode{40113, "USER_EXPIRED", "用户已过期"}
	PasswordError   = &ErrorCode{40114, "PASSWORD_ERROR", "密码错误"}
	PasswordExpired = &ErrorCode{40115, "PASSWORD_EXPIRED", "密码已过期"}
	PasswordWeak    = &ErrorCode{40116, "PASSWORD_WEAK", "密码强度不足"}
	PasswordSame    = &ErrorCode{40117, "PASSWORD_SAME", "新旧密码相同"}
	AccountNotExist = &ErrorCode{40118, "ACCOUNT_NOT_EXIST", "账户不存在"}
	AccountFrozen   = &ErrorCode{40119, "ACCOUNT_FROZEN", "账户已冻结"}
)

// ============================== 验证码错误 (401xx) ==============================

var (
	CaptchaError   = &ErrorCode{40120, "CAPTCHA_ERROR", "验证码错误"}
	CaptchaExpired = &ErrorCode{40121, "CAPTCHA_EXPIRED", "验证码过期"}
	CaptchaInvalid = &ErrorCode{40122, "CAPTCHA_INVALID", "验证码无效"}
	CaptchaMissing = &ErrorCode{40123, "CAPTCHA_MISSING", "验证码缺失"}
	CaptchaFreq    = &ErrorCode{40124, "CAPTCHA_FREQUENCY", "验证码发送频繁"}
	SmsCodeError   = &ErrorCode{40125, "SMS_CODE_ERROR", "短信验证码错误"}
	EmailCodeError = &ErrorCode{40126, "EMAIL_CODE_ERROR", "邮箱验证码错误"}
)

// ============================== 403 Forbidden (403xx) ==============================

var (
	Forbidden        = &ErrorCode{40300, "FORBIDDEN", "禁止访问"}
	PermissionDenied = &ErrorCode{40301, "PERMISSION_DENIED", "权限不足"}
	AccessLimited    = &ErrorCode{40302, "ACCESS_LIMITED", "访问受限"}
	RoleNotFound     = &ErrorCode{40303, "ROLE_NOT_FOUND", "角色不存在"}
	RoleDenied       = &ErrorCode{40304, "ROLE_DENIED", "角色被拒绝"}
	ResourceDenied   = &ErrorCode{40305, "RESOURCE_DENIED", "资源访问被拒绝"}
	IpBlocked        = &ErrorCode{40306, "IP_BLOCKED", "IP已被封禁"}
	RegionBlocked    = &ErrorCode{40307, "REGION_BLOCKED", "地区访问受限"}
	DeviceBlocked    = &ErrorCode{40308, "DEVICE_BLOCKED", "设备已被封禁"}
	ActionForbidden  = &ErrorCode{40309, "ACTION_FORBIDDEN", "操作被禁止"}
)

// ============================== 404 Not Found (404xx) ==============================

var (
	NotFound         = &ErrorCode{40400, "NOT_FOUND", "未找到"}
	PageNotFound     = &ErrorCode{40401, "PAGE_NOT_FOUND", "页面未找到"}
	ResourceNotFound = &ErrorCode{40402, "RESOURCE_NOT_FOUND", "资源未找到"}
	ApiNotFound      = &ErrorCode{40403, "API_NOT_FOUND", "接口未找到"}
	RouteNotFound    = &ErrorCode{40404, "ROUTE_NOT_FOUND", "路由未找到"}
	ServiceNotFound  = &ErrorCode{40405, "SERVICE_NOT_FOUND", "服务未找到"}
	RecordNotFound   = &ErrorCode{40406, "RECORD_NOT_FOUND", "记录未找到"}
	ConfigNotFound   = &ErrorCode{40407, "CONFIG_NOT_FOUND", "配置未找到"}
)

// ============================== 405 Method Not Allowed (405xx) ==============================

var (
	MethodNotAllowed = &ErrorCode{40500, "METHOD_NOT_ALLOWED", "方法不允许"}
)

// ============================== 406 Not Acceptable (406xx) ==============================

var (
	NotAcceptable     = &ErrorCode{40600, "NOT_ACCEPTABLE", "不可接受"}
	ContentTypeError  = &ErrorCode{40601, "CONTENT_TYPE_ERROR", "内容类型错误"}
	AcceptHeaderError = &ErrorCode{40602, "ACCEPT_HEADER_ERROR", "Accept头错误"}
)

// ============================== 408 Request Timeout (408xx) ==============================

var (
	Timeout        = &ErrorCode{40800, "TIMEOUT", "请求超时"}
	ConnectTimeout = &ErrorCode{40801, "CONNECT_TIMEOUT", "连接超时"}
	ReadTimeout    = &ErrorCode{40802, "READ_TIMEOUT", "读取超时"}
	WriteTimeout   = &ErrorCode{40803, "WRITE_TIMEOUT", "写入超时"}
	ExecuteTimeout = &ErrorCode{40804, "EXECUTE_TIMEOUT", "执行超时"}
	IdleTimeout    = &ErrorCode{40805, "IDLE_TIMEOUT", "空闲超时"}
)

// ============================== 409 Conflict (409xx) ==============================

var (
	Conflict         = &ErrorCode{40900, "CONFLICT", "冲突"}
	VersionConflict  = &ErrorCode{40901, "VERSION_CONFLICT", "版本冲突"}
	ConcurrencyError = &ErrorCode{40902, "CONCURRENCY_ERROR", "并发错误"}
	LockConflict     = &ErrorCode{40903, "LOCK_CONFLICT", "锁冲突"}
	ResourceConflict = &ErrorCode{40904, "RESOURCE_CONFLICT", "资源冲突"}
	OptimisticLock   = &ErrorCode{40905, "OPTIMISTIC_LOCK", "乐观锁冲突"}
	DuplicateEntry   = &ErrorCode{40906, "DUPLICATE_ENTRY", "重复条目"}
	UniqueConstraint = &ErrorCode{40907, "UNIQUE_CONSTRAINT", "唯一约束冲突"}
	ForeignKeyError  = &ErrorCode{40908, "FOREIGN_KEY_ERROR", "外键约束错误"}
	IntegrityError   = &ErrorCode{40909, "INTEGRITY_ERROR", "完整性约束错误"}
)

// ============================== 410 Gone (410xx) ==============================

var (
	Gone           = &ErrorCode{41000, "GONE", "资源已删除"}
	ResourceGone   = &ErrorCode{41001, "RESOURCE_GONE", "资源已不存在"}
	DeprecatedApi  = &ErrorCode{41002, "DEPRECATED_API", "接口已废弃"}
	VersionExpired = &ErrorCode{41003, "VERSION_EXPIRED", "版本已过期"}
)

// ============================== 413 Payload Too Large (413xx) ==============================

var (
	PayloadTooLarge = &ErrorCode{41300, "PAYLOAD_TOO_LARGE", "请求体过大"}
	RequestTooLarge = &ErrorCode{41301, "REQUEST_TOO_LARGE", "请求过大"}
	UploadTooLarge  = &ErrorCode{41302, "UPLOAD_TOO_LARGE", "上传内容过大"}
)

// ============================== 415 Unsupported Media Type (415xx) ==============================

var (
	UnsupportedMedia = &ErrorCode{41500, "UNSUPPORTED_MEDIA", "不支持的媒体类型"}
	UnsupportedType  = &ErrorCode{41501, "UNSUPPORTED_TYPE", "不支持的类型"}
)

// ============================== 422 Unprocessable Entity (422xx) ==============================

var (
	UnprocessableEntity = &ErrorCode{42200, "UNPROCESSABLE_ENTITY", "无法处理的实体"}
	SemanticError       = &ErrorCode{42201, "SEMANTIC_ERROR", "语义错误"}
	LogicError          = &ErrorCode{42202, "LOGIC_ERROR", "逻辑错误"}
)

// ============================== 429 Too Many Requests (429xx) ==============================

var (
	TooManyRequests  = &ErrorCode{42900, "TOO_MANY_REQUESTS", "请求过多"}
	RateLimited      = &ErrorCode{42901, "RATE_LIMITED", "请求频率过高"}
	ThrottleExceeded = &ErrorCode{42902, "THROTTLE_EXCEEDED", "限流阈值超出"}
	ConcurrentLimit  = &ErrorCode{42903, "CONCURRENT_LIMIT", "并发数超限"}
	DailyLimitExceed = &ErrorCode{42904, "DAILY_LIMIT_EXCEEDED", "日请求量超限"}
)

// ============================== 500 Internal Server Error (500xx) ==============================

var (
	ServerInternalError = &ErrorCode{50000, "SERVER_INTERNAL_ERROR", "服务器内部错误"}
	DatabaseError       = &ErrorCode{50001, "DATABASE_ERROR", "数据库错误"}
	CacheError          = &ErrorCode{50002, "CACHE_ERROR", "缓存错误"}
	FileError           = &ErrorCode{50003, "FILE_ERROR", "文件错误"}
	StorageError        = &ErrorCode{50004, "STORAGE_ERROR", "存储错误"}
	RemoteError         = &ErrorCode{50005, "REMOTE_ERROR", "远程调用错误"}
	ConfigError         = &ErrorCode{50006, "CONFIG_ERROR", "配置错误"}
	NetworkError        = &ErrorCode{50007, "NETWORK_ERROR", "网络错误"}
	EncryptError        = &ErrorCode{50008, "ENCRYPT_ERROR", "加密错误"}
	DecryptError        = &ErrorCode{50009, "DECRYPT_ERROR", "解密错误"}
	SerializeError      = &ErrorCode{50010, "SERIALIZE_ERROR", "序列化错误"}
	DeserializeErr      = &ErrorCode{50011, "DESERIALIZE_ERROR", "反序列化错误"}
	JsonError           = &ErrorCode{50012, "JSON_ERROR", "JSON处理错误"}
	XmlError            = &ErrorCode{50013, "XML_ERROR", "XML处理错误"}
	IoError             = &ErrorCode{50014, "IO_ERROR", "IO错误"}
	MemoryError         = &ErrorCode{50015, "MEMORY_ERROR", "内存错误"}
	ThreadError         = &ErrorCode{50016, "THREAD_ERROR", "线程错误"}
	PoolExhausted       = &ErrorCode{50017, "POOL_EXHAUSTED", "连接池耗尽"}
	QueueFull           = &ErrorCode{50018, "QUEUE_FULL", "队列已满"}
	UnknownError        = &ErrorCode{50099, "UNKNOWN_ERROR", "未知错误"}
)

// ============================== 第三方服务错误 (500xx) ==============================

var (
	ThirdPartyError = &ErrorCode{50020, "THIRD_PARTY_ERROR", "第三方服务错误"}
	ApiCallFailed   = &ErrorCode{50021, "API_CALL_FAILED", "API调用失败"}
	CallbackError   = &ErrorCode{50022, "CALLBACK_ERROR", "回调错误"}
	WebhookError    = &ErrorCode{50023, "WEBHOOK_ERROR", "Webhook错误"}
	SmsError        = &ErrorCode{50024, "SMS_ERROR", "短信服务错误"}
	EmailError      = &ErrorCode{50025, "EMAIL_ERROR", "邮件服务错误"}
	PaymentError    = &ErrorCode{50026, "PAYMENT_ERROR", "支付服务错误"}
	OssError        = &ErrorCode{50027, "OSS_ERROR", "对象存储错误"}
	CdnError        = &ErrorCode{50028, "CDN_ERROR", "CDN服务错误"}
	PushError       = &ErrorCode{50029, "PUSH_ERROR", "推送服务错误"}
)

// ============================== 502 Bad Gateway (502xx) ==============================

var (
	GatewayError     = &ErrorCode{50200, "GATEWAY_ERROR", "网关错误"}
	UpstreamError    = &ErrorCode{50201, "UPSTREAM_ERROR", "上游服务错误"}
	ProxyError       = &ErrorCode{50202, "PROXY_ERROR", "代理错误"}
	LoadBalanceError = &ErrorCode{50203, "LOAD_BALANCE_ERROR", "负载均衡错误"}
)

// ============================== 503 Service Unavailable (503xx) ==============================

var (
	ServiceUnavailable = &ErrorCode{50300, "SERVICE_UNAVAILABLE", "服务不可用"}
	SystemMaintenance  = &ErrorCode{50301, "SYSTEM_MAINTENANCE", "系统维护中"}
	ResourceExhausted  = &ErrorCode{50302, "RESOURCE_EXHAUSTED", "资源耗尽"}
	ServiceOverload    = &ErrorCode{50303, "SERVICE_OVERLOAD", "服务过载"}
	CircuitBreaker     = &ErrorCode{50304, "CIRCUIT_BREAKER", "熔断保护"}
	ServiceDegraded    = &ErrorCode{50305, "SERVICE_DEGRADED", "服务降级"}
)

// ============================== 504 Gateway Timeout (504xx) ==============================

var (
	GatewayTimeout  = &ErrorCode{50400, "GATEWAY_TIMEOUT", "网关超时"}
	UpstreamTimeout = &ErrorCode{50401, "UPSTREAM_TIMEOUT", "上游服务超时"}
)
