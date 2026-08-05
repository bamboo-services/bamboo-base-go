package xOptDatabase

import (
	"fmt"

	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

// OracleFromEnv 从环境变量拼装 Oracle godror logfmt DSN。
//
// 格式: user="..." password="..." connectString="host:port/service" libDir="..."
//
// 读取的环境变量:
//   - DATABASE_USER          (默认 system)
//   - DATABASE_PASS          (默认 空)
//   - DATABASE_HOST          (默认 localhost)
//   - DATABASE_PORT          (默认 1521)
//   - DATABASE_SERVICE_NAME  (默认 ORCLPDB1)
//   - DATABASE_LIB_DIR       (默认 空；Linux 下留空由 ldconfig 兜底)
//
// 该函数仅负责拼装 DSN 字符串，不涉及任何驱动包。显式构造请使用 [WithDriver]
// 并 import 对应驱动插件（如 plugins/database/oracle），[FromEnv] 会自动调用本函数。
func OracleFromEnv() string {
	user := xEnv.GetEnvString(xEnv.DatabaseUser, "system")
	pass := xEnv.GetEnvString(xEnv.DatabasePass, "")
	host := xEnv.GetEnvString(xEnv.DatabaseHost, "localhost")
	port := xEnv.GetEnvString(xEnv.DatabasePort, "1521")
	service := xEnv.GetEnvString(xEnv.DatabaseServiceName, "ORCLPDB1")
	connectString := fmt.Sprintf("%s:%s/%s", host, port, service)
	dsn := fmt.Sprintf(`user=%q password=%q connectString=%q`, user, pass, connectString)
	if libDir := xEnv.GetEnvString(xEnv.DatabaseLibDir, ""); libDir != "" {
		dsn += fmt.Sprintf(` libDir=%q`, libDir)
	}
	return dsn
}
