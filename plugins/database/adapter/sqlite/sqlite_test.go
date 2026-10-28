package sqlite_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/sqlite/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
	"gorm.io/gorm"
)

// TestAdapter_InterfaceAndRegistration 验证 Adapter 接口实现与 init 全局注册查找。
func TestAdapter_InterfaceAndRegistration(t *testing.T) {
	adp, err := dialect.Lookup(dialect.DriverSQLite)
	if err != nil {
		t.Fatalf("dialect.Lookup(DriverSQLite) 失败: %v", err)
	}
	if adp == nil {
		t.Fatal("期望返回非空 Adapter，实际为 nil")
	}
	if got := adp.Name(); got != string(dialect.DriverSQLite) {
		t.Fatalf("Name() = %q, 期望 %q", got, dialect.DriverSQLite)
	}

	dialector, err := adp.Open()
	if err != nil {
		t.Fatalf("Open() 失败: %v", err)
	}
	if dialector == nil {
		t.Fatal("期望返回非空 gorm.Dialector，实际为 nil")
	}
	if dialector.Name() != "sqlite" {
		t.Fatalf("Dialector.Name() = %q, 期望 sqlite", dialector.Name())
	}
}

// TestNew_ExplicitDSNAndErrors 验证 New(dsn) 的正常与空字符串/空白字符错误延迟返回。
func TestNew_ExplicitDSNAndErrors(t *testing.T) {
	tests := []struct {
		name      string
		dsn       string
		expectErr bool
	}{
		{
			name:      "内存数据库",
			dsn:       ":memory:",
			expectErr: false,
		},
		{
			name:      "文件路径带空格修剪",
			dsn:       "  file:test.db?cache=shared  ",
			expectErr: false,
		},
		{
			name:      "空字符串错误延迟",
			dsn:       "",
			expectErr: true,
		},
		{
			name:      "纯空白字符串错误延迟",
			dsn:       "   \t\n  ",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adp := sqlite.New(tt.dsn)
			if adp.Name() != "sqlite" {
				t.Fatalf("Name() = %q, 期望 sqlite", adp.Name())
			}

			dialector, err := adp.Open()
			if tt.expectErr {
				if err == nil {
					t.Fatalf("期望 Open 返回错误，实际返回 nil，dialector=%v", dialector)
				}
				if dialector != nil {
					t.Fatalf("发生错误时 dialector 应为 nil，实际为 %v", dialector)
				}
			} else {
				if err != nil {
					t.Fatalf("Open 失败: %v", err)
				}
				if dialector == nil {
					t.Fatal("期望 dialector 非 nil")
				}
			}
		})
	}
}

// TestFromEnv_FieldPrecedenceAndDefaults 验证环境变量优先级及缺省值逻辑。
func TestFromEnv_FieldPrecedenceAndDefaults(t *testing.T) {
	// 1. 无任何环境变量设置，默认 ":memory:"
	_ = os.Unsetenv(string(env.DatabaseDSN))
	_ = os.Unsetenv(string(env.DatabasePath))

	adp := sqlite.FromEnv()
	dialector, err := adp.Open()
	if err != nil {
		t.Fatalf("默认 FromEnv 失败: %v", err)
	}
	if dialector == nil {
		t.Fatal("默认 dialector 期望非空")
	}

	// 2. 仅设置 DATABASE_PATH
	t.Setenv(string(env.DatabasePath), "custom.db")
	_ = os.Unsetenv(string(env.DatabaseDSN))
	adp = sqlite.FromEnv()
	dialector, err = adp.Open()
	if err != nil {
		t.Fatalf("设置 DATABASE_PATH 时 Open 失败: %v", err)
	}
	if dialector == nil {
		t.Fatal("dialector 期望非空")
	}

	// 3. 同时设置 DATABASE_DSN 和 DATABASE_PATH，DATABASE_DSN 优先
	t.Setenv(string(env.DatabaseDSN), "override.db?cache=shared&mode=rwc")
	t.Setenv(string(env.DatabasePath), "ignored.db")
	adp = sqlite.FromEnv()
	dialector, err = adp.Open()
	if err != nil {
		t.Fatalf("设置 DATABASE_DSN 时 Open 失败: %v", err)
	}
	if dialector == nil {
		t.Fatal("dialector 期望非空")
	}

	// 4. DATABASE_PATH 设置为空白字符，期望 Open 报告错误
	_ = os.Unsetenv(string(env.DatabaseDSN))
	t.Setenv(string(env.DatabasePath), "   ")
	adp = sqlite.FromEnv()
	dialector, err = adp.Open()
	if err == nil {
		t.Fatalf("DATABASE_PATH 为空白时 Open 应返回错误，实际返回 nil")
	}
	if dialector != nil {
		t.Fatalf("发生错误时 dialector 应为 nil，实际为 %v", dialector)
	}
}

// TestNoNetworkOrConnectionOnConstruction 验证 New/FromEnv/Open 不主动建连或连网络。
func TestNoNetworkOrConnectionOnConstruction(t *testing.T) {
	// 指向一个当前不存在的深层目录（若建连尝试创建则可能失败或产生副作用）
	nonExistentPath := filepath.Join(os.TempDir(), "non-existent-sub-dir-12345", "test.db")
	adp := sqlite.New(nonExistentPath)

	// 验证 Open 只是构造 Dialector，不报错也不尝试 touch 磁盘文件或建连
	dialector, err := adp.Open()
	if err != nil {
		t.Fatalf("Open 不应报错: %v", err)
	}
	if dialector == nil {
		t.Fatal("dialector 期望非空")
	}

	// 文件不应在 Open 时被创建
	if _, statErr := os.Stat(nonExistentPath); !os.IsNotExist(statErr) {
		t.Fatalf("Open 阶段不应创建数据库文件: %s", nonExistentPath)
	}
}

// TestSQLite_RealReadWrite_TempFile 验证基于临时文件的真实 SQLite 基础读写。
func TestSQLite_RealReadWrite_TempFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "sqlite_test.db")

	adp := sqlite.New(dbFile)
	dialector, err := adp.Open()
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}

	// 用真实的 gorm.Open 验证此 Dialector
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open 失败: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取底层 sql.DB 失败: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("sqlDB.Ping 失败: %v", err)
	}

	// 真实建表与读写操作
	type TestItem struct {
		ID   uint   `gorm:"primaryKey"`
		Name string `gorm:"size:255"`
	}

	if err := db.AutoMigrate(&TestItem{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}

	item := TestItem{Name: "bamboo-v2-sqlite"}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("db.Create 失败: %v", err)
	}
	if item.ID == 0 {
		t.Fatal("期望写入后 ID > 0")
	}

	var found TestItem
	if err := db.First(&found, item.ID).Error; err != nil {
		t.Fatalf("db.First 失败: %v", err)
	}
	if found.Name != "bamboo-v2-sqlite" {
		t.Fatalf("查询数据不匹配: got %q, want %q", found.Name, "bamboo-v2-sqlite")
	}
}

// TestOtherEngines_DefaultSkip 验证其它真实外部引擎测试默认 skip，未开启 BAMBOO_TEST_* 开关时不计验收。
func TestOtherEngines_DefaultSkip(t *testing.T) {
	engines := []struct {
		envKey string
		name   string
	}{
		{"BAMBOO_TEST_MYSQL", "MySQL"},
		{"BAMBOO_TEST_POSTGRES", "PostgreSQL"},
		{"BAMBOO_TEST_ORACLE", "Oracle"},
		{"BAMBOO_TEST_SQLSERVER", "SQLServer"},
	}

	for _, eng := range engines {
		t.Run(eng.name, func(t *testing.T) {
			if strings.TrimSpace(os.Getenv(eng.envKey)) == "" {
				t.Skipf("未设置 %s，跳过 %s 真实引擎测试（默认 skip，不连生产/默认地址，不启容器）", eng.envKey, eng.name)
			}
			t.Logf("已开启 %s 测试", eng.name)
		})
	}
}
