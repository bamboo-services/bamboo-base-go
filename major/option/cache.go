package option

import (
	xOptionCache "github.com/bamboo-services/bamboo-base-go/major/option/cache"
)

// 以下类型为 [xOptionCache] 子包类型的别名重导出，保持父包对外 API 兼容，
// 使 init 包与业务侧可继续通过 xOption.CacheConfig / xOption.CacheType 等访问。
//
// 构造函数（WithRedis / WithMemory / FromEnv 等）已迁移至 [xOptionCache] 子包，
// 业务侧应使用 xOption.WithCache(xOptionCache.WithRedis(...)) 的两层调用形态。
type (
	// CacheType 缓存实现类型，详见 [xOptionCache.CacheType]。
	CacheType = xOptionCache.CacheType

	// CacheConfig 缓存配置，详见 [xOptionCache.CacheConfig]。
	CacheConfig = xOptionCache.CacheConfig

	// CacheOption 缓存二级选项，详见 [xOptionCache.CacheOption]。
	CacheOption = xOptionCache.CacheOption

	// RedisOptions Redis 缓存连接参数，详见 [xOptionCache.RedisOptions]。
	RedisOptions = xOptionCache.RedisOptions

	// MemoryOptions 程序内内存缓存参数，详见 [xOptionCache.MemoryOptions]。
	MemoryOptions = xOptionCache.MemoryOptions
)

// 缓存类型常量重导出，保持 xOption.CacheTypeRedis 等旧引用兼容。
const (
	CacheTypeRedis  = xOptionCache.CacheTypeRedis
	CacheTypeMemory = xOptionCache.CacheTypeMemory
	CacheTypeNone   = xOptionCache.CacheTypeNone
)

// WithCache 将 [xOptionCache.CacheOption] 包裹为顶层 [Option]，供 Runner 使用。
//
// 该函数对标 [WithDatabase]，是 cache 两层设计的顶层入口。
// 内部逐个执行传入的 CacheOption 修改 [CacheConfig]，完成后写入聚合 Config。
// nil CacheOption 会被跳过，支持条件构造（如 cond && xOptionCache.WithRedis(...)）。
//
// 使用示例：
//
//	// Redis 后端
//	xOption.WithCache(xOptionCache.WithRedis("localhost:6379"))
//	// Memory 后端
//	xOption.WithCache(xOptionCache.WithMemory(xOptionCache.WithMemoryDefaultTTL(30*time.Minute)))
//	// 从环境变量装配
//	xOption.WithCache(xOptionCache.FromEnv())
func WithCache(opts ...xOptionCache.CacheOption) Option {
	return func(c *Config) {
		for _, o := range opts {
			if o != nil {
				o(&c.cache)
			}
		}
	}
}
