package mysql_test

import (
	"os"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/mysql/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TestAdapter_Interface 验证 mysql.New 与 mysql.FromEnv 满足 dialect.Adapter 接口契约。
func TestAdapter_Interface(t *testing.T) {
	assertAdapter := func(dialect.Adapter) {}
	assertAdapter(mysql.New("root:pass@tcp(localhost:3306)/bamboo?charset=utf8mb4&parseTime=True&loc=Local"))
	assertAdapter(mysql.FromEnv())
}

// TestAdapter_Name 验证适配器 Name() 返回驱动名 "mysql"。
func TestAdapter_Name(t *testing.T) {
	ad := mysql.New("root:pass@tcp(localhost:3306)/bamboo?charset=utf8mb4&parseTime=True&loc=Local")
	if got := ad.Name(); got != "mysql" {
		t.Fatalf("expected Name() to be 'mysql', got %q", got)
	}

	adEnv := mysql.FromEnv()
	if got := adEnv.Name(); got != "mysql" {
		t.Fatalf("expected FromEnv().Name() to be 'mysql', got %q", got)
	}
}

// TestAdapter_Open_Type 验证 Open 返回的 Dialector 类型为 gorm.io/driver/mysql.Dialector。
func TestAdapter_Open_Type(t *testing.T) {
	ad := mysql.New("root:pass@tcp(localhost:3306)/bamboo?charset=utf8mb4&parseTime=True&loc=Local")
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	if dialector == nil {
		t.Fatal("expected non-nil dialector")
	}

	if _, ok := dialector.(*gormmysql.Dialector); !ok {
		t.Fatalf("expected *gormmysql.Dialector, got %T", dialector)
	}
	if dialector.Name() != "mysql" {
		t.Fatalf("expected dialector.Name() to be 'mysql', got %q", dialector.Name())
	}
}

// TestAdapter_FromEnv_Assembly 验证从环境变量各字段正常拼装 DSN。
func TestAdapter_FromEnv_Assembly(t *testing.T) {
	t.Setenv("DATABASE_USER", "custom_user")
	t.Setenv("DATABASE_PASS", "custom_pass")
	t.Setenv("DATABASE_HOST", "db.example.com")
	t.Setenv("DATABASE_PORT", "3307")
	t.Setenv("DATABASE_NAME", "custom_db")
	t.Setenv("DATABASE_CHARSET", "utf8")
	_ = os.Unsetenv("DATABASE_DSN")

	ad := mysql.FromEnv()
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	md, ok := dialector.(*gormmysql.Dialector)
	if !ok {
		t.Fatalf("expected *gormmysql.Dialector, got %T", dialector)
	}

	wantDSN := "custom_user:custom_pass@tcp(db.example.com:3307)/custom_db?charset=utf8&parseTime=True&loc=Local"
	if md.DSN != wantDSN {
		t.Fatalf("DSN mismatch: got %q, want %q", md.DSN, wantDSN)
	}
}

// TestAdapter_FromEnv_Defaults 验证环境变量未设置时使用标准默认值。
func TestAdapter_FromEnv_Defaults(t *testing.T) {
	_ = os.Unsetenv("DATABASE_DSN")
	_ = os.Unsetenv("DATABASE_USER")
	_ = os.Unsetenv("DATABASE_PASS")
	_ = os.Unsetenv("DATABASE_HOST")
	_ = os.Unsetenv("DATABASE_PORT")
	_ = os.Unsetenv("DATABASE_NAME")
	_ = os.Unsetenv("DATABASE_CHARSET")

	ad := mysql.FromEnv()
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	md, ok := dialector.(*gormmysql.Dialector)
	if !ok {
		t.Fatalf("expected *gormmysql.Dialector, got %T", dialector)
	}

	wantDSN := "root:@tcp(localhost:3306)/bamboo?charset=utf8mb4&parseTime=True&loc=Local"
	if md.DSN != wantDSN {
		t.Fatalf("DSN mismatch: got %q, want %q", md.DSN, wantDSN)
	}
}

// TestAdapter_FromEnv_DirectDSN 验证 DATABASE_DSN 覆盖分项配置。
func TestAdapter_FromEnv_DirectDSN(t *testing.T) {
	t.Setenv("DATABASE_USER", "ignored_user")
	t.Setenv("DATABASE_DSN", "override_user:override_pass@tcp(remote-host:3306)/override_db?charset=utf8mb4&parseTime=True&loc=Local")

	ad := mysql.FromEnv()
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	md, ok := dialector.(*gormmysql.Dialector)
	if !ok {
		t.Fatalf("expected *gormmysql.Dialector, got %T", dialector)
	}

	wantDSN := "override_user:override_pass@tcp(remote-host:3306)/override_db?charset=utf8mb4&parseTime=True&loc=Local"
	if md.DSN != wantDSN {
		t.Fatalf("DSN mismatch: got %q, want %q", md.DSN, wantDSN)
	}
}

// TestAdapter_DSNValidationAndNoPasswordLeakage 验证配置错误保存并在 Open() 时返回，且错误信息绝不泄露密码。
func TestAdapter_DSNValidationAndNoPasswordLeakage(t *testing.T) {
	sensitivePassword := "SuperSecretPasswordP@ss123!"

	tests := []struct {
		name string
		ad   dialect.Adapter
	}{
		{
			name: "empty DSN via New",
			ad:   mysql.New("   "),
		},
		{
			name: "missing at sign with sensitive password",
			ad:   mysql.New("root:" + sensitivePassword + "tcp(localhost:3306)/db"),
		},
		{
			name: "missing slash with sensitive password",
			ad:   mysql.New("root:" + sensitivePassword + "@tcp(localhost:3306)db"),
		},
		{
			name: "invalid net addr format with sensitive password",
			ad:   mysql.New("root:" + sensitivePassword + "@localhost:3306/db"),
		},
		{
			name: "empty tcp addr with sensitive password",
			ad:   mysql.New("root:" + sensitivePassword + "@tcp()/db"),
		},
		{
			name: "empty port via FromEnv",
			ad: func() dialect.Adapter {
				_ = os.Unsetenv("DATABASE_DSN")
				t.Setenv("DATABASE_PASS", sensitivePassword)
				t.Setenv("DATABASE_PORT", "")
				return mysql.FromEnv()
			}(),
		},
		{
			name: "empty host via FromEnv",
			ad: func() dialect.Adapter {
				_ = os.Unsetenv("DATABASE_DSN")
				t.Setenv("DATABASE_PASS", sensitivePassword)
				t.Setenv("DATABASE_HOST", "")
				return mysql.FromEnv()
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialector, err := tt.ad.Open()
			if err == nil {
				t.Fatal("expected Open() to return error, got nil")
			}
			if dialector != nil {
				t.Fatalf("expected nil dialector on error, got %v", dialector)
			}
			errMsg := err.Error()
			if strings.Contains(errMsg, sensitivePassword) {
				t.Fatalf("error message leaked sensitive password: %q", errMsg)
			}
		})
	}
}

// TestAdapter_InitRegistration 验证 init 函数向 dialect.Register 成功注册 DriverMySQL。
func TestAdapter_InitRegistration(t *testing.T) {
	_ = os.Unsetenv("DATABASE_DSN")
	t.Setenv("DATABASE_USER", "registry_user")
	t.Setenv("DATABASE_PASS", "reg_pass")
	t.Setenv("DATABASE_HOST", "reg_host")
	t.Setenv("DATABASE_PORT", "3306")
	t.Setenv("DATABASE_NAME", "reg_db")
	t.Setenv("DATABASE_CHARSET", "utf8mb4")

	ad, err := dialect.Lookup(dialect.DriverMySQL)
	if err != nil {
		t.Fatalf("expected DriverMySQL to be registered in dialect, got error: %v", err)
	}
	if ad == nil {
		t.Fatal("expected non-nil adapter from Lookup")
	}
	if ad.Name() != "mysql" {
		t.Fatalf("expected adapter.Name() to be 'mysql', got %q", ad.Name())
	}

	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open error: %v", err)
	}
	md, ok := dialector.(*gormmysql.Dialector)
	if !ok {
		t.Fatalf("expected *gormmysql.Dialector, got %T", dialector)
	}

	wantDSN := "registry_user:reg_pass@tcp(reg_host:3306)/reg_db?charset=utf8mb4&parseTime=True&loc=Local"
	if md.DSN != wantDSN {
		t.Fatalf("DSN mismatch from registered factory: got %q, want %q", md.DSN, wantDSN)
	}
}

// TestNoNetworkOrConnectionOnConstruction 验证 New/FromEnv/Open 不主动建连或连网络。
func TestNoNetworkOrConnectionOnConstruction(t *testing.T) {
	// 指向一个当前不可达的端口和 IP
	nonExistentDSN := "root:pass@tcp(192.0.2.1:3306)/test_db?charset=utf8mb4&parseTime=True&loc=Local"
	adp := mysql.New(nonExistentDSN)

	// 验证 Open 只是构造 Dialector，不报错也不尝试网络连接
	dialector, err := adp.Open()
	if err != nil {
		t.Fatalf("Open 不应报错: %v", err)
	}
	if dialector == nil {
		t.Fatal("dialector 期望非空")
	}
}

// TestRealEngine_MySQL 真实 MySQL 引擎测试：
// 仅在显式设置 BAMBOO_TEST_MYSQL_DSN 或 BAMBOO_TEST_MYSQL 时执行，默认自动 skip。
// 不连生产/默认地址，不启动容器。
func TestRealEngine_MySQL(t *testing.T) {
	dsn := os.Getenv("BAMBOO_TEST_MYSQL_DSN")
	if dsn == "" {
		dsn = os.Getenv("BAMBOO_TEST_MYSQL")
	}
	if strings.TrimSpace(dsn) == "" {
		t.Skip("skipping real MySQL engine test; set BAMBOO_TEST_MYSQL_DSN to run")
	}

	ad := mysql.New(dsn)
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open failed: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() failed: %v", err)
	}
	defer func() {
		_ = sqlDB.Close()
	}()

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("sqlDB.Ping failed: %v", err)
	}

	var one int
	if err := db.Raw("SELECT 1").Scan(&one).Error; err != nil {
		t.Fatalf("query SELECT 1 failed: %v", err)
	}
	if one != 1 {
		t.Fatalf("expected 1, got %d", one)
	}
}
