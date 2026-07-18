package option

import (
	xOptionDB "github.com/bamboo-services/bamboo-base-go/major/option/database"
)

// WithDatabase 包装一个已构造的 [xOptionDB.Config] 为 [Option]。
//
// 通常与 [xOptionDB.MySQL] / [xOptionDB.Postgres] / [xOptionDB.SQLite] 配合使用：
//
//	xOption.WithDatabase(xOptionDB.MySQL("user:pass@tcp(localhost:3306)/db"))
//
// 这种拆分使得驱动专属构造逻辑与 Option 装配逻辑分离，
// [xOptionDB] 子包不依赖 option 父包，避免循环依赖。
func WithDatabase(cfg xOptionDB.Config) Option {
	return func(c *Config) { c.database = cfg }
}

// WithMySQL 便捷构造 MySQL 配置的 [Option]，等价于
// [WithDatabase]([xOptionDB.MySQL](dsn, opts...)).
func WithMySQL(dsn string, opts ...xOptionDB.CommonOption) Option {
	return WithDatabase(xOptionDB.MySQL(dsn, opts...))
}

// WithPostgres 便捷构造 PostgreSQL 配置的 [Option]，等价于
// [WithDatabase]([xOptionDB.Postgres](dsn, opts...)).
func WithPostgres(dsn string, opts ...xOptionDB.CommonOption) Option {
	return WithDatabase(xOptionDB.Postgres(dsn, opts...))
}

// WithSQLite 便捷构造 SQLite 配置的 [Option]，等价于
// [WithDatabase]([xOptionDB.SQLite](dsn, opts...)).
func WithSQLite(dsn string, opts ...xOptionDB.CommonOption) Option {
	return WithDatabase(xOptionDB.SQLite(dsn, opts...))
}

// WithDatabaseFromEnv 从环境变量自动装配数据库配置的 [Option]。
//
// 委托给 [xOptionDB.FromEnv] 完成 DSN 拼装与 TablePrefix 读取，
// 返回 nil Option 表示未启用内置数据库（DATABASE_DRIVER 为空或 "none"）。
//
// 详见 [xOptionDB.FromEnv] 的优先级说明。
func WithDatabaseFromEnv(opts ...xOptionDB.CommonOption) Option {
	cfg, ok := xOptionDB.FromEnv(opts...)
	if !ok {
		return nil
	}
	return WithDatabase(cfg)
}
