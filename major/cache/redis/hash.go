package xCacheRedis

import (
	"context"
	"fmt"
	"reflect"
	"time"

	xCacheDriver "github.com/bamboo-services/bamboo-base-go/major/cache/driver"
	"github.com/redis/go-redis/v9"
)

// HashCache [xCacheDriver.HashCache] 的 Redis 实现。
//
// 使用 Redis Hash 命令操作。field 通过 [xCacheDriver.KeyEncoder] 转为 string 作为 Redis hash field。
type HashCache[K any, F comparable, V any, S any] struct {
	rdb   *redis.Client
	codec xCacheDriver.Codec
	enc   xCacheDriver.KeyEncoder
	ttl   time.Duration
}

// NewHashCache 构造一个基于 Redis 的 [xCacheDriver.HashCache] 实现。
func NewHashCache[K any, F comparable, V any, S any](rdb *redis.Client, codec xCacheDriver.Codec, enc xCacheDriver.KeyEncoder, ttl time.Duration) xCacheDriver.HashCache[K, F, V, S] {
	if codec == nil {
		codec = xCacheDriver.JSONCodec{}
	}
	return &HashCache[K, F, V, S]{rdb: rdb, codec: codec, enc: enc, ttl: ttl}
}

// refreshTTL 在写操作后按需续期。
func (c *HashCache[K, F, V, S]) refreshTTL(ctx context.Context, key K) {
	if c.ttl > 0 {
		_ = c.rdb.Expire(ctx, xCacheDriver.EncodeKey(c.enc, key), c.ttl)
	}
}

// Get 获取单个字段的值。
func (c *HashCache[K, F, V, S]) Get(ctx context.Context, key K, field F) (*V, bool, error) {
	k := xCacheDriver.EncodeKey(c.enc, key)
	data, err := c.rdb.HGet(ctx, k, xCacheDriver.EncodeKey(c.enc, field)).Bytes()
	if err == redis.Nil {
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

// Set 设置单个字段的值。
//
// value 为 nil 时等价于 [Remove] 该 field，与 KeyCache.Set 的 nil 删除语义对齐。
func (c *HashCache[K, F, V, S]) Set(ctx context.Context, key K, field F, value *V) error {
	if value == nil {
		return c.Remove(ctx, key, field)
	}
	data, err := c.codec.Marshal(*value)
	if err != nil {
		return err
	}
	k := xCacheDriver.EncodeKey(c.enc, key)
	if err := c.rdb.HSet(ctx, k, xCacheDriver.EncodeKey(c.enc, field), data).Err(); err != nil {
		return err
	}
	c.refreshTTL(ctx, key)
	return nil
}

// GetAll 获取所有字段及值，以 map[F]V 形式返回。
//
// Redis HGETALL 返回 map[string]string，要求 F 为 string 兼容类型。
// F 非 string 时返回错误，避免静默返回空 map 与「key 不存在」混淆。
func (c *HashCache[K, F, V, S]) GetAll(ctx context.Context, key K) (map[F]V, error) {
	result := make(map[F]V)
	var fZero F
	if t := reflect.TypeOf(fZero); t == nil || t.Kind() != reflect.String {
		return nil, fmt.Errorf("redis HashCache.GetAll: F must be string kind, got %T", fZero)
	}
	k := xCacheDriver.EncodeKey(c.enc, key)
	raw, err := c.rdb.HGetAll(ctx, k).Result()
	if err != nil {
		return nil, err
	}
	for f, data := range raw {
		var v V
		if err := c.codec.Unmarshal([]byte(data), &v); err != nil {
			return nil, err
		}
		result[any(f).(F)] = v
	}
	return result, nil
}

// GetAllStruct 获取所有字段到结构体 S。
//
// 先通过 [GetAll] 解码为 map[string]V（F 必须为 string），再用 codec 把 map 序列化后
// 反序列化为 S。与 memory 后端的 GetAllStruct 走等价路径，避免直接拼接 raw string
// 导致双重编码。
func (c *HashCache[K, F, V, S]) GetAllStruct(ctx context.Context, key K) (S, error) {
	var s S
	m, err := c.GetAll(ctx, key)
	if err != nil {
		return s, err
	}
	if len(m) == 0 {
		return s, nil
	}
	combined, err := c.codec.Marshal(m)
	if err != nil {
		return s, err
	}
	if err := c.codec.Unmarshal(combined, &s); err != nil {
		return s, err
	}
	return s, nil
}

// SetAll 批量设置字段。
func (c *HashCache[K, F, V, S]) SetAll(ctx context.Context, key K, fields map[F]*V) error {
	if len(fields) == 0 {
		return nil
	}
	k := xCacheDriver.EncodeKey(c.enc, key)
	args := make([]any, 0, len(fields)*2)
	for f, v := range fields {
		if v == nil {
			continue
		}
		data, err := c.codec.Marshal(*v)
		if err != nil {
			return err
		}
		args = append(args, xCacheDriver.EncodeKey(c.enc, f), data)
	}
	if len(args) == 0 {
		return nil
	}
	if err := c.rdb.HSet(ctx, k, args...).Err(); err != nil {
		return err
	}
	c.refreshTTL(ctx, key)
	return nil
}

// SetAllStruct 用结构体批量设置字段。
//
// 先把 S 序列化为 []byte，再反序列化为 map[string]V（要求 field 名为 string），
// 最后逐字段以 codec 编码为 []byte 通过 HSET 写入。
func (c *HashCache[K, F, V, S]) SetAllStruct(ctx context.Context, key K, value S) error {
	data, err := c.codec.Marshal(value)
	if err != nil {
		return err
	}
	var m map[string]V
	if err := c.codec.Unmarshal(data, &m); err != nil {
		return err
	}
	k := xCacheDriver.EncodeKey(c.enc, key)
	args := make([]any, 0, len(m)*2)
	for f, v := range m {
		vData, err := c.codec.Marshal(v)
		if err != nil {
			return err
		}
		args = append(args, f, vData)
	}
	if len(args) == 0 {
		return nil
	}
	if err := c.rdb.HSet(ctx, k, args...).Err(); err != nil {
		return err
	}
	c.refreshTTL(ctx, key)
	return nil
}

// Exists 判断字段是否存在。
func (c *HashCache[K, F, V, S]) Exists(ctx context.Context, key K, field F) (bool, error) {
	n, err := c.rdb.HExists(ctx, xCacheDriver.EncodeKey(c.enc, key), xCacheDriver.EncodeKey(c.enc, field)).Result()
	if err != nil {
		return false, err
	}
	return n, nil
}

// Remove 移除指定字段。
func (c *HashCache[K, F, V, S]) Remove(ctx context.Context, key K, fields ...F) error {
	if len(fields) == 0 {
		return nil
	}
	args := make([]string, 0, len(fields))
	for _, f := range fields {
		args = append(args, xCacheDriver.EncodeKey(c.enc, f))
	}
	return c.rdb.HDel(ctx, xCacheDriver.EncodeKey(c.enc, key), args...).Err()
}

// Delete 删除整个 hash。
func (c *HashCache[K, F, V, S]) Delete(ctx context.Context, key K) error {
	return c.rdb.Del(ctx, xCacheDriver.EncodeKey(c.enc, key)).Err()
}
