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

// Config 缓存模块配置，描述后端类型及各后端详细连接/存储参数。
type Config struct {
	TypeVal       Type
	Redis         RedisOptions
	Memory        MemoryOptions
	FromEnv       bool
	Logger        *logging.Logger
	validationErr error
}

// Option 定义 Cache 模块与 Manager 的通用配置函数。
type Option func(*Config)

// RedisOptions Redis 缓存连接参数。
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

// RedisOption 是 [RedisOptions] 的二级选项。
type RedisOption func(*RedisOptions)

// MemoryOption 是 [MemoryOptions] 的二级选项。
type MemoryOption func(*MemoryOptions)

// WithType 显式指定缓存实现类型。
func WithType(t Type) Option {
	return func(c *Config) { c.TypeVal = t }
}

// WithRedis 配置 Redis 作为缓存后端。
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

// WithMemory 配置程序内内存作为缓存后端。
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

// Get 从上下文中提取缓存管理器实例 *Manager。
// 基于 application.Get[*Manager](ctx, ctxkey.CacheManagerKey) 实现，不设全局 fallback。
func Get(ctx context.Context) (*Manager, *apperr.Error) {
	return application.Get[*Manager](ctx, ctxkey.CacheManagerKey)
}

// MustGet 从上下文中提取缓存管理器实例 *Manager，若获取失败则 panic。
func MustGet(ctx context.Context) *Manager {
	mgr, err := Get(ctx)
	if err != nil {
		panic(err)
	}
	return mgr
}
