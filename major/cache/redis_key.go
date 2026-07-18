package xCache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisKeyCache [KeyCache] 的 Redis 实现。
type redisKeyCache[K any, V any] struct {
	rdb   *redis.Client
	codec Codec
	enc   KeyEncoder
	ttl   time.Duration
}

// NewRedisKeyCache 构造一个基于 Redis 的 [KeyCache] 实现。
//
// rdb 必须非空；codec/enc 为 nil 时回退到 [JSONCodec] / [DefaultKeyEncoder]。
// ttl 为每次 Set 的默认过期时间，0 表示永不过期。
func NewRedisKeyCache[K any, V any](rdb *redis.Client, codec Codec, enc KeyEncoder, ttl time.Duration) KeyCache[K, V] {
	if codec == nil {
		codec = JSONCodec{}
	}
	return &redisKeyCache[K, V]{rdb: rdb, codec: codec, enc: enc, ttl: ttl}
}

// Get 从 Redis 读取键对应的值。
//
// 键不存在时（redis.Nil）返回 nil, false, nil。
func (c *redisKeyCache[K, V]) Get(ctx context.Context, key K) (*V, bool, error) {
	k := EncodeKey(c.enc, key)
	data, err := c.rdb.Get(ctx, k).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var v V
	if err := c.codec.Unmarshal(data, &v); err != nil {
		return nil, false, err
	}
	return &v, true, nil
}

// Set 将值序列化后写入 Redis。value 为 nil 时等价于删除。
func (c *redisKeyCache[K, V]) Set(ctx context.Context, key K, value *V) error {
	if value == nil {
		return c.Delete(ctx, key)
	}
	data, err := c.codec.Marshal(*value)
	if err != nil {
		return err
	}
	k := EncodeKey(c.enc, key)
	if c.ttl > 0 {
		return c.rdb.Set(ctx, k, data, c.ttl).Err()
	}
	return c.rdb.Set(ctx, k, data, 0).Err()
}

// Exists 判断键是否存在。
func (c *redisKeyCache[K, V]) Exists(ctx context.Context, key K) (bool, error) {
	n, err := c.rdb.Exists(ctx, EncodeKey(c.enc, key)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Delete 删除键。
func (c *redisKeyCache[K, V]) Delete(ctx context.Context, key K) error {
	return c.rdb.Del(ctx, EncodeKey(c.enc, key)).Err()
}
