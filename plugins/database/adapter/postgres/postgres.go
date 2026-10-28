package postgres

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DriverName 定义 PostgreSQL 方言驱动名称。
const DriverName = "postgres"

// adapter 实现 dialect.Adapter 接口。
type adapter struct {
	dsn string
	err error
}

func init() {
	dialect.Register(dialect.DriverPostgres, FromEnv)
}

// New 接收显式 DSN 构造 PostgreSQL dialect.Adapter。
//
// 该函数不建立网络连接。如果 DSN 为空或格式非法，将错误推迟至 Open 返回，
// 且保证返回的错误信息不包含敏感密码。
func New(dsn string) dialect.Adapter {
	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" {
		return &adapter{
			err: errors.New("postgres: dsn cannot be empty"),
		}
	}

	if err := validateDSN(trimmed); err != nil {
		return &adapter{
			err: err,
		}
	}

	return &adapter{
		dsn: trimmed,
	}
}

// FromEnv 从环境变量构造 PostgreSQL dialect.Adapter。
//
// 优先使用 DATABASE_DSN；若未设置，则从 DATABASE_HOST, DATABASE_USER,
// DATABASE_PASS, DATABASE_NAME, DATABASE_PORT, DATABASE_TIMEZONE 拼装。
// 格式完全对齐 major/option/database/fromenv_postgres.go:
// host=xxx user=xxx password=xxx dbname=xxx port=xxx TimeZone=xxx sslmode=disable。
// 若字段中包含空格、单引号或反斜杠等特殊字符，安全使用单引号转义包裹。
// 缺配置或解析错误保存于 adapter 中并在 Open 时返回，避免在构造或 init 期产生网络连接或伪造有效 DSN。
func FromEnv() dialect.Adapter {
	if rawDSN, ok := env.Lookup(env.DatabaseDSN); ok && strings.TrimSpace(string(rawDSN)) != "" {
		trimmed := strings.TrimSpace(string(rawDSN))
		if err := validateDSN(trimmed); err != nil {
			return &adapter{err: err}
		}
		return &adapter{dsn: trimmed}
	}

	host := "localhost"
	if val, ok := env.Lookup(env.DatabaseHost); ok {
		host = string(val)
	}

	user := "postgres"
	if val, ok := env.Lookup(env.DatabaseUser); ok {
		user = string(val)
	}

	pass := ""
	if val, ok := env.Lookup(env.DatabasePass); ok {
		pass = string(val)
	}

	name := "postgres"
	if val, ok := env.Lookup(env.DatabaseName); ok {
		name = string(val)
	}

	port := "5432"
	if val, ok := env.Lookup(env.DatabasePort); ok {
		port = string(val)
	}

	timezone := "Asia/Shanghai"
	if val, ok := env.Lookup(env.DatabaseTimezone); ok {
		timezone = string(val)
	}

	if strings.TrimSpace(port) == "" {
		return &adapter{err: errors.New("postgres: port cannot be empty")}
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s TimeZone=%s sslmode=disable",
		escapeKeyValue(host),
		escapeKeyValue(user),
		escapeKeyValue(pass),
		escapeKeyValue(name),
		escapeKeyValue(port),
		escapeKeyValue(timezone),
	)

	if err := validateDSN(dsn); err != nil {
		return &adapter{err: err}
	}

	return &adapter{dsn: dsn}
}

// Name 返回适配器驱动名称。
func (a *adapter) Name() string {
	return DriverName
}

// Open 构造 gorm.Dialector 实例。
//
// 仅做 Dialector 构造，不执行数据库真实建连或网络操作。
func (a *adapter) Open() (gorm.Dialector, error) {
	if a.err != nil {
		return nil, a.err
	}
	return gormpostgres.Open(a.dsn), nil
}

// escapeKeyValue 对 key=value 格式中可能包含空格、单引号或反斜杠的值进行安全转义与加单引号。
func escapeKeyValue(val string) string {
	if val == "" {
		return ""
	}
	if !strings.ContainsAny(val, " '\"\\") {
		return val
	}
	escaped := strings.ReplaceAll(val, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	return "'" + escaped + "'"
}

// validateDSN 验证 DSN 结构合法性，脱敏任何可能包含凭据的错误信息。
func validateDSN(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return errors.New("postgres: dsn cannot be empty")
	}

	// 支持 URL 格式 (postgres:// 或 postgresql://)
	if strings.HasPrefix(trimmed, "postgres://") || strings.HasPrefix(trimmed, "postgresql://") {
		idx := strings.Index(trimmed, "://")
		scheme := trimmed[:idx]
		rest := trimmed[idx+3:]
		if rest == "" {
			return errors.New("postgres: invalid DSN format")
		}
		atIdx := strings.LastIndex(rest, "@")
		hostPart := rest
		if atIdx != -1 {
			hostPart = rest[atIdx+1:]
		}
		slashIdx := strings.Index(hostPart, "/")
		qIdx := strings.Index(hostPart, "?")
		endIdx := len(hostPart)
		if slashIdx != -1 && slashIdx < endIdx {
			endIdx = slashIdx
		}
		if qIdx != -1 && qIdx < endIdx {
			endIdx = qIdx
		}
		host := strings.TrimSpace(hostPart[:endIdx])
		if host == "" {
			return errors.New("postgres: host cannot be empty")
		}
		_ = scheme
		return nil
	}

	// 支持 key=value 连接串格式
	if strings.Contains(trimmed, "=") {
		return nil
	}

	return errors.New("postgres: invalid DSN format")
}
