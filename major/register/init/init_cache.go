package xInit

import (
	"context"
	"fmt"
	"log/slog"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtx "github.com/bamboo-services/bamboo-base-go/defined/context"
	xCache "github.com/bamboo-services/bamboo-base-go/major/cache"
	xCacheMemory "github.com/bamboo-services/bamboo-base-go/major/cache/memory"
	xOption "github.com/bamboo-services/bamboo-base-go/major/option"
	xRegNode "github.com/bamboo-services/bamboo-base-go/major/register/node"
	"github.com/redis/go-redis/v9"
)

// CacheInit 根据传入的 [xOption.CacheConfig] 构造缓存初始化节点。
//
// 返回的 Node 会根据 [CacheConfig.Type] 选择对应后端：
//   - CacheTypeRedis：使用 go-redis 构造 *redis.Client 并 Ping 验证，封装进 [*xCache.Manager]
//   - CacheTypeMemory：构造 [*xCacheMemory.Store]（含分片 + TTL + janitor），封装进 [*xCache.Manager]
//
// 返回值统一为 [*xCache.Manager]，由调用方注册到 [xCtx.CacheManagerKey]。
// 若 Type 为 CacheTypeNone，调用方应跳过此工厂。
//
// Redis 后端兼容性：为保持与历史代码（从 [xCtx.RedisClientKey] 取 *redis.Client）兼容，
// 调用方应在装配 Manager 后，额外通过 [RedisClientFromManager] 把 *redis.Client
// 补注册到 [xCtx.RedisClientKey]。
func CacheInit(cfg xOption.CacheConfig) xRegNode.Node {
	return func(ctx context.Context) (any, error) {
		log := xLog.WithName(xLog.NamedINIT)
		log.Debug(ctx, "正在连接缓存", slog.String("type", string(cfg.Type())))

		switch cfg.Type() {
		case xOption.CacheTypeRedis:
			manager, err := initRedisCache(ctx, cfg.Redis(), log)
			if err != nil {
				return nil, err
			}
			return manager, nil

		case xOption.CacheTypeMemory:
			manager := initMemoryCache(cfg.Memory(), log)
			log.Info(ctx, "缓存连接成功", slog.String("type", string(cfg.Type())))
			return manager, nil

		default:
			return nil, fmt.Errorf("不支持的缓存类型: %s", cfg.Type())
		}
	}
}

// initRedisCache 构造 Redis 客户端并验证连通性，返回封装后的 [*xCache.Manager]。
func initRedisCache(ctx context.Context, rOpts xOption.RedisOptions, log *xLog.LogNamedLogger) (*xCache.Manager, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         rOpts.Addr,
		Username:     rOpts.Username,
		Password:     rOpts.Password,
		DB:           rOpts.DB,
		PoolSize:     rOpts.PoolSize,
		MinIdleConns: rOpts.MinIdleConns,
		DialTimeout:  rOpts.DialTimeout,
		ReadTimeout:  rOpts.ReadTimeout,
		WriteTimeout: rOpts.WriteTimeout,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis 连接失败: %w", err)
	}
	log.Info(ctx, "缓存连接成功", slog.String("type", string(xCache.CacheTypeRedis)))
	return xCache.NewManager(xCache.CacheTypeRedis,
		xCache.WithRedisClient(client),
		xCache.WithLogger(log),
	), nil
}

// initMemoryCache 构造内存存储实例并封装进 [*xCache.Manager]。
func initMemoryCache(mOpts xOption.MemoryOptions, log *xLog.LogNamedLogger) *xCache.Manager {
	store := xCacheMemory.NewStore(mOpts.ShardCount, mOpts.MaxEntries, mOpts.DefaultTTL)
	return xCache.NewManager(xCache.CacheTypeMemory,
		xCache.WithMemoryStore(store),
		xCache.WithManagerTTL(mOpts.DefaultTTL),
		xCache.WithLogger(log),
	)
}

// RedisClientFromManager 返回一个 Node，从已注册的 [xCtx.CacheManagerKey] 中
// 提取 [*xCache.Manager]，再返回其持有的 *redis.Client。
//
// 仅供 Redis 后端使用，用于把 *redis.Client 补注册到 [xCtx.RedisClientKey]，
// 保持与历史代码（[xCtxUtil.MustGetRDB] / [xCtxUtil.GetRDB]）的兼容性。
// 若 Manager 不存在或后端非 Redis，返回 nil。
func RedisClientFromManager() xRegNode.Node {
	return func(ctx context.Context) (any, error) {
		val := ctx.Value(xCtx.CacheManagerKey)
		if val == nil {
			return nil, nil
		}
		manager, ok := val.(*xCache.Manager)
		if !ok {
			return nil, nil
		}
		return manager.Redis(), nil
	}
}
