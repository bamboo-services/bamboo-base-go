package option

import (
	"time"

	xCache "github.com/bamboo-services/bamboo-base-go/major/cache"
)

// CacheType 缓存实现类型，标识框架内置缓存的后端选择。
//
// 类型定义为 [xCache.CacheType] 的别名，保持 option 包对外 API 向后兼容的同时，
// 让缓存类型定义统一收敛到 major/cache 包，避免重复声明。
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
// 仅通过 Type / Redis / Memory 暴露只读视图，避免下游直接修改内部状态。
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
// Runner 据此决定是否装配内置缓存节点。
func (c CacheConfig) Enabled() bool {
	return c.typeVal != "" && c.typeVal != CacheTypeNone
}

// Redis 返回 Redis 缓存选项。仅当 Type 为 CacheTypeRedis 时有效。
func (c CacheConfig) Redis() RedisOptions { return c.redis }

// Memory 返回内存缓存选项。仅当 Type 为 CacheTypeMemory 时有效。
func (c CacheConfig) Memory() MemoryOptions { return c.memory }

// RedisOptions Redis 缓存连接参数。
//
// 字段语义与 github.com/redis/go-redis/v9 的 Options 对齐，
// 由 Runner 内部据此构造 redis.Client。
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

// MemoryOptions 程序内内存缓存参数。
type MemoryOptions struct {
	DefaultTTL time.Duration // 默认过期时间，0 表示永不过期
	MaxEntries int           // 最大条目数，0 表示无上限
	ShardCount int           // 分片数（提升并发），0 表示使用默认分片
}

// WithCacheType 直接指定缓存实现类型。
//
// 通常无需手动调用，WithRedis / WithMemory 会隐式设置类型。
// 仅在需要显式声明 CacheTypeNone（禁用内置缓存）时使用。
func WithCacheType(t CacheType) Option {
	return func(c *Config) { c.cache.typeVal = t }
}

// WithRedis 配置 Redis 作为缓存后端，并隐式将缓存类型置为 CacheTypeRedis。
//
// addr 为必填项（host:port）；其余参数通过 RedisOption 可变参数按需设置。
func WithRedis(addr string, opts ...RedisOption) Option {
	return func(c *Config) {
		c.cache.typeVal = CacheTypeRedis
		c.cache.redis = RedisOptions{Addr: addr}
		for _, o := range opts {
			if o != nil {
				o(&c.cache.redis)
			}
		}
	}
}

// WithMemory 配置程序内内存作为缓存后端，并隐式将缓存类型置为 CacheTypeMemory。
//
// 可通过 MemoryOption 可变参数调整默认 TTL、容量等。
func WithMemory(opts ...MemoryOption) Option {
	return func(c *Config) {
		c.cache.typeVal = CacheTypeMemory
		c.cache.memory = MemoryOptions{DefaultTTL: 30 * time.Minute}
		for _, o := range opts {
			if o != nil {
				o(&c.cache.memory)
			}
		}
	}
}

// RedisOption 是 RedisOptions 的二级选项，避免 WithRedis 参数列表过长。
type RedisOption func(*RedisOptions)

// WithRedisUsername 设置 Redis ACL 用户名。
func WithRedisUsername(u string) RedisOption { return func(r *RedisOptions) { r.Username = u } }

// WithRedisPassword 设置 Redis 密码。
func WithRedisPassword(p string) RedisOption { return func(r *RedisOptions) { r.Password = p } }

// WithRedisDB 设置 Redis 数据库序号。
func WithRedisDB(db int) RedisOption { return func(r *RedisOptions) { r.DB = db } }

// WithRedisPoolSize 设置连接池大小。
func WithRedisPoolSize(n int) RedisOption { return func(r *RedisOptions) { r.PoolSize = n } }

// WithRedisMinIdleConns 设置最小空闲连接数。
func WithRedisMinIdleConns(n int) RedisOption { return func(r *RedisOptions) { r.MinIdleConns = n } }

// WithRedisDialTimeout 设置连接建立超时。
func WithRedisDialTimeout(d time.Duration) RedisOption { return func(r *RedisOptions) { r.DialTimeout = d } }

// WithRedisReadTimeout 设置读操作超时。
func WithRedisReadTimeout(d time.Duration) RedisOption { return func(r *RedisOptions) { r.ReadTimeout = d } }

// WithRedisWriteTimeout 设置写操作超时。
func WithRedisWriteTimeout(d time.Duration) RedisOption { return func(r *RedisOptions) { r.WriteTimeout = d } }

// MemoryOption 是 MemoryOptions 的二级选项。
type MemoryOption func(*MemoryOptions)

// WithMemoryDefaultTTL 设置默认过期时间。
func WithMemoryDefaultTTL(d time.Duration) MemoryOption { return func(m *MemoryOptions) { m.DefaultTTL = d } }

// WithMemoryMaxEntries 设置最大条目数。
func WithMemoryMaxEntries(n int) MemoryOption { return func(m *MemoryOptions) { m.MaxEntries = n } }

// WithMemoryShardCount 设置分片数。
func WithMemoryShardCount(n int) MemoryOption { return func(m *MemoryOptions) { m.ShardCount = n } }
