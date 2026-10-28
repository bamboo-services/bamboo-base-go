package postgres_test

import (
	"os"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/postgres/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestAdapter_Interface 验证 postgres.New 与 FromEnv 满足 dialect.Adapter 接口契约。
func TestAdapter_Interface(t *testing.T) {
	assertAdapter := func(dialect.Adapter) {}
	assertAdapter(postgres.New("host=localhost user=postgres password=test dbname=postgres port=5432 sslmode=disable"))
	assertAdapter(postgres.FromEnv())
}

// TestAdapter_Name 验证适配器 Name() 返回正确驱动名称 "postgres"。
func TestAdapter_Name(t *testing.T) {
	ad := postgres.New("host=localhost user=postgres password=test dbname=postgres port=5432 sslmode=disable")
	if got := ad.Name(); got != "postgres" {
		t.Fatalf("expected Name() to be 'postgres', got %q", got)
	}

	adEnv := postgres.FromEnv()
	if got := adEnv.Name(); got != "postgres" {
		t.Fatalf("expected FromEnv().Name() to be 'postgres', got %q", got)
	}
}

// TestAdapter_Open_Type 验证 Open 返回的 Dialector 类型为 gorm.io/driver/postgres.Dialector。
func TestAdapter_Open_Type(t *testing.T) {
	ad := postgres.New("host=localhost user=postgres password=test dbname=postgres port=5432 sslmode=disable")
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	if dialector == nil {
		t.Fatal("expected non-nil dialector")
	}

	if _, ok := dialector.(*gormpostgres.Dialector); !ok {
		t.Fatalf("expected *gormpostgres.Dialector, got %T", dialector)
	}
	if dialector.Name() != "postgres" {
		t.Fatalf("expected dialector.Name() to be 'postgres', got %q", dialector.Name())
	}
}

// TestAdapter_FromEnv_Assembly 验证从环境变量各字段正常拼装 DSN。
func TestAdapter_FromEnv_Assembly(t *testing.T) {
	t.Setenv("DATABASE_USER", "custom_pg")
	t.Setenv("DATABASE_PASS", "Secret123")
	t.Setenv("DATABASE_HOST", "db.pg.internal")
	t.Setenv("DATABASE_PORT", "54320")
	t.Setenv("DATABASE_NAME", "my_pg_db")
	t.Setenv("DATABASE_TIMEZONE", "UTC")
	_ = os.Unsetenv("DATABASE_DSN")

	ad := postgres.FromEnv()
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	pd, ok := dialector.(*gormpostgres.Dialector)
	if !ok {
		t.Fatalf("expected *gormpostgres.Dialector, got %T", dialector)
	}

	wantDSN := "host=db.pg.internal user=custom_pg password=Secret123 dbname=my_pg_db port=54320 TimeZone=UTC sslmode=disable"
	if pd.DSN != wantDSN {
		t.Fatalf("DSN mismatch: got %q, want %q", pd.DSN, wantDSN)
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
	_ = os.Unsetenv("DATABASE_TIMEZONE")

	ad := postgres.FromEnv()
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	pd, ok := dialector.(*gormpostgres.Dialector)
	if !ok {
		t.Fatalf("expected *gormpostgres.Dialector, got %T", dialector)
	}

	wantDSN := "host=localhost user=postgres password= dbname=postgres port=5432 TimeZone=Asia/Shanghai sslmode=disable"
	if pd.DSN != wantDSN {
		t.Fatalf("DSN mismatch: got %q, want %q", pd.DSN, wantDSN)
	}
}

// TestAdapter_FromEnv_Escaping 验证字段包含空格或特殊字符时能够正确转义。
func TestAdapter_FromEnv_Escaping(t *testing.T) {
	t.Setenv("DATABASE_USER", "user with space")
	t.Setenv("DATABASE_PASS", "pass'quote\\slash")
	t.Setenv("DATABASE_HOST", "localhost")
	t.Setenv("DATABASE_PORT", "5432")
	t.Setenv("DATABASE_NAME", "my db")
	t.Setenv("DATABASE_TIMEZONE", "Asia/Shanghai")
	_ = os.Unsetenv("DATABASE_DSN")

	ad := postgres.FromEnv()
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	pd, ok := dialector.(*gormpostgres.Dialector)
	if !ok {
		t.Fatalf("expected *gormpostgres.Dialector, got %T", dialector)
	}

	wantDSN := "host=localhost user='user with space' password='pass\\'quote\\\\slash' dbname='my db' port=5432 TimeZone=Asia/Shanghai sslmode=disable"
	if pd.DSN != wantDSN {
		t.Fatalf("DSN mismatch: got %q, want %q", pd.DSN, wantDSN)
	}
}

// TestAdapter_FromEnv_DirectDSN 验证 DATABASE_DSN 覆盖分项配置。
func TestAdapter_FromEnv_DirectDSN(t *testing.T) {
	t.Setenv("DATABASE_USER", "ignored_user")
	t.Setenv("DATABASE_DSN", "postgres://override_user:pass@remote:5432/override_db?sslmode=disable")

	ad := postgres.FromEnv()
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	pd, ok := dialector.(*gormpostgres.Dialector)
	if !ok {
		t.Fatalf("expected *gormpostgres.Dialector, got %T", dialector)
	}

	wantDSN := "postgres://override_user:pass@remote:5432/override_db?sslmode=disable"
	if pd.DSN != wantDSN {
		t.Fatalf("DSN mismatch: got %q, want %q", pd.DSN, wantDSN)
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
			ad:   postgres.New("   "),
		},
		{
			name: "invalid URL without host",
			ad:   postgres.New("postgres://postgres:" + sensitivePassword + "@/dbname"),
		},
		{
			name: "invalid DSN format without key value or url scheme",
			ad:   postgres.New("invalid_dsn_plain_text_" + sensitivePassword),
		},
		{
			name: "empty port via FromEnv",
			ad: func() dialect.Adapter {
				_ = os.Unsetenv("DATABASE_DSN")
				t.Setenv("DATABASE_PASS", sensitivePassword)
				t.Setenv("DATABASE_PORT", "   ")
				return postgres.FromEnv()
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

// TestAdapter_InitRegistration 验证 init 函数向 dialect.Register 成功注册 DriverPostgres。
func TestAdapter_InitRegistration(t *testing.T) {
	_ = os.Unsetenv("DATABASE_DSN")
	t.Setenv("DATABASE_USER", "registry_user")
	t.Setenv("DATABASE_PASS", "reg_pass")
	t.Setenv("DATABASE_HOST", "reg_host")
	t.Setenv("DATABASE_PORT", "5432")
	t.Setenv("DATABASE_NAME", "reg_db")
	t.Setenv("DATABASE_TIMEZONE", "Asia/Shanghai")

	ad, err := dialect.Lookup(dialect.DriverPostgres)
	if err != nil {
		t.Fatalf("expected DriverPostgres to be registered in dialect, got error: %v", err)
	}
	if ad == nil {
		t.Fatal("expected non-nil adapter from Lookup")
	}
	if ad.Name() != "postgres" {
		t.Fatalf("expected adapter.Name() to be 'postgres', got %q", ad.Name())
	}

	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open error: %v", err)
	}
	pd, ok := dialector.(*gormpostgres.Dialector)
	if !ok {
		t.Fatalf("expected *gormpostgres.Dialector, got %T", dialector)
	}

	wantDSN := "host=reg_host user=registry_user password=reg_pass dbname=reg_db port=5432 TimeZone=Asia/Shanghai sslmode=disable"
	if pd.DSN != wantDSN {
		t.Fatalf("DSN mismatch from registered factory: got %q, want %q", pd.DSN, wantDSN)
	}
}

// TestRealEngine_Postgres 真实 PostgreSQL 引擎测试：
// 仅在显式设置 BAMBOO_TEST_POSTGRES_DSN 时执行真实连接测试，默认自动 skip。
func TestRealEngine_Postgres(t *testing.T) {
	dsn := os.Getenv("BAMBOO_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("skipping real PostgreSQL engine test; set BAMBOO_TEST_POSTGRES_DSN to run")
	}

	ad := postgres.New(dsn)
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
	defer func() { _ = sqlDB.Close() }()

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
