package oracle

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
	gormoracle "github.com/oracle-samples/gorm-oracle/oracle"
	"gorm.io/gorm"
)

// DriverName 定义 Oracle 方言驱动在全局注册表中的标识名称。
const DriverName = "oracle"

// adapter 实现 dialect.Adapter 接口。
type adapter struct {
	dsn string
	err error
}

func init() {
	dialect.Register(dialect.DriverOracle, FromEnv)
}

// New 接收显式 DSN 字符串构造 Oracle 数据库方言适配器 [dialect.Adapter]。
//
// 该函数为纯构造操作，不建立底层网络连接。支持 URL 格式（oracle:// 或 oracledb://）、
// godror logfmt 格式以及 Easy Connect 简写格式。如果 DSN 为空或格式非法，
// 错误将延迟推迟至 [dialect.Adapter.Open] 方法调用时返回，且保证错误信息不泄露敏感密码。
func New(dsn string) dialect.Adapter {
	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" {
		return &adapter{
			err: errors.New("oracle: dsn cannot be empty"),
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

// FromEnv 从标准环境变量构造 Oracle 数据库方言适配器 [dialect.Adapter]。
//
// 函数优先读取 DATABASE_DSN；若未设置，则从环境变量中按 godror logfmt 格式拼装：
// user="..." password="..." connectString="host:port/service" [libDir="..."]
//
// 读取的环境变量:
//   - DATABASE_USER (默认 "system")
//   - DATABASE_PASS (默认 "")
//   - DATABASE_HOST (默认 "localhost")
//   - DATABASE_PORT (默认 "1521")
//   - DATABASE_SERVICE_NAME (默认 "ORCLPDB1")
//   - DATABASE_LIB_DIR (可选，指定时追加 libDir="...")
//
// 任何配置缺失或解析错误均保存于适配器实例中并推迟至 [dialect.Adapter.Open] 时返回，
// 避免在构造或 init 期产生网络 I/O 或不可控 panic。
func FromEnv() dialect.Adapter {
	if rawDSN, ok := env.Lookup(env.DatabaseDSN); ok && strings.TrimSpace(string(rawDSN)) != "" {
		trimmed := strings.TrimSpace(string(rawDSN))
		if err := validateDSN(trimmed); err != nil {
			return &adapter{err: err}
		}
		return &adapter{dsn: trimmed}
	}

	user := "system"
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

	port := "1521"
	if val, ok := env.Lookup(env.DatabasePort); ok {
		port = string(val)
	}

	service := "ORCLPDB1"
	if val, ok := env.Lookup(env.DatabaseServiceName); ok {
		service = string(val)
	}

	if strings.TrimSpace(port) == "" {
		return &adapter{err: errors.New("oracle: port cannot be empty")}
	}
	if strings.TrimSpace(host) == "" {
		return &adapter{err: errors.New("oracle: host cannot be empty")}
	}
	if strings.TrimSpace(service) == "" {
		return &adapter{err: errors.New("oracle: service name cannot be empty")}
	}

	connectString := fmt.Sprintf("%s:%s/%s", host, port, service)
	dsn := fmt.Sprintf(`user=%q password=%q connectString=%q`, user, pass, connectString)

	if val, ok := env.Lookup(env.DatabaseLibDir); ok && strings.TrimSpace(string(val)) != "" {
		dsn += fmt.Sprintf(` libDir=%q`, string(val))
	}

	if err := validateDSN(dsn); err != nil {
		return &adapter{err: err}
	}

	return &adapter{dsn: dsn}
}

// Name 返回适配器驱动名称 "oracle"。
func (a *adapter) Name() string {
	return DriverName
}

// Open 构造 gorm.Dialector 实例。
//
// 仅做 Dialector 构造，不执行数据库真实建连或网络操作。
// 若在构造或环境变量解析阶段存在配置错误，在此处返回保存的错误。
func (a *adapter) Open() (gorm.Dialector, error) {
	if a.err != nil {
		return nil, a.err
	}
	return gormoracle.Open(a.dsn), nil
}

// validateDSN 验证 Oracle DSN 的基本结构合法性，脱敏任何可能包含凭据的错误信息。
func validateDSN(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return errors.New("oracle: dsn cannot be empty")
	}

	// 1. 支持标准 URL 格式 (oracle:// 或 oracledb://)
	if strings.HasPrefix(trimmed, "oracle://") || strings.HasPrefix(trimmed, "oracledb://") {
		idx := strings.Index(trimmed, "://")
		rest := trimmed[idx+3:]
		if rest == "" {
			return errors.New("oracle: invalid DSN format")
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
			return errors.New("oracle: host cannot be empty")
		}
		return nil
	}

	// 2. 支持 godror logfmt 格式 (如 user="scott" password="..." connectString="...")
	// 或 key=value 格式 (如 user/password@host:port/service 或 connectString=...)
	if strings.Contains(trimmed, "connectString=") || strings.Contains(trimmed, "user=") {
		return nil
	}

	// 3. 支持简写 Easy Connect 格式 (如 user/pass@host:port/service)
	if strings.Contains(trimmed, "@") && strings.Contains(trimmed, "/") {
		return nil
	}

	return errors.New("oracle: invalid DSN format")
}
