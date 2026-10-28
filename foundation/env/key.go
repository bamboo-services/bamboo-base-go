package env

import (
	"database/sql/driver"
	"encoding/xml"
	"fmt"
	"os"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/typed"
)

// Key 是基于 typed.String 派生的强类型环境变量键。
type Key typed.String

var (
	_ typed.Operable    = Key("")
	_ typed.Convertible = (*Key)(nil)
)

// ============================== 系统与预算配置 ==============================

const (
	// Env 应用环境 (APP_ENV，如 local/dev/prod)。
	Env Key = "APP_ENV"
	// AppName 应用名称 (APP_NAME)。
	AppName Key = "APP_NAME"
	// AppVersion 应用版本号 (APP_VERSION)。
	AppVersion Key = "APP_VERSION"
	// Timezone 应用默认时区 (APP_TIMEZONE)。
	Timezone Key = "APP_TIMEZONE"

	// Debug 调试模式 (APP_DEBUG)。
	Debug Key = "APP_DEBUG"
	// Host 监听地址 (HOST)。
	Host Key = "HOST"
	// Port 监听端口 (PORT)。
	Port Key = "PORT"

	// StartTimeout 应用启动超时预算键 (APP_START_TIMEOUT)。
	StartTimeout Key = "APP_START_TIMEOUT"
	// GraceTimeout 应用平滑停机等待预算键 (APP_GRACE_TIMEOUT)。
	GraceTimeout Key = "APP_GRACE_TIMEOUT"
	// CleanupTimeout 应用清理超时预算键 (APP_CLEANUP_TIMEOUT)。
	CleanupTimeout Key = "APP_CLEANUP_TIMEOUT"
	// ShutdownTimeout 应用总关机超时预算键 (APP_SHUTDOWN_TIMEOUT)。
	ShutdownTimeout Key = "APP_SHUTDOWN_TIMEOUT"

	// GrpcPort gRPC 端口 (GRPC_PORT)。
	GrpcPort Key = "GRPC_PORT"
	// GrpcReflection gRPC 服务反射开关 (GRPC_REFLECTION)。
	GrpcReflection Key = "GRPC_REFLECTION"
)

// ============================== 数据库配置 ==============================

const (
	// DatabaseDriver 数据库驱动 (mysql/postgres/sqlite/oracle/sqlserver)。
	DatabaseDriver Key = "DATABASE_DRIVER"
	// DatabaseDSN 数据库完整连接串。
	DatabaseDSN Key = "DATABASE_DSN"
	// DatabaseHost 数据库主机地址。
	DatabaseHost Key = "DATABASE_HOST"
	// DatabasePort 数据库端口。
	DatabasePort Key = "DATABASE_PORT"
	// DatabaseUser 数据库用户名。
	DatabaseUser Key = "DATABASE_USER"
	// DatabasePass 数据库密码。
	DatabasePass Key = "DATABASE_PASS"
	// DatabaseName 数据库名称。
	DatabaseName Key = "DATABASE_NAME"
	// DatabaseCharset 数据库字符集。
	DatabaseCharset Key = "DATABASE_CHARSET"
	// DatabaseTimezone 数据库时区。
	DatabaseTimezone Key = "DATABASE_TIMEZONE"
	// DatabasePrefix 数据库表前缀。
	DatabasePrefix Key = "DATABASE_PREFIX"
	// DatabasePath SQLite 数据库文件路径。
	DatabasePath Key = "DATABASE_PATH"
	// DatabaseServiceName Oracle 服务名。
	DatabaseServiceName Key = "DATABASE_SERVICE_NAME"
	// DatabaseLibDir Oracle Instant Client 目录。
	DatabaseLibDir Key = "DATABASE_LIB_DIR"
)

// ============================== 缓存/NoSQL配置 ==============================

const (
	// NoSQLDriver 缓存驱动类型 (redis/memory/none)。
	NoSQLDriver Key = "NOSQL_DRIVER"
	// NoSQLHost Redis 主机地址。
	NoSQLHost Key = "NOSQL_HOST"
	// NoSQLPort Redis 端口。
	NoSQLPort Key = "NOSQL_PORT"
	// NoSQLUser Redis 用户名。
	NoSQLUser Key = "NOSQL_USER"
	// NoSQLPass Redis 密码。
	NoSQLPass Key = "NOSQL_PASS"
	// NoSQLDatabase Redis 数据库索引。
	NoSQLDatabase Key = "NOSQL_DATABASE"
	// NoSQLPoolSize Redis 连接池大小。
	NoSQLPoolSize Key = "NOSQL_POOL_SIZE"
	// NoSQLPrefix Redis 键前缀。
	NoSQLPrefix Key = "NOSQL_PREFIX"

	// NoSQLMemoryDefaultTTL 内存缓存默认过期时间。
	NoSQLMemoryDefaultTTL Key = "NOSQL_MEMORY_DEFAULT_TTL"
	// NoSQLMemoryMaxEntries 内存缓存最大条目数。
	NoSQLMemoryMaxEntries Key = "NOSQL_MEMORY_MAX_ENTRIES"
	// NoSQLMemoryShardCount 内存缓存分片数。
	NoSQLMemoryShardCount Key = "NOSQL_MEMORY_SHARD_COUNT"
)

// ============================== 雪花算法配置 ==============================

const (
	// SnowflakeDatacenterID 数据中心 ID。
	SnowflakeDatacenterID Key = "SNOWFLAKE_DATACENTER_ID"
	// SnowflakeNodeID 节点 ID。
	SnowflakeNodeID Key = "SNOWFLAKE_NODE_ID"
)

// ============================== 日志配置 ==============================

const (
	// LogLevel 日志级别 (debug/info/warn/error)。
	LogLevel Key = "LOG_LEVEL"
	// LogPath 日志文件路径。
	LogPath Key = "LOG_PATH"
	// LogMaxSize 日志文件最大大小（MB）。
	LogMaxSize Key = "LOG_MAX_SIZE"
	// LogMaxAge 日志文件最大保留天数。
	LogMaxAge Key = "LOG_MAX_AGE"
	// LogMaxBackups 日志文件最大备份数。
	LogMaxBackups Key = "LOG_MAX_BACKUPS"
	// LogCompress 是否压缩日志文件。
	LogCompress Key = "LOG_COMPRESS"
)

// ============================== 第三方服务配置 ==============================

const (
	// SmsAccessKey 短信服务 AccessKey。
	SmsAccessKey Key = "SMS_ACCESS_KEY"
	// SmsSecretKey 短信服务 SecretKey。
	SmsSecretKey Key = "SMS_SECRET_KEY"
	// SmsSignName 短信签名。
	SmsSignName Key = "SMS_SIGN_NAME"

	// EmailHost 邮件服务器地址。
	EmailHost Key = "EMAIL_HOST"
	// EmailPort 邮件服务器端口。
	EmailPort Key = "EMAIL_PORT"
	// EmailUser 邮件用户名。
	EmailUser Key = "EMAIL_USER"
	// EmailPass 邮件密码。
	EmailPass Key = "EMAIL_PASS"
	// EmailFrom 发件人地址。
	EmailFrom Key = "EMAIL_FROM"

	// EmailTLS 邮件 TLS 策略。
	EmailTLS Key = "EMAIL_TLS"
	// EmailFromName 发件人名称。
	EmailFromName Key = "EMAIL_FROM_NAME"
	// EmailTemplateDir 外部邮件模板目录。
	EmailTemplateDir Key = "EMAIL_TEMPLATE_DIR"

	// OssEndpoint OSS 端点。
	OssEndpoint Key = "OSS_ENDPOINT"
	// OssAccessKey OSS AccessKey。
	OssAccessKey Key = "OSS_ACCESS_KEY"
	// OssSecretKey OSS SecretKey。
	OssSecretKey Key = "OSS_SECRET_KEY"
	// OssBucket OSS 存储桶。
	OssBucket Key = "OSS_BUCKET"
)

// Lookup 查询指定环境变量键的值。
// 返回原始字符串封装的 typed.String 以及是否存在布尔值。
// 如果环境变量未设置，返回 ("", false)；如果设置为空串，返回 ("", true)。
// 原始环境变量值不进行受控键名语法过滤。
func Lookup(key Key) (typed.String, bool) {
	val, ok := os.LookupEnv(string(key))
	if !ok {
		return "", false
	}
	return typed.String(val), true
}

// String 返回底层原生 string。
func (k Key) String() string {
	return string(k)
}

// TypedString 返回对应的 typed.String。
func (k Key) TypedString() typed.String {
	return typed.String(k)
}

// IsEmpty 报告键是否为空。
func (k Key) IsEmpty() bool {
	return len(k) == 0
}

// IsValid 校验受控键名语法是否合法。
func (k Key) IsValid() bool {
	return typed.String(k).IsValid()
}

// MarshalText 实现 encoding.TextMarshaler 接口。
func (k Key) MarshalText() ([]byte, error) {
	return k.TypedString().MarshalText()
}

// UnmarshalText 实现 encoding.TextUnmarshaler 接口。
func (k *Key) UnmarshalText(text []byte) error {
	if k == nil {
		return fmt.Errorf("env.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.UnmarshalText(text); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}

// MarshalBinary 实现 encoding.BinaryMarshaler 接口。
func (k Key) MarshalBinary() ([]byte, error) {
	return k.TypedString().MarshalBinary()
}

// UnmarshalBinary 实现 encoding.BinaryUnmarshaler 接口。
func (k *Key) UnmarshalBinary(data []byte) error {
	if k == nil {
		return fmt.Errorf("env.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.UnmarshalBinary(data); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}

// MarshalJSON 实现 json.Marshaler 接口。
func (k Key) MarshalJSON() ([]byte, error) {
	return k.TypedString().MarshalJSON()
}

// UnmarshalJSON 实现 json.Unmarshaler 接口。
func (k *Key) UnmarshalJSON(raw []byte) error {
	if k == nil {
		return fmt.Errorf("env.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.UnmarshalJSON(raw); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}

// MarshalXML 实现 xml.Marshaler 接口。
func (k Key) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return k.TypedString().MarshalXML(e, start)
}

// UnmarshalXML 实现 xml.Unmarshaler 接口。
func (k *Key) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if k == nil {
		return fmt.Errorf("env.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.UnmarshalXML(d, start); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}

// MarshalXMLAttr 实现 xml.MarshalerAttr 接口。
func (k Key) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return k.TypedString().MarshalXMLAttr(name)
}

// UnmarshalXMLAttr 实现 xml.UnmarshalerAttr 接口。
func (k *Key) UnmarshalXMLAttr(attr xml.Attr) error {
	if k == nil {
		return fmt.Errorf("env.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.UnmarshalXMLAttr(attr); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}

// Value 实现 driver.Valuer 接口。
func (k Key) Value() (driver.Value, error) {
	return k.TypedString().Value()
}

// Scan 实现 sql.Scanner 接口。
func (k *Key) Scan(src any) error {
	if k == nil {
		return fmt.Errorf("env.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.Scan(src); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}

// Set 实现 flag.Value 接口。
func (k *Key) Set(raw string) error {
	if k == nil {
		return fmt.Errorf("env.Key: nil receiver")
	}
	var candidate typed.String
	if err := candidate.Set(raw); err != nil {
		return err
	}
	*k = Key(candidate)
	return nil
}
