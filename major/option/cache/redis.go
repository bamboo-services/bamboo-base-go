package cache

import "time"

// WithRedis 配置 Redis 作为缓存后端，返回 [CacheOption]。
//
// 隐式将缓存类型置为 CacheTypeRedis。addr 为必填项（host:port）；
// 其余参数通过 [RedisOption] 可变参数按需设置。
//
// 使用示例：
//
//	xOption.WithCache(xOptionCache.WithRedis("localhost:6379", xOptionCache.WithRedisPassword("xxx")))
func WithRedis(addr string, opts ...RedisOption) CacheOption {
	return func(c *CacheConfig) {
		c.typeVal = CacheTypeRedis
		c.redis = RedisOptions{Addr: addr}
		for _, o := range opts {
			if o != nil {
				o(&c.redis)
			}
		}
	}
}

// RedisOption 是 [RedisOptions] 的二级选项，避免 [WithRedis] 参数列表过长。
type RedisOption func(*RedisOptions)

// WithRedisUsername 设置 Redis ACL 用户名。
func WithRedisUsername(u string) RedisOption { return func(r *RedisOptions) { r.Username = u } }

// WithRedisPassword 设置 Redis 密码。
func WithRedisPassword(p string) RedisOption { return func(r *RedisOptions) { r.Password = p } }

// WithRedisDB 设置 Redis 数据库序号。
func WithRedisDB(db int) RedisOption { return func(r *RedisOptions) { r.DB = db } }

// WithRedisPoolSize 设置连接池大小。
func WithRedisPoolSize(n int) RedisOption { return func(r *RedisOptions) { r.PoolSize = n } }

// WithRedisMinIdleConns 设置最小空闲连接数。
func WithRedisMinIdleConns(n int) RedisOption { return func(r *RedisOptions) { r.MinIdleConns = n } }

// WithRedisDialTimeout 设置连接建立超时。
func WithRedisDialTimeout(d time.Duration) RedisOption {
	return func(r *RedisOptions) { r.DialTimeout = d }
}

// WithRedisReadTimeout 设置读操作超时。
func WithRedisReadTimeout(d time.Duration) RedisOption {
	return func(r *RedisOptions) { r.ReadTimeout = d }
}

// WithRedisWriteTimeout 设置写操作超时。
func WithRedisWriteTimeout(d time.Duration) RedisOption {
	return func(r *RedisOptions) { r.WriteTimeout = d }
}
