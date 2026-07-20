package xOptDatabase

import (
	"fmt"

	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

// SQLServer 构造 SQL Server 的 [DatabaseOption]。
//
// dsn 为 SQL Server URL 格式连接串，例如：
//
//	sqlserver://user:pass@host:port?database=dbname
//
// 如需从环境变量自动拼装，改用 [FromEnv] 并设置 DATABASE_DRIVER=sqlserver。
// 如需调整连接池参数，配合 [WithMaxOpenConns] 等二级选项使用。
func SQLServer(dsn string, opts ...DatabaseOption) DatabaseOption {
	return func(c *DatabaseConfig) {
		c.driver = DriverSQLServer
		c.dsn = dsn
		applyDatabase(c, opts...)
	}
}

// SQLServerFromEnv 从环境变量拼装 SQL Server URL DSN。
//
// 格式: sqlserver://user:pass@host:port?database=name
//
// 读取的环境变量:
//   - DATABASE_USER   (默认 sa)
//   - DATABASE_PASS   (默认 空)
//   - DATABASE_HOST   (默认 localhost)
//   - DATABASE_PORT   (默认 1433)
//   - DATABASE_NAME   (默认 master)
func SQLServerFromEnv() string {
	return fmt.Sprintf(
		"sqlserver://%s:%s@%s:%s?database=%s",
		xEnv.GetEnvString(xEnv.DatabaseUser, "sa"),
		xEnv.GetEnvString(xEnv.DatabasePass, ""),
		xEnv.GetEnvString(xEnv.DatabaseHost, "localhost"),
		xEnv.GetEnvString(xEnv.DatabasePort, "1433"),
		xEnv.GetEnvString(xEnv.DatabaseName, "master"),
	)
}
