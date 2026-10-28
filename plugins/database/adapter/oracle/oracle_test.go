package oracle_test

import (
	"os"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/oracle/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
	gormoracle "github.com/oracle-samples/gorm-oracle/oracle"
	"gorm.io/gorm"
)

// TestAdapter_Interface 验证 oracle.New 与 FromEnv 满足 dialect.Adapter 接口契约。
func TestAdapter_Interface(t *testing.T) {
	assertAdapter := func(dialect.Adapter) {}
	assertAdapter(oracle.New(`user="system" password="pwd" connectString="localhost:1521/ORCLPDB1"`))
	assertAdapter(oracle.FromEnv())
}

// TestAdapter_Name 验证适配器 Name() 返回正确驱动名称 "oracle"。
func TestAdapter_Name(t *testing.T) {
	ad := oracle.New(`user="system" password="pwd" connectString="localhost:1521/ORCLPDB1"`)
	if got := ad.Name(); got != "oracle" {
		t.Fatalf("expected Name() to be 'oracle', got %q", got)
	}

	adEnv := oracle.FromEnv()
	if got := adEnv.Name(); got != "oracle" {
		t.Fatalf("expected FromEnv().Name() to be 'oracle', got %q", got)
	}
}

// TestAdapter_Open_Type 验证 Open 返回的 Dialector 类型为 *oracle.Dialector。
func TestAdapter_Open_Type(t *testing.T) {
	ad := oracle.New(`user="system" password="pwd" connectString="localhost:1521/ORCLPDB1"`)
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	if dialector == nil {
		t.Fatal("expected non-nil dialector")
	}

	if _, ok := dialector.(*gormoracle.Dialector); !ok {
		t.Fatalf("expected *gormoracle.Dialector, got %T", dialector)
	}
	if dialector.Name() != "oracle" {
		t.Fatalf("expected dialector.Name() to be 'oracle', got %q", dialector.Name())
	}
}

// TestAdapter_FromEnv_Assembly 验证从环境变量各字段正常拼装 DSN。
func TestAdapter_FromEnv_Assembly(t *testing.T) {
	t.Setenv("DATABASE_USER", "scott")
	t.Setenv("DATABASE_PASS", "tiger")
	t.Setenv("DATABASE_HOST", "dbhost")
	t.Setenv("DATABASE_PORT", "1521")
	t.Setenv("DATABASE_SERVICE_NAME", "orclpdb1")
	t.Setenv("DATABASE_LIB_DIR", "/opt/instantclient")
	_ = os.Unsetenv("DATABASE_DSN")

	ad := oracle.FromEnv()
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	od, ok := dialector.(*gormoracle.Dialector)
	if !ok {
		t.Fatalf("expected *gormoracle.Dialector, got %T", dialector)
	}

	wantDSN := `user="scott" password="tiger" connectString="dbhost:1521/orclpdb1" libDir="/opt/instantclient"`
	if od.DataSourceName != wantDSN {
		t.Fatalf("DSN mismatch: got %q, want %q", od.DataSourceName, wantDSN)
	}
}

// TestAdapter_FromEnv_Defaults 验证环境变量未设置时使用标准默认值且无 libDir。
func TestAdapter_FromEnv_Defaults(t *testing.T) {
	_ = os.Unsetenv("DATABASE_DSN")
	_ = os.Unsetenv("DATABASE_USER")
	_ = os.Unsetenv("DATABASE_PASS")
	_ = os.Unsetenv("DATABASE_HOST")
	_ = os.Unsetenv("DATABASE_PORT")
	_ = os.Unsetenv("DATABASE_SERVICE_NAME")
	_ = os.Unsetenv("DATABASE_LIB_DIR")

	ad := oracle.FromEnv()
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	od, ok := dialector.(*gormoracle.Dialector)
	if !ok {
		t.Fatalf("expected *gormoracle.Dialector, got %T", dialector)
	}

	wantDSN := `user="system" password="" connectString="localhost:1521/ORCLPDB1"`
	if od.DataSourceName != wantDSN {
		t.Fatalf("DSN mismatch: got %q, want %q", od.DataSourceName, wantDSN)
	}
	if strings.Contains(od.DataSourceName, "libDir") {
		t.Fatalf("expected no libDir in default DSN, got %q", od.DataSourceName)
	}
}

// TestAdapter_FromEnv_DirectDSN 验证 DATABASE_DSN 覆盖分项配置。
func TestAdapter_FromEnv_DirectDSN(t *testing.T) {
	t.Setenv("DATABASE_USER", "ignored_user")
	t.Setenv("DATABASE_DSN", `user="custom" password="secret" connectString="remote:1521/custompdb"`)

	ad := oracle.FromEnv()
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	od, ok := dialector.(*gormoracle.Dialector)
	if !ok {
		t.Fatalf("expected *gormoracle.Dialector, got %T", dialector)
	}

	wantDSN := `user="custom" password="secret" connectString="remote:1521/custompdb"`
	if od.DataSourceName != wantDSN {
		t.Fatalf("DSN mismatch: got %q, want %q", od.DataSourceName, wantDSN)
	}
}

// TestAdapter_FromEnv_Escaping 验证用户名和密码包含引号等字符时使用 %q 安全转义。
func TestAdapter_FromEnv_Escaping(t *testing.T) {
	t.Setenv("DATABASE_USER", `user"quote`)
	t.Setenv("DATABASE_PASS", `pass\slash"quote`)
	t.Setenv("DATABASE_HOST", "localhost")
	t.Setenv("DATABASE_PORT", "1521")
	t.Setenv("DATABASE_SERVICE_NAME", "ORCLPDB1")
	_ = os.Unsetenv("DATABASE_LIB_DIR")
	_ = os.Unsetenv("DATABASE_DSN")

	ad := oracle.FromEnv()
	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	od, ok := dialector.(*gormoracle.Dialector)
	if !ok {
		t.Fatalf("expected *gormoracle.Dialector, got %T", dialector)
	}

	wantDSN := `user="user\"quote" password="pass\\slash\"quote" connectString="localhost:1521/ORCLPDB1"`
	if od.DataSourceName != wantDSN {
		t.Fatalf("DSN mismatch: got %q, want %q", od.DataSourceName, wantDSN)
	}
}

// TestAdapter_DSNValidationAndNoPasswordLeakage 验证配置错误保存并在 Open() 时返回，且错误信息绝不泄露密码。
func TestAdapter_DSNValidationAndNoPasswordLeakage(t *testing.T) {
	sensitivePassword := "SuperSecretOraclePassword123!"

	tests := []struct {
		name string
		ad   dialect.Adapter
	}{
		{
			name: "empty DSN via New",
			ad:   oracle.New("   "),
		},
		{
			name: "invalid DSN plain text with sensitive password",
			ad:   oracle.New("invalid_dsn_string_with_password_" + sensitivePassword),
		},
		{
			name: "invalid URL without host",
			ad:   oracle.New("oracle://scott:" + sensitivePassword + "@/orcl"),
		},
		{
			name: "empty port via FromEnv",
			ad: func() dialect.Adapter {
				_ = os.Unsetenv("DATABASE_DSN")
				t.Setenv("DATABASE_PASS", sensitivePassword)
				t.Setenv("DATABASE_PORT", "   ")
				return oracle.FromEnv()
			}(),
		},
		{
			name: "empty host via FromEnv",
			ad: func() dialect.Adapter {
				_ = os.Unsetenv("DATABASE_DSN")
				t.Setenv("DATABASE_PASS", sensitivePassword)
				t.Setenv("DATABASE_HOST", "   ")
				return oracle.FromEnv()
			}(),
		},
		{
			name: "empty service name via FromEnv",
			ad: func() dialect.Adapter {
				_ = os.Unsetenv("DATABASE_DSN")
				t.Setenv("DATABASE_PASS", sensitivePassword)
				t.Setenv("DATABASE_SERVICE_NAME", "   ")
				return oracle.FromEnv()
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

// TestAdapter_InitRegistration 验证 init 函数向 dialect.Register 成功注册 DriverOracle。
func TestAdapter_InitRegistration(t *testing.T) {
	_ = os.Unsetenv("DATABASE_DSN")
	t.Setenv("DATABASE_USER", "registry_user")
	t.Setenv("DATABASE_PASS", "reg_pass")
	t.Setenv("DATABASE_HOST", "reg_host")
	t.Setenv("DATABASE_PORT", "1521")
	t.Setenv("DATABASE_SERVICE_NAME", "reg_pdb")
	_ = os.Unsetenv("DATABASE_LIB_DIR")

	ad, err := dialect.Lookup(dialect.DriverOracle)
	if err != nil {
		t.Fatalf("expected DriverOracle to be registered in dialect, got error: %v", err)
	}
	if ad == nil {
		t.Fatal("expected non-nil adapter from Lookup")
	}
	if ad.Name() != "oracle" {
		t.Fatalf("expected adapter.Name() to be 'oracle', got %q", ad.Name())
	}

	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("unexpected Open error: %v", err)
	}
	od, ok := dialector.(*gormoracle.Dialector)
	if !ok {
		t.Fatalf("expected *gormoracle.Dialector, got %T", dialector)
	}

	wantDSN := `user="registry_user" password="reg_pass" connectString="reg_host:1521/reg_pdb"`
	if od.DataSourceName != wantDSN {
		t.Fatalf("DSN mismatch from registered factory: got %q, want %q", od.DataSourceName, wantDSN)
	}
}

// TestNoNetworkOrConnectionOnConstruction 验证 New/FromEnv/Open 不主动建连或连网络。
func TestNoNetworkOrConnectionOnConstruction(t *testing.T) {
	// 指向一个不可达的保留测试 IP 与端口
	nonExistentDSN := `user="system" password="pwd" connectString="192.0.2.1:1521/NONEXISTENT"`
	adp := oracle.New(nonExistentDSN)

	// 验证 Open 只是构造 Dialector，不报错也不尝试网络连接
	dialector, err := adp.Open()
	if err != nil {
		t.Fatalf("Open 不应报错: %v", err)
	}
	if dialector == nil {
		t.Fatal("dialector 期望非空")
	}
}

// TestRealEngine_Oracle 真实 Oracle 引擎测试：
// 仅在显式设置 BAMBOO_TEST_ORACLE_DSN 或 BAMBOO_TEST_ORACLE 时执行，默认自动 skip。
// 不连生产/默认地址，不启动容器，未设置时跳过且不计验收。
func TestRealEngine_Oracle(t *testing.T) {
	dsn := os.Getenv("BAMBOO_TEST_ORACLE_DSN")
	if dsn == "" {
		dsn = os.Getenv("BAMBOO_TEST_ORACLE")
	}
	if strings.TrimSpace(dsn) == "" {
		t.Skip("skipping real Oracle engine test; set BAMBOO_TEST_ORACLE_DSN to run")
	}

	ad := oracle.New(dsn)
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
	if err := db.Raw("SELECT 1 FROM DUAL").Scan(&one).Error; err != nil {
		t.Fatalf("query SELECT 1 FROM DUAL failed: %v", err)
	}
	if one != 1 {
		t.Fatalf("expected 1, got %d", one)
	}
}
