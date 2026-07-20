package xOptDatabase_test

import (
	"os"
	"testing"

	xOptDatabase "github.com/bamboo-services/bamboo-base-go/major/option/database"
)

// TestFromEnv_OracleDispatch 验证 DATABASE_DRIVER=oracle 时 FromEnv 进入 Oracle 分支，
// 返回非 nil DatabaseOption，且 apply 后 driver 与 DSN 正确。
func TestFromEnv_OracleDispatch(t *testing.T) {
	t.Setenv("DATABASE_DRIVER", "oracle")
	t.Setenv("DATABASE_USER", "scott")
	t.Setenv("DATABASE_PASS", "tiger")
	t.Setenv("DATABASE_HOST", "dbhost")
	t.Setenv("DATABASE_PORT", "1521")
	t.Setenv("DATABASE_SERVICE_NAME", "orclpdb1")
	t.Setenv("DATABASE_LIB_DIR", "/opt/ic")
	os.Unsetenv("DATABASE_DSN")

	opt := xOptDatabase.FromEnv()
	if opt == nil {
		t.Fatal("Oracle FromEnv 返回 nil，预期非 nil")
	}
	cfg := xOptDatabase.DatabaseConfig{}
	opt(&cfg)
	if cfg.Driver() != xOptDatabase.DriverOracle {
		t.Errorf("Driver 不匹配: got=%q want=%q", cfg.Driver(), xOptDatabase.DriverOracle)
	}
	wantDSN := `user="scott" password="tiger" connectString="dbhost:1521/orclpdb1" libDir="/opt/ic"`
	if cfg.DSN() != wantDSN {
		t.Errorf("DSN 不匹配\n got=%q\nwant=%q", cfg.DSN(), wantDSN)
	}
}

// TestFromEnv_SQLServerDispatch 验证 DATABASE_DRIVER=sqlserver 时 FromEnv 进入 SQLServer 分支。
func TestFromEnv_SQLServerDispatch(t *testing.T) {
	t.Setenv("DATABASE_DRIVER", "sqlserver")
	t.Setenv("DATABASE_USER", "sa")
	t.Setenv("DATABASE_PASS", "Pa55w0rd")
	t.Setenv("DATABASE_HOST", "localhost")
	t.Setenv("DATABASE_PORT", "1433")
	t.Setenv("DATABASE_NAME", "mydb")
	os.Unsetenv("DATABASE_DSN")

	opt := xOptDatabase.FromEnv()
	if opt == nil {
		t.Fatal("SQLServer FromEnv 返回 nil，预期非 nil")
	}
	cfg := xOptDatabase.DatabaseConfig{}
	opt(&cfg)
	if cfg.Driver() != xOptDatabase.DriverSQLServer {
		t.Errorf("Driver 不匹配: got=%q want=%q", cfg.Driver(), xOptDatabase.DriverSQLServer)
	}
	wantDSN := "sqlserver://sa:Pa55w0rd@localhost:1433?database=mydb"
	if cfg.DSN() != wantDSN {
		t.Errorf("DSN 不匹配\n got=%q\nwant=%q", cfg.DSN(), wantDSN)
	}
}

// TestFromEnv_NoneReturnsNil 验证 DATABASE_DRIVER=none 时 FromEnv 返回 nil。
func TestFromEnv_NoneReturnsNil(t *testing.T) {
	t.Setenv("DATABASE_DRIVER", "none")
	if opt := xOptDatabase.FromEnv(); opt != nil {
		t.Errorf("none 驱动应返回 nil，got non-nil: %T", opt)
	}
}
