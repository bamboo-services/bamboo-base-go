package cache

import (
	"sync"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
	"github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver"
	"github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory"
	"github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis"
	goredis "github.com/redis/go-redis/v9"
)

// 对外 API 重新导出 driver 包的类型与别名，方便业务侧统一使用 package cache。
type (
	// KeyCache 等价于 [driver.KeyCache]。
	KeyCache[K any, V any] = driver.KeyCache[K, V]
	// HashCache 等价于 [driver.HashCache]。
	HashCache[K any, F comparable, V any, S any] = driver.HashCache[K, F, V, S]
	// SetCache 等价于 [driver.SetCache]。
	SetCache[K any, V any] = driver.SetCache[K, V]
	// ListCache 等价于 [driver.ListCache]。
	ListCache[K any, V any] = driver.ListCache[K, V]
	// Codec 等价于 [driver.Codec]。
	Codec = driver.Codec
	// JSONCodec 等价于 [driver.JSONCodec]。
	JSONCodec = driver.JSONCodec
	// KeyEncoder 等价于 [driver.KeyEncoder]。
	KeyEncoder = driver.KeyEncoder
	// DefaultKeyEncoder 等价于 [driver.DefaultKeyEncoder]。
	DefaultKeyEncoder = driver.DefaultKeyEncoder
	// Type 等价于 [driver.CacheType]。
	Type = driver.CacheType
	// SetOption 等价于 [driver.SetOption]，写操作的函数式选项。
	SetOption = driver.SetOption
	// SetConfig 等价于 [driver.SetConfig]。
	SetConfig = driver.SetConfig
)

// 常量与函数别名：可直接引用 driver 包常量与工具函数。
const (
	// CacheTypeRedis 使用 Redis 作为缓存后端，适用于分布式与跨进程共享场景。
	CacheTypeRedis = driver.CacheTypeRedis
	// CacheTypeMemory 使用程序内内存作为缓存后端，适用于单实例或可接受最终一致性的场景。
	CacheTypeMemory = driver.CacheTypeMemory
	// CacheTypeNone 不启用内置缓存实现，业务侧可自行装配缓存节点。
	CacheTypeNone = driver.CacheTypeNone
)

// EncodeKey 等价于 [driver.EncodeKey]。
func EncodeKey(enc KeyEncoder, key any) string { return driver.EncodeKey(enc, key) }

// ApplySet 等价于 [driver.ApplySet]，将默认 TTL 与选项列表合成最终配置。
func ApplySet(base time.Duration, opts []SetOption) driver.SetConfig {
	return driver.ApplySet(base, opts)
}

// WithTTL 等价于 [driver.WithTTL]，设置本次写入的过期时间覆盖默认 TTL。
// ttl > 0 时按此值过期；ttl <= 0 时表示永不过期。
func WithTTL(ttl time.Duration) SetOption { return driver.WithTTL(ttl) }

// WithNX 设置仅当键不存在时写入的条件。
func WithNX() SetOption { return driver.WithNX() }

// WithXX 设置仅当键已存在时写入的条件。
func WithXX() SetOption { return driver.WithXX() }

// WithKeepTTL 保留原有 TTL，覆盖值但不重设过期时间。
func WithKeepTTL() SetOption { return driver.WithKeepTTL() }

// WithNoSlide 写入但不续期（不滑动 TTL）。
func WithNoSlide() SetOption { return driver.WithNoSlide() }

// Manager 缓存统一管理器，作为业务侧访问缓存能力的门面入口。
//
// 根据 [Type] 持有对应的底层后端实例（Redis *goredis.Client 或 Memory *memory.Store），
// 通过泛型工厂方法 [KeyCacheOf] / [HashCacheOf] / [SetCacheOf] / [ListCacheOf] 返回对应后端的接口实现。
// 同时暴露 [Redis] / [Memory] 直接返回底层实例，供需要后端特有能力的场景使用。
type Manager struct {
	kind  Type
	rdb   *goredis.Client
	mem   *memory.Store
	codec Codec
	enc   KeyEncoder
	ttl   time.Duration
	log   *logging.LogNamedLogger

	closeOnce sync.Once
}

// ManagerOption 是 [Manager] 的函数式选项。
type ManagerOption func(*Manager)

// WithRedisClient 注入 Redis 客户端，并将 kind 置为 [CacheTypeRedis]。
func WithRedisClient(rdb *goredis.Client) ManagerOption {
	return func(m *Manager) {
		m.rdb = rdb
		m.kind = CacheTypeRedis
	}
}

// WithMemoryStore 注入内存存储实例，并将 kind 置为 [CacheTypeMemory]。
func WithMemoryStore(store *memory.Store) ManagerOption {
	return func(m *Manager) {
		m.mem = store
		m.kind = CacheTypeMemory
	}
}

// WithManagerTTL 设置默认 TTL，所有未显式指定 TTL 的写入操作使用此值。
func WithManagerTTL(ttl time.Duration) ManagerOption {
	return func(m *Manager) { m.ttl = ttl }
}

// WithCodec 设置序列化器，默认 [JSONCodec]。
func WithCodec(codec Codec) ManagerOption {
	return func(m *Manager) { m.codec = codec }
}

// WithKeyEncoder 设置键编码器，默认 [DefaultKeyEncoder]。
func WithKeyEncoder(enc KeyEncoder) ManagerOption {
	return func(m *Manager) { m.enc = enc }
}

// WithManagerLogger 设置命名日志器，用于缓存操作的调试与错误日志。
func WithManagerLogger(log *logging.LogNamedLogger) ManagerOption {
	return func(m *Manager) { m.log = log }
}

// NewManager 构造缓存管理器。
func NewManager(kind Type, opts ...ManagerOption) *Manager {
	m := &Manager{
		kind:  kind,
		codec: JSONCodec{},
	}
	for _, o := range opts {
		if o != nil {
			o(m)
		}
	}
	return m
}

// Type 返回当前缓存后端类型。
func (m *Manager) Type() Type { return m.kind }

// Redis 返回底层 Redis 客户端。
// 仅当 Type 为 [CacheTypeRedis] 时返回非 nil；其他类型返回 nil。
func (m *Manager) Redis() *goredis.Client { return m.rdb }

// Memory 返回底层内存存储实例。
// 仅当 Type 为 [CacheTypeMemory] 时返回非 nil；其他类型返回 nil。
func (m *Manager) Memory() *memory.Store { return m.mem }

// Codec 返回当前使用的序列化器。
func (m *Manager) Codec() Codec { return m.codec }

// KeyEncoder 返回当前使用的键编码器。
func (m *Manager) KeyEncoder() KeyEncoder { return m.enc }

// TTL 返回默认过期时间，0 表示永不过期。
func (m *Manager) TTL() time.Duration { return m.ttl }

// Logger 返回命名日志器，可能为 nil（未注入时）。
func (m *Manager) Logger() *logging.LogNamedLogger { return m.log }

// KeyCacheOf 返回基于当前后端的 [KeyCache] 泛型实现。
//
// 当 m 为 nil、底层驱动未注入或后端类型未支持时，安全返回 nil。
func KeyCacheOf[K any, V any](m *Manager) KeyCache[K, V] {
	if m == nil {
		return nil
	}
	switch m.kind {
	case CacheTypeRedis:
		if m.rdb == nil {
			return nil
		}
		return redis.NewKeyCache[K, V](m.rdb, m.codec, m.enc, m.ttl)
	case CacheTypeMemory:
		if m.mem == nil {
			return nil
		}
		return memory.NewKeyCache[K, V](m.mem, m.codec, m.enc, m.ttl)
	default:
		return nil
	}
}

// HashCacheOf 返回基于当前后端的 [HashCache] 泛型实现。
//
// 当 m 为 nil、底层驱动未注入或后端类型未支持时，安全返回 nil。
func HashCacheOf[K any, F comparable, V any, S any](m *Manager) HashCache[K, F, V, S] {
	if m == nil {
		return nil
	}
	switch m.kind {
	case CacheTypeRedis:
		if m.rdb == nil {
			return nil
		}
		return redis.NewHashCache[K, F, V, S](m.rdb, m.codec, m.enc, m.ttl)
	case CacheTypeMemory:
		if m.mem == nil {
			return nil
		}
		return memory.NewHashCache[K, F, V, S](m.mem, m.codec, m.enc, m.ttl)
	default:
		return nil
	}
}

// SetCacheOf 返回基于当前后端的 [SetCache] 泛型实现。
//
// 当 m 为 nil、底层驱动未注入或后端类型未支持时，安全返回 nil。
func SetCacheOf[K any, V any](m *Manager) SetCache[K, V] {
	if m == nil {
		return nil
	}
	switch m.kind {
	case CacheTypeRedis:
		if m.rdb == nil {
			return nil
		}
		return redis.NewSetCache[K, V](m.rdb, m.codec, m.enc, m.ttl)
	case CacheTypeMemory:
		if m.mem == nil {
			return nil
		}
		return memory.NewSetCache[K, V](m.mem, m.codec, m.enc, m.ttl)
	default:
		return nil
	}
}

// ListCacheOf 返回基于当前后端的 [ListCache] 泛型实现。
//
// 当 m 为 nil、底层驱动未注入或后端类型未支持时，安全返回 nil。
func ListCacheOf[K any, V any](m *Manager) ListCache[K, V] {
	if m == nil {
		return nil
	}
	switch m.kind {
	case CacheTypeRedis:
		if m.rdb == nil {
			return nil
		}
		return redis.NewListCache[K, V](m.rdb, m.codec, m.enc, m.ttl)
	case CacheTypeMemory:
		if m.mem == nil {
			return nil
		}
		return memory.NewListCache[K, V](m.mem, m.codec, m.enc, m.ttl)
	default:
		return nil
	}
}

// Close 释放底层资源。
// 对 Memory 后端停止 janitor goroutine；对 Redis 后端不主动关闭连接（由拥有者管理）。
func (m *Manager) Close() {
	m.closeOnce.Do(func() {
		if m.mem != nil {
			m.mem.Close()
		}
	})
}
