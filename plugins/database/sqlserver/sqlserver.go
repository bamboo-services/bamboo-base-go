// Package xSQLServer 提供 SQL Server 数据库驱动插件。
//
// 本插件在 init() 中将 DriverSQLServer 注册到 common 层的驱动工厂注册表，
// 使 major 框架的 FromEnv / WithDriver 可解析 SQL Server 驱动。业务端只需
// import 本插件（可空白导入）即可将 SQL Server 驱动编入二进制。
package xSQLServer

import (
	xDatabase "github.com/bamboo-services/bamboo-base-go/common/database"
	"gorm.io/driver/sqlserver"
)

// init 注册 DriverSQLServer → sqlserver.Open 工厂，服务 FromEnv / WithDriver。
func init() {
	xDatabase.RegisterDialector(xDatabase.DriverSQLServer, sqlserver.Open)
}
