package xOptDatabase

import (
	"fmt"

	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

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
//
// 该函数仅负责拼装 DSN 字符串，不涉及任何驱动包。显式构造请使用 [WithDriver]
// 并 import 对应驱动插件（如 plugins/database/sqlserver），[FromEnv] 会自动调用本函数。
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
