package xEnv

// EnvKey 环境变量键类型
type EnvKey string

// String 获取环境变量键字符串
func (e EnvKey) String() string {
	return string(e)
}

// ============================== 系统配置 ==============================

const (
	Debug EnvKey = "XLF_DEBUG" // 调试模式 (true/false)
	Host  EnvKey = "XLF_HOST"  // 监听地址
	Port  EnvKey = "XLF_PORT"  // 监听端口

	GrpcPort       EnvKey = "GRPC_PORT"
	GrpcReflection EnvKey = "GRPC_REFLECTION"
)

// ============================== 数据库配置 ==============================

const (
	DatabaseDriver      EnvKey = "DATABASE_DRIVER"       // 数据库驱动 (mysql/postgres/sqlite/oracle/sqlserver)
	DatabaseDSN         EnvKey = "DATABASE_DSN"          // 数据库完整连接串（设置后忽略下方分项配置）
	DatabaseHost        EnvKey = "DATABASE_HOST"         // 数据库主机地址
	DatabasePort        EnvKey = "DATABASE_PORT"         // 数据库端口
	DatabaseUser        EnvKey = "DATABASE_USER"         // 数据库用户名
	DatabasePass        EnvKey = "DATABASE_PASS"         // 数据库密码
	DatabaseName        EnvKey = "DATABASE_NAME"         // 数据库名称
	DatabaseCharset     EnvKey = "DATABASE_CHARSET"      // 数据库字符集
	DatabaseTimezone    EnvKey = "DATABASE_TIMEZONE"     // 数据库时区
	DatabasePrefix      EnvKey = "DATABASE_PREFIX"       // 数据库表前缀
	DatabasePath        EnvKey = "DATABASE_PATH"         // SQLite 数据库文件路径（仅 sqlite 驱动有效）
	DatabaseServiceName EnvKey = "DATABASE_SERVICE_NAME" // Oracle 服务名（仅 oracle 驱动有效）
	DatabaseLibDir      EnvKey = "DATABASE_LIB_DIR"      // Oracle Instant Client 目录（macOS/Windows 有效，Linux 需用 ldconfig 配置系统库搜索路径）
)

// ============================== Redis/缓存配置 ==============================

const (
	NoSqlDriver   EnvKey = "NOSQL_DRIVER"    // 缓存驱动类型 (redis/memory/none)
	NoSqlHost     EnvKey = "NOSQL_HOST"      // Redis 主机地址
	NoSqlPort     EnvKey = "NOSQL_PORT"      // Redis 端口
	NoSqlUser     EnvKey = "NOSQL_USER"      // Redis 用户名 (ACL 模式)
	NoSqlPass     EnvKey = "NOSQL_PASS"      // Redis 密码
	NoSqlDatabase EnvKey = "NOSQL_DATABASE"  // Redis 数据库索引 (0-15)
	NoSqlPoolSize EnvKey = "NOSQL_POOL_SIZE" // Redis 连接池大小
	NoSqlPrefix   EnvKey = "NOSQL_PREFIX"    // Redis 键前缀

	NoSqlMemoryDefaultTTL EnvKey = "NOSQL_MEMORY_DEFAULT_TTL" // NOSQL_DRIVER=memory时生效 内存缓存默认过期时间（Go Duration 字符串，如 30m/1h，0=永不过期）
	NoSqlMemoryMaxEntries EnvKey = "NOSQL_MEMORY_MAX_ENTRIES" // NOSQL_DRIVER=memory时生效 内存缓存最大条目数（0=无上限）
	NoSqlMemoryShardCount EnvKey = "NOSQL_MEMORY_SHARD_COUNT" // NOSQL_DRIVER=memory时生效 内存缓存分片数（0=使用默认分片）
)

// ============================== 雪花算法配置 ==============================

const (
	SnowflakeDatacenterID EnvKey = "SNOWFLAKE_DATACENTER_ID" // 数据中心 ID (0-31)
	SnowflakeNodeID       EnvKey = "SNOWFLAKE_NODE_ID"       // 节点 ID (0-31)
)

// ============================== 日志配置 ==============================

const (
	LogLevel      EnvKey = "LOG_LEVEL"       // 日志级别 (debug/info/warn/error)
	LogPath       EnvKey = "LOG_PATH"        // 日志文件路径
	LogMaxSize    EnvKey = "LOG_MAX_SIZE"    // 日志文件最大大小（MB）
	LogMaxAge     EnvKey = "LOG_MAX_AGE"     // 日志文件最大保留天数
	LogMaxBackups EnvKey = "LOG_MAX_BACKUPS" // 日志文件最大备份数
	LogCompress   EnvKey = "LOG_COMPRESS"    // 是否压缩日志文件
)

// ============================== 第三方服务配置 ==============================

const (
	SmsAccessKey EnvKey = "SMS_ACCESS_KEY" // 短信服务 AccessKey
	SmsSecretKey EnvKey = "SMS_SECRET_KEY" // 短信服务 SecretKey
	SmsSignName  EnvKey = "SMS_SIGN_NAME"  // 短信签名

	EmailHost EnvKey = "EMAIL_HOST" // 邮件服务器地址
	EmailPort EnvKey = "EMAIL_PORT" // 邮件服务器端口
	EmailUser EnvKey = "EMAIL_USER" // 邮件用户名
	EmailPass EnvKey = "EMAIL_PASS" // 邮件密码
	EmailFrom EnvKey = "EMAIL_FROM" // 发件人地址

	EmailTLS         EnvKey = "EMAIL_TLS"          // 邮件 TLS 策略
	EmailFromName    EnvKey = "EMAIL_FROM_NAME"    // 发件人名称
	EmailTemplateDir EnvKey = "EMAIL_TEMPLATE_DIR" // 外部邮件模板目录

	OssEndpoint  EnvKey = "OSS_ENDPOINT"   // OSS 端点
	OssAccessKey EnvKey = "OSS_ACCESS_KEY" // OSS AccessKey
	OssSecretKey EnvKey = "OSS_SECRET_KEY" // OSS SecretKey
	OssBucket    EnvKey = "OSS_BUCKET"     // OSS 存储桶
)

// ============================== 运行环境配置 ==============================

const (
	Env        EnvKey = "ENV"         // 运行环境 (development/testing/staging/production)
	AppName    EnvKey = "APP_NAME"    // 应用名称
	AppVersion EnvKey = "APP_VERSION" // 应用版本
	Timezone   EnvKey = "TIMEZONE"    // 系统时区
)
