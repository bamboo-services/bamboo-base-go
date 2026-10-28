package env

// 系统与预算配置环境变量键。
const (
	Env             Key = "APP_ENV"              // 应用环境 (APP_ENV，如 local/dev/prod)。
	AppName         Key = "APP_NAME"             // 应用名称 (APP_NAME)。
	AppVersion      Key = "APP_VERSION"          // 应用版本号 (APP_VERSION)。
	Timezone        Key = "APP_TIMEZONE"         // 应用默认时区 (APP_TIMEZONE)。
	Debug           Key = "APP_DEBUG"            // 调试模式 (APP_DEBUG)。
	Host            Key = "HOST"                 // 监听地址 (HOST)。
	Port            Key = "PORT"                 // 监听端口 (PORT)。
	StartTimeout    Key = "APP_START_TIMEOUT"    // 应用启动超时预算键 (APP_START_TIMEOUT)。
	GraceTimeout    Key = "APP_GRACE_TIMEOUT"    // 应用平滑停机等待预算键 (APP_GRACE_TIMEOUT)。
	CleanupTimeout  Key = "APP_CLEANUP_TIMEOUT"  // 应用清理超时预算键 (APP_CLEANUP_TIMEOUT)。
	ShutdownTimeout Key = "APP_SHUTDOWN_TIMEOUT" // 应用总关机超时预算键 (APP_SHUTDOWN_TIMEOUT)。
	GrpcPort        Key = "GRPC_PORT"            // gRPC 端口 (GRPC_PORT)。
	GrpcReflection  Key = "GRPC_REFLECTION"      // gRPC 服务反射开关 (GRPC_REFLECTION)。
)

// 数据库配置环境变量键。
const (
	DatabaseDriver      Key = "DATABASE_DRIVER"       // 数据库驱动 (mysql/postgres/sqlite/oracle/sqlserver)。
	DatabaseDSN         Key = "DATABASE_DSN"          // 数据库完整连接串。
	DatabaseHost        Key = "DATABASE_HOST"         // 数据库主机地址。
	DatabasePort        Key = "DATABASE_PORT"         // 数据库端口。
	DatabaseUser        Key = "DATABASE_USER"         // 数据库用户名。
	DatabasePass        Key = "DATABASE_PASS"         // 数据库密码。
	DatabaseName        Key = "DATABASE_NAME"         // 数据库名称。
	DatabaseCharset     Key = "DATABASE_CHARSET"      // 数据库字符集。
	DatabaseTimezone    Key = "DATABASE_TIMEZONE"     // 数据库时区。
	DatabasePrefix      Key = "DATABASE_PREFIX"       // 数据库表前缀。
	DatabasePath        Key = "DATABASE_PATH"         // SQLite 数据库文件路径。
	DatabaseServiceName Key = "DATABASE_SERVICE_NAME" // Oracle 服务名。
	DatabaseLibDir      Key = "DATABASE_LIB_DIR"      // Oracle Instant Client 目录。
)

// 缓存与 NoSQL 配置环境变量键。
const (
	NoSQLDriver           Key = "NOSQL_DRIVER"             // 缓存驱动类型 (redis/memory/none)。
	NoSQLHost             Key = "NOSQL_HOST"               // Redis 主机地址。
	NoSQLPort             Key = "NOSQL_PORT"               // Redis 端口。
	NoSQLUser             Key = "NOSQL_USER"               // Redis 用户名。
	NoSQLPass             Key = "NOSQL_PASS"               // Redis 密码。
	NoSQLDatabase         Key = "NOSQL_DATABASE"           // Redis 数据库索引。
	NoSQLPoolSize         Key = "NOSQL_POOL_SIZE"          // Redis 连接池大小。
	NoSQLPrefix           Key = "NOSQL_PREFIX"             // Redis 键前缀。
	NoSQLMemoryDefaultTTL Key = "NOSQL_MEMORY_DEFAULT_TTL" // 内存缓存默认过期时间。
	NoSQLMemoryMaxEntries Key = "NOSQL_MEMORY_MAX_ENTRIES" // 内存缓存最大条目数。
	NoSQLMemoryShardCount Key = "NOSQL_MEMORY_SHARD_COUNT" // 内存缓存分片数。
)

// 雪花算法配置环境变量键。
const (
	SnowflakeDatacenterID Key = "SNOWFLAKE_DATACENTER_ID" // 数据中心 ID。
	SnowflakeNodeID       Key = "SNOWFLAKE_NODE_ID"       // 节点 ID。
)

// 日志配置环境变量键。
const (
	LogLevel      Key = "LOG_LEVEL"       // 日志级别 (debug/info/warn/error)。
	LogPath       Key = "LOG_PATH"        // 日志文件路径。
	LogMaxSize    Key = "LOG_MAX_SIZE"    // 日志文件最大大小（MB）。
	LogMaxAge     Key = "LOG_MAX_AGE"     // 日志文件最大保留天数。
	LogMaxBackups Key = "LOG_MAX_BACKUPS" // 日志文件最大备份数。
	LogCompress   Key = "LOG_COMPRESS"    // 是否压缩日志文件。
)

// 第三方服务配置环境变量键。
const (
	SmsAccessKey Key = "SMS_ACCESS_KEY" // 短信服务 AccessKey。
	SmsSecretKey Key = "SMS_SECRET_KEY" // 短信服务 SecretKey。
	SmsSignName  Key = "SMS_SIGN_NAME"  // 短信签名。

	EmailHost        Key = "EMAIL_HOST"         // 邮件服务器地址。
	EmailPort        Key = "EMAIL_PORT"         // 邮件服务器端口。
	EmailUser        Key = "EMAIL_USER"         // 邮件用户名。
	EmailPass        Key = "EMAIL_PASS"         // 邮件密码。
	EmailFrom        Key = "EMAIL_FROM"         // 发件人地址。
	EmailTLS         Key = "EMAIL_TLS"          // 邮件 TLS 策略。
	EmailFromName    Key = "EMAIL_FROM_NAME"    // 发件人名称。
	EmailTemplateDir Key = "EMAIL_TEMPLATE_DIR" // 外部邮件模板目录。

	OssEndpoint  Key = "OSS_ENDPOINT"   // OSS 端点。
	OssAccessKey Key = "OSS_ACCESS_KEY" // OSS AccessKey。
	OssSecretKey Key = "OSS_SECRET_KEY" // OSS SecretKey。
	OssBucket    Key = "OSS_BUCKET"     // OSS 存储桶。
)
