package option

import (
	xOptCache "github.com/bamboo-services/bamboo-base-go/major/option/cache"
)

// 以下类型为 [xOptCache] 子包类型的别名重导出，保持父包对外 API 兼容，
// 使 init 包与业务侧可继续通过 xOption.CacheConfig / xOption.CacheType 等访问。
//
// 构造函数（WithRedis / WithMemory / FromEnv 等）已迁移至 [xOptCache] 子包，
// 业务侧应使用 xOption.WithCache(xOptCache.WithRedis(...)) 的两层调用形态。
type (
	// CacheType 缓存实现类型，详见 [xOptCache.CacheType]。
	CacheType = xOptCache.CacheType

	// CacheConfig 缓存配置，详见 [xOptCache.CacheConfig]。
	CacheConfig = xOptCache.CacheConfig

	// CacheOption 缓存二级选项，详见 [xOptCache.CacheOption]。
	CacheOption = xOptCache.CacheOption

	// RedisOptions Redis 缓存连接参数，详见 [xOptCache.RedisOptions]。
	RedisOptions = xOptCache.RedisOptions

	// MemoryOptions 程序内内存缓存参数，详见 [xOptCache.MemoryOptions]。
	MemoryOptions = xOptCache.MemoryOptions
)

// 缓存类型常量重导出，保持 xOption.CacheTypeRedis 等旧引用兼容。
const (
	CacheTypeRedis  = xOptCache.CacheTypeRedis
	CacheTypeMemory = xOptCache.CacheTypeMemory
	CacheTypeNone   = xOptCache.CacheTypeNone
)

// WithCache 将 [xOptCache.CacheOption] 包裹为顶层 [Option]，供 Register 使用。
//
// 该函数对标 [WithDatabase]，是 cache 两层设计的顶层入口。
// 内部逐个执行传入的 CacheOption 修改 [CacheConfig]，完成后写入聚合 Config。
// nil CacheOption 会被跳过，支持条件构造（如 cond && xOptCache.WithRedis(...)）。
//
// 使用示例：
//
//	// Redis 后端
//	xOption.WithCache(xOptCache.WithRedis("localhost:6379"))
//	// Memory 后端
//	xOption.WithCache(xOptCache.WithMemory(xOptCache.WithMemoryDefaultTTL(30*time.Minute)))
//	// 从环境变量装配
//	xOption.WithCache(xOptCache.FromEnv())
func WithCache(opts ...xOptCache.CacheOption) Option {
	return func(c *Config) {
		for _, o := range opts {
			if o != nil {
				o(&c.cache)
			}
		}
	}
}
