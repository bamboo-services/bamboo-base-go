package xReg

import (
	"context"
	"testing"

	xCtx "github.com/bamboo-services/bamboo-base-go/defined/context"
	xCache "github.com/bamboo-services/bamboo-base-go/major/cache"
	xOption "github.com/bamboo-services/bamboo-base-go/major/option"
	xOptCache "github.com/bamboo-services/bamboo-base-go/major/option/cache"
	xOptDatabase "github.com/bamboo-services/bamboo-base-go/major/option/database"
	xRegNode "github.com/bamboo-services/bamboo-base-go/major/register/node"
	"gorm.io/gorm"
)

// TestRegisterOptionUptoExec 验证业务节点在 Register 带上 DB+Cache opts 后，
// 能从 ctx 拿到 *gorm.DB 和 *xCache.Manager，且 reg.Init.Ctx 也包含它们。
func TestRegisterOptionUptoExec(t *testing.T) {
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
		xOption.WithDatabase(xOptDatabase.SQLite(":memory:")),
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