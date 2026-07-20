package xOptDatabase

import (
	"fmt"

	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

// Postgres 构造 PostgreSQL 的 [DatabaseOption]。
//
// dsn 为完整的 PostgreSQL 连接串，格式如：
//
//	host=localhost user=postgres password=secret dbname=app port=5432 TimeZone=Asia/Shanghai sslmode=disable
//
// 如需从环境变量自动拼装，改用 [FromEnv] 并设置 DATABASE_DRIVER=postgres。
// 如需调整连接池参数，配合 [WithMaxOpenConns] 等二级选项使用。
func Postgres(dsn string, opts ...DatabaseOption) DatabaseOption {
	return func(c *DatabaseConfig) {
		c.driver = DriverPostgres
		c.dsn = dsn
		applyDatabase(c, opts...)
	}
}

// PostgresFromEnv 从环境变量拼装 PostgreSQL DSN。
//
// 格式: host=xxx user=xxx password=xxx dbname=xxx port=xxx TimeZone=xxx sslmode=disable
//
// 读取的环境变量:
//   - DATABASE_HOST      (默认 localhost)
//   - DATABASE_USER      (默认 postgres)
//   - DATABASE_PASS      (默认 空)
//   - DATABASE_NAME      (默认 postgres)
//   - DATABASE_PORT      (默认 5432)
//   - DATABASE_TIMEZONE  (默认 Asia/Shanghai)
func PostgresFromEnv() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s TimeZone=%s sslmode=disable",
		xEnv.GetEnvString(xEnv.DatabaseHost, "localhost"),
		xEnv.GetEnvString(xEnv.DatabaseUser, "postgres"),
		xEnv.GetEnvString(xEnv.DatabasePass, ""),
		xEnv.GetEnvString(xEnv.DatabaseName, "postgres"),
		xEnv.GetEnvString(xEnv.DatabasePort, "5432"),
		xEnv.GetEnvString(xEnv.DatabaseTimezone, "Asia/Shanghai"),
	)
}
