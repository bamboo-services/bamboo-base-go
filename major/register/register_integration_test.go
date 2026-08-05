package xReg

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"

	xDB "github.com/bamboo-services/bamboo-base-go/common/database"
	xCtx "github.com/bamboo-services/bamboo-base-go/defined/context"
	xCache "github.com/bamboo-services/bamboo-base-go/major/cache"
	xOption "github.com/bamboo-services/bamboo-base-go/major/option"
	xOptCache "github.com/bamboo-services/bamboo-base-go/major/option/cache"
	xOptDatabase "github.com/bamboo-services/bamboo-base-go/major/option/database"
	xRegNode "github.com/bamboo-services/bamboo-base-go/major/register/node"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// mockSQLDriver / mockConnector 提供内存占位连接池，使 gorm.Open 能持有 *sql.DB，
// 供 DatabaseInit 的 applyPool 配置连接池参数（不实际执行任何 SQL）。
type mockSQLDriver struct{}

func (mockSQLDriver) Open(string) (driver.Conn, error) { return nil, nil }

type mockConnector struct{}

func (mockConnector) Connect(context.Context) (driver.Conn, error) { return nil, nil }
func (mockConnector) Driver() driver.Driver                        { return mockSQLDriver{} }

// mockDialector 测试用 Dialector，验证 Register 装配链路（无需真实数据库驱动）。
type mockDialector struct{}

func (mockDialector) Name() string { return "mock" }
func (mockDialector) Initialize(db *gorm.DB) error {
	db.ConnPool = sql.OpenDB(mockConnector{})
	return nil
}
func (mockDialector) Migrator(*gorm.DB) gorm.Migrator               { return nil }
func (mockDialector) DataTypeOf(*schema.Field) string               { return "" }
func (mockDialector) DefaultValueOf(*schema.Field) clause.Expression { return nil }
func (mockDialector) BindVarTo(clause.Writer, *gorm.Statement, interface{}) {}
func (mockDialector) QuoteTo(clause.Writer, string)                  {}
func (mockDialector) Explain(sql string, vars ...interface{}) string { return "" }

// TestRegisterOptionUptoExec 验证业务节点在 Register 带上 DB+Cache opts 后，
// 能从 ctx 拿到 *gorm.DB 和 *xCache.Manager，且 reg.Init.Ctx 也包含它们。
//
// 使用注册到 common 驱动注册表的 mock Dialector 验证装配链路，框架 core 不依赖
// 任何真实数据库驱动；真实驱动的端到端验证由各驱动插件（plugins/database/*）负责。
func TestRegisterOptionUptoExec(t *testing.T) {
	// 注册测试专用 mock 驱动到 common 注册表，验证 openByDriver 的注册表路径。
	const mockDriver = xDB.Driver("mock-register-test")
	xDB.RegisterDialector(mockDriver, func(dsn string) gorm.Dialector { return mockDialector{} })

	// 业务节点：断言 ctx 含 DB 和 CacheManager
	var gotDB, gotCache any
	businessNode := xRegNode.RegNodeList{
		Key: xCtx.ContextKey("test_business_node"),
		Node: func(ctx context.Context) (any, error) {
			gotDB = ctx.Value(xCtx.DatabaseKey)
			gotCache = ctx.Value(xCtx.CacheManagerKey)
			return nil, nil
		},
	}

	reg := Register(context.Background(),
		[]xRegNode.RegNodeList{businessNode},
		xOption.WithDatabase(xOptDatabase.WithDriver(mockDriver, ":memory:")),
		xOption.WithCache(xOptCache.WithMemory()),
	)

	// 断言业务节点拿到了 DB
	if gotDB == nil {
		t.Fatal("业务节点未从 ctx 拿到 *gorm.DB（DatabaseKey 为空）")
	}
	if _, ok := gotDB.(*gorm.DB); !ok {
		t.Fatalf("业务节点拿到的 DatabaseKey 不是 *gorm.DB，实际类型: %T", gotDB)
	}

	// 断言业务节点拿到了 Cache
	if gotCache == nil {
		t.Fatal("业务节点未从 ctx 拿到 *xCache.Manager（CacheManagerKey 为空）")
	}
	if _, ok := gotCache.(*xCache.Manager); !ok {
		t.Fatalf("业务节点拿到的 CacheManagerKey 不是 *xCache.Manager，实际类型: %T", gotCache)
	}

	// 断言 Register 返回的 reg 也包含组件
	if reg.Init == nil || reg.Serve == nil {
		t.Fatal("Register 返回的 reg.Init 或 reg.Serve 为空")
	}
	if reg.Init.Ctx.Value(xCtx.DatabaseKey) == nil {
		t.Fatal("reg.Init.Ctx 未包含 DatabaseKey")
	}
	if reg.Init.Ctx.Value(xCtx.CacheManagerKey) == nil {
		t.Fatal("reg.Init.Ctx 未包含 CacheManagerKey")
	}
}
