package xCache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisListCache [ListCache] 的 Redis 实现。
type redisListCache[K any, V any] struct {
	rdb   *redis.Client
	codec Codec
	enc   KeyEncoder
	ttl   time.Duration
}

// NewRedisListCache 构造一个基于 Redis 的 [ListCache] 实现。
func NewRedisListCache[K any, V any](rdb *redis.Client, codec Codec, enc KeyEncoder, ttl time.Duration) ListCache[K, V] {
	if codec == nil {
		codec = JSONCodec{}
	}
	return &redisListCache[K, V]{rdb: rdb, codec: codec, enc: enc, ttl: ttl}
}

// refreshTTL 在写操作后按需续期。
func (c *redisListCache[K, V]) refreshTTL(ctx context.Context, key K) {
	if c.ttl > 0 {
		_ = c.rdb.Expire(ctx, EncodeKey(c.enc, key), c.ttl)
	}
}

// encodeValues 把多个值序列化为 Redis 命令可接受的 []any 参数。
func (c *redisListCache[K, V]) encodeValues(values []V) ([]any, error) {
	args := make([]any, 0, len(values))
	for _, v := range values {
		data, err := c.codec.Marshal(v)
		if err != nil {
			return nil, err
		}
		args = append(args, data)
	}
	return args, nil
}

// Prepend 将一个或多个值插入到列表头部（左侧）。
//
// Redis LPUSH 按参数顺序从左插入，最终顺序与参数顺序相反。
// 为对齐业务语义（参数顺序即头部顺序），这里反转参数。
func (c *redisListCache[K, V]) Prepend(ctx context.Context, key K, values ...V) error {
	if len(values) == 0 {
		return nil
	}
	reversed := make([]V, len(values))
	for i, v := range values {
		reversed[len(values)-1-i] = v
	}
	args, err := c.encodeValues(reversed)
	if err != nil {
		return err
	}
	if err := c.rdb.LPush(ctx, EncodeKey(c.enc, key), args...).Err(); err != nil {
		return err
	}
	c.refreshTTL(ctx, key)
	return nil
}

// Append 将一个或多个值追加到列表尾部（右侧）。
func (c *redisListCache[K, V]) Append(ctx context.Context, key K, values ...V) error {
	if len(values) == 0 {
		return nil
	}
	args, err := c.encodeValues(values)
	if err != nil {
		return err
	}
	if err := c.rdb.RPush(ctx, EncodeKey(c.enc, key), args...).Err(); err != nil {
		return err
	}
	c.refreshTTL(ctx, key)
	return nil
}

// Range 按索引范围获取列表元素，支持负数索引。
func (c *redisListCache[K, V]) Range(ctx context.Context, key K, start int64, end int64) ([]V, error) {
	raw, err := c.rdb.LRange(ctx, EncodeKey(c.enc, key), start, end).Result()
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

// Index 获取指定索引位置的元素，支持负数索引。越界时返回 nil, nil。
func (c *redisListCache[K, V]) Index(ctx context.Context, key K, index int64) (*V, error) {
	data, err := c.rdb.LIndex(ctx, EncodeKey(c.enc, key), index).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var v V
	if err := c.codec.Unmarshal([]byte(data), &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Len 获取列表的长度。
func (c *redisListCache[K, V]) Len(ctx context.Context, key K) (int64, error) {
	return c.rdb.LLen(ctx, EncodeKey(c.enc, key)).Result()
}

// Pop 从列表头部弹出一个元素并返回。列表为空时返回 nil, nil。
func (c *redisListCache[K, V]) Pop(ctx context.Context, key K) (*V, error) {
	data, err := c.rdb.LPop(ctx, EncodeKey(c.enc, key)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var v V
	if err := c.codec.Unmarshal([]byte(data), &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// PopLast 从列表尾部弹出一个元素并返回。列表为空时返回 nil, nil。
func (c *redisListCache[K, V]) PopLast(ctx context.Context, key K) (*V, error) {
	data, err := c.rdb.RPop(ctx, EncodeKey(c.enc, key)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var v V
	if err := c.codec.Unmarshal([]byte(data), &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Remove 从列表中移除指定数量的匹配元素。
//
// count > 0 从头部开始；count < 0 从尾部开始；count = 0 移除全部。
// 直接映射到 Redis LREM 命令。
func (c *redisListCache[K, V]) Remove(ctx context.Context, key K, count int64, value V) error {
	data, err := c.codec.Marshal(value)
	if err != nil {
		return err
	}
	return c.rdb.LRem(ctx, EncodeKey(c.enc, key), count, data).Err()
}

// Delete 删除整个列表。
func (c *redisListCache[K, V]) Delete(ctx context.Context, key K) error {
	return c.rdb.Del(ctx, EncodeKey(c.enc, key)).Err()
}
