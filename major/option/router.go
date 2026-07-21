package option

import (
	"context"

	"github.com/gin-gonic/gin"
)

// RouteRegistrar 路由注册器，接收已装配依赖的上下文与 Gin 引擎进行路由挂载。
//
// 多个注册器可叠加，由 Register 在 Exec + engineInit 后按注册顺序逐个执行。
// 插件可直接暴露 RouteRegistrar 供业务侧通过 [WithRoute] 导入，
// 实现「插件自带路由、业务侧一行 Option 接入」的装配方式。
//
// ctx 为 Register 装配完成后的上下文（reg.Init.Ctx，未经 Runner 的 WithCancel 包裹）。RouteRegistrar 不应依赖此 ctx 的 Done() 信号做后台任务——应使用 per-request gin.Context 或 Runner 的 goroutineFunc 入口。已包含通过
// [WithDatabase] / [WithCache] 等声明的 DB、缓存管理器等组件实例，
// 业务侧可直接用 xCtxUtil.MustGetDB(ctx) / MustGetCacheManager(ctx) 取用，
// 无需自行从 *xReg.Reg 读取，避免 context 值语义导致的「装配前捕获」陷阱。
type RouteRegistrar func(ctx context.Context, serve *gin.Engine)

// WithRoute 注册一个或多个路由注册器到配置中。
//
// 多次调用可叠加多个注册器，执行顺序与调用顺序一致。nil 注册器会被跳过，
// 确保条件构造（如 cond && WithRoute(r)）的安全性。
// 支持一次传入多个注册器，底层循环追加到 routes 列表。
func WithRoute(rs ...RouteRegistrar) Option {
	return func(c *Config) {
		for _, r := range rs {
			if r != nil {
				c.routes = append(c.routes, r)
			}
		}
	}
}

// WithRouteGroup 注册一个带前缀的路由组注册器。
//
// 等价于在 [WithRoute] 内手动调用 serve.Group(prefix) 后再注册子路由，
// 作为语法糖使用，便于按业务模块划分路由前缀。prefix 为空串时退化为 [WithRoute]。
// r 闭包内若需访问 DB/缓存等组件，应通过 gin.Context 的请求上下文取用
// （InjectContext 中间件已把 reg.Init.Ctx 注入每个请求），而非依赖注册阶段的外层 ctx。
func WithRouteGroup(prefix string, r func(rg *gin.RouterGroup)) Option {
	return func(c *Config) {
		if r == nil {
			return
		}
		c.routes = append(c.routes, func(ctx context.Context, serve *gin.Engine) {
			r(serve.Group(prefix))
		})
	}
}
