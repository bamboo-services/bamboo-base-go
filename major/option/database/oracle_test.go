package xOptDatabase_test

import (
	"os"
	"strings"
	"testing"

	xOptDatabase "github.com/bamboo-services/bamboo-base-go/major/option/database"
)

// TestOracleFromEnv_DSNAssembly 验证 Oracle godror logfmt DSN 拼装，
// 含 libDir 时应完整拼入连接串。
func TestOracleFromEnv_DSNAssembly(t *testing.T) {
	t.Setenv("DATABASE_USER", "scott")
	t.Setenv("DATABASE_PASS", "tiger")
	t.Setenv("DATABASE_HOST", "dbhost")
	t.Setenv("DATABASE_PORT", "1521")
	t.Setenv("DATABASE_SERVICE_NAME", "orclpdb1")
	t.Setenv("DATABASE_LIB_DIR", "/opt/instantclient")

	got := xOptDatabase.OracleFromEnv()
	want := `user="scott" password="tiger" connectString="dbhost:1521/orclpdb1" libDir="/opt/instantclient"`
	if got != want {
		t.Errorf("OracleFromEnv DSN 拼装不匹配\n got=%q\nwant=%q", got, want)
	}
}

// TestOracleFromEnv_NoLibDir 验证 libDir 未设置时不追加该字段（Linux 路径），
// 仅保留 user/password/connectString 三段。
func TestOracleFromEnv_NoLibDir(t *testing.T) {
	t.Setenv("DATABASE_USER", "u")
	t.Setenv("DATABASE_PASS", "p")
	t.Setenv("DATABASE_HOST", "h")
	t.Setenv("DATABASE_PORT", "1521")
	t.Setenv("DATABASE_SERVICE_NAME", "svc")
	os.Unsetenv("DATABASE_LIB_DIR")

	got := xOptDatabase.OracleFromEnv()
	if strings.Contains(got, "libDir") {
		t.Errorf("未设置 libDir 却被拼入: %q", got)
	}
	want := `user="u" password="p" connectString="h:1521/svc"`
	if got != want {
		t.Errorf("OracleFromEnv 无 libDir 时 DSN 拼装不匹配\n got=%q\nwant=%q", got, want)
	}
}

// TestOracle_ConstructOption 验证 Oracle() 构造的 DatabaseOption 正确设置 driver 与 dsn。
func TestOracle_ConstructOption(t *testing.T) {
	dsn := `user="scott" password="tiger" connectString="dbhost:1521/orclpdb1"`
	opt := xOptDatabase.Oracle(dsn)
	cfg := xOptDatabase.DatabaseConfig{}
	opt(&cfg)
	if cfg.Driver() != xOptDatabase.DriverOracle {
		t.Errorf("Driver 不匹配: got=%q want=%q", cfg.Driver(), xOptDatabase.DriverOracle)
	}
	if cfg.DSN() != dsn {
		t.Errorf("DSN 不匹配: got=%q want=%q", cfg.DSN(), dsn)
	}
}
