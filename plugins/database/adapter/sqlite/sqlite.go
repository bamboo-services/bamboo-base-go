package sqlite

import (
	"errors"
	"strings"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
	libtnbsqlite "github.com/libtnb/sqlite"
	"gorm.io/gorm"
)

func init() {
	dialect.Register(dialect.DriverSQLite, func() dialect.Adapter {
		return FromEnv()
	})
}

// adapter 是 SQLite 的 dialect.Adapter 内部实现。
type adapter struct {
	dsn string
	err *apperr.Error
}

// New 使用显式指定的 DSN 或文件路径构造 SQLite 数据库方言适配器 [dialect.Adapter]。
//
// 该函数为纯构造操作，仅在内存中保存 DSN，不建立数据库连接、不访问磁盘文件。
// 若 dsn 为空或仅包含空白字符，将以错误码 [dialect.AdapterConfigInvalid]
// 记录配置非法，并在 [dialect.Adapter.Open] 时返回，避免伪造无效连接句柄。
//
// SQLite 的 DSN 既可以是 ":memory:"，也可以是文件路径或带查询串的 URI，
// 本适配器不做任何 DSN 字符串格式校验，非空白 DSN 一律原样交给驱动处理。
func New(dsn string) dialect.Adapter {
	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" {
		return &adapter{
			err: configInvalid("SQLite 数据源路径不能为空", errors.New("sqlite: dsn cannot be empty")),
		}
	}
	return &adapter{
		dsn: trimmed,
	}
}

// FromEnv 从标准环境变量构造 SQLite 数据库方言适配器 [dialect.Adapter]。
//
// 函数优先读取 DATABASE_DSN；若未设置或为空，则读取 DATABASE_PATH；
// 若两者均未配置，默认采用内存数据库 ":memory:"。
// 若显式配置的路径仅包含空白字符，将以错误码 [dialect.AdapterConfigInvalid]
// 记录配置非法，并在 [dialect.Adapter.Open] 时返回明确的配置缺失错误。
func FromEnv() dialect.Adapter {
	if val, ok := env.Lookup(env.DatabaseDSN); ok {
		trimmed := strings.TrimSpace(val.String())
		if trimmed != "" {
			return &adapter{dsn: trimmed}
		}
	}

	dsn := ":memory:"
	if val, ok := env.Lookup(env.DatabasePath); ok {
		trimmed := strings.TrimSpace(val.String())
		if trimmed != "" {
			dsn = trimmed
		} else {
			return &adapter{
				err: configInvalid("SQLite 数据库路径不能为空", errors.New("sqlite: database path is empty")),
			}
		}
	}

	return &adapter{
		dsn: dsn,
	}
}

// Name 返回适配器驱动名称 "sqlite"。
func (a *adapter) Name() string {
	return string(dialect.DriverSQLite)
}

// Open 仅构造并返回 GORM Dialector，不在此建立数据库物理连接或 Ping。
//
// 若适配器实例为空，返回错误码 [dialect.AdapterInvalid]；
// 若在构造或环境变量解析阶段存在配置错误，返回错误码 [dialect.AdapterConfigInvalid]
// 并原样透传已记录的强类型错误；成功时返回的 nil 是真正的 nil 错误指针。
func (a *adapter) Open() (gorm.Dialector, *apperr.Error) {
	if a == nil {
		return nil, apperr.New(dialect.AdapterInvalid, "SQLite 适配器实例为空", errors.New("sqlite: adapter instance is nil"))
	}
	if a.err != nil {
		return nil, a.err
	}
	return libtnbsqlite.Open(a.dsn), nil
}

// configInvalid 构造 SQLite 适配器配置非法错误。
//
// msg 为安全的中文说明，cause 为已脱敏的底层原因，二者均不携带 DSN 明文或文件路径。
func configInvalid(msg string, cause error) *apperr.Error {
	return apperr.New(dialect.AdapterConfigInvalid, apperr.ErrMessage(msg), cause)
}
