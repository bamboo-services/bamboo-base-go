package redis

import (
	"context"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	driver "github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver"
	goredis "github.com/redis/go-redis/v9"
)

// SetCache 是 [driver.SetCache] 的 Redis 存储实现。
type SetCache[K any, V any] struct {
	rdb   *goredis.Client
	codec driver.Codec
	enc   driver.KeyEncoder
	ttl   time.Duration
}

// NewSetCache 构造一个基于 Redis 的 [driver.SetCache] 实现。
func NewSetCache[K any, V any](rdb *goredis.Client, codec driver.Codec, enc driver.KeyEncoder, ttl time.Duration) driver.SetCache[K, V] {
	if codec == nil {
		codec = driver.JSONCodec{}
	}
	return &SetCache[K, V]{rdb: rdb, codec: codec, enc: enc, ttl: ttl}
}

// ready 校验缓存实例是否可用，避免 nil 接收器或未注入 Redis 客户端时 panic。
func (c *SetCache[K, V]) ready() *apperr.Error {
	if c == nil || c.rdb == nil {
		return notInitializedErr()
	}
	return nil
}

// refreshTTL 在写操作后按需续期。
func (c *SetCache[K, V]) refreshTTL(ctx context.Context, key K, ttl time.Duration) {
	if ttl > 0 {
		_ = c.rdb.Expire(ctx, driver.EncodeKey(c.enc, key), ttl)
	}
}

// encodeMembers 把多个成员序列化为 Redis 命令可接受的 []any 参数。
func (c *SetCache[K, V]) encodeMembers(members []V) ([]any, *apperr.Error) {
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

// Add 将一组成员添加到集合中，已存在的成员会被忽略。
//
// opts 用于在单次调用覆盖默认 TTL 或附加条件：
//   - NX：仅当 key 不存在时写入（先 Exists 预检，存在则跳过）
//   - XX：仅当 key 已存在时写入（先 Exists 预检，不存在则跳过）
//   - NoSlide/KeepTTL：写入但不续期（跳过 refreshTTL）
func (c *SetCache[K, V]) Add(ctx context.Context, key K, members []V, opts ...driver.SetOption) *apperr.Error {
	if err := c.ready(); err != nil {
		return err
	}
	if len(members) == 0 {
		return nil
	}
	cfg := driver.ApplySet(c.ttl, opts)
	k := driver.EncodeKey(c.enc, key)
	// NX/XX 预检：基于 key 是否存在决定是否跳过写入
	if cfg.NX || cfg.XX {
		exists, err := c.rdb.Exists(ctx, k).Result()
		if err != nil {
			return opErr(err)
		}
		if cfg.NX && exists > 0 {
			return nil
		}
		if cfg.XX && exists == 0 {
			return nil
		}
	}
	args, err := c.encodeMembers(members)
	if err != nil {
		return err
	}
	if err := c.rdb.SAdd(ctx, k, args...).Err(); err != nil {
		return opErr(err)
	}
	// NoSlide/KeepTTL 时跳过续期，保留原有 TTL
	if !cfg.NoSlide && !cfg.KeepTTL {
		c.refreshTTL(ctx, key, cfg.TTL)
	}
	return nil
}

// Members 获取集合中的所有成员。
func (c *SetCache[K, V]) Members(ctx context.Context, key K) ([]V, *apperr.Error) {
	if err := c.ready(); err != nil {
		return nil, err
	}
	raw, err := c.rdb.SMembers(ctx, driver.EncodeKey(c.enc, key)).Result()
	if err != nil {
		return nil, opErr(err)
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
func (c *SetCache[K, V]) IsMember(ctx context.Context, key K, member V) (bool, *apperr.Error) {
	if err := c.ready(); err != nil {
		return false, err
	}
	data, err := c.codec.Marshal(member)
	if err != nil {
		return false, err
	}
	found, cmdErr := c.rdb.SIsMember(ctx, driver.EncodeKey(c.enc, key), data).Result()
	if cmdErr != nil {
		return false, opErr(cmdErr)
	}
	return found, nil
}

// Count 获取集合中的成员数量。
func (c *SetCache[K, V]) Count(ctx context.Context, key K) (int64, *apperr.Error) {
	if err := c.ready(); err != nil {
		return 0, err
	}
	count, err := c.rdb.SCard(ctx, driver.EncodeKey(c.enc, key)).Result()
	if err != nil {
		return 0, opErr(err)
	}
	return count, nil
}

// Remove 从集合中移除指定的成员。
func (c *SetCache[K, V]) Remove(ctx context.Context, key K, members ...V) *apperr.Error {
	if err := c.ready(); err != nil {
		return err
	}
	if len(members) == 0 {
		return nil
	}
	args, err := c.encodeMembers(members)
	if err != nil {
		return err
	}
	return opErr(c.rdb.SRem(ctx, driver.EncodeKey(c.enc, key), args...).Err())
}

// Delete 删除整个集合。
func (c *SetCache[K, V]) Delete(ctx context.Context, key K) *apperr.Error {
	if err := c.ready(); err != nil {
		return err
	}
	return opErr(c.rdb.Del(ctx, driver.EncodeKey(c.enc, key)).Err())
}
