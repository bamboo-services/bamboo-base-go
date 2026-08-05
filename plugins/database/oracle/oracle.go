// Package xOracle 提供 Oracle 数据库驱动插件。
//
// 本插件在 init() 中将 DriverOracle 注册到 common 层的驱动工厂注册表，
// 使 major 框架的 FromEnv / WithDriver 可解析 Oracle 驱动。业务端只需
// import 本插件（可空白导入）即可将 Oracle 驱动编入二进制。
//
// 注意：本驱动基于 godror，需要 CGO 与 Oracle Instant Client。仅当业务端
// 真正需要 Oracle 时才应引入本插件，避免将 CGO 工具链强加给其他应用。
package xOracle

import (
	xDatabase "github.com/bamboo-services/bamboo-base-go/common/database"
	"github.com/oracle-samples/gorm-oracle/oracle"
)

// init 注册 DriverOracle → oracle.Open 工厂，服务 FromEnv / WithDriver。
func init() {
	xDatabase.RegisterDialector(xDatabase.DriverOracle, oracle.Open)
}
