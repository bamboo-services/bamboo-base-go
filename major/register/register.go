package xReg

import (
	"context"

	xCtx "github.com/bamboo-services/bamboo-base-go/defined/context"
	xOption "github.com/bamboo-services/bamboo-base-go/major/option"
	xInit "github.com/bamboo-services/bamboo-base-go/major/register/init"
	xRegNode "github.com/bamboo-services/bamboo-base-go/major/register/node"
	"github.com/gin-gonic/gin"
)

// Reg 表示应用程序的核心注册结构，包含所有初始化后的组件实例。
type Reg struct {
	Serve *gin.Engine       // Gin 引擎实例
	Init  *xRegNode.RegNode // 初始化节点
}

// New 创建并返回一个未初始化的 `Reg` 实例。
//
// 该函数仅分配内存并返回 `Reg` 类型的初始值，
// 需要调用者进一步初始化相关字段。
//
// 返回值:
//   - `*Reg`: 返回一个新的 `Reg` 实例。
func newReg(ctx context.Context) *Reg {
	return &Reg{
		Init: xRegNode.NewRegNode(ctx),
	}
}

// Register 创建并初始化应用程序的核心注册中心，返回完全就绪的 Reg 实例。
//
// 该函数在一次调用内完成环境加载、日志器创建、内置基础设施装配、业务节点注册、
// Exec 执行、Gin 引擎构建与路由挂载，调用方拿到 *Reg 后即可直接交给 Runner 启动。
//
// 装配顺序（严格固定）：
//  1. 配置器与日志器（configInit / loggerInit）
//  2. 雪花算法节点（SnowflakeNodeKey，框架强制注册）
//  3. opts 中的数据库节点（DatabaseKey，仅当 DatabaseConfig.Enabled()）
//  4. opts 中的缓存节点（CacheManagerKey，仅当 CacheConfig.Enabled()）；
//     若为 Redis 后端，额外补注册 *redis.Client 到 RedisClientKey
//  5. nodeList 中的业务节点（按传入顺序）
//  6. 一次 Exec() 完成全部装配
//  7. Gin 引擎构建（engineInit）
//  8. opts 中的路由注册器逐个挂载到 Gin 引擎
//
// 参数:
//   - ctx: 根上下文，会随组件装配逐步 WithValue 演进
//   - nodeList: 业务侧自定义组件节点列表，按顺序在框架基础设施之后注册
//   - opts: 声明式配置选项，控制数据库/缓存/路由等内置组件的装配；
//     传入 nil 或空切片表示不启用任何内置实现
//
// 业务节点可通过 ctx.Value(xCtx.DatabaseKey) / ctx.Value(xCtx.CacheManagerKey)
// 等访问已装配的基础设施组件。
//
// 框架保留 ContextKey 约束：nodeList 不得包含 SnowflakeNodeKey / DatabaseKey /
// CacheManagerKey / RedisClientKey，否则 Use 会因重复注册 panic。框架会在
// nodeList 之前注册这些保留键。
func Register(ctx context.Context, nodeList []xRegNode.RegNodeList, opts ...xOption.Option) *Reg {
	reg := newReg(ctx)
	reg.configInit()
	reg.loggerInit()

	cfg := xOption.Apply(opts...)

	// 基础设施：雪花
	reg.Init.Use(xCtx.SnowflakeNodeKey, xInit.SnowflakeInit)
	// 基础设施：数据库（来自 opts）
	if dc := cfg.Database(); dc.Enabled() {
		reg.Init.Use(xCtx.DatabaseKey, xInit.DatabaseInit(dc))
	}
	// 基础设施：缓存（来自 opts）
	if cc := cfg.Cache(); cc.Enabled() {
		reg.Init.Use(xCtx.CacheManagerKey, xInit.CacheInit(cc))
		if cc.Type() == xOption.CacheTypeRedis {
			reg.Init.Use(xCtx.RedisClientKey, xInit.RedisClientFromManager())
		}
	}
	// 业务节点（来自 nodeList）
	for _, node := range nodeList {
		reg.Init.Use(node.Key, node.Node)
	}
	// 一次 Exec 完成全部装配
	reg.Init.Exec()

	// Gin 引擎
	reg.engineInit()

	// 路由注册（engineInit 之后，ctx 已含全部组件）
	// 注意：此处捕获的 reg.Init.Ctx 来自 Register 阶段，尚未被 Runner 的 WithCancel 包裹。
	// 组件值等价，但 RouteRegistrar 不应依赖此 ctx 的 Done() 信号驱动后台任务——
	// 后台任务请使用每请求的 gin.Context，或通过 Runner 的 goroutineFunc 入口接收被取消的 ctx。
	for _, registrar := range cfg.Routes() {
		registrar(reg.Init.Ctx, reg.Serve)
	}

	return reg
}
