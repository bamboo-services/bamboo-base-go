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
)

// ============================== 数据库配置 ==============================

const (
	DatabaseHost     EnvKey = "DATABASE_HOST"     // 数据库主机地址
	DatabasePort     EnvKey = "DATABASE_PORT"     // 数据库端口
	DatabaseUser     EnvKey = "DATABASE_USER"     // 数据库用户名
	DatabasePass     EnvKey = "DATABASE_PASS"     // 数据库密码
	DatabaseName     EnvKey = "DATABASE_NAME"     // 数据库名称
	DatabaseCharset  EnvKey = "DATABASE_CHARSET"  // 数据库字符集
	DatabaseTimezone EnvKey = "DATABASE_TIMEZONE" // 数据库时区
	DatabasePrefix   EnvKey = "DATABASE_PREFIX"   // 数据库表前缀
)

// ============================== Redis 配置 ==============================

const (
	NoSqlHost     EnvKey = "NOSQL_HOST"      // NoSQL 主机地址
	NoSqlPort     EnvKey = "NOSQL_PORT"      // NoSQL 端口
	NoSqlPass     EnvKey = "NOSQL_PASS"      // NoSQL 密码
	NoSqlDatabase EnvKey = "NOSQL_DATABASE"  // NoSQL 数据库索引
	NoSqlPoolSize EnvKey = "NOSQL_POOL_SIZE" // NoSQL 连接池大小
	NoSqlPrefix   EnvKey = "NOSQL_PREFIX"    // NoSQL 连接池大小
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
