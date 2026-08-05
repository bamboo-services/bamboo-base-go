package xOptDatabase

import (
	"fmt"

	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

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
//
// 该函数仅负责拼装 DSN 字符串，不涉及任何驱动包。显式构造请使用 [WithDriver]
// 并 import 对应驱动插件（如 plugins/database/mysql），[FromEnv] 会自动调用本函数。
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
