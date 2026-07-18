package xCache

import (
	"sync"
	"time"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/redis/go-redis/v9"
)

// Manager 缓存统一管理器，作为业务侧访问缓存能力的唯一入口。
//
// 根据 [CacheType] 持有对应的底层后端实例（Redis *redis.Client 或 Memory *memory.Store），
// 通过泛型工厂方法 [KeyCache] / [HashCache] / [SetCache] / [ListCache] 返回对应后端的
// 接口实现。业务侧无需感知后端差异，切换后端只需更换 [option.WithRedis] / [option.WithMemory]。
//
// 同时暴露 [Redis] / [Memory] 直接返回底层实例，供需要后端特有能力的场景使用
// （如 Redis 的 Pub/Sub、Pipeline；Memory 的 Len/Close 监控）。
//
// 生命周期：由 [init.CacheInit] 在应用启动时构造，注册到 [xCtx.CacheManagerKey]，
// 应用退出时由 [Manager.Close] 释放资源（如 Memory 的 janitor goroutine）。
type Manager struct {
	kind  CacheType
	rdb   *redis.Client
	mem   *memoryStore
	codec Codec
	enc   KeyEncoder
	ttl   time.Duration
	log   *xLog.LogNamedLogger

	closeOnce sync.Once
}

// ManagerOption 是 [Manager] 的函数式选项，避免构造函数参数列表过长。
type ManagerOption func(*Manager)

// WithRedisClient 注入 Redis 客户端，并将 kind 置为 [CacheTypeRedis]。
//
// 通常由 [init.CacheInit] 在 Redis 后端装配时调用，业务侧无需直接使用。
func WithRedisClient(rdb *redis.Client) ManagerOption {
	return func(m *Manager) {
		m.rdb = rdb
		m.kind = CacheTypeRedis
	}
}

// WithMemoryStore 注入内存存储实例，并将 kind 置为 [CacheTypeMemory]。
func WithMemoryStore(store *memoryStore) ManagerOption {
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

// WithLogger 设置命名日志器，用于缓存操作的调试与错误日志。
func WithLogger(log *xLog.LogNamedLogger) ManagerOption {
	return func(m *Manager) { m.log = log }
}

// NewManager 构造缓存管理器。
//
// kind 为 [CacheTypeRedis] / [CacheTypeMemory]，需配合对应的 WithRedisClient /
// WithMemoryStore 选项注入底层实例。kind 与注入实例不匹配时，对应工厂方法会返回 nil。
//
// 示例：
//
//	m := xCache.NewManager(xCache.CacheTypeRedis,
//	    xCache.WithRedisClient(rdb),
//	    xCache.WithManagerTTL(30*time.Minute),
//	)
//	kc := m.KeyCache[string, User]()
func NewManager(kind CacheType, opts ...ManagerOption) *Manager {
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
func (m *Manager) Type() CacheType { return m.kind }

// Redis 返回底层 Redis 客户端。
//
// 仅当 Type 为 [CacheTypeRedis] 时返回非 nil；其他类型返回 nil。
// 业务侧可通过此方法直接调用 Redis 特有命令（如 Pipeline、Pub/Sub）。
func (m *Manager) Redis() *redis.Client { return m.rdb }

// Memory 返回底层内存存储实例。
//
// 仅当 Type 为 [CacheTypeMemory] 时返回非 nil；其他类型返回 nil。
// 业务侧可通过此方法访问 Store 的监控方法（如 Len、Close）。
func (m *Manager) Memory() *memoryStore { return m.mem }

// Codec 返回当前使用的序列化器。
func (m *Manager) Codec() Codec { return m.codec }

// TTL 返回默认过期时间，0 表示永不过期。
func (m *Manager) TTL() time.Duration { return m.ttl }

// Logger 返回命名日志器，可能为 nil（未注入时）。
func (m *Manager) Logger() *xLog.LogNamedLogger { return m.log }

// KeyCacheOf 返回基于当前后端的 [KeyCache] 实现。
//
// Go 1.18+ 禁止接口/方法带类型参数，故采用包级泛型函数而非 Manager 方法，
// 这是 GORM Dialector 模式在 Go 泛型限制下的编译期类型安全版本：
// Manager 作为统一门面持有底层 driver（Redis *redis.Client 或 Memory *memoryStore），
// 本函数按 [Manager.Type] 分发到对应实现。
//
// 泛型参数：
//   - K: 业务键类型，通过 [KeyEncoder] 转为 string 作为底层 key
//   - V: 值类型，通过 [Codec] 序列化为 []byte 存储
//
// 后端未装配时返回 nil。
//
// 使用示例：
//
//	kc := xCache.KeyCacheOf[string, User](manager)
//	_ = kc.Set(ctx, "user:1", &User{Name: "筱锋"})
//	v, ok, _ := kc.Get(ctx, "user:1")
func KeyCacheOf[K any, V any](m *Manager) KeyCache[K, V] {
	if m == nil {
		return nil
	}
	switch m.kind {
	case CacheTypeRedis:
		if m.rdb == nil {
			return nil
		}
		return NewRedisKeyCache[K, V](m.rdb, m.codec, m.enc, m.ttl)
	case CacheTypeMemory:
		if m.mem == nil {
			return nil
		}
		return NewMemoryKeyCache[K, V](m.mem, m.codec, m.enc, m.ttl)
	default:
		return nil
	}
}

// HashCacheOf 返回基于当前后端的 [HashCache] 实现。
//
// 泛型参数：
//   - K: 哈希键类型
//   - F: 字段类型（必须 comparable）
//   - V: 字段值类型
//   - S: GetAllStruct/SetAllStruct 使用的结构体类型
func HashCacheOf[K any, F comparable, V any, S any](m *Manager) HashCache[K, F, V, S] {
	if m == nil {
		return nil
	}
	switch m.kind {
	case CacheTypeRedis:
		if m.rdb == nil {
			return nil
		}
		return NewRedisHashCache[K, F, V, S](m.rdb, m.codec, m.enc, m.ttl)
	case CacheTypeMemory:
		if m.mem == nil {
			return nil
		}
		return NewMemoryHashCache[K, F, V, S](m.mem, m.codec, m.enc, m.ttl)
	default:
		return nil
	}
}

// SetCacheOf 返回基于当前后端的 [SetCache] 实现。
func SetCacheOf[K any, V any](m *Manager) SetCache[K, V] {
	if m == nil {
		return nil
	}
	switch m.kind {
	case CacheTypeRedis:
		if m.rdb == nil {
			return nil
		}
		return NewRedisSetCache[K, V](m.rdb, m.codec, m.enc, m.ttl)
	case CacheTypeMemory:
		if m.mem == nil {
			return nil
		}
		return NewMemorySetCache[K, V](m.mem, m.codec, m.enc, m.ttl)
	default:
		return nil
	}
}

// ListCacheOf 返回基于当前后端的 [ListCache] 实现。
func ListCacheOf[K any, V any](m *Manager) ListCache[K, V] {
	if m == nil {
		return nil
	}
	switch m.kind {
	case CacheTypeRedis:
		if m.rdb == nil {
			return nil
		}
		return NewRedisListCache[K, V](m.rdb, m.codec, m.enc, m.ttl)
	case CacheTypeMemory:
		if m.mem == nil {
			return nil
		}
		return NewMemoryListCache[K, V](m.mem, m.codec, m.enc, m.ttl)
	default:
		return nil
	}
}

// Close 释放底层资源。
//
// 当前仅对 Memory 后端有意义（停止 janitor goroutine）；Redis 后端关闭由
// 调用方自行管理（通常跟随应用生命周期）。可安全多次调用。
func (m *Manager) Close() {
	m.closeOnce.Do(func() {
		if m.mem != nil {
			m.mem.Close()
		}
	})
}
