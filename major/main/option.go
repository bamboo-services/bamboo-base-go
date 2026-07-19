package xMain

import (
	xCtx "github.com/bamboo-services/bamboo-base-go/defined/context"
	xOption "github.com/bamboo-services/bamboo-base-go/major/option"
	xInit "github.com/bamboo-services/bamboo-base-go/major/register/init"
)

// initOption 应用配置选项，装配内置组件并挂载路由。
//
// 执行流程：
//  1. 通过 [xOption.Apply] 装配所有 Option，得到聚合配置
//  2. 若数据库配置已启用，通过 [xInit.DatabaseInit] 构造初始化节点并调用
//     [RegNode.UseAfterExec] 在 Exec 完成后补注册到 DatabaseKey
//  3. 若缓存配置已启用：
//     - 通过 [xInit.CacheInit] 构造 [*xCache.Manager]，注册到 CacheManagerKey
//     - 若为 Redis 后端，额外通过 [xInit.RedisClientFromManager] 把 *redis.Client
//       补注册到 RedisClientKey，保持与历史代码兼容
//  4. 按配置中的路由注册器列表顺序，逐个对 Gin 引擎挂载路由；每个注册器
//     接收 reg.Init.Ctx（已含 database/cache 等装配结果）与 reg.Serve
//
// 装配顺序固定为 database → cache → routes，确保数据库与缓存先于路由可用。
func (runner *mainRunner) initOption() {
	cfg := xOption.Apply(runner.opts...)

	if dc := cfg.Database(); dc.Enabled() {
		runner.reg.Init.UseAfterExec(xCtx.DatabaseKey, xInit.DatabaseInit(dc))
	}
	if cc := cfg.Cache(); cc.Enabled() {
		runner.reg.Init.UseAfterExec(xCtx.CacheManagerKey, xInit.CacheInit(cc))
		// Redis 后端：补注册 *redis.Client 到 RedisClientKey，兼容历史 API
		if cc.Type() == xOption.CacheTypeRedis {
			runner.reg.Init.UseAfterExec(xCtx.RedisClientKey, xInit.RedisClientFromManager())
		}
	}

	for _, registrar := range cfg.Routes() {
		registrar(runner.reg.Init.Ctx, runner.reg.Serve)
	}
}
