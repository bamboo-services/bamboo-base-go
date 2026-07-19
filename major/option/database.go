package option

import (
	xOptionDB "github.com/bamboo-services/bamboo-base-go/major/option/database"
)

// 以下类型为 [xOptionDB] 子包类型的别名重导出，保持父包对外 API 兼容，
// 使 init 包与业务侧可继续通过 xOption.DatabaseConfig / xOption.Driver 等访问。
//
// 构造函数（MySQL / Postgres / SQLite / FromEnv / WithAutoMigrate 等）已迁移至 [xOptionDB] 子包，
// 业务侧应使用 xOption.WithDatabase(xOptionDB.MySQL(...)) 的两层调用形态。
type (
	// Driver 数据库驱动类型，详见 [xOptionDB.Driver]。
	Driver = xOptionDB.Driver

	// DatabaseConfig 数据库配置，详见 [xOptionDB.DatabaseConfig]。
	DatabaseConfig = xOptionDB.DatabaseConfig

	// DatabaseOption 数据库二级选项，详见 [xOptionDB.DatabaseOption]。
	DatabaseOption = xOptionDB.DatabaseOption

	// CommonOptions 数据库通用连接池参数，详见 [xOptionDB.CommonOptions]。
	CommonOptions = xOptionDB.CommonOptions

	// PrepareFunc 建表后数据初始化回调签名，详见 [xOptionDB.PrepareFunc]。
	PrepareFunc = xOptionDB.PrepareFunc
)

// 数据库驱动常量重导出，保持 xOption.DriverMySQL 等旧引用兼容。
const (
	DriverMySQL    = xOptionDB.DriverMySQL
	DriverPostgres = xOptionDB.DriverPostgres
	DriverSQLite   = xOptionDB.DriverSQLite
	DriverNone     = xOptionDB.DriverNone
)

// WithDatabase 将 [xOptionDB.DatabaseOption] 包裹为顶层 [Option]，供 Runner 使用。
//
// 该函数对标 [WithCache]，是 database 两层设计的顶层入口。
// 内部逐个执行传入的 DatabaseOption 修改 [DatabaseConfig]，完成后写入聚合 Config。
// nil DatabaseOption 会被跳过，支持条件构造。
//
// 使用示例：
//
//	// 显式 MySQL DSN
//	xOption.WithDatabase(xOptionDB.MySQL("user:pass@tcp(localhost:3306)/db"))
//	// 从环境变量装配 + 声明迁移与数据初始化
//	xOption.WithDatabase(
//	    xOptionDB.FromEnv(),
//	    xOptionDB.WithAutoMigrate(&entity.Role{}, &entity.User{}),
//	    xOptionDB.WithPrepare(seedRoles),
//	)
func WithDatabase(opts ...xOptionDB.DatabaseOption) Option {
	return func(c *Config) {
		for _, o := range opts {
			if o != nil {
				o(&c.database)
			}
		}
	}
}
