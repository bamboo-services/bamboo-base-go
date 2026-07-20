package xOptDatabase

import (
	"fmt"

	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

// Oracle 构造 Oracle 的 [DatabaseOption]。
//
// dsn 为 godror logfmt 格式连接串，例如：
//
//	user="scott" password="tiger" connectString="dbhost:1521/orclpdb1" libDir="/path/to/instantclient"
//
// 注意：Oracle 驱动底层为 godror，依赖 Oracle Instant Client（ODPI-C）。
// macOS/Windows 需在 dsn 中指定 libDir 或通过环境变量 DATABASE_LIB_DIR 配置；
// Linux 需用 ldconfig 将 Instant Client 加入系统库搜索路径。
// 如需从环境变量自动拼装，改用 [FromEnv] 并设置 DATABASE_DRIVER=oracle。
// 如需调整连接池参数，配合 [WithMaxOpenConns] 等二级选项使用。
func Oracle(dsn string, opts ...DatabaseOption) DatabaseOption {
	return func(c *DatabaseConfig) {
		c.driver = DriverOracle
		c.dsn = dsn
		applyDatabase(c, opts...)
	}
}

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
