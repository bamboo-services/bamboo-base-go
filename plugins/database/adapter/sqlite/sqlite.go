package sqlite

import (
	"errors"
	"strings"

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
	err error
}

// New 使用显式指定的 DSN 构造 SQLite dialect.Adapter。
//
// 仅保存 DSN，不在此处建立数据库连接或验证文件。若 dsn 为空，
// 则将错误延迟保存至 Open() 时返回，避免伪造有效连接串。
func New(dsn string) dialect.Adapter {
	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" {
		return &adapter{
			err: errors.New("sqlite: dsn cannot be empty"),
		}
	}
	return &adapter{
		dsn: trimmed,
	}
}

// FromEnv 从环境变量构造 SQLite dialect.Adapter。
//
// 优先读取 DATABASE_DSN，若设置且非空则直接使用；
// 未设置或为空时读取 DATABASE_PATH（未设置或为空时默认 ":memory:"）。
// 若最终解析得到的路径为空（如显式配置仅含空白），Open() 将返回明确的配置缺失错误。
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
				err: errors.New("sqlite: database path is empty"),
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
// 若在构造或环境变量解析阶段存在配置错误，在此处返回保存的错误。
func (a *adapter) Open() (gorm.Dialector, error) {
	if a.err != nil {
		return nil, a.err
	}
	return libtnbsqlite.Open(a.dsn), nil
}
