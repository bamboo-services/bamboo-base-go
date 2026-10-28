package redis_test

import (
	"context"
	"time"

	"github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver"
	"github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/redis"
	goredis "github.com/redis/go-redis/v9"
)

// ExampleNewKeyCache 演示基于已有 Redis 客户端构造泛型 KeyCache。
//
// 涉及网络 I/O 的调用作为编译型示例，不验证确定性标准输出。
func ExampleNewKeyCache() {
	rdb := goredis.NewClient(&goredis.Options{
		Addr: "localhost:6379",
	})
	defer func() { _ = rdb.Close() }()

	ctx := context.Background()
	keyCache := redis.NewKeyCache[string, string](rdb, nil, nil, 10*time.Minute)

	val := "session_data"
	// 写入带条件控制的键
	_ = keyCache.Set(ctx, "session:123", &val, driver.WithNX())
	_, _, _ = keyCache.Get(ctx, "session:123")
}

// ExampleNewHashCache 演示基于 Redis 客户端构造泛型 HashCache 并批量操作字段。
func ExampleNewHashCache() {
	rdb := goredis.NewClient(&goredis.Options{
		Addr: "localhost:6379",
	})
	defer func() { _ = rdb.Close() }()

	ctx := context.Background()
	type UserInfo struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	hashCache := redis.NewHashCache[string, string, string, UserInfo](rdb, nil, nil, 30*time.Minute)

	val := "Alice"
	_ = hashCache.Set(ctx, "user:profile", "name", &val)
	_, _ = hashCache.GetAll(ctx, "user:profile")
}
