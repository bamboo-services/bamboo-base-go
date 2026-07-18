package xCacheRedis

import (
	"context"
	"time"

	xCacheDriver "github.com/bamboo-services/bamboo-base-go/major/cache/driver"
	"github.com/redis/go-redis/v9"
)

// SetCache [xCacheDriver.SetCache] 的 Redis 实现。
type SetCache[K any, V any] struct {
	rdb   *redis.Client
	codec xCacheDriver.Codec
	enc   xCacheDriver.KeyEncoder
	ttl   time.Duration
}

// NewSetCache 构造一个基于 Redis 的 [xCacheDriver.SetCache] 实现。
func NewSetCache[K any, V any](rdb *redis.Client, codec xCacheDriver.Codec, enc xCacheDriver.KeyEncoder, ttl time.Duration) xCacheDriver.SetCache[K, V] {
	if codec == nil {
		codec = xCacheDriver.JSONCodec{}
	}
	return &SetCache[K, V]{rdb: rdb, codec: codec, enc: enc, ttl: ttl}
}

// refreshTTL 在写操作后按需续期。
func (c *SetCache[K, V]) refreshTTL(ctx context.Context, key K) {
	if c.ttl > 0 {
		_ = c.rdb.Expire(ctx, xCacheDriver.EncodeKey(c.enc, key), c.ttl)
	}
}

// encodeMembers 把多个成员序列化为 Redis 命令可接受的 []any 参数。
func (c *SetCache[K, V]) encodeMembers(members []V) ([]any, error) {
	args := make([]any, 0, len(members))
	for _, m := range members {
		data, err := c.codec.Marshal(m)
		if err != nil {
			return nil, err
		}
		args = append(args, data)
	}
	return args, nil
}

// Add 将一个或多个成员添加到集合中，已存在的成员会被忽略。
func (c *SetCache[K, V]) Add(ctx context.Context, key K, members ...V) error {
	if len(members) == 0 {
		return nil
	}
	args, err := c.encodeMembers(members)
	if err != nil {
		return err
	}
	if err := c.rdb.SAdd(ctx, xCacheDriver.EncodeKey(c.enc, key), args...).Err(); err != nil {
		return err
	}
	c.refreshTTL(ctx, key)
	return nil
}

// Members 获取集合中的所有成员。
func (c *SetCache[K, V]) Members(ctx context.Context, key K) ([]V, error) {
	raw, err := c.rdb.SMembers(ctx, xCacheDriver.EncodeKey(c.enc, key)).Result()
	if err != nil {
		return nil, err
	}
	result := make([]V, 0, len(raw))
	for _, data := range raw {
		var v V
		if err := c.codec.Unmarshal([]byte(data), &v); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, nil
}

// IsMember 检查指定成员是否存在于集合中。
func (c *SetCache[K, V]) IsMember(ctx context.Context, key K, member V) (bool, error) {
	data, err := c.codec.Marshal(member)
	if err != nil {
		return false, err
	}
	return c.rdb.SIsMember(ctx, xCacheDriver.EncodeKey(c.enc, key), data).Result()
}

// Count 获取集合中的成员数量。
func (c *SetCache[K, V]) Count(ctx context.Context, key K) (int64, error) {
	return c.rdb.SCard(ctx, xCacheDriver.EncodeKey(c.enc, key)).Result()
}

// Remove 从集合中移除指定的成员。
func (c *SetCache[K, V]) Remove(ctx context.Context, key K, members ...V) error {
	if len(members) == 0 {
		return nil
	}
	args, err := c.encodeMembers(members)
	if err != nil {
		return err
	}
	return c.rdb.SRem(ctx, xCacheDriver.EncodeKey(c.enc, key), args...).Err()
}

// Delete 删除整个集合。
func (c *SetCache[K, V]) Delete(ctx context.Context, key K) error {
	return c.rdb.Del(ctx, xCacheDriver.EncodeKey(c.enc, key)).Err()
}
