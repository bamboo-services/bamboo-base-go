package xOptDatabase

import (
	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

// SQLiteFromEnv 从环境变量读取 SQLite 数据库路径。
//
// 直接返回 DATABASE_PATH 的值（默认 ":memory:"）。SQLite 无传统意义的 DSN，
// 该值即作为 GORM sqlite 驱动的 dsn 参数。
//
// 读取的环境变量:
//   - DATABASE_PATH  (默认 :memory:)
//
// 该函数仅负责拼装 DSN 字符串，不涉及任何驱动包。显式构造请使用 [WithDriver]
// 并 import 对应驱动插件（如 plugins/database/sqlite），[FromEnv] 会自动调用本函数。
func SQLiteFromEnv() string {
	return xEnv.GetEnvString(xEnv.DatabasePath, ":memory:")
}
