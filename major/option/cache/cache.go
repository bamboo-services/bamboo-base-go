// Package xOptCache 缓存配置子包，定义 [CacheConfig] 与 [CacheOption] 及各后端构造策略。
//
// 与 [github.com/bamboo-services/bamboo-base-go/major/option/database] 子包对称：
//   - 外层 [CacheConfig] 为数据载体，字段小写只读，仅通过 getter 暴露
//   - [CacheOption] 为修改函数，直接作用于 *CacheConfig
//   - [WithRedis] / [WithMemory] / [FromEnv] 均返回 [CacheOption]，由父包 [option.WithCache] 包裹为顶层 Option
//
// 该子包不 import option 父包，避免循环依赖。
package xOptCache

import (
	"time"

	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
	xCache "github.com/bamboo-services/bamboo-base-go/major/cache"
)

// CacheType 缓存实现类型，标识框架内置缓存的后端选择。
//
// 类型定义为 [xCache.CacheType] 的别名，让缓存类型定义统一收敛到 major/cache 包。
// 由 [WithRedis]、[WithMemory] 隐式设置，或由 [WithCacheType] 显式声明。
// 零值空串等价于 [CacheTypeNone]，表示不启用内置缓存实现。
type CacheType = xCache.CacheType

const (
	// CacheTypeRedis 使用 Redis 作为缓存后端，适用于分布式与跨进程共享场景。
	CacheTypeRedis = xCache.CacheTypeRedis
	// CacheTypeMemory 使用程序内内存作为缓存后端，适用于单实例或可接受最终一致性的场景。
	CacheTypeMemory = xCache.CacheTypeMemory
	// CacheTypeNone 不启用内置缓存实现，业务侧可自行通过 Register 注册缓存节点。
	CacheTypeNone = xCache.CacheTypeNone
)

// CacheConfig 缓存配置，描述缓存后端类型及对应实现的具体参数。
//
// 字段均为小写，仅通过 getter 暴露只读视图，避免下游直接修改内部状态。
// 构造请使用对应后端的构造函数（[WithRedis] / [WithMemory]）或 [FromEnv]。
type CacheConfig struct {
	typeVal CacheType
	redis   RedisOptions
	memory  MemoryOptions
}

// Type 返回缓存实现类型。
func (c CacheConfig) Type() CacheType { return c.typeVal }

// Enabled 返回是否启用了内置缓存实现。
//
// 零值（未设置任何缓存选项）与显式声明 CacheTypeNone 均视为未启用，
// Register 据此决定是否装配内置缓存节点。
func (c CacheConfig) Enabled() bool {
	return c.typeVal != "" && c.typeVal != CacheTypeNone
}

// Redis 返回 Redis 缓存选项。仅当 Type 为 CacheTypeRedis 时有效。
func (c CacheConfig) Redis() RedisOptions { return c.redis }

// Memory 返回内存缓存选项。仅当 Type 为 CacheTypeMemory 时有效。
func (c CacheConfig) Memory() MemoryOptions { return c.memory }

// RedisOptions Redis 缓存连接参数（[CacheConfig] 的 Redis 后端专属配置）。
//
// 字段语义与 github.com/redis/go-redis/v9 的 Options 对齐，
// 由 Register 内部据此构造 redis.Client。
type RedisOptions struct {
	Addr         string        // 主机地址，格式 host:port，如 "localhost:6379"
	Username     string        // 用户名（Redis 6+ ACL），留空表示无
	Password     string        // 密码，留空表示无
	DB           int           // 数据库序号
	PoolSize     int           // 连接池大小，0 表示使用客户端默认值
	MinIdleConns int           // 最小空闲连接数
	DialTimeout  time.Duration // 连接建立超时
	ReadTimeout  time.Duration // 读操作超时
	WriteTimeout time.Duration // 写操作超时
}

// MemoryOptions 程序内内存缓存参数（[CacheConfig] 的 Memory 后端专属配置）。
type MemoryOptions struct {
	DefaultTTL time.Duration // 默认过期时间，0 表示永不过期
	MaxEntries int           // 最大条目数，0 表示无上限
	ShardCount int           // 分片数（提升并发），0 表示使用默认分片
}

// CacheOption 是 [CacheConfig] 的统一二级选项。
//
// 直接作用于 [CacheConfig] 整体，[WithRedis] / [WithMemory] / [WithCacheType] / [FromEnv]
// 均返回 CacheOption，由父包 option.WithCache 包裹为顶层 Option。
type CacheOption func(*CacheConfig)

// WithCacheType 直接指定缓存实现类型，返回 [CacheOption]。
//
// 通常无需手动调用，WithRedis / WithMemory 会隐式设置类型。
// 仅在需要显式声明 CacheTypeNone（禁用内置缓存）时使用。
func WithCacheType(t CacheType) CacheOption {
	return func(c *CacheConfig) { c.typeVal = t }
}

// FromEnv 从环境变量自动装配缓存配置的 [CacheOption]。
//
// 读取顺序与优先级:
//   - NOSQL_DRIVER 为 "redis" 时，按 NOSQL_HOST/PORT/USER/PASS/DATABASE/POOL_SIZE
//     自动拼装 Redis 连接参数并装配 Redis 后端
//   - NOSQL_DRIVER 为 "memory" 时，按 NOSQL_MEMORY_DEFAULT_TTL/MAX_ENTRIES/SHARD_COUNT
//     装配程序内内存缓存后端
//   - NOSQL_DRIVER 为空或 "none" 时返回 nil，表示不启用内置缓存
//
// Redis 连接池超时等高级参数暂未从环境变量读取（保持 env 列表精简），如需调整请配合
// [WithRedisDialTimeout] / [WithRedisReadTimeout] 等二级选项显式设置。
// Memory 的分片等参数同理，可用 [WithMemoryShardCount] 等二级选项覆盖。
//
// 该函数依赖 .env 已在 Register 阶段通过 godotenv 加载完成。
//
// 返回值可能为 nil（未启用内置缓存），父包 [option.WithCache] 会跳过 nil 选项。
func FromEnv() CacheOption {
	switch xCache.CacheType(xEnv.GetEnvString(xEnv.NoSqlDriver, "none")) {
	case CacheTypeRedis:
		return redisFromEnvOption()
	case CacheTypeMemory:
		return memoryFromEnvOption()
	default:
		return nil
	}
}

// redisFromEnvOption 从环境变量拼装 Redis 连接参数，返回 [CacheOption]。
//
// 读取的环境变量:
//   - NOSQL_HOST       (默认 localhost)
//   - NOSQL_PORT       (默认 6379)
//   - NOSQL_USER       (默认 空，非 ACL 模式留空)
//   - NOSQL_PASS       (默认 空)
//   - NOSQL_DATABASE   (默认 0)
//   - NOSQL_POOL_SIZE  (默认 0，go-redis 按 CPU 数自适应)
//
// addr 拼装为 host:port，其余参数通过 [RedisOption] 二级选项按需叠加，
// 空值/零值项不传递，保持 go-redis 默认行为。
func redisFromEnvOption() CacheOption {
	addr := xEnv.GetEnvString(xEnv.NoSqlHost, "localhost") + ":" +
		xEnv.GetEnvString(xEnv.NoSqlPort, "6379")

	opts := []RedisOption{
		WithRedisUsername(xEnv.GetEnvString(xEnv.NoSqlUser, "")),
		WithRedisPassword(xEnv.GetEnvString(xEnv.NoSqlPass, "")),
		WithRedisDB(xEnv.GetEnvInt(xEnv.NoSqlDatabase, 0)),
	}
	if poolSize := xEnv.GetEnvInt(xEnv.NoSqlPoolSize, 0); poolSize > 0 {
		opts = append(opts, WithRedisPoolSize(poolSize))
	}

	return WithRedis(addr, opts...)
}

// memoryFromEnvOption 从环境变量拼装内存缓存参数，返回 [CacheOption]。
//
// 读取的环境变量及零值语义（与 [xCacheMemory.NewStore] 兜底行为对齐）:
//   - NOSQL_MEMORY_DEFAULT_TTL  支持 Go Duration 字符串如 30m/1h/0；0 表示永不过期。
//     未设置时保持 [WithMemory] 默认值 30m。
//   - NOSQL_MEMORY_MAX_ENTRIES  0 表示无上限。未设置或 <= 0 时均不传递，
//     由 [xCacheMemory.NewStore] 兜底为无上限，避免依赖 [WithMemory] 零值巧合。
//   - NOSQL_MEMORY_SHARD_COUNT  0 表示使用默认分片数。未设置或 <= 0 时均不传递，
//     由 [xCacheMemory.NewStore] 兜底为 16 分片。
//
// 仅在环境变量显式设置且语义有效（> 0）时才叠加对应 [MemoryOption]，
// 使 .env 中 "0=用默认/无上限" 的注释与代码行为直接对齐。
func memoryFromEnvOption() CacheOption {
	var opts []MemoryOption
	if ttl, exists := xEnv.GetEnv(xEnv.NoSqlMemoryDefaultTTL); exists {
		if d, err := time.ParseDuration(ttl); err == nil {
			opts = append(opts, WithMemoryDefaultTTL(d))
		}
	}
	if _, exists := xEnv.GetEnv(xEnv.NoSqlMemoryMaxEntries); exists {
		if maxEntries := xEnv.GetEnvInt(xEnv.NoSqlMemoryMaxEntries, 0); maxEntries > 0 {
			opts = append(opts, WithMemoryMaxEntries(maxEntries))
		}
	}
	if _, exists := xEnv.GetEnv(xEnv.NoSqlMemoryShardCount); exists {
		if shardCount := xEnv.GetEnvInt(xEnv.NoSqlMemoryShardCount, 0); shardCount > 0 {
			opts = append(opts, WithMemoryShardCount(shardCount))
		}
	}
	return WithMemory(opts...)
}
