package sqlserver

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
	gormsqlserver "gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

// DriverName 定义 SQL Server 方言驱动在全局注册表中的标识名称。
const DriverName = "sqlserver"

// adapter 实现 dialect.Adapter 接口。
type adapter struct {
	dsn string
	err error
}

func init() {
	dialect.Register(dialect.DriverSQLServer, FromEnv)
}

// New 接收显式 DSN 字符串构造 SQL Server 数据库方言适配器 [dialect.Adapter]。
//
// 该函数为纯构造操作，不建立底层网络连接。要求 DSN 采用 sqlserver:// 协议格式。
// 如果 DSN 为空或格式非法，错误将延迟推迟至 [dialect.Adapter.Open] 方法调用时返回，
// 且保证返回的错误信息不包含敏感密码。
func New(dsn string) dialect.Adapter {
	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" {
		return &adapter{
			err: errors.New("sqlserver: dsn cannot be empty"),
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

// FromEnv 从标准环境变量构造 SQL Server 数据库方言适配器 [dialect.Adapter]。
//
// 函数优先读取 DATABASE_DSN；若未设置，则从 DATABASE_USER (默认 "sa")、DATABASE_PASS (默认 "")、
// DATABASE_HOST (默认 "localhost")、DATABASE_PORT (默认 "1433")、DATABASE_NAME (默认 "master") 拼装。
// 凭据使用 [net/url.URL] 及 [net/url.UserPassword] 进行标准编码，并使用 [net.JoinHostPort]
// 规范处理包含 IPv6 等各种 Host 格式。
// 任何解析或配置错误均保存于适配器实例中并在 [dialect.Adapter.Open] 时返回，避免在构造或 init 期产生网络连接或 panic。
func FromEnv() dialect.Adapter {
	if rawDSN, ok := env.Lookup(env.DatabaseDSN); ok && strings.TrimSpace(string(rawDSN)) != "" {
		trimmed := strings.TrimSpace(string(rawDSN))
		if err := validateDSN(trimmed); err != nil {
			return &adapter{err: err}
		}
		return &adapter{dsn: trimmed}
	}

	user := "sa"
	if val, ok := env.Lookup(env.DatabaseUser); ok {
		user = string(val)
	}

	pass := ""
	if val, ok := env.Lookup(env.DatabasePass); ok {
		pass = string(val)
	}

	host := "localhost"
	if val, ok := env.Lookup(env.DatabaseHost); ok {
		host = string(val)
	}

	port := "1433"
	if val, ok := env.Lookup(env.DatabasePort); ok {
		port = string(val)
	}

	dbName := "master"
	if val, ok := env.Lookup(env.DatabaseName); ok {
		dbName = string(val)
	}

	// 端口与主机格式校验
	if strings.TrimSpace(port) == "" {
		return &adapter{err: errors.New("sqlserver: port cannot be empty")}
	}
	trimmedHost := strings.TrimSpace(host)
	if trimmedHost == "" {
		return &adapter{err: errors.New("sqlserver: host cannot be empty")}
	}

	u := &url.URL{
		Scheme: "sqlserver",
		User:   url.UserPassword(user, pass),
		Host:   net.JoinHostPort(trimmedHost, strings.TrimSpace(port)),
	}
	q := u.Query()
	q.Set("database", dbName)
	u.RawQuery = q.Encode()

	dsn := u.String()
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
	return gormsqlserver.Open(a.dsn), nil
}

// validateDSN 验证 DSN 结构合法性，脱敏任何可能包含凭据的错误信息。
func validateDSN(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("sqlserver: invalid DSN format")
	}
	if u.Scheme != "sqlserver" {
		return fmt.Errorf("sqlserver: unsupported scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return errors.New("sqlserver: host cannot be empty")
	}
	return nil
}
