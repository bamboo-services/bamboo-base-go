package sqlserver_test

import (
	"net"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/sqlserver/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
	gormsqlserver "gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

// cleanEnv 恢复测试期间变更的环境变量，避免污染其他测试。
func cleanEnv(t *testing.T, keys ...string) {
	t.Helper()
	orig := make(map[string]*string, len(keys))
	for _, k := range keys {
		if val, exists := os.LookupEnv(k); exists {
			v := val
			orig[k] = &v
		} else {
			orig[k] = nil
		}
	}
	t.Cleanup(func() {
		for k, v := range orig {
			if v != nil {
				_ = os.Setenv(k, *v)
			} else {
				_ = os.Unsetenv(k)
			}
		}
	})
}

// TestAdapter_Interface 验证 sqlserver.New 与 FromEnv 满足 dialect.Adapter 接口契约。
func TestAdapter_Interface(t *testing.T) {
	assertAdapter := func(dialect.Adapter) {}
	assertAdapter(sqlserver.New("sqlserver://sa:pass@localhost:1433?database=master"))
	assertAdapter(sqlserver.FromEnv())
}

// TestAdapter_Name 验证适配器 Name() 返回正确驱动名。
func TestAdapter_Name(t *testing.T) {
	ad := sqlserver.New("sqlserver://sa:pass@localhost:1433?database=master")
	if got := ad.Name(); got != "sqlserver" {
		t.Fatalf("expected Name() to be 'sqlserver', got %q", got)
	}

	adEnv := sqlserver.FromEnv()
	if got := adEnv.Name(); got != "sqlserver" {
		t.Fatalf("expected FromEnv().Name() to be 'sqlserver', got %q", got)
	}
}

// TestAdapter_Open_Type 验证 Open 返回的 Dialector 类型为 gorm.io/driver/sqlserver.Dialector。
func TestAdapter_Open_Type(t *testing.T) {
	ad := sqlserver.New("sqlserver://sa:pass@localhost:1433?database=master")
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	if dialector == nil {
		t.Fatal("expected non-nil dialector")
	}

	if _, ok := dialector.(*gormsqlserver.Dialector); !ok {
		t.Fatalf("expected *gormsqlserver.Dialector, got %T", dialector)
	}
	if dialector.Name() != "sqlserver" {
		t.Fatalf("expected dialector.Name() to be 'sqlserver', got %q", dialector.Name())
	}
}

// TestAdapter_FromEnv_Assembly 验证从环境变量各字段正常拼装 DSN。
func TestAdapter_FromEnv_Assembly(t *testing.T) {
	cleanEnv(t, "DATABASE_USER", "DATABASE_PASS", "DATABASE_HOST", "DATABASE_PORT", "DATABASE_NAME", "DATABASE_DSN")
	t.Setenv("DATABASE_USER", "custom_sa")
	t.Setenv("DATABASE_PASS", "Secret123")
	t.Setenv("DATABASE_HOST", "db.internal")
	t.Setenv("DATABASE_PORT", "14330")
	t.Setenv("DATABASE_NAME", "my_app_db")
	_ = os.Unsetenv("DATABASE_DSN")

	ad := sqlserver.FromEnv()
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	sd, ok := dialector.(*gormsqlserver.Dialector)
	if !ok {
		t.Fatalf("expected *gormsqlserver.Dialector, got %T", dialector)
	}

	wantDSN := "sqlserver://custom_sa:Secret123@db.internal:14330?database=my_app_db"
	if sd.DSN != wantDSN {
		t.Fatalf("DSN mismatch: got %q, want %q", sd.DSN, wantDSN)
	}
}

// TestAdapter_FromEnv_SpecialCredentials 验证包含特殊字符（/、@、#、?、%或:）的凭据能安全编码并在解析后取回同值。
func TestAdapter_FromEnv_SpecialCredentials(t *testing.T) {
	cleanEnv(t, "DATABASE_USER", "DATABASE_PASS", "DATABASE_HOST", "DATABASE_PORT", "DATABASE_NAME", "DATABASE_DSN")
	user := "user/@#?:%test"
	pass := "p@ss/word#123?%val:extra"

	t.Setenv("DATABASE_USER", user)
	t.Setenv("DATABASE_PASS", pass)
	t.Setenv("DATABASE_HOST", "db.local")
	t.Setenv("DATABASE_PORT", "1433")
	t.Setenv("DATABASE_NAME", "special_db")
	_ = os.Unsetenv("DATABASE_DSN")

	ad := sqlserver.FromEnv()
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	sd, ok := dialector.(*gormsqlserver.Dialector)
	if !ok {
		t.Fatalf("expected *gormsqlserver.Dialector, got %T", dialector)
	}

	u, err := url.Parse(sd.DSN)
	if err != nil {
		t.Fatalf("failed to parse generated DSN %q: %v", sd.DSN, err)
	}

	if u.User == nil {
		t.Fatal("expected non-nil User in parsed DSN")
	}
	if gotUser := u.User.Username(); gotUser != user {
		t.Fatalf("user mismatch: got %q, want %q", gotUser, user)
	}
	gotPass, hasPass := u.User.Password()
	if !hasPass || gotPass != pass {
		t.Fatalf("password mismatch: got %q (has=%v), want %q", gotPass, hasPass, pass)
	}
	if q := u.Query().Get("database"); q != "special_db" {
		t.Fatalf("database query param mismatch: got %q, want %q", q, "special_db")
	}
}

// TestAdapter_FromEnv_IPv6 验证 IPv6 地址使用 net.JoinHostPort 能够正确规范组装。
func TestAdapter_FromEnv_IPv6(t *testing.T) {
	cleanEnv(t, "DATABASE_USER", "DATABASE_PASS", "DATABASE_HOST", "DATABASE_PORT", "DATABASE_NAME", "DATABASE_DSN")
	t.Setenv("DATABASE_USER", "sa")
	t.Setenv("DATABASE_PASS", "pass")
	t.Setenv("DATABASE_HOST", "::1")
	t.Setenv("DATABASE_PORT", "1433")
	t.Setenv("DATABASE_NAME", "master")
	_ = os.Unsetenv("DATABASE_DSN")

	ad := sqlserver.FromEnv()
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	sd, ok := dialector.(*gormsqlserver.Dialector)
	if !ok {
		t.Fatalf("expected *gormsqlserver.Dialector, got %T", dialector)
	}

	u, err := url.Parse(sd.DSN)
	if err != nil {
		t.Fatalf("failed to parse generated DSN %q: %v", sd.DSN, err)
	}

	h, p, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatalf("failed to split host port from %q: %v", u.Host, err)
	}
	if h != "::1" || p != "1433" {
		t.Fatalf("expected host '::1' and port '1433', got host=%q, port=%q", h, p)
	}
}

// TestAdapter_FromEnv_Defaults 验证环境变量未设置时使用标准默认值。
func TestAdapter_FromEnv_Defaults(t *testing.T) {
	cleanEnv(t, "DATABASE_DSN", "DATABASE_USER", "DATABASE_PASS", "DATABASE_HOST", "DATABASE_PORT", "DATABASE_NAME")
	_ = os.Unsetenv("DATABASE_DSN")
	_ = os.Unsetenv("DATABASE_USER")
	_ = os.Unsetenv("DATABASE_PASS")
	_ = os.Unsetenv("DATABASE_HOST")
	_ = os.Unsetenv("DATABASE_PORT")
	_ = os.Unsetenv("DATABASE_NAME")

	ad := sqlserver.FromEnv()
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	sd, ok := dialector.(*gormsqlserver.Dialector)
	if !ok {
		t.Fatalf("expected *gormsqlserver.Dialector, got %T", dialector)
	}

	wantDSN := "sqlserver://sa:@localhost:1433?database=master"
	if sd.DSN != wantDSN {
		t.Fatalf("DSN mismatch: got %q, want %q", sd.DSN, wantDSN)
	}
}

// TestAdapter_FromEnv_DirectDSN 验证 DATABASE_DSN 覆盖分项配置。
func TestAdapter_FromEnv_DirectDSN(t *testing.T) {
	cleanEnv(t, "DATABASE_USER", "DATABASE_DSN")
	t.Setenv("DATABASE_USER", "ignored_sa")
	t.Setenv("DATABASE_DSN", "sqlserver://override_user:pass@remote:1433?database=prod")

	ad := sqlserver.FromEnv()
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	sd, ok := dialector.(*gormsqlserver.Dialector)
	if !ok {
		t.Fatalf("expected *gormsqlserver.Dialector, got %T", dialector)
	}

	wantDSN := "sqlserver://override_user:pass@remote:1433?database=prod"
	if sd.DSN != wantDSN {
		t.Fatalf("DSN mismatch: got %q, want %q", sd.DSN, wantDSN)
	}
}

// TestAdapter_DSNValidationAndNoPasswordLeakage 验证配置错误保存并在 Open() 时返回，且错误信息绝不泄露密码。
func TestAdapter_DSNValidationAndNoPasswordLeakage(t *testing.T) {
	cleanEnv(t, "DATABASE_DSN", "DATABASE_PASS", "DATABASE_PORT")
	sensitivePassword := "SuperSecretPasswordP@ss123!"

	tests := []struct {
		name string
		ad   dialect.Adapter
	}{
		{
			name: "empty DSN via New",
			ad:   sqlserver.New("   "),
		},
		{
			name: "invalid scheme with sensitive password",
			ad:   sqlserver.New("http://sa:" + sensitivePassword + "@localhost:1433?database=test"),
		},
		{
			name: "missing host with sensitive password",
			ad:   sqlserver.New("sqlserver://sa:" + sensitivePassword + "@"),
		},
		{
			name: "empty port via FromEnv",
			ad: func() dialect.Adapter {
				_ = os.Unsetenv("DATABASE_DSN")
				t.Setenv("DATABASE_PASS", sensitivePassword)
				t.Setenv("DATABASE_PORT", "")
				return sqlserver.FromEnv()
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

// TestAdapter_InitRegistration 验证 init 函数向 dialect.Register 成功注册 DriverSQLServer。
func TestAdapter_InitRegistration(t *testing.T) {
	cleanEnv(t, "DATABASE_DSN", "DATABASE_USER", "DATABASE_PASS", "DATABASE_HOST", "DATABASE_PORT", "DATABASE_NAME")
	_ = os.Unsetenv("DATABASE_DSN")
	t.Setenv("DATABASE_USER", "registry_user")
	t.Setenv("DATABASE_PASS", "reg_pass")
	t.Setenv("DATABASE_HOST", "reg_host")
	t.Setenv("DATABASE_PORT", "1433")
	t.Setenv("DATABASE_NAME", "reg_db")

	ad, err := dialect.Lookup(dialect.DriverSQLServer)
	if err != nil {
		t.Fatalf("expected DriverSQLServer to be registered in dialect, got error: %v", err)
	}
	if ad == nil {
		t.Fatal("expected non-nil adapter from Lookup")
	}
	if ad.Name() != "sqlserver" {
		t.Fatalf("expected adapter.Name() to be 'sqlserver', got %q", ad.Name())
	}

	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open error: %v", err)
	}
	sd, ok := dialector.(*gormsqlserver.Dialector)
	if !ok {
		t.Fatalf("expected *gormsqlserver.Dialector, got %T", dialector)
	}

	wantDSN := "sqlserver://registry_user:reg_pass@reg_host:1433?database=reg_db"
	if sd.DSN != wantDSN {
		t.Fatalf("DSN mismatch from registered factory: got %q, want %q", sd.DSN, wantDSN)
	}
}

// TestRealEngine_SQLServer 真实 SQL Server 引擎测试：
// 仅在设置了 BAMBOO_TEST_SQLSERVER_DSN 时执行真实连接测试，默认自动 skip。
func TestRealEngine_SQLServer(t *testing.T) {
	dsn := os.Getenv("BAMBOO_TEST_SQLSERVER_DSN")
	if dsn == "" {
		t.Skip("skipping real SQL Server engine test; set BAMBOO_TEST_SQLSERVER_DSN to run")
	}

	ad := sqlserver.New(dsn)
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
