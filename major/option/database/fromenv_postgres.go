package xOptDatabase

import (
	"fmt"

	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

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
//
// 该函数仅负责拼装 DSN 字符串，不涉及任何驱动包。显式构造请使用 [WithDriver]
// 并 import 对应驱动插件（如 plugins/database/postgres），[FromEnv] 会自动调用本函数。
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
