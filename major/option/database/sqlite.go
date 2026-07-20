package xOptDatabase

import (
	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

// SQLite 构造 SQLite 的 [DatabaseOption]。
//
// dsn 通常为文件路径（如 "test.db"）或 ":memory:"（内存数据库，仅用于开发与测试）。
// 如需从环境变量自动读取，改用 [FromEnv] 并设置 DATABASE_DRIVER=sqlite。
// 如需调整连接池参数，配合 [WithMaxOpenConns] 等二级选项使用。
func SQLite(dsn string, opts ...DatabaseOption) DatabaseOption {
	return func(c *DatabaseConfig) {
		c.driver = DriverSQLite
		c.dsn = dsn
		applyDatabase(c, opts...)
	}
}

// SQLiteFromEnv 从环境变量读取 SQLite 数据库路径。
//
// 直接返回 DATABASE_PATH 的值（默认 ":memory:"）。SQLite 无传统意义的 DSN，
// 该值即作为 GORM sqlite 驱动的 dsn 参数。
//
// 读取的环境变量:
//   - DATABASE_PATH  (默认 :memory:)
func SQLiteFromEnv() string {
	return xEnv.GetEnvString(xEnv.DatabasePath, ":memory:")
}
