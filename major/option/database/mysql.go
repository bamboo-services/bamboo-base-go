package database

import (
	"fmt"

	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

// MySQL 构造 MySQL 的 [DatabaseOption]。
//
// dsn 为完整的 MySQL 连接串，格式如：
//
//	user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
//
// 如需从环境变量自动拼装，改用 [FromEnv] 并设置 DATABASE_DRIVER=mysql。
// 如需调整连接池参数，配合 [WithMaxOpenConns] 等二级选项使用。
func MySQL(dsn string, opts ...DatabaseOption) DatabaseOption {
	return func(c *DatabaseConfig) {
		c.driver = DriverMySQL
		c.dsn = dsn
		applyDatabase(c, opts...)
	}
}

// MySQLFromEnv 从环境变量拼装 MySQL DSN。
//
// 格式: user:pass@tcp(host:port)/name?charset=xxx&parseTime=True&loc=Local
//
// 读取的环境变量:
//   - DATABASE_USER     (默认 root)
//   - DATABASE_PASS     (默认 空)
//   - DATABASE_HOST     (默认 localhost)
//   - DATABASE_PORT     (默认 3306)
//   - DATABASE_NAME     (默认 bamboo)
//   - DATABASE_CHARSET  (默认 utf8mb4)
func MySQLFromEnv() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local",
		xEnv.GetEnvString(xEnv.DatabaseUser, "root"),
		xEnv.GetEnvString(xEnv.DatabasePass, ""),
		xEnv.GetEnvString(xEnv.DatabaseHost, "localhost"),
		xEnv.GetEnvString(xEnv.DatabasePort, "3306"),
		xEnv.GetEnvString(xEnv.DatabaseName, "bamboo"),
		xEnv.GetEnvString(xEnv.DatabaseCharset, "utf8mb4"),
	)
}
