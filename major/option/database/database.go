package xOptDatabase

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

// Driver 数据库驱动类型，标识框架内置数据库的实现选择。
//
// 零值空串等价于 [DriverNone]，表示不启用内置数据库实现。
type Driver string

const (
	// DriverNone 表示不启用内置数据库实现。Register 会跳过数据库节点装配。
	DriverNone Driver = "none"

	// DriverMySQL 表示使用 MySQL 驱动（github.com/go-sql-driver/mysql）。
	DriverMySQL Driver = "mysql"

	// DriverPostgres 表示使用 PostgreSQL 驱动（github.com/jackc/pgx/v5）。
	DriverPostgres Driver = "postgres"

	// DriverSQLite 表示使用 SQLite 驱动（github.com/glebarez/sqlite）。
	DriverSQLite Driver = "sqlite"

	// DriverOracle 表示使用 Oracle 驱动（github.com/oracle-samples/gorm-oracle，底层 godror）。
	DriverOracle Driver = "oracle"

	// DriverSQLServer 表示使用 SQL Server 驱动（gorm.io/driver/sqlserver，底层 microsoft/go-mssqldb）。
	DriverSQLServer Driver = "sqlserver"
)

// DatabaseConfig 数据库配置，描述驱动类型、DSN、连接池参数、表迁移及数据初始化。
//
// 字段均为小写，仅通过 getter 暴露只读视图，避免下游直接修改内部状态。
// 构造请使用对应驱动的构造函数（[MySQL] / [Postgres] / [SQLite] / [Oracle] / [SQLServer]）或 [FromEnv]。
type DatabaseConfig struct {
	driver         Driver
	dsn            string
	common         CommonOptions
	tablePrefix    string
	migrateTables  []interface{} // AutoMigrate 目标表，可多次叠加
	prepareFuncs   []PrepareFunc // 建表后数据初始化回调，按注册顺序执行
}

// Driver 返回数据库驱动类型。
func (c DatabaseConfig) Driver() Driver { return c.driver }

// Enabled 返回是否启用了内置数据库实现。
//
// 零值（未设置任何数据库选项）与显式声明 DriverNone 均视为未启用，
// Register 据此决定是否装配内置数据库节点。
func (c DatabaseConfig) Enabled() bool {
	return c.driver != "" && c.driver != DriverNone
}

// DSN 返回数据库连接串。
func (c DatabaseConfig) DSN() string { return c.dsn }

// Common 返回数据库通用连接池参数。
func (c DatabaseConfig) Common() CommonOptions { return c.common }

// TablePrefix 返回数据库表名前缀，由 GORM NamingStrategy.TablePrefix 使用。
//
// 空串表示无前缀。
func (c DatabaseConfig) TablePrefix() string { return c.tablePrefix }

// AutoMigrateTables 返回 AutoMigrate 声明的目标表列表。
//
// 由 [WithAutoMigrate] 叠加，在 [DatabaseInit] 中建连后自动执行。
func (c DatabaseConfig) AutoMigrateTables() []interface{} { return c.migrateTables }

// Prepares 返回建表后数据初始化回调列表。
//
// 由 [WithPrepare] 注册，在 AutoMigrate 成功后按注册顺序执行。
// 任一回调返回 error 会中断启动。
func (c DatabaseConfig) Prepares() []PrepareFunc { return c.prepareFuncs }

// CommonOptions 数据库通用连接池参数，与具体驱动无关。
//
// 字段语义与 gorm.io/gorm 的 DB 连接池配置对齐。
type CommonOptions struct {
	MaxOpenConns    int           // 最大打开连接数，0 表示使用默认值
	MaxIdleConns    int           // 最大空闲连接数，0 表示使用默认值
	ConnMaxLifetime time.Duration // 连接最大存活时间
	ConnMaxIdleTime time.Duration // 连接最大空闲时间
}

// DatabaseOption 是 [DatabaseConfig] 的二级选项，避免各驱动构造函数参数列表过长。
//
// 直接作用于 [DatabaseConfig] 整体，可承载连接池、迁移、数据初始化等跨字段选项。
type DatabaseOption func(*DatabaseConfig)

// WithMaxOpenConns 设置最大打开连接数。
func WithMaxOpenConns(n int) DatabaseOption {
	return func(c *DatabaseConfig) { c.common.MaxOpenConns = n }
}

// WithMaxIdleConns 设置最大空闲连接数。
func WithMaxIdleConns(n int) DatabaseOption {
	return func(c *DatabaseConfig) { c.common.MaxIdleConns = n }
}

// WithConnMaxLifetime 设置连接最大存活时间。
func WithConnMaxLifetime(d time.Duration) DatabaseOption {
	return func(c *DatabaseConfig) { c.common.ConnMaxLifetime = d }
}

// WithConnMaxIdleTime 设置连接最大空闲时间。
func WithConnMaxIdleTime(d time.Duration) DatabaseOption {
	return func(c *DatabaseConfig) { c.common.ConnMaxIdleTime = d }
}

// WithTablePrefix 显式设置数据库表名前缀。
//
// [FromEnv] 已自动从 DATABASE_PREFIX 读取，此函数供显式构造非 env 路径时覆盖。
func WithTablePrefix(prefix string) DatabaseOption {
	return func(c *DatabaseConfig) { c.tablePrefix = prefix }
}

// WithAutoMigrate 声明 AutoMigrate 目标表。
//
// 可多次调用叠加，每次追加的表都会并入迁移列表。
// [DatabaseInit] 在建连成功后按声明顺序执行 db.AutoMigrate(tables...)。
func WithAutoMigrate(tables ...interface{}) DatabaseOption {
	return func(c *DatabaseConfig) {
		c.migrateTables = append(c.migrateTables, tables...)
	}
}

// PrepareFunc 数据库建表后的数据初始化回调签名。
//
// ctx 含已注册组件；db 为已建连且已完成 AutoMigrate 的 *gorm.DB。
// 返回 error 会中断启动流程。
type PrepareFunc func(ctx context.Context, db *gorm.DB) error

// WithPrepare 注册一个或多个建表后数据初始化回调。
//
// 可多次调用叠加，回调按注册顺序在 AutoMigrate 成功后依次执行。
// 传入 nil 回调会被静默跳过。任一回调返回 error 会中断启动。
// 支持一次传入多个回调，底层循环追加到 prepareFuncs 列表。
func WithPrepare(fns ...PrepareFunc) DatabaseOption {
	return func(c *DatabaseConfig) {
		for _, fn := range fns {
			if fn != nil {
				c.prepareFuncs = append(c.prepareFuncs, fn)
			}
		}
	}
}

// applyDatabase 将二级选项应用到 DatabaseConfig，供各驱动构造函数复用。
func applyDatabase(target *DatabaseConfig, opts ...DatabaseOption) {
	for _, o := range opts {
		if o != nil {
			o(target)
		}
	}
}

// FromEnv 从环境变量构造数据库配置的 [DatabaseOption]。
//
// 读取顺序与优先级:
//  1. 若设置了 DATABASE_DSN，则直接使用该完整连接串，驱动由 DATABASE_DRIVER 决定
//  2. 否则按 DATABASE_DRIVER 调用对应驱动的 env DSN 拼装函数（见 mysql.go / postgres.go / sqlite.go / oracle.go / sqlserver.go）
//  3. DATABASE_DRIVER 为空或 "none" 时返回 nil，表示不启用内置数据库
//
// DATABASE_PREFIX 若已设置，会同步写入 TablePrefix，由 DatabaseInit 应用到 GORM NamingStrategy。
// 连接池参数暂未从环境变量读取（保持 env 列表精简），如需调整请配合
// [WithMaxOpenConns] 等二级选项显式设置。
//
// 该函数依赖 .env 已在 Register 阶段通过 godotenv 加载完成。
//
// 返回的 DatabaseOption 可能为 nil（未启用内置数据库），父包 [option.WithDatabase] 会跳过 nil 选项。
// opts 中的二级选项会与 env 装配结果一起透传到 [DatabaseConfig]，可与 [WithAutoMigrate] / [WithPrepare]
// 等叠加使用。
func FromEnv(opts ...DatabaseOption) DatabaseOption {
	driver := Driver(xEnv.GetEnvString(xEnv.DatabaseDriver, "none"))
	if driver == "" || driver == DriverNone {
		return nil
	}
	dsn := xEnv.GetEnvString(xEnv.DatabaseDSN, "")
	if dsn == "" {
		dsn = resolveDSNFromEnv(driver)
	}
	prefix := xEnv.GetEnvString(xEnv.DatabasePrefix, "")
	return func(c *DatabaseConfig) {
		c.driver = driver
		c.dsn = dsn
		c.tablePrefix = prefix
		applyDatabase(c, opts...)
	}
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
	case DriverOracle:
		return OracleFromEnv()
	case DriverSQLServer:
		return SQLServerFromEnv()
	case DriverSQLite:
		return SQLiteFromEnv()
	default:
		panic(fmt.Sprintf("不支持的数据库驱动: %s", driver))
	}
}