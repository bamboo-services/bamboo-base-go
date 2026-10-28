package mysql

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DriverName 定义 MySQL 方言驱动名称。
const DriverName = "mysql"

// adapter 是 MySQL 的 dialect.Adapter 内部实现。
type adapter struct {
	dsn string
	err error
}

func init() {
	dialect.Register(dialect.DriverMySQL, FromEnv)
}

// New 接收显式 DSN 构造 MySQL dialect.Adapter。
//
// 该函数不建立网络连接，如果 DSN 为空或格式无效，将错误推迟至 Open 返回，
// 且保证返回的错误信息不包含敏感密码。
func New(dsn string) dialect.Adapter {
	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" {
		return &adapter{
			err: errors.New("mysql: dsn cannot be empty"),
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

// FromEnv 从环境变量构造 MySQL dialect.Adapter。
//
// 优先使用 DATABASE_DSN；若未设置，则从 DATABASE_USER (默认 "root"), DATABASE_PASS (默认 ""),
// DATABASE_HOST (默认 "localhost"), DATABASE_PORT (默认 "3306"), DATABASE_NAME (默认 "bamboo"),
// DATABASE_CHARSET (默认 "utf8mb4") 拼装标准 DSN：
// user:pass@tcp(host:port)/name?charset=xxx&parseTime=True&loc=Local
//
// 任何解析或配置错误均保存于 adapter 中并在 Open 时返回，避免在构造或 init 期产生网络连接或 panic。
func FromEnv() dialect.Adapter {
	if rawDSN, ok := env.Lookup(env.DatabaseDSN); ok && strings.TrimSpace(string(rawDSN)) != "" {
		trimmed := strings.TrimSpace(string(rawDSN))
		if err := validateDSN(trimmed); err != nil {
			return &adapter{err: err}
		}
		return &adapter{dsn: trimmed}
	}

	user := "root"
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

	port := "3306"
	if val, ok := env.Lookup(env.DatabasePort); ok {
		port = string(val)
	}

	dbName := "bamboo"
	if val, ok := env.Lookup(env.DatabaseName); ok {
		dbName = string(val)
	}

	charset := "utf8mb4"
	if val, ok := env.Lookup(env.DatabaseCharset); ok {
		charset = string(val)
	}

	if strings.TrimSpace(port) == "" {
		return &adapter{err: errors.New("mysql: port cannot be empty")}
	}

	if strings.TrimSpace(host) == "" {
		return &adapter{err: errors.New("mysql: host cannot be empty")}
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local",
		user,
		pass,
		host,
		port,
		dbName,
		charset,
	)

	if err := validateDSN(dsn); err != nil {
		return &adapter{err: err}
	}

	return &adapter{dsn: dsn}
}

// Name 返回适配器驱动名称 "mysql"。
func (a *adapter) Name() string {
	return DriverName
}

// Open 构造并返回 gorm.Dialector 实例。
//
// 仅做 Dialector 构造，不执行数据库真实建连或网络操作。
// 若在构造或环境变量解析阶段存在配置错误，在此处返回保存的错误。
func (a *adapter) Open() (gorm.Dialector, error) {
	if a.err != nil {
		return nil, a.err
	}
	return gormmysql.Open(a.dsn), nil
}

// validateDSN 验证 MySQL DSN 的基本结构合法性，脱敏任何可能包含凭据的错误信息。
func validateDSN(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return errors.New("mysql: dsn cannot be empty")
	}

	atIdx := strings.LastIndex(trimmed, "@")
	if atIdx == -1 {
		return errors.New("mysql: invalid DSN format: missing '@'")
	}

	right := trimmed[atIdx+1:]
	slashIdx := strings.Index(right, "/")
	if slashIdx == -1 {
		return errors.New("mysql: invalid DSN format: missing '/'")
	}

	netAddr := right[:slashIdx]
	if !strings.HasPrefix(netAddr, "tcp(") || !strings.HasSuffix(netAddr, ")") {
		return errors.New("mysql: invalid DSN format: expected tcp(host:port)")
	}

	addr := strings.TrimSuffix(strings.TrimPrefix(netAddr, "tcp("), ")")
	if strings.TrimSpace(addr) == "" {
		return errors.New("mysql: invalid DSN format: empty tcp address")
	}

	return nil
}
