package xOptDatabase_test

import (
	"os"
	"testing"

	xOptDatabase "github.com/bamboo-services/bamboo-base-go/major/option/database"
)

// TestSQLServerFromEnv_DSNAssembly 验证 SQL Server URL DSN 拼装，
// 五个分项字段应正确映射到 sqlserver://user:pass@host:port?database=name 格式。
func TestSQLServerFromEnv_DSNAssembly(t *testing.T) {
	t.Setenv("DATABASE_USER", "sa")
	t.Setenv("DATABASE_PASS", "Pa55w0rd")
	t.Setenv("DATABASE_HOST", "localhost")
	t.Setenv("DATABASE_PORT", "1433")
	t.Setenv("DATABASE_NAME", "mydb")

	got := xOptDatabase.SQLServerFromEnv()
	want := "sqlserver://sa:Pa55w0rd@localhost:1433?database=mydb"
	if got != want {
		t.Errorf("SQLServerFromEnv DSN 拼装不匹配\n got=%q\nwant=%q", got, want)
	}
}

// TestSQLServerFromEnv_Defaults 验证环境变量未设置时使用 SQL Server 安装惯例默认值：
// 用户 sa、端口 1433、库名 master。GetEnvString 仅在 key 不存在时返回默认值，
// 故用 os.Unsetenv 模拟「未设置」而非 t.Setenv("...", "")（后者会得到空串）。
func TestSQLServerFromEnv_Defaults(t *testing.T) {
	os.Unsetenv("DATABASE_USER")
	os.Unsetenv("DATABASE_PASS")
	os.Unsetenv("DATABASE_HOST")
	os.Unsetenv("DATABASE_PORT")
	os.Unsetenv("DATABASE_NAME")

	got := xOptDatabase.SQLServerFromEnv()
	want := "sqlserver://sa:@localhost:1433?database=master"
	if got != want {
		t.Errorf("SQLServerFromEnv 默认值拼装不匹配\n got=%q\nwant=%q", got, want)
	}
}

// TestSQLServer_ConstructOption 验证 SQLServer() 构造的 DatabaseOption 正确设置 driver 与 dsn。
func TestSQLServer_ConstructOption(t *testing.T) {
	dsn := "sqlserver://sa:Pa55w0rd@localhost:1433?database=mydb"
	opt := xOptDatabase.SQLServer(dsn)
	cfg := xOptDatabase.DatabaseConfig{}
	opt(&cfg)
	if cfg.Driver() != xOptDatabase.DriverSQLServer {
		t.Errorf("Driver 不匹配: got=%q want=%q", cfg.Driver(), xOptDatabase.DriverSQLServer)
	}
	if cfg.DSN() != dsn {
		t.Errorf("DSN 不匹配: got=%q want=%q", cfg.DSN(), dsn)
	}
}
