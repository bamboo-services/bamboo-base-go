package database

import (
	"time"

	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

// Driver 数据库驱动类型，标识框架内置数据库的实现选择。
//
// 零值空串等价于 [DriverNone]，表示不启用内置数据库实现。
type Driver string

const (
	// DriverMySQL 使用 MySQL 作为数据库。
	DriverMySQL Driver = "mysql"
	// DriverPostgres 使用 PostgreSQL 作为数据库。
	DriverPostgres Driver = "postgres"
	// DriverSQLite 使用 SQLite 作为数据库，通常仅用于开发与测试环境。
	DriverSQLite Driver = "sqlite"
	// DriverNone 不启用内置数据库实现，业务侧可自行通过 Register 注册数据库节点。
	DriverNone Driver = "none"
)

// Config 数据库配置，描述驱动类型、DSN 及通用连接池参数。
//
// 各驱动共享统一的 DSN 字符串与通用参数，驱动类型仅决定方言与底层打开方式。
// 字段均为小写，仅通过 getter 暴露只读视图，避免下游直接修改内部状态。
// 构造请使用对应驱动的构造函数（[MySQL] / [Postgres] / [SQLite]）或 [FromEnv]。
type Config struct {
	driver      Driver
	dsn         string
	common      CommonOptions
	tablePrefix string
}

// Driver 返回数据库驱动类型。
func (c Config) Driver() Driver { return c.driver }

// Enabled 返回是否启用了内置数据库实现。
//
// 零值（未设置任何数据库选项）与显式声明 DriverNone 均视为未启用，
// Runner 据此决定是否装配内置数据库节点。
func (c Config) Enabled() bool {
	return c.driver != "" && c.driver != DriverNone
}

// DSN 返回数据库连接串。
func (c Config) DSN() string { return c.dsn }

// Common 返回数据库通用连接池参数。
func (c Config) Common() CommonOptions { return c.common }

// TablePrefix 返回数据库表名前缀，由 GORM NamingStrategy.TablePrefix 使用。
//
// 空串表示无前缀。
func (c Config) TablePrefix() string { return c.tablePrefix }

// CommonOptions 数据库通用连接池参数，与具体驱动无关。
//
// 字段语义与 gorm.io/gorm 的 DB 连接池配置对齐。
type CommonOptions struct {
	MaxOpenConns    int           // 最大打开连接数，0 表示使用默认值
	MaxIdleConns    int           // 最大空闲连接数，0 表示使用默认值
	ConnMaxLifetime time.Duration // 连接最大存活时间
	ConnMaxIdleTime time.Duration // 连接最大空闲时间
}

// CommonOption 是 [CommonOptions] 的二级选项，避免各驱动构造函数参数列表过长。
type CommonOption func(*CommonOptions)

// WithMaxOpenConns 设置最大打开连接数。
func WithMaxOpenConns(n int) CommonOption { return func(o *CommonOptions) { o.MaxOpenConns = n } }

// WithMaxIdleConns 设置最大空闲连接数。
func WithMaxIdleConns(n int) CommonOption { return func(o *CommonOptions) { o.MaxIdleConns = n } }

// WithConnMaxLifetime 设置连接最大存活时间。
func WithConnMaxLifetime(d time.Duration) CommonOption { return func(o *CommonOptions) { o.ConnMaxLifetime = d } }

// WithConnMaxIdleTime 设置连接最大空闲时间。
func WithConnMaxIdleTime(d time.Duration) CommonOption { return func(o *CommonOptions) { o.ConnMaxIdleTime = d } }

// applyCommon 将二级选项应用到 CommonOptions，供各驱动构造函数复用。
func applyCommon(target *CommonOptions, opts ...CommonOption) {
	for _, o := range opts {
		if o != nil {
			o(target)
		}
	}
}

// FromEnv 从环境变量构造数据库配置。
//
// 读取顺序与优先级:
//  1. 若设置了 DATABASE_DSN，则直接使用该完整连接串，驱动由 DATABASE_DRIVER 决定
//  2. 否则按 DATABASE_DRIVER 调用对应驱动的 env DSN 拼装函数（见 mysql.go / postgres.go / sqlite.go）
//  3. DATABASE_DRIVER 为空或 "none" 时返回零值 Config 与 false，表示不启用内置数据库
//
// DATABASE_PREFIX 若已设置，会同步写入 TablePrefix，由 DatabaseInit 应用到 GORM NamingStrategy。
// 连接池参数暂未从环境变量读取（保持 env 列表精简），如需调整请配合
// [WithMaxOpenConns] 等二级选项显式设置。
//
// 该函数依赖 .env 已在 Register 阶段通过 godotenv 加载完成。
//
// 返回值:
//   - Config: 装配完成的数据库配置
//   - bool: 是否启用内置数据库（false 时调用方应跳过装配）
func FromEnv(opts ...CommonOption) (Config, bool) {
	driver := Driver(xEnv.GetEnvString(xEnv.DatabaseDriver, "none"))
	if driver == "" || driver == DriverNone {
		return Config{}, false
	}

	cfg := Config{
		driver:      driver,
		dsn:         resolveDSNFromEnv(driver),
		tablePrefix: xEnv.GetEnvString(xEnv.DatabasePrefix, ""),
	}
	applyCommon(&cfg.common, opts...)
	return cfg, true
}

// resolveDSNFromEnv 根据驱动类型从环境变量拼装 DSN。
//
// 若 DATABASE_DSN 已设置则直接返回（覆盖分项配置）；否则委托给各驱动的专属拼装函数。
func resolveDSNFromEnv(driver Driver) string {
	if dsn := xEnv.GetEnvString(xEnv.DatabaseDSN, ""); dsn != "" {
		return dsn
	}
	switch driver {
	case DriverMySQL:
		return MySQLFromEnv()
	case DriverPostgres:
		return PostgresFromEnv()
	case DriverSQLite:
		return SQLiteFromEnv()
	default:
		return ""
	}
}
