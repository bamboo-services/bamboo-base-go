package xCacheMemory

import (
	"context"
	"time"

	xCacheDriver "github.com/bamboo-services/bamboo-base-go/major/cache/driver"
)

// HashCache [xCacheDriver.HashCache] 的内存实现。
//
// 内存中以 map[F][]byte 存储字段（field → 已序列化的 value），整体作为 [memoryEntry.Value]
// 存入 [Store]，复用 Store 的 TTL / LRU 能力。F 声明为 comparable，可直接作为
// 运行时 map key，无需转换为 string。
//
// GetAllStruct / SetAllStruct 依赖 [xCacheDriver.Codec] 在 struct 与 map[F]V 之间的转换能力
// （JSONCodec 原生支持）。
type HashCache[K any, F comparable, V any, S any] struct {
	store *Store
	codec xCacheDriver.Codec
	enc   xCacheDriver.KeyEncoder
	ttl   time.Duration
}

// NewHashCache 构造一个基于内存的 [xCacheDriver.HashCache] 实现。
func NewHashCache[K any, F comparable, V any, S any](store *Store, codec xCacheDriver.Codec, enc xCacheDriver.KeyEncoder, ttl time.Duration) xCacheDriver.HashCache[K, F, V, S] {
	if codec == nil {
		codec = xCacheDriver.JSONCodec{}
	}
	return &HashCache[K, F, V, S]{store: store, codec: codec, enc: enc, ttl: ttl}
}

// loadOrCreate 仅用于 Get 路径（只读）。写路径必须走 [Store.Update] 保证原子性。
func (c *HashCache[K, F, V, S]) loadOrCreate(key K) (map[F][]byte, bool) {
	k := xCacheDriver.EncodeKey(c.enc, key)
	if value, ok := c.store.Get(k); ok {
		if m, ok := value.(map[F][]byte); ok {
			return m, false
		}
	}
	return make(map[F][]byte), true
}

// Get 获取单个字段的值。
func (c *HashCache[K, F, V, S]) Get(ctx context.Context, key K, field F) (*V, bool, error) {
	k := xCacheDriver.EncodeKey(c.enc, key)
	value, ok := c.store.Get(k)
	if !ok {
		return nil, false, nil
	}
	m, ok := value.(map[F][]byte)
	if !ok {
		return nil, false, nil
	}
	data, ok := m[field]
	if !ok {
		return nil, false, nil
	}
	var v V
	if err := c.codec.Unmarshal(data, &v); err != nil {
		return nil, false, err
	}
	return &v, true, nil
}

// Set 设置单个字段的值。
//
// value 为 nil 时等价于 [Remove] 该 field，与 KeyCache.Set 的 nil 删除语义对齐。
// 通过 [Store.Update] 在单把锁内完成读-改-写，避免并发 panic。
func (c *HashCache[K, F, V, S]) Set(ctx context.Context, key K, field F, value *V) error {
	if value == nil {
		return c.Remove(ctx, key, field)
	}
	data, err := c.codec.Marshal(*value)
	if err != nil {
		return err
	}
	k := xCacheDriver.EncodeKey(c.enc, key)
	c.store.Update(k, c.ttl, func(old any) any {
		m, _ := old.(map[F][]byte)
		if m == nil {
			m = make(map[F][]byte)
		}
		m[field] = data
		return m
	})
	return nil
}

// GetAll 获取所有字段及值，以 map[F]V 形式返回。
func (c *HashCache[K, F, V, S]) GetAll(ctx context.Context, key K) (map[F]V, error) {
	result := make(map[F]V)
	k := xCacheDriver.EncodeKey(c.enc, key)
	value, ok := c.store.Get(k)
	if !ok {
		return result, nil
	}
	m, ok := value.(map[F][]byte)
	if !ok {
		return result, nil
	}
	for f, data := range m {
		var v V
		if err := c.codec.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		result[f] = v
	}
	return result, nil
}

// GetAllStruct 获取所有字段到结构体 S。
//
// 先通过 [GetAll] 拿到 map[F]V，再用 codec 把 map 序列化后反序列化为 S。
func (c *HashCache[K, F, V, S]) GetAllStruct(ctx context.Context, key K) (S, error) {
	var s S
	m, err := c.GetAll(ctx, key)
	if err != nil {
		return s, err
	}
	data, err := c.codec.Marshal(m)
	if err != nil {
		return s, err
	}
	if err := c.codec.Unmarshal(data, &s); err != nil {
		return s, err
	}
	return s, nil
}

// SetAll 批量设置字段。
//
// 通过 [Store.Update] 保证原子性。value 为 nil 的 field 会被删除。
// 修改后若 map 为空则整个 hash 被删除。
func (c *HashCache[K, F, V, S]) SetAll(ctx context.Context, key K, fields map[F]*V) error {
	if len(fields) == 0 {
		return nil
	}
	// 预编码非 nil 值，避免在 Update 闭包内做可能失败的 I/O
	encoded := make(map[F][]byte, len(fields))
	var deleteFields []F
	for f, v := range fields {
		if v == nil {
			deleteFields = append(deleteFields, f)
			continue
		}
		data, err := c.codec.Marshal(*v)
		if err != nil {
			return err
		}
		encoded[f] = data
	}
	k := xCacheDriver.EncodeKey(c.enc, key)
	c.store.Update(k, c.ttl, func(old any) any {
		m, _ := old.(map[F][]byte)
		if m == nil {
			m = make(map[F][]byte)
		}
		for f, data := range encoded {
			m[f] = data
		}
		for _, f := range deleteFields {
			delete(m, f)
		}
		if len(m) == 0 {
			return nil
		}
		return m
	})
	return nil
}

// SetAllStruct 用结构体批量设置字段。
func (c *HashCache[K, F, V, S]) SetAllStruct(ctx context.Context, key K, value S) error {
	data, err := c.codec.Marshal(value)
	if err != nil {
		return err
	}
	var m map[F]V
	if err := c.codec.Unmarshal(data, &m); err != nil {
		return err
	}
	fields := make(map[F]*V, len(m))
	for f := range m {
		v := m[f]
		fields[f] = &v
	}
	return c.SetAll(ctx, key, fields)
}

// Exists 判断字段是否存在。
func (c *HashCache[K, F, V, S]) Exists(ctx context.Context, key K, field F) (bool, error) {
	k := xCacheDriver.EncodeKey(c.enc, key)
	value, ok := c.store.Get(k)
	if !ok {
		return false, nil
	}
	m, ok := value.(map[F][]byte)
	if !ok {
		return false, nil
	}
	_, ok = m[field]
	return ok, nil
}

// Remove 移除指定字段。
//
// 通过 [Store.Update] 保证原子性。移除后若 map 为空则整个 hash 被删除。
func (c *HashCache[K, F, V, S]) Remove(ctx context.Context, key K, fields ...F) error {
	if len(fields) == 0 {
		return nil
	}
	k := xCacheDriver.EncodeKey(c.enc, key)
	c.store.Update(k, c.ttl, func(old any) any {
		m, _ := old.(map[F][]byte)
		if m == nil {
			return nil
		}
		for _, f := range fields {
			delete(m, f)
		}
		if len(m) == 0 {
			return nil
		}
		return m
	})
	return nil
}

// Delete 删除整个 hash。
func (c *HashCache[K, F, V, S]) Delete(ctx context.Context, key K) error {
	c.store.Delete(xCacheDriver.EncodeKey(c.enc, key))
	return nil
}
