package xCacheMemory

import (
	"context"
	"time"

	xCacheDriver "github.com/bamboo-services/bamboo-base-go/major/cache/driver"
)

// SetCache [xCacheDriver.SetCache] 的内存实现。
//
// 内存中以 map[string]struct{} 存储成员（序列化后的 string 作为 key），
// 整体作为 [memoryEntry.Value] 存入 [Store]。
type SetCache[K any, V any] struct {
	store *Store
	codec xCacheDriver.Codec
	enc   xCacheDriver.KeyEncoder
	ttl   time.Duration
}

// NewSetCache 构造一个基于内存的 [xCacheDriver.SetCache] 实现。
func NewSetCache[K any, V any](store *Store, codec xCacheDriver.Codec, enc xCacheDriver.KeyEncoder, ttl time.Duration) xCacheDriver.SetCache[K, V] {
	if codec == nil {
		codec = xCacheDriver.JSONCodec{}
	}
	return &SetCache[K, V]{store: store, codec: codec, enc: enc, ttl: ttl}
}

// loadOrCreate 仅用于 Get/Count/IsMember 等只读路径。写路径走 [Store.Update]。
func (c *SetCache[K, V]) loadOrCreate(key K) (map[string]struct{}, bool) {
	k := xCacheDriver.EncodeKey(c.enc, key)
	if value, ok := c.store.Get(k); ok {
		if m, ok := value.(map[string]struct{}); ok {
			return m, false
		}
	}
	return make(map[string]struct{}), true
}

// encodeMember 把成员序列化为 string key。
func (c *SetCache[K, V]) encodeMember(member V) (string, error) {
	data, err := c.codec.Marshal(member)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Add 将一组成员添加到集合中，已存在的成员会被忽略。
//
// 通过 [Store.Update] / [Store.UpdateKeepExpireAt] 在单把锁内完成读-改-写，避免并发 panic。
//
// opts 支持以下条件写入（存在任意条件选项时走 [Store.UpdateKeepExpireAt]）：
//   - NX：仅当 key 不存在时写入（条件不满足返回 [UpdateNoChange]，不刷新 TTL）
//   - XX：仅当 key 已存在时写入（条件不满足返回 [UpdateNoChange]，不刷新 TTL）
//   - KeepTTL/NoSlide：追加数据但保留原 ExpireAt（不滑动 key 的整体 TTL）
//
// 无任何条件选项时走 [Store.Update] 保持原有行为。
func (c *SetCache[K, V]) Add(ctx context.Context, key K, members []V, opts ...xCacheDriver.SetOption) error {
	if len(members) == 0 {
		return nil
	}
	// 预编码，避免在 Update 闭包内做可能失败的 I/O
	encoded := make([]string, 0, len(members))
	for _, m := range members {
		data, err := c.codec.Marshal(m)
		if err != nil {
			return err
		}
		encoded = append(encoded, string(data))
	}
	k := xCacheDriver.EncodeKey(c.enc, key)
	cfg := xCacheDriver.ApplySet(c.ttl, opts)
	hasCond := cfg.NX || cfg.XX || cfg.KeepTTL || cfg.NoSlide
	if hasCond {
		// 条件写入：NX/XX 在闭包内按 key 原子判断，NoSlide/KeepTTL 保留原 ExpireAt
		c.store.UpdateKeepExpireAt(k, cfg.TTL, func(old any) any {
			set, _ := old.(map[string]struct{})
			// NX：key 已存在则跳过（不刷新 TTL）
			if cfg.NX && set != nil {
				return UpdateNoChange
			}
			// XX：key 不存在则跳过（不刷新 TTL）
			if cfg.XX && set == nil {
				return UpdateNoChange
			}
			if set == nil {
				set = make(map[string]struct{})
			}
			for _, mk := range encoded {
				set[mk] = struct{}{}
			}
			return set
		})
		return nil
	}
	c.store.Update(k, cfg.TTL, func(old any) any {
		set, _ := old.(map[string]struct{})
		if set == nil {
			set = make(map[string]struct{})
		}
		for _, mk := range encoded {
			set[mk] = struct{}{}
		}
		return set
	})
	return nil
}

// Members 获取集合中的所有成员。
func (c *SetCache[K, V]) Members(ctx context.Context, key K) ([]V, error) {
	k := xCacheDriver.EncodeKey(c.enc, key)
	value, ok := c.store.Get(k)
	if !ok {
		return nil, nil
	}
	m, ok := value.(map[string]struct{})
	if !ok {
		return nil, nil
	}
	result := make([]V, 0, len(m))
	for mk := range m {
		var v V
		if err := c.codec.Unmarshal([]byte(mk), &v); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, nil
}

// IsMember 检查指定成员是否存在于集合中。
func (c *SetCache[K, V]) IsMember(ctx context.Context, key K, member V) (bool, error) {
	k := xCacheDriver.EncodeKey(c.enc, key)
	value, ok := c.store.Get(k)
	if !ok {
		return false, nil
	}
	m, ok := value.(map[string]struct{})
	if !ok {
		return false, nil
	}
	mk, err := c.encodeMember(member)
	if err != nil {
		return false, err
	}
	_, ok = m[mk]
	return ok, nil
}

// Count 获取集合中的成员数量。
func (c *SetCache[K, V]) Count(ctx context.Context, key K) (int64, error) {
	k := xCacheDriver.EncodeKey(c.enc, key)
	value, ok := c.store.Get(k)
	if !ok {
		return 0, nil
	}
	m, ok := value.(map[string]struct{})
	if !ok {
		return 0, nil
	}
	return int64(len(m)), nil
}

// Remove 从集合中移除指定的成员。
//
// 通过 [Store.Update] 保证原子性。移除后若集合为空则整个 key 被删除。
func (c *SetCache[K, V]) Remove(ctx context.Context, key K, members ...V) error {
	if len(members) == 0 {
		return nil
	}
	encoded := make([]string, 0, len(members))
	for _, m := range members {
		data, err := c.codec.Marshal(m)
		if err != nil {
			return err
		}
		encoded = append(encoded, string(data))
	}
	k := xCacheDriver.EncodeKey(c.enc, key)
	c.store.Update(k, c.ttl, func(old any) any {
		set, _ := old.(map[string]struct{})
		if set == nil {
			return nil
		}
		for _, mk := range encoded {
			delete(set, mk)
		}
		if len(set) == 0 {
			return nil
		}
		return set
	})
	return nil
}

// Delete 删除整个集合。
func (c *SetCache[K, V]) Delete(ctx context.Context, key K) error {
	c.store.Delete(xCacheDriver.EncodeKey(c.enc, key))
	return nil
}
