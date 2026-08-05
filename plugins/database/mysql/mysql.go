// Package xMysql 提供 MySQL 数据库驱动插件。
//
// 本插件在 init() 中将 DriverMySQL 注册到 common 层的驱动工厂注册表，
// 使 major 框架的 FromEnv / WithDriver 可解析 MySQL 驱动。业务端只需 import
// 本插件（可空白导入）即可将 MySQL 驱动编入二进制。
package xMysql

import (
	xDatabase "github.com/bamboo-services/bamboo-base-go/common/database"
	"gorm.io/driver/mysql"
)

// init 注册 DriverMySQL → mysql.Open 工厂，服务 FromEnv / WithDriver。
func init() {
	xDatabase.RegisterDialector(xDatabase.DriverMySQL, mysql.Open)
}
