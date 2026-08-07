package xInit_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"strings"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"

	xOption "github.com/bamboo-services/bamboo-base-go/major/option"
	xOptDatabase "github.com/bamboo-services/bamboo-base-go/major/option/database"
	xInit "github.com/bamboo-services/bamboo-base-go/major/register/init"
)

// mockSQLDriver / mockConnector 提供内存占位连接池，使 gorm.Open 能持有 *sql.DB，
// 供 DatabaseInit 的 applyPool 配置连接池参数（不实际执行任何 SQL）。
type mockSQLDriver struct{}

func (mockSQLDriver) Open(string) (driver.Conn, error) { return nil, nil }

type mockConnector struct{}

func (mockConnector) Connect(context.Context) (driver.Conn, error) { return nil, nil }
func (mockConnector) Driver() driver.Driver                        { return mockSQLDriver{} }

// mockDialector 测试用最小 Dialector 实现，仅用于验证 DatabaseInit 的 Dialector 注入路径。
//
// 框架 core 不内置任何数据库驱动，真实驱动的集成测试位于对应驱动插件
// （如 plugins/database/sqlite）中，此处仅验证注入与装配逻辑。
type mockDialector struct{}

func (mockDialector) Name() string { return "mock" }
func (mockDialector) Initialize(db *gorm.DB) error {
	db.ConnPool = sql.OpenDB(mockConnector{})
	return nil
}
func (mockDialector) Migrator(*gorm.DB) gorm.Migrator                       { return nil }
func (mockDialector) DataTypeOf(*schema.Field) string                       { return "" }
func (mockDialector) DefaultValueOf(*schema.Field) clause.Expression        { return nil }
func (mockDialector) BindVarTo(clause.Writer, *gorm.Statement, interface{}) {}
func (mockDialector) QuoteTo(clause.Writer, string)                         {}
func (mockDialector) Explain(sql string, vars ...interface{}) string        { return "" }

// TestDatabaseInit_UnregisteredDriver 验证 WithDriver 但未 import 对应驱动插件时，
// DatabaseInit 在 openByDriver 阶段 fail-fast 报错（不 panic、不产生 *gorm.DB）。
func TestDatabaseInit_UnregisteredDriver(t *testing.T) {
	cfg := xOption.Apply(
		xOption.WithDatabase(
			xOptDatabase.WithDriver(xOptDatabase.DriverSQLite, ":memory:"),
		),
	).Database()

	node := xInit.DatabaseInit(cfg)
	result, err := node(context.Background())
	if err == nil {
		t.Fatalf("未注册驱动应报错，实际返回 nil err, result=%v", result)
	}
	if !strings.Contains(err.Error(), "不支持的数据库驱动") {
		t.Errorf("错误文案未提示驱动未注册: %v", err)
	}
}

// TestDatabaseInit_CustomDialector 验证 WithDialector 注入自定义驱动时 DatabaseInit 成功，
// 且仅注入 Dialector（无驱动枚举）时 Enabled() 仍为 true。
func TestDatabaseInit_CustomDialector(t *testing.T) {
	cfg := xOption.Apply(
		xOption.WithDatabase(
			xOptDatabase.WithDialector(mockDialector{}),
		),
	).Database()
	if !cfg.Enabled() {
		t.Error("注入 Dialector 后 Enabled() 应为 true")
	}

	node := xInit.DatabaseInit(cfg)
	result, err := node(context.Background())
	if err != nil {
		t.Fatalf("DatabaseInit 失败: %v", err)
	}
	if _, ok := result.(*gorm.DB); !ok {
		t.Fatalf("返回类型非 *gorm.DB: %T", result)
	}
}

// TestFromEnv_WithDialectorWhenDriverUnset 验证 FromEnv 在 DATABASE_DRIVER 未设置时，
// 仍应用显式传入的 opts（如 WithDialector），使 FromEnv(WithDialector(...)) 组合形式生效。
func TestFromEnv_WithDialectorWhenDriverUnset(t *testing.T) {
	t.Setenv("DATABASE_DRIVER", "none")

	cfg := xOption.Apply(
		xOption.WithDatabase(
			xOptDatabase.FromEnv(xOptDatabase.WithDialector(mockDialector{})),
		),
	).Database()
	if !cfg.Enabled() {
		t.Error("FromEnv(WithDialector) 在 driver 为空时应启用数据库")
	}
	if cfg.Dialector() == nil {
		t.Error("FromEnv 应保留 WithDialector 注入的 Dialector")
	}
}

// TestFromEnv_UnknownDriverNoPanic 验证 DATABASE_DRIVER 拼写错误时 FromEnv 不 panic，
// 由后续 DatabaseInit 阶段统一报"不支持的数据库驱动"。
func TestFromEnv_UnknownDriverNoPanic(t *testing.T) {
	t.Setenv("DATABASE_DRIVER", "mysqll")

	cfg := xOption.Apply(
		xOption.WithDatabase(xOptDatabase.FromEnv()),
	).Database()
	if !cfg.Enabled() {
		t.Error("未知 driver 也应设置 driver 使 Enabled() 为 true")
	}

	// DatabaseInit 应返回错误而非 panic
	node := xInit.DatabaseInit(cfg)
	_, err := node(context.Background())
	if err == nil {
		t.Fatal("未知 driver 时 DatabaseInit 应报错")
	}
	if !strings.Contains(err.Error(), "不支持的数据库驱动") {
		t.Errorf("错误文案未提示驱动不支持: %v", err)
	}
}
