package xOptDatabase

import (
	"context"
	"time"

	"gorm.io/gorm"

	xDB "github.com/bamboo-services/bamboo-base-go/common/database"
	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

// Driver 数据库驱动类型，标识业务端选择接入的 GORM 驱动实现。
//
// 定义于 common 层（[xDB.Driver]），此处为类型别名重导出以保持既有引用兼容。
// 零值空串等价于 [DriverNone]，表示不启用数据库。
type Driver = xDB.Driver

// 数据库驱动枚举常量，重导出自 common 层 [xDB]，保持 xOptDatabase.DriverMySQL 等旧引用兼容。
const (
	// DriverNone 表示不启用数据库。Register 会跳过数据库节点装配。
	DriverNone Driver = xDB.DriverNone

	// DriverMySQL 表示使用 MySQL 驱动（github.com/go-sql-driver/mysql）。
	DriverMySQL Driver = xDB.DriverMySQL

	// DriverPostgres 表示使用 PostgreSQL 驱动（github.com/jackc/pgx/v5）。
	DriverPostgres Driver = xDB.DriverPostgres

	// DriverSQLite 表示使用 SQLite 驱动（github.com/libtnb/sqlite，纯 Go 实现）。
	DriverSQLite Driver = xDB.DriverSQLite

	// DriverOracle 表示使用 Oracle 驱动（github.com/oracle-samples/gorm-oracle，底层 godror）。
	DriverOracle Driver = xDB.DriverOracle

	// DriverSQLServer 表示使用 SQL Server 驱动（gorm.io/driver/sqlserver，底层 microsoft/go-mssqldb）。
	DriverSQLServer Driver = xDB.DriverSQLServer
)

// DatabaseConfig 数据库配置，描述驱动类型、DSN、GORM Dialector、连接池参数、表迁移及数据初始化。
//
// 字段均为小写，仅通过 getter 暴露只读视图，避免下游直接修改内部状态。
// 构造请使用 [WithDriver] / [WithDialector] / [FromEnv]：
//   - [WithDriver] 指定驱动枚举 + DSN，打开连接时从 common 层驱动注册表解析 Dialector 工厂
//     （业务端需 import 对应驱动插件，如 plugins/database/mysql）；
//   - [WithDialector] 直接注入任意 GORM 驱动（TiDB / ClickHouse 等自定义驱动，无需注册）；
//   - [FromEnv] 从环境变量自动装配。
type DatabaseConfig struct {
	driver        Driver
	dsn           string
	dialector     gorm.Dialector // 显式注入的 GORM 驱动拨号器（WithDialector / 自定义驱动）
	common        CommonOptions
	tablePrefix   string
	migrateTables []interface{} // AutoMigrate 目标表，可多次叠加
	prepareFuncs  []PrepareFunc // 建表后数据初始化回调，按注册顺序执行
}

// Driver 返回数据库驱动类型。
func (c DatabaseConfig) Driver() Driver { return c.driver }

// Dialector 返回显式注入的 GORM 驱动拨号器。
//
// 非 nil 表示业务端通过 [WithDialector] 或自定义方式注入了驱动，打开连接时
// 优先使用它，不再依赖 common 层驱动注册表。nil 表示需按 [DatabaseConfig.Driver]
// 从注册表解析。
func (c DatabaseConfig) Dialector() gorm.Dialector { return c.dialector }

// Enabled 返回是否启用了数据库。
//
// 满足任一条件即视为启用：
//   - 设置了驱动枚举（非空且非 [DriverNone]）；
//   - 显式注入了 [gorm.Dialector]（自定义驱动场景）。
//
// 零值（未设置任何数据库选项）与显式声明 DriverNone 均视为未启用，
// Register 据此决定是否装配数据库节点。
func (c DatabaseConfig) Enabled() bool {
	return (c.driver != "" && c.driver != DriverNone) || c.dialector != nil
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

// DatabaseOption 是 [DatabaseConfig] 的二级选项，避免各驱动配置参数列表过长。
//
// 直接作用于 [DatabaseConfig] 整体，可承载驱动、DSN、连接池、迁移、数据初始化等跨字段选项。
type DatabaseOption func(*DatabaseConfig)

// WithDriver 显式指定驱动枚举与 DSN，替代旧的内置构造函数（MySQL / Postgres / SQLite / Oracle / SQLServer）。
//
// dsn 为完整的数据库连接串（各驱动格式见对应插件文档）。打开连接时，
// [DatabaseInit] 会从 common 层驱动注册表按 driver 枚举解析 Dialector 工厂，
// 因此业务端必须 import 对应驱动插件（如 plugins/database/mysql）触发注册。
//
// 如需注入无法通过注册表解析的自定义驱动，改用 [WithDialector]。
func WithDriver(d Driver, dsn string, opts ...DatabaseOption) DatabaseOption {
	return func(c *DatabaseConfig) {
		c.driver = d
		c.dsn = dsn
		applyDatabase(c, opts...)
	}
}

// WithDialector 直接注入 GORM 驱动拨号器，用于接入自定义驱动（TiDB / ClickHouse 等）。
//
// 注入后 [DatabaseInit] 优先使用该 Dialector 打开连接，不依赖驱动注册表与插件。
// 也可由驱动子模块组合使用。与 [WithDriver] 同时使用时，dialector 优先级更高。
func WithDialector(d gorm.Dialector) DatabaseOption {
	return func(c *DatabaseConfig) { c.dialector = d }
}

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

// applyDatabase 将二级选项应用到 DatabaseConfig，供各配置构造函数复用。
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
//  2. 否则按 DATABASE_DRIVER 调用对应驱动的 env DSN 拼装函数（见 fromenv_mysql.go / fromenv_postgres.go 等）
//  3. DATABASE_DRIVER 为空或 "none" 时，闭包不修改 [DatabaseConfig]，等价于不启用数据库
//
// DATABASE_PREFIX 若已设置，会同步写入 TablePrefix，由 DatabaseInit 应用到 GORM NamingStrategy。
// 连接池参数暂未从环境变量读取（保持 env 列表精简），如需调整请配合
// [WithMaxOpenConns] 等二级选项显式设置。
//
// 环境变量读取被延迟到 Option 闭包执行时（即 [xOption.Apply] 在 [xReg.Register] 内部
// 调用阶段），此时 configInit() 已通过 godotenv.Load() 加载 .env，保证 FromEnv 可用于
// 业务侧在 Register 调用前构造 opts 切片的常见写法。
//
// 注意：FromEnv 仅装配驱动枚举与 DSN 字符串，打开连接时仍需按驱动枚举从 common 层
// 驱动注册表解析 Dialector 工厂，因此业务端必须 import 对应驱动插件
// （如 plugins/database/mysql）触发注册，否则启动时会在 DatabaseInit 阶段报错。
//
// 未启用数据库时（driver 为空或 none），返回的闭包不修改 [DatabaseConfig]，
// [DatabaseConfig.Enabled] 返回 false，Register 阶段会跳过数据库节点注册。
// opts 中的二级选项会与 env 装配结果一起透传到 [DatabaseConfig]，可与 [WithAutoMigrate] / [WithPrepare]
// 等叠加使用。
func FromEnv(opts ...DatabaseOption) DatabaseOption {
	return func(c *DatabaseConfig) {
		driver := Driver(xEnv.GetEnvString(xEnv.DatabaseDriver, "none"))
		// 驱动未启用时仍应用显式传入的 opts（如 WithDialector 注入自定义驱动），
		// 使 FromEnv(WithDialector(...)) 的组合形式可用；未传 opts 时行为与原来一致。
		if driver != "" && driver != DriverNone {
			dsn := xEnv.GetEnvString(xEnv.DatabaseDSN, "")
			if dsn == "" {
				dsn = resolveDSNFromEnv(driver)
			}
			c.driver = driver
			c.dsn = dsn
			c.tablePrefix = xEnv.GetEnvString(xEnv.DatabasePrefix, "")
		}
		applyDatabase(c, opts...)
	}
}

// resolveDSNFromEnv 根据驱动类型从环境变量拼装 DSN。
//
// 若 DATABASE_DSN 已设置则直接返回（覆盖分项配置）；否则委托给各驱动的专属拼装函数。
// 未知驱动返回空串，不 panic——由 [DatabaseInit] 在打开连接阶段统一报"不支持的数据库驱动"。
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
		return ""
	}
}
