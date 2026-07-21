package option

import (
	xOptDatabase "github.com/bamboo-services/bamboo-base-go/major/option/database"
)

// 以下类型为 [xOptDatabase] 子包类型的别名重导出，保持父包对外 API 兼容，
// 使 init 包与业务侧可继续通过 xOption.DatabaseConfig / xOption.Driver 等访问。
//
// 构造函数（MySQL / Postgres / SQLite / FromEnv / WithAutoMigrate 等）已迁移至 [xOptDatabase] 子包，
// 业务侧应使用 xOption.WithDatabase(xOptDatabase.MySQL(...)) 的两层调用形态。
type (
	// Driver 数据库驱动类型，详见 [xOptDatabase.Driver]。
	Driver = xOptDatabase.Driver

	// DatabaseConfig 数据库配置，详见 [xOptDatabase.DatabaseConfig]。
	DatabaseConfig = xOptDatabase.DatabaseConfig

	// DatabaseOption 数据库二级选项，详见 [xOptDatabase.DatabaseOption]。
	DatabaseOption = xOptDatabase.DatabaseOption

	// CommonOptions 数据库通用连接池参数，详见 [xOptDatabase.CommonOptions]。
	CommonOptions = xOptDatabase.CommonOptions

	// PrepareFunc 建表后数据初始化回调签名，详见 [xOptDatabase.PrepareFunc]。
	PrepareFunc = xOptDatabase.PrepareFunc
)

// 数据库驱动常量重导出，保持 xOption.DriverMySQL 等旧引用兼容。
const (
	DriverMySQL     = xOptDatabase.DriverMySQL
	DriverPostgres  = xOptDatabase.DriverPostgres
	DriverSQLite    = xOptDatabase.DriverSQLite
	DriverOracle    = xOptDatabase.DriverOracle
	DriverSQLServer = xOptDatabase.DriverSQLServer
	DriverNone      = xOptDatabase.DriverNone
)

// WithDatabase 将 [xOptDatabase.DatabaseOption] 包裹为顶层 [Option]，供 Register 使用。
//
// 该函数对标 [WithCache]，是 database 两层设计的顶层入口。
// 内部逐个执行传入的 DatabaseOption 修改 [DatabaseConfig]，完成后写入聚合 Config。
// nil DatabaseOption 会被跳过，支持条件构造。
// 使用示例：
//
//	// 显式 MySQL DSN
//	xOption.WithDatabase(xOptDatabase.MySQL("user:pass@tcp(localhost:3306)/db"))
//	// 从环境变量装配 + 声明迁移与数据初始化
//	xOption.WithDatabase(
//	    xOptDatabase.FromEnv(),
//	    xOptDatabase.WithAutoMigrate(&entity.Role{}, &entity.User{}),
//	    xOptDatabase.WithPrepare(seedRoles),
//	)
func WithDatabase(opts ...xOptDatabase.DatabaseOption) Option {
	return func(c *Config) {
		for _, o := range opts {
			if o != nil {
				o(&c.database)
			}
		}
	}
}
