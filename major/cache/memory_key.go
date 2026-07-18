package xCache

import (
	"context"
	"time"
)

// memoryKeyCache [KeyCache] 的内存实现。
type memoryKeyCache[K any, V any] struct {
	store *memoryStore
	codec Codec
	enc   KeyEncoder
	ttl   time.Duration
}

// NewMemoryKeyCache 构造一个基于内存的 [KeyCache] 实现。
//
// ttl 为单次写入的过期时间，0 表示使用 Store 的 defaultTTL（若也为 0 则永不过期）。
// codec/enc 为 nil 时回退到 [JSONCodec] / [DefaultKeyEncoder]。
func NewMemoryKeyCache[K any, V any](store *memoryStore, codec Codec, enc KeyEncoder, ttl time.Duration) KeyCache[K, V] {
	if codec == nil {
		codec = JSONCodec{}
	}
	return &memoryKeyCache[K, V]{store: store, codec: codec, enc: enc, ttl: ttl}
}

// Get 从内存中读取键对应的值。
//
// 不存在或已过期时返回 nil, false, nil。反序列化失败返回 nil, false, err。
// [memoryStore.Get] 在锁内返回 Value 的引用拷贝，与并发 Set 不产生 data race。
func (c *memoryKeyCache[K, V]) Get(ctx context.Context, key K) (*V, bool, error) {
	k := EncodeKey(c.enc, key)
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
func (c *memoryKeyCache[K, V]) Set(ctx context.Context, key K, value *V) error {
	if value == nil {
		return c.Delete(ctx, key)
	}
	k := EncodeKey(c.enc, key)
	data, err := c.codec.Marshal(*value)
	if err != nil {
		return err
	}
	c.store.Set(k, data, c.ttl)
	return nil
}

// Exists 判断键是否存在且未过期。
func (c *memoryKeyCache[K, V]) Exists(ctx context.Context, key K) (bool, error) {
	return c.store.Exists(EncodeKey(c.enc, key)), nil
}

// Delete 删除键。键不存在时不报错。
func (c *memoryKeyCache[K, V]) Delete(ctx context.Context, key K) error {
	c.store.Delete(EncodeKey(c.enc, key))
	return nil
}
