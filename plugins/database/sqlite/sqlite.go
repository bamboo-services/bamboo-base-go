// Package xSQLite 提供 SQLite 数据库驱动插件。
//
// 本插件在 init() 中将 DriverSQLite 注册到 common 层的驱动工厂注册表，
// 使 major 框架的 FromEnv / WithDriver 可解析 SQLite 驱动。业务端只需
// import 本插件（可空白导入）即可将 SQLite 驱动编入二进制。
package xSQLite

import (
	xDatabase "github.com/bamboo-services/bamboo-base-go/common/database"
	"github.com/libtnb/sqlite"
)

// init 注册 DriverSQLite → sqlite.Open 工厂，服务 FromEnv / WithDriver。
func init() {
	xDatabase.RegisterDialector(xDatabase.DriverSQLite, sqlite.Open)
}
