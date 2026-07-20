package option

import (
	xOptDatabase "github.com/bamboo-services/bamboo-base-go/major/option/database"
)

// Option 定义应用级配置选项，采用函数式选项模式（functional options）。
//
// 该模式与 plugins/grpc/runner、plugins/cron/runner 中的 Option 保持一致，
// 用于在 Runner 启动阶段声明式地选择框架内置组件的实现（缓存后端、数据库驱动等），
// 而非由业务侧自行编写初始化节点。Option 仅描述「选哪种实现 + 用什么参数」，
// 具体的装配逻辑由 Runner 内部根据 [Config] 完成。
//
// 使用示例：
//
//	cfg := option.Apply(
//	    option.WithCache(xOptCache.WithRedis("localhost:6379", xOptCache.WithRedisPassword("xxx"))),
//	    option.WithDatabase(
//	        xOptDatabase.FromEnv(),
//	        xOptDatabase.WithAutoMigrate(&entity.Role{}, &entity.User{}),
//	    ),
//	)
type Option func(*Config)

// Config 是应用运行期配置的聚合体，由 Runner 消费并据此装配内置组件。
//
// 所有字段均为小写且不可变（仅通过 getter 暴露只读视图），避免下游
// 在拿到 Config 后直接修改内部状态，保证配置在装配阶段的一致性。
type Config struct {
	cache    CacheConfig
	database xOptDatabase.DatabaseConfig
	routes   []RouteRegistrar
}

// Apply 将传入的选项逐个应用到 [Config]，返回装配完成的配置实例。
//
// nil 选项会被跳过，确保 WithXxx 条件构造（如 cond && WithRedis(...)）的安全性。
// 未设置的组件保持零值，对应 Type/Driver 为 "none"，表示不启用该内置实现。
func Apply(opts ...Option) *Config {
	cfg := &Config{}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
	return cfg
}

// Cache 返回缓存配置的只读视图。
func (c *Config) Cache() CacheConfig { return c.cache }

// Database 返回数据库配置的只读视图。
func (c *Config) Database() xOptDatabase.DatabaseConfig { return c.database }

// Routes 返回路由注册器列表，按 WithRoute / WithRouteGroup 的调用顺序排列。
//
// Runner 会在启动 HTTP 前按此顺序逐个执行，每个 [RouteRegistrar] 接收
// 已装配依赖的 reg.Init.Ctx（含 DB/缓存等组件）与 Gin 引擎。返回 nil 表示无路由需注册。
func (c *Config) Routes() []RouteRegistrar { return c.routes }
