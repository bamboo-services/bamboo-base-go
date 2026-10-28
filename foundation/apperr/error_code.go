package apperr

// ============================== 通用错误码 (400xx) ==============================

// 通用错误码定义。
var (
	NotExist = &ErrorCode{40000, "NOT_EXIST", "内容不存在"} // 内容不存在错误码 (40000)。
	Existed  = &ErrorCode{40001, "EXISTED", "内容已存在"}   // 内容已存在错误码 (40001)。
	Expired  = &ErrorCode{40002, "EXPIRED", "内容已过期"}   // 内容已过期错误码 (40002)。
	Disabled = &ErrorCode{40003, "DISABLED", "内容已禁用"}  // 内容已禁用错误码 (40003)。
	Locked   = &ErrorCode{40004, "LOCKED", "内容已锁定"}    // 内容已锁定错误码 (40004)。
	Pending  = &ErrorCode{40005, "PENDING", "内容待处理"}   // 内容待处理错误码 (40005)。
	Rejected = &ErrorCode{40006, "REJECTED", "内容已拒绝"}  // 内容已拒绝错误码 (40006)。
	Canceled = &ErrorCode{40007, "CANCELED", "内容已取消"}  // 内容已取消错误码 (40007)。
)

// ============================== 400 Bad Request (400xx) ==============================

// 请求错误码定义。
var (
	BadRequest     = &ErrorCode{40010, "BAD_REQUEST", "错误请求"}      // 错误请求错误码 (40010)。
	ParameterError = &ErrorCode{40011, "PARAMETER_ERROR", "参数错误"}  // 参数错误错误码 (40011)。
	ParameterEmpty = &ErrorCode{40012, "PARAMETER_EMPTY", "参数缺失"}  // 参数缺失错误码 (40012)。
	ParameterType  = &ErrorCode{40013, "PARAMETER_TYPE", "参数类型错误"} // 参数类型错误错误码 (40013)。
	BodyError      = &ErrorCode{40014, "BODY_ERROR", "请求体错误"}      // 请求体错误错误码 (40014)。
	BodyEmpty      = &ErrorCode{40015, "BODY_EMPTY", "请求体缺失"}      // 请求体缺失错误码 (40015)。
	BodyType       = &ErrorCode{40016, "BODY_TYPE", "请求体类型错误"}     // 请求体类型错误错误码 (40016)。
	HeaderError    = &ErrorCode{40017, "HEADER_ERROR", "请求头错误"}    // 请求头错误错误码 (40017)。
	HeaderEmpty    = &ErrorCode{40018, "HEADER_EMPTY", "请求头缺失"}    // 请求头缺失错误码 (40018)。
	HeaderType     = &ErrorCode{40019, "HEADER_TYPE", "请求头类型错误"}   // 请求头类型错误错误码 (40019)。
)

// ============================== 操作错误 (400xx) ==============================

// 操作错误码定义。
var (
	OperationError   = &ErrorCode{40020, "OPERATION_ERROR", "操作错误"}         // 操作错误错误码 (40020)。
	OperationFailed  = &ErrorCode{40021, "OPERATION_FAILED", "操作失败"}        // 操作失败错误码 (40021)。
	OperationDenied  = &ErrorCode{40022, "OPERATION_DENIED", "操作被拒绝"}       // 操作被拒绝错误码 (40022)。
	OperationInvalid = &ErrorCode{40023, "OPERATION_INVALID", "操作无效"}       // 操作无效错误码 (40023)。
	DeveloperError   = &ErrorCode{40024, "DEVELOPER_ERROR", "开发者操作错误"}      // 开发者操作错误错误码 (40024)。
	RepeatOperation  = &ErrorCode{40025, "REPEAT_OPERATION", "重复操作"}        // 重复操作错误码 (40025)。
	UnsupportedOp    = &ErrorCode{40026, "UNSUPPORTED_OPERATION", "不支持的操作"} // 不支持的操作错误码 (40026)。
)

// ============================== 验证错误 (400xx) ==============================

// 验证错误码定义。
var (
	ValidationError = &ErrorCode{40030, "VALIDATION_ERROR", "验证错误"} // 验证错误错误码 (40030)。
	FormatError     = &ErrorCode{40031, "FORMAT_ERROR", "格式错误"}     // 格式错误错误码 (40031)。
	LengthError     = &ErrorCode{40032, "LENGTH_ERROR", "长度错误"}     // 长度错误错误码 (40032)。
	RangeError      = &ErrorCode{40033, "RANGE_ERROR", "范围错误"}      // 范围错误错误码 (40033)。
	PatternError    = &ErrorCode{40034, "PATTERN_ERROR", "格式不匹配"}   // 格式不匹配错误码 (40034)。
	TypeMismatch    = &ErrorCode{40035, "TYPE_MISMATCH", "类型不匹配"}   // 类型不匹配错误码 (40035)。
	InvalidValue    = &ErrorCode{40036, "INVALID_VALUE", "无效值"}     // 无效值错误码 (40036)。
	InvalidFormat   = &ErrorCode{40037, "INVALID_FORMAT", "无效格式"}   // 无效格式错误码 (40037)。
	InvalidState    = &ErrorCode{40038, "INVALID_STATE", "无效状态"}    // 无效状态错误码 (40038)。
	InvalidInput    = &ErrorCode{40039, "INVALID_INPUT", "无效输入"}    // 无效输入错误码 (40039)。
)

// ============================== 数据错误 (400xx) ==============================

// 数据错误码定义。
var (
	DataError      = &ErrorCode{40040, "DATA_ERROR", "数据错误"}          // 数据错误错误码 (40040)。
	DataInvalid    = &ErrorCode{40041, "DATA_INVALID", "数据无效"}        // 数据无效错误码 (40041)。
	DataConflict   = &ErrorCode{40042, "DATA_CONFLICT", "数据冲突"}       // 数据冲突错误码 (40042)。
	DataDuplicate  = &ErrorCode{40043, "DATA_DUPLICATE", "数据重复"}      // 数据重复错误码 (40043)。
	DataNotMatch   = &ErrorCode{40044, "DATA_NOT_MATCH", "数据不匹配"}     // 数据不匹配错误码 (40044)。
	DataIncomplete = &ErrorCode{40045, "DATA_INCOMPLETE", "数据不完整"}    // 数据不完整错误码 (40045)。
	DataCorrupted  = &ErrorCode{40046, "DATA_CORRUPTED", "数据损坏"}      // 数据损坏错误码 (40046)。
	DataOutOfRange = &ErrorCode{40047, "DATA_OUT_OF_RANGE", "数据超出范围"} // 数据超出范围错误码 (40047)。
	DataTooLarge   = &ErrorCode{40048, "DATA_TOO_LARGE", "数据过大"}      // 数据过大错误码 (40048)。
	DataTooSmall   = &ErrorCode{40049, "DATA_TOO_SMALL", "数据过小"}      // 数据过小错误码 (40049)。
)

// ============================== 文件错误 (400xx) ==============================

// 文件错误码定义。
var (
	FileUploadError    = &ErrorCode{40050, "FILE_UPLOAD_ERROR", "文件上传错误"}      // 文件上传错误错误码 (40050)。
	FileDownloadError  = &ErrorCode{40051, "FILE_DOWNLOAD_ERROR", "文件下载错误"}    // 文件下载错误错误码 (40051)。
	FileSizeExceeded   = &ErrorCode{40052, "FILE_SIZE_EXCEEDED", "文件大小超限"}     // 文件大小超限错误码 (40052)。
	FileTypeNotAllowed = &ErrorCode{40053, "FILE_TYPE_NOT_ALLOWED", "文件类型不允许"} // 文件类型不允许错误码 (40053)。
	FileNotFound       = &ErrorCode{40054, "FILE_NOT_FOUND", "文件未找到"}          // 文件未找到错误码 (40054)。
	FileReadError      = &ErrorCode{40055, "FILE_READ_ERROR", "文件读取错误"}        // 文件读取错误错误码 (40055)。
	FileWriteError     = &ErrorCode{40056, "FILE_WRITE_ERROR", "文件写入错误"}       // 文件写入错误错误码 (40056)。
	FileDeleteError    = &ErrorCode{40057, "FILE_DELETE_ERROR", "文件删除错误"}      // 文件删除错误错误码 (40057)。
	FileFormatError    = &ErrorCode{40058, "FILE_FORMAT_ERROR", "文件格式错误"}      // 文件格式错误错误码 (40058)。
	FileEmpty          = &ErrorCode{40059, "FILE_EMPTY", "文件为空"}               // 文件为空错误码 (40059)。
)

// ============================== 业务错误 (400xx) ==============================

// 业务错误码定义。
var (
	BusinessError     = &ErrorCode{40060, "BUSINESS_ERROR", "业务错误"}       // 业务错误错误码 (40060)。
	TransactionFailed = &ErrorCode{40061, "TRANSACTION_FAILED", "事务失败"}   // 事务失败错误码 (40061)。
	StateError        = &ErrorCode{40062, "STATE_ERROR", "状态错误"}          // 状态错误错误码 (40062)。
	FlowError         = &ErrorCode{40063, "FLOW_ERROR", "流程错误"}           // 流程错误错误码 (40063)。
	RuleViolation     = &ErrorCode{40064, "RULE_VIOLATION", "规则违反"}       // 规则违反错误码 (40064)。
	QuotaExceeded     = &ErrorCode{40065, "QUOTA_EXCEEDED", "配额超限"}       // 配额超限错误码 (40065)。
	LimitExceeded     = &ErrorCode{40066, "LIMIT_EXCEEDED", "限制超出"}       // 限制超出错误码 (40066)。
	BalanceInsuff     = &ErrorCode{40067, "BALANCE_INSUFFICIENT", "余额不足"} // 余额不足错误码 (40067)。
	StockInsuff       = &ErrorCode{40068, "STOCK_INSUFFICIENT", "库存不足"}   // 库存不足错误码 (40068)。
	ConditionNotMet   = &ErrorCode{40069, "CONDITION_NOT_MET", "条件不满足"}   // 条件不满足错误码 (40069)。
)

// ============================== 401 Unauthorized (401xx) ==============================

// 授权与认证错误码定义。
var (
	Unauthorized     = &ErrorCode{40100, "UNAUTHORIZED", "未授权"}       // 未授权错误码 (40100)。
	LoginFailed      = &ErrorCode{40101, "LOGIN_FAILED", "登录失败"}      // 登录失败错误码 (40101)。
	TokenInvalid     = &ErrorCode{40102, "TOKEN_INVALID", "令牌无效"}     // 令牌无效错误码 (40102)。
	TokenExpired     = &ErrorCode{40103, "TOKEN_EXPIRED", "令牌过期"}     // 令牌过期错误码 (40103)。
	TokenMissing     = &ErrorCode{40104, "TOKEN_MISSING", "令牌缺失"}     // 令牌缺失错误码 (40104)。
	SessionExpired   = &ErrorCode{40105, "SESSION_EXPIRED", "会话过期"}   // 会话过期错误码 (40105)。
	SessionInvalid   = &ErrorCode{40106, "SESSION_INVALID", "会话无效"}   // 会话无效错误码 (40106)。
	CredentialError  = &ErrorCode{40107, "CREDENTIAL_ERROR", "凭证错误"}  // 凭证错误错误码 (40107)。
	SignatureInvalid = &ErrorCode{40108, "SIGNATURE_INVALID", "签名无效"} // 签名无效错误码 (40108)。
	SignatureExpired = &ErrorCode{40109, "SIGNATURE_EXPIRED", "签名过期"} // 签名过期错误码 (40109)。
)

// ============================== 用户错误 (401xx) ==============================

// 用户与账户错误码定义。
var (
	UserNotFound    = &ErrorCode{40110, "USER_NOT_FOUND", "用户不存在"}    // 用户不存在错误码 (40110)。
	UserDisabled    = &ErrorCode{40111, "USER_DISABLED", "用户已禁用"}     // 用户已禁用错误码 (40111)。
	UserLocked      = &ErrorCode{40112, "USER_LOCKED", "用户已锁定"}       // 用户已锁定错误码 (40112)。
	UserExpired     = &ErrorCode{40113, "USER_EXPIRED", "用户已过期"}      // 用户已过期错误码 (40113)。
	PasswordError   = &ErrorCode{40114, "PASSWORD_ERROR", "密码错误"}     // 密码错误错误码 (40114)。
	PasswordExpired = &ErrorCode{40115, "PASSWORD_EXPIRED", "密码已过期"}  // 密码已过期错误码 (40115)。
	PasswordWeak    = &ErrorCode{40116, "PASSWORD_WEAK", "密码强度不足"}    // 密码强度不足错误码 (40116)。
	PasswordSame    = &ErrorCode{40117, "PASSWORD_SAME", "新旧密码相同"}    // 新旧密码相同错误码 (40117)。
	AccountNotExist = &ErrorCode{40118, "ACCOUNT_NOT_EXIST", "账户不存在"} // 账户不存在错误码 (40118)。
	AccountFrozen   = &ErrorCode{40119, "ACCOUNT_FROZEN", "账户已冻结"}    // 账户已冻结错误码 (40119)。
)

// ============================== 验证码错误 (401xx) ==============================

// 验证码错误码定义。
var (
	CaptchaError   = &ErrorCode{40120, "CAPTCHA_ERROR", "验证码错误"}       // 验证码错误错误码 (40120)。
	CaptchaExpired = &ErrorCode{40121, "CAPTCHA_EXPIRED", "验证码过期"}     // 验证码过期错误码 (40121)。
	CaptchaInvalid = &ErrorCode{40122, "CAPTCHA_INVALID", "验证码无效"}     // 验证码无效错误码 (40122)。
	CaptchaMissing = &ErrorCode{40123, "CAPTCHA_MISSING", "验证码缺失"}     // 验证码缺失错误码 (40123)。
	CaptchaFreq    = &ErrorCode{40124, "CAPTCHA_FREQUENCY", "验证码发送频繁"} // 验证码发送频繁错误码 (40124)。
	SmsCodeError   = &ErrorCode{40125, "SMS_CODE_ERROR", "短信验证码错误"}    // 短信验证码错误错误码 (40125)。
	EmailCodeError = &ErrorCode{40126, "EMAIL_CODE_ERROR", "邮箱验证码错误"}  // 邮箱验证码错误错误码 (40126)。
)

// ============================== 403 Forbidden (403xx) ==============================

// 权限与访问控制错误码定义。
var (
	Forbidden        = &ErrorCode{40300, "FORBIDDEN", "禁止访问"}          // 禁止访问错误码 (40300)。
	PermissionDenied = &ErrorCode{40301, "PERMISSION_DENIED", "权限不足"}  // 权限不足错误码 (40301)。
	AccessLimited    = &ErrorCode{40302, "ACCESS_LIMITED", "访问受限"}     // 访问受限错误码 (40302)。
	RoleNotFound     = &ErrorCode{40303, "ROLE_NOT_FOUND", "角色不存在"}    // 角色不存在错误码 (40303)。
	RoleDenied       = &ErrorCode{40304, "ROLE_DENIED", "角色被拒绝"}       // 角色被拒绝错误码 (40304)。
	ResourceDenied   = &ErrorCode{40305, "RESOURCE_DENIED", "资源访问被拒绝"} // 资源访问被拒绝错误码 (40305)。
	IPBlocked        = &ErrorCode{40306, "IP_BLOCKED", "IP已被封禁"}       // IP已被封禁错误码 (40306)。
	RegionBlocked    = &ErrorCode{40307, "REGION_BLOCKED", "地区访问受限"}   // 地区访问受限错误码 (40307)。
	DeviceBlocked    = &ErrorCode{40308, "DEVICE_BLOCKED", "设备已被封禁"}   // 设备已被封禁错误码 (40308)。
	ActionForbidden  = &ErrorCode{40309, "ACTION_FORBIDDEN", "操作被禁止"}  // 操作被禁止错误码 (40309)。
)

// ============================== 404 Not Found (404xx) ==============================

// 资源不存在错误码定义。
var (
	NotFound         = &ErrorCode{40400, "NOT_FOUND", "未找到"}            // 未找到错误码 (40400)。
	PageNotFound     = &ErrorCode{40401, "PAGE_NOT_FOUND", "页面未找到"}     // 页面未找到错误码 (40401)。
	ResourceNotFound = &ErrorCode{40402, "RESOURCE_NOT_FOUND", "资源未找到"} // 资源未找到错误码 (40402)。
	APINotFound      = &ErrorCode{40403, "API_NOT_FOUND", "接口未找到"}      // 接口未找到错误码 (40403)。
	RouteNotFound    = &ErrorCode{40404, "ROUTE_NOT_FOUND", "路由未找到"}    // 路由未找到错误码 (40404)。
	ServiceNotFound  = &ErrorCode{40405, "SERVICE_NOT_FOUND", "服务未找到"}  // 服务未找到错误码 (40405)。
	RecordNotFound   = &ErrorCode{40406, "RECORD_NOT_FOUND", "记录未找到"}   // 记录未找到错误码 (40406)。
	ConfigNotFound   = &ErrorCode{40407, "CONFIG_NOT_FOUND", "配置未找到"}   // 配置未找到错误码 (40407)。
)

// ============================== 405 Method Not Allowed (405xx) ==============================

// HTTP 方法不支持错误码定义。
var (
	MethodNotAllowed = &ErrorCode{40500, "METHOD_NOT_ALLOWED", "方法不允许"} // 方法不允许错误码 (40500)。
)

// ============================== 406 Not Acceptable (406xx) ==============================

// 请求内容协商错误码定义。
var (
	NotAcceptable     = &ErrorCode{40600, "NOT_ACCEPTABLE", "不可接受"}           // 不可接受错误码 (40600)。
	ContentTypeError  = &ErrorCode{40601, "CONTENT_TYPE_ERROR", "内容类型错误"}     // 内容类型错误错误码 (40601)。
	AcceptHeaderError = &ErrorCode{40602, "ACCEPT_HEADER_ERROR", "Accept头错误"} // Accept头错误错误码 (40602)。
)

// ============================== 408 Request Timeout (408xx) ==============================

// 请求超时错误码定义。
var (
	Timeout        = &ErrorCode{40800, "TIMEOUT", "请求超时"}         // 请求超时错误码 (40800)。
	ConnectTimeout = &ErrorCode{40801, "CONNECT_TIMEOUT", "连接超时"} // 连接超时错误码 (40801)。
	ReadTimeout    = &ErrorCode{40802, "READ_TIMEOUT", "读取超时"}    // 读取超时错误码 (40802)。
	WriteTimeout   = &ErrorCode{40803, "WRITE_TIMEOUT", "写入超时"}   // 写入超时错误码 (40803)。
	ExecuteTimeout = &ErrorCode{40804, "EXECUTE_TIMEOUT", "执行超时"} // 执行超时错误码 (40804)。
	IdleTimeout    = &ErrorCode{40805, "IDLE_TIMEOUT", "空闲超时"}    // 空闲超时错误码 (40805)。
)

// ============================== 409 Conflict (409xx) ==============================

// 状态冲突错误码定义。
var (
	Conflict         = &ErrorCode{40900, "CONFLICT", "冲突"}              // 冲突错误码 (40900)。
	VersionConflict  = &ErrorCode{40901, "VERSION_CONFLICT", "版本冲突"}    // 版本冲突错误码 (40901)。
	ConcurrencyError = &ErrorCode{40902, "CONCURRENCY_ERROR", "并发错误"}   // 并发错误错误码 (40902)。
	LockConflict     = &ErrorCode{40903, "LOCK_CONFLICT", "锁冲突"}        // 锁冲突错误码 (40903)。
	ResourceConflict = &ErrorCode{40904, "RESOURCE_CONFLICT", "资源冲突"}   // 资源冲突错误码 (40904)。
	OptimisticLock   = &ErrorCode{40905, "OPTIMISTIC_LOCK", "乐观锁冲突"}    // 乐观锁冲突错误码 (40905)。
	DuplicateEntry   = &ErrorCode{40906, "DUPLICATE_ENTRY", "重复条目"}     // 重复条目错误码 (40906)。
	UniqueConstraint = &ErrorCode{40907, "UNIQUE_CONSTRAINT", "唯一约束冲突"} // 唯一约束冲突错误码 (40907)。
	ForeignKeyError  = &ErrorCode{40908, "FOREIGN_KEY_ERROR", "外键约束错误"} // 外键约束错误错误码 (40908)。
	IntegrityError   = &ErrorCode{40909, "INTEGRITY_ERROR", "完整性约束错误"}  // 完整性约束错误错误码 (40909)。
)

// ============================== 410 Gone (410xx) ==============================

// 资源永久失效错误码定义。
var (
	Gone           = &ErrorCode{41000, "GONE", "资源已删除"}            // 资源已删除错误码 (41000)。
	ResourceGone   = &ErrorCode{41001, "RESOURCE_GONE", "资源已不存在"}  // 资源已不存在错误码 (41001)。
	DeprecatedAPI  = &ErrorCode{41002, "DEPRECATED_API", "接口已废弃"}  // 接口已废弃错误码 (41002)。
	VersionExpired = &ErrorCode{41003, "VERSION_EXPIRED", "版本已过期"} // 版本已过期错误码 (41003)。
)

// ============================== 413 Payload Too Large (413xx) ==============================

// 请求体过大错误码定义。
var (
	PayloadTooLarge = &ErrorCode{41300, "PAYLOAD_TOO_LARGE", "请求体过大"} // 请求体过大错误码 (41300)。
	RequestTooLarge = &ErrorCode{41301, "REQUEST_TOO_LARGE", "请求过大"}  // 请求过大错误码 (41301)。
	UploadTooLarge  = &ErrorCode{41302, "UPLOAD_TOO_LARGE", "上传内容过大"} // 上传内容过大错误码 (41302)。
)

// ============================== 415 Unsupported Media Type (415xx) ==============================

// 媒体类型不支持错误码定义。
var (
	UnsupportedMedia = &ErrorCode{41500, "UNSUPPORTED_MEDIA", "不支持的媒体类型"} // 不支持的媒体类型错误码 (41500)。
	UnsupportedType  = &ErrorCode{41501, "UNSUPPORTED_TYPE", "不支持的类型"}    // 不支持的类型错误码 (41501)。
)

// ============================== 422 Unprocessable Entity (422xx) ==============================

// 实体无法处理错误码定义。
var (
	UnprocessableEntity = &ErrorCode{42200, "UNPROCESSABLE_ENTITY", "无法处理的实体"} // 无法处理的实体错误码 (42200)。
	SemanticError       = &ErrorCode{42201, "SEMANTIC_ERROR", "语义错误"}          // 语义错误错误码 (42201)。
	LogicError          = &ErrorCode{42202, "LOGIC_ERROR", "逻辑错误"}             // 逻辑错误错误码 (42202)。
)

// ============================== 429 Too Many Requests (429xx) ==============================

// 请求限流错误码定义。
var (
	TooManyRequests  = &ErrorCode{42900, "TOO_MANY_REQUESTS", "请求过多"}      // 请求过多错误码 (42900)。
	RateLimited      = &ErrorCode{42901, "RATE_LIMITED", "请求频率过高"}         // 请求频率过高错误码 (42901)。
	ThrottleExceeded = &ErrorCode{42902, "THROTTLE_EXCEEDED", "限流阈值超出"}    // 限流阈值超出错误码 (42902)。
	ConcurrentLimit  = &ErrorCode{42903, "CONCURRENT_LIMIT", "并发数超限"}      // 并发数超限错误码 (42903)。
	DailyLimitExceed = &ErrorCode{42904, "DAILY_LIMIT_EXCEEDED", "日请求量超限"} // 日请求量超限错误码 (42904)。
)

// ============================== 500 Internal Server Error (500xx) ==============================

// 服务器内部错误码定义。
var (
	ServerInternalError = &ErrorCode{50000, "SERVER_INTERNAL_ERROR", "服务器内部错误"} // 服务器内部错误错误码 (50000)。
	DatabaseError       = &ErrorCode{50001, "DATABASE_ERROR", "数据库错误"}          // 数据库错误错误码 (50001)。
	CacheError          = &ErrorCode{50002, "CACHE_ERROR", "缓存错误"}              // 缓存错误错误码 (50002)。
	FileError           = &ErrorCode{50003, "FILE_ERROR", "文件错误"}               // 文件错误错误码 (50003)。
	StorageError        = &ErrorCode{50004, "STORAGE_ERROR", "存储错误"}            // 存储错误错误码 (50004)。
	RemoteError         = &ErrorCode{50005, "REMOTE_ERROR", "远程调用错误"}           // 远程调用错误错误码 (50005)。
	ConfigError         = &ErrorCode{50006, "CONFIG_ERROR", "配置错误"}             // 配置错误错误码 (50006)。
	NetworkError        = &ErrorCode{50007, "NETWORK_ERROR", "网络错误"}            // 网络错误错误码 (50007)。
	EncryptError        = &ErrorCode{50008, "ENCRYPT_ERROR", "加密错误"}            // 加密错误错误码 (50008)。
	DecryptError        = &ErrorCode{50009, "DECRYPT_ERROR", "解密错误"}            // 解密错误错误码 (50009)。
	SerializeError      = &ErrorCode{50010, "SERIALIZE_ERROR", "序列化错误"}         // 序列化错误错误码 (50010)。
	DeserializeErr      = &ErrorCode{50011, "DESERIALIZE_ERROR", "反序列化错误"}      // 反序列化错误错误码 (50011)。
	JSONError           = &ErrorCode{50012, "JSON_ERROR", "JSON处理错误"}           // JSON处理错误错误码 (50012)。
	XMLError            = &ErrorCode{50013, "XML_ERROR", "XML处理错误"}             // XML处理错误错误码 (50013)。
	IoError             = &ErrorCode{50014, "IO_ERROR", "IO错误"}                 // IO错误错误码 (50014)。
	MemoryError         = &ErrorCode{50015, "MEMORY_ERROR", "内存错误"}             // 内存错误错误码 (50015)。
	ThreadError         = &ErrorCode{50016, "THREAD_ERROR", "线程错误"}             // 线程错误错误码 (50016)。
	PoolExhausted       = &ErrorCode{50017, "POOL_EXHAUSTED", "连接池耗尽"}          // 连接池耗尽错误码 (50017)。
	QueueFull           = &ErrorCode{50018, "QUEUE_FULL", "队列已满"}               // 队列已满错误码 (50018)。
	UnknownError        = &ErrorCode{50099, "UNKNOWN_ERROR", "未知错误"}            // 未知错误错误码 (50099)。
)

// ============================== 第三方服务错误 (500xx) ==============================

// 第三方服务调用错误码定义。
var (
	ThirdPartyError = &ErrorCode{50020, "THIRD_PARTY_ERROR", "第三方服务错误"} // 第三方服务错误错误码 (50020)。
	APICallFailed   = &ErrorCode{50021, "API_CALL_FAILED", "API调用失败"}   // API调用失败错误码 (50021)。
	CallbackError   = &ErrorCode{50022, "CALLBACK_ERROR", "回调错误"}       // 回调错误错误码 (50022)。
	WebhookError    = &ErrorCode{50023, "WEBHOOK_ERROR", "Webhook错误"}   // Webhook错误错误码 (50023)。
	SmsError        = &ErrorCode{50024, "SMS_ERROR", "短信服务错误"}          // 短信服务错误错误码 (50024)。
	EmailError      = &ErrorCode{50025, "EMAIL_ERROR", "邮件服务错误"}        // 邮件服务错误错误码 (50025)。
	PaymentError    = &ErrorCode{50026, "PAYMENT_ERROR", "支付服务错误"}      // 支付服务错误错误码 (50026)。
	OssError        = &ErrorCode{50027, "OSS_ERROR", "对象存储错误"}          // 对象存储错误错误码 (50027)。
	CdnError        = &ErrorCode{50028, "CDN_ERROR", "CDN服务错误"}         // CDN服务错误错误码 (50028)。
	PushError       = &ErrorCode{50029, "PUSH_ERROR", "推送服务错误"}         // 推送服务错误错误码 (50029)。
)

// ============================== 502 Bad Gateway (502xx) ==============================

// 网关与代理错误码定义。
var (
	GatewayError     = &ErrorCode{50200, "GATEWAY_ERROR", "网关错误"}        // 网关错误错误码 (50200)。
	UpstreamError    = &ErrorCode{50201, "UPSTREAM_ERROR", "上游服务错误"}     // 上游服务错误错误码 (50201)。
	ProxyError       = &ErrorCode{50202, "PROXY_ERROR", "代理错误"}          // 代理错误错误码 (50202)。
	LoadBalanceError = &ErrorCode{50203, "LOAD_BALANCE_ERROR", "负载均衡错误"} // 负载均衡错误错误码 (50203)。
)

// ============================== 503 Service Unavailable (503xx) ==============================

// 服务可用性错误码定义。
var (
	ServiceUnavailable = &ErrorCode{50300, "SERVICE_UNAVAILABLE", "服务不可用"} // 服务不可用错误码 (50300)。
	SystemMaintenance  = &ErrorCode{50301, "SYSTEM_MAINTENANCE", "系统维护中"}  // 系统维护中错误码 (50301)。
	ResourceExhausted  = &ErrorCode{50302, "RESOURCE_EXHAUSTED", "资源耗尽"}   // 资源耗尽错误码 (50302)。
	ServiceOverload    = &ErrorCode{50303, "SERVICE_OVERLOAD", "服务过载"}     // 服务过载错误码 (50303)。
	CircuitBreaker     = &ErrorCode{50304, "CIRCUIT_BREAKER", "熔断保护"}      // 熔断保护错误码 (50304)。
	ServiceDegraded    = &ErrorCode{50305, "SERVICE_DEGRADED", "服务降级"}     // 服务降级错误码 (50305)。
)

// ============================== 504 Gateway Timeout (504xx) ==============================

// 网关超时错误码定义。
var (
	GatewayTimeout  = &ErrorCode{50400, "GATEWAY_TIMEOUT", "网关超时"}    // 网关超时错误码 (50400)。
	UpstreamTimeout = &ErrorCode{50401, "UPSTREAM_TIMEOUT", "上游服务超时"} // 上游服务超时错误码 (50401)。
)
