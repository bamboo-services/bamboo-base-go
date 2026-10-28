package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
)

// Config 描述缓存模块的配置参数，包含后端选择、连接参数与存储选项。
type Config struct {
	TypeVal Type            // 指定当前启用的缓存实现类型（如 [CacheTypeRedis] 或 [CacheTypeMemory]）。
	Redis   RedisOptions    // 保存 Redis 模式下的网络连接、鉴权与连接池参数。
	Memory  MemoryOptions   // 保存内存模式下的分片、淘汰与容量参数。
	FromEnv bool            // 标识是否启用了从环境变量自动识别与填充配置。
	Logger  *logging.Logger // 传入应用日志实例，用于打印缓存内部调试与生命周期日志。

	validationErr error
}

// Option 定义作用于 [*Config] 的函数式配置选项。
type Option func(*Config)

// RedisOptions 定义 Redis 后端的网络连接、鉴权与连接池参数。
type RedisOptions struct {
	Addr         string        // Redis 服务的网络地址，格式为 "host:port"，如 "localhost:6379"。
	Username     string        // Redis 6+ ACL 认证用户名，未开启 ACL 时留空。
	Password     string        // Redis 连接认证密码，无密码时留空。
	DB           int           // Redis 数据库序号，默认为 0。
	PoolSize     int           // 客户端连接池最大连接数，0 表示使用客户端默认值。
	MinIdleConns int           // 连接池保持的最小空闲连接数。
	DialTimeout  time.Duration // 建立网络连接的超时时间。
	ReadTimeout  time.Duration // 执行读操作的超时时间。
	WriteTimeout time.Duration // 执行写操作的超时时间。
}

// MemoryOptions 定义程序内分片内存后端的容量与淘汰参数。
type MemoryOptions struct {
	DefaultTTL time.Duration // 未显式指定写入 TTL 时的默认过期时间；0 表示永不过期。
	MaxEntries int           // 每个分片允许容纳的最大条目数；0 表示无上限。
	ShardCount int           // 内部并发分片数，必须为 2 的幂；0 表示使用默认 16 分片。
}

// RedisOption 是 [RedisOptions] 的二级选项。
type RedisOption func(*RedisOptions)

// MemoryOption 是 [MemoryOptions] 的二级选项。
type MemoryOption func(*MemoryOptions)

// WithType 显式指定缓存实现类型。
//
// 传入未支持的类型将在生命周期 Start 阶段返回错误。
func WithType(t Type) Option {
	return func(c *Config) { c.TypeVal = t }
}

// WithRedis 配置 Redis 作为缓存后端。
//
// addr 为必须提供的网络地址（格式为 host:port）；opts 为可选的连接与连接池配置。
// 若此前调用过 [WithMemory]，本选项将覆盖其后端设置。
func WithRedis(addr string, opts ...RedisOption) Option {
	return func(c *Config) {
		c.TypeVal = CacheTypeRedis
		c.Redis = RedisOptions{Addr: addr}
		for _, o := range opts {
			if o != nil {
				o(&c.Redis)
			}
		}
	}
}

// WithMemory 配置程序内分片内存作为缓存后端。
//
// 默认过期时间初始化为 30 分钟，可通过 opts 中的 [WithMemoryDefaultTTL] 调整。
// 若此前调用过 [WithRedis]，本选项将覆盖其后端设置。
func WithMemory(opts ...MemoryOption) Option {
	return func(c *Config) {
		c.TypeVal = CacheTypeMemory
		c.Memory = MemoryOptions{DefaultTTL: 30 * time.Minute}
		for _, o := range opts {
			if o != nil {
				o(&c.Memory)
			}
		}
	}
}

// FromEnv 声明从环境变量自动装配缓存配置。
//
// 对应环境变量:
//   - NOSQL_DRIVER: redis / memory / none
//   - NOSQL_HOST / NOSQL_PORT / NOSQL_USER / NOSQL_PASS / NOSQL_DATABASE / NOSQL_POOL_SIZE
//   - NOSQL_MEMORY_DEFAULT_TTL / NOSQL_MEMORY_MAX_ENTRIES / NOSQL_MEMORY_SHARD_COUNT
func FromEnv() Option {
	return func(c *Config) {
		c.FromEnv = true
		c.applyFromEnv()
	}
}

// WithLogger 显式注入应用 Logger。
//
// 注入后缓存模块将派生名为 "CACH" 的子日志记录器输出内部日志。
func WithLogger(l *logging.Logger) Option {
	return func(c *Config) {
		c.Logger = l
	}
}

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
func WithRedisDialTimeout(d time.Duration) RedisOption {
	return func(r *RedisOptions) { r.DialTimeout = d }
}

// WithRedisReadTimeout 设置读操作超时。
func WithRedisReadTimeout(d time.Duration) RedisOption {
	return func(r *RedisOptions) { r.ReadTimeout = d }
}

// WithRedisWriteTimeout 设置写操作超时。
func WithRedisWriteTimeout(d time.Duration) RedisOption {
	return func(r *RedisOptions) { r.WriteTimeout = d }
}

// WithMemoryDefaultTTL 设置内存缓存默认过期时间。
func WithMemoryDefaultTTL(d time.Duration) MemoryOption {
	return func(m *MemoryOptions) { m.DefaultTTL = d }
}

// WithMemoryMaxEntries 设置内存缓存最大条目数。
func WithMemoryMaxEntries(n int) MemoryOption {
	return func(m *MemoryOptions) { m.MaxEntries = n }
}

// WithMemoryShardCount 设置内存缓存分片数。
func WithMemoryShardCount(n int) MemoryOption {
	return func(m *MemoryOptions) { m.ShardCount = n }
}

func (c *Config) applyFromEnv() {
	driverStr := "none"
	if raw, ok := env.Lookup(env.NoSQLDriver); ok && !raw.IsEmpty() {
		driverStr = raw.String()
	}

	switch Type(driverStr) {
	case CacheTypeRedis:
		host := "localhost"
		if raw, ok := env.Lookup(env.NoSQLHost); ok && !raw.IsEmpty() {
			host = raw.String()
		}
		port := "6379"
		if raw, ok := env.Lookup(env.NoSQLPort); ok && !raw.IsEmpty() {
			port = raw.String()
		}

		c.TypeVal = CacheTypeRedis
		c.Redis.Addr = host + ":" + port

		if raw, ok := env.Lookup(env.NoSQLUser); ok {
			c.Redis.Username = raw.String()
		}
		if raw, ok := env.Lookup(env.NoSQLPass); ok {
			c.Redis.Password = raw.String()
		}
		if raw, ok := env.Lookup(env.NoSQLDatabase); ok {
			val, err := raw.Int()
			if err != nil {
				c.validationErr = errors.Join(c.validationErr, fmt.Errorf("invalid NOSQL_DATABASE: %w", err))
			} else {
				c.Redis.DB = val
			}
		}
		if raw, ok := env.Lookup(env.NoSQLPoolSize); ok {
			val, err := raw.Int()
			if err != nil {
				c.validationErr = errors.Join(c.validationErr, fmt.Errorf("invalid NOSQL_POOL_SIZE: %w", err))
			} else if val > 0 {
				c.Redis.PoolSize = val
			}
		}

	case CacheTypeMemory:
		c.TypeVal = CacheTypeMemory
		if c.Memory.DefaultTTL <= 0 {
			c.Memory.DefaultTTL = 30 * time.Minute
		}
		if raw, ok := env.Lookup(env.NoSQLMemoryDefaultTTL); ok {
			if d, err := time.ParseDuration(raw.String()); err == nil {
				c.Memory.DefaultTTL = d
			} else {
				c.validationErr = errors.Join(c.validationErr, fmt.Errorf("invalid NOSQL_MEMORY_DEFAULT_TTL: %w", err))
			}
		}
		if raw, ok := env.Lookup(env.NoSQLMemoryMaxEntries); ok {
			val, err := raw.Int()
			if err != nil {
				c.validationErr = errors.Join(c.validationErr, fmt.Errorf("invalid NOSQL_MEMORY_MAX_ENTRIES: %w", err))
			} else if val > 0 {
				c.Memory.MaxEntries = val
			}
		}
		if raw, ok := env.Lookup(env.NoSQLMemoryShardCount); ok {
			val, err := raw.Int()
			if err != nil {
				c.validationErr = errors.Join(c.validationErr, fmt.Errorf("invalid NOSQL_MEMORY_SHARD_COUNT: %w", err))
			} else if val > 0 {
				c.Memory.ShardCount = val
			}
		}

	default:
		c.TypeVal = CacheTypeNone
	}
}

// Get 从请求或应用上下文中提取强类型的缓存管理器实例 [*Manager]。
//
// 底层基于 [application.Get] 实现，查找 [ctxkey.CacheManagerKey]。
// 当插件未注册或上下文类型不符时返回结构化错误，不使用任何全局备选单例。
func Get(ctx context.Context) (*Manager, *apperr.Error) {
	return application.Get[*Manager](ctx, ctxkey.CacheManagerKey)
}

// MustGet 从上下文中提取缓存管理器实例 [*Manager]，若获取失败则立即 panic。
//
// 仅建议在初始化阶段或确信缓存必已就绪的核心流程中使用。
func MustGet(ctx context.Context) *Manager {
	mgr, err := Get(ctx)
	if err != nil {
		panic(err)
	}
	return mgr
}
