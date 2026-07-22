package xCacheMemory

import (
	"context"
	"time"

	xCacheDriver "github.com/bamboo-services/bamboo-base-go/major/cache/driver"
)

// KeyCache [xCacheDriver.KeyCache] 的内存实现。
type KeyCache[K any, V any] struct {
	store *Store
	codec xCacheDriver.Codec
	enc   xCacheDriver.KeyEncoder
	ttl   time.Duration
}

// NewKeyCache 构造一个基于内存的 [xCacheDriver.KeyCache] 实现。
//
// ttl 为单次写入的过期时间，0 表示使用 Store 的 defaultTTL（若也为 0 则永不过期）。
// codec/enc 为 nil 时回退到 [xCacheDriver.JSONCodec] / [xCacheDriver.DefaultKeyEncoder]。
func NewKeyCache[K any, V any](store *Store, codec xCacheDriver.Codec, enc xCacheDriver.KeyEncoder, ttl time.Duration) xCacheDriver.KeyCache[K, V] {
	if codec == nil {
		codec = xCacheDriver.JSONCodec{}
	}
	return &KeyCache[K, V]{store: store, codec: codec, enc: enc, ttl: ttl}
}

// Get 从内存中读取键对应的值。
//
// 不存在或已过期时返回 nil, false, nil。反序列化失败返回 nil, false, err。
// [Store.Get] 在锁内返回 Value 的引用拷贝，与并发 Set 不产生 data race。
func (c *KeyCache[K, V]) Get(ctx context.Context, key K) (*V, bool, error) {
	k := xCacheDriver.EncodeKey(c.enc, key)
	value, ok := c.store.Get(k)
	if !ok {
		return nil, false, nil
	}
	data, ok := value.([]byte)
	if !ok {
		return nil, false, nil
	}
	var v V
	if err := c.codec.Unmarshal(data, &v); err != nil {
		return nil, false, err
	}
	return &v, true, nil
}

// Set 将值序列化后写入内存。value 为 nil 时等价于删除。
//
// opts 用于在单次调用覆盖默认 TTL，并支持 NX/XX/KeepTTL 条件写入：
//   - NX：仅当 key 不存在时写入（条件不满足时直接返回，不刷新 TTL）
//   - XX：仅当 key 已存在时写入（条件不满足时直接返回，不刷新 TTL）
//   - KeepTTL：覆盖值但保留原 ExpireAt（等价于 Redis SET KEEPTTL）
//
// NoSlide 对 KeyCache 无意义（Set 本身就是整体覆盖），传入时被忽略。
// 无任何条件选项时走 [Store.Set] 保持原有行为。
func (c *KeyCache[K, V]) Set(ctx context.Context, key K, value *V, opts ...xCacheDriver.SetOption) error {
	if value == nil {
		return c.Delete(ctx, key)
	}
	k := xCacheDriver.EncodeKey(c.enc, key)
	data, err := c.codec.Marshal(*value)
	if err != nil {
		return err
	}
	cfg := xCacheDriver.ApplySet(c.ttl, opts)
	// 存在 NX/XX/KeepTTL 条件时走 SetCond，否则保持原 Set 路径
	if cfg.NX || cfg.XX || cfg.KeepTTL {
		c.store.SetCond(k, data, cfg.TTL, cfg.NX, cfg.XX, cfg.KeepTTL)
		return nil
	}
	c.store.Set(k, data, cfg.TTL)
	return nil
}

// Exists 判断键是否存在且未过期。
func (c *KeyCache[K, V]) Exists(ctx context.Context, key K) (bool, error) {
	return c.store.Exists(xCacheDriver.EncodeKey(c.enc, key)), nil
}

// Delete 删除键。键不存在时不报错。
func (c *KeyCache[K, V]) Delete(ctx context.Context, key K) error {
	c.store.Delete(xCacheDriver.EncodeKey(c.enc, key))
	return nil
}
