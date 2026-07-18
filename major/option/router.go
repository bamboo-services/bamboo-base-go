package option

import "github.com/gin-gonic/gin"

// RouteRegistrar 路由注册器，接收 Gin 引擎进行路由挂载。
//
// 多个注册器可叠加，由 Runner 在启动 HTTP 前按注册顺序逐个执行。
// 插件可直接暴露 RouteRegistrar 供业务侧通过 [WithRoute] 导入，
// 实现「插件自带路由、业务侧一行 Option 接入」的装配方式。
type RouteRegistrar func(serve *gin.Engine)

// WithRoute 注册一个路由注册器到配置中。
//
// 多次调用可叠加多个注册器，执行顺序与调用顺序一致。nil 注册器会被跳过，
// 确保条件构造（如 cond && WithRoute(r)）的安全性。
func WithRoute(r RouteRegistrar) Option {
	return func(c *Config) {
		if r != nil {
			c.routes = append(c.routes, r)
		}
	}
}

// WithRouteGroup 注册一个带前缀的路由组注册器。
//
// 等价于在 [WithRoute] 内手动调用 serve.Group(prefix) 后再注册子路由，
// 作为语法糖使用，便于按业务模块划分路由前缀。prefix 为空串时退化为 [WithRoute]。
func WithRouteGroup(prefix string, r func(rg *gin.RouterGroup)) Option {
	return func(c *Config) {
		if r == nil {
			return
		}
		c.routes = append(c.routes, func(serve *gin.Engine) {
			r(serve.Group(prefix))
		})
	}
}
