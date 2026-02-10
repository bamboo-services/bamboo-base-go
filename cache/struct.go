package xCache

import (
	"time"

	xLog "github.com/bamboo-services/bamboo-base-go/log"
	"github.com/redis/go-redis/v9"
)

// Cache 封装了一个用于缓存数据的 Redis 客户端。
//
// 它使用预配置的生存时间（TTL）来管理键值的过期策略。
type Cache struct {
	RDB *redis.Client
	TTL time.Duration
	Log *xLog.LogNamedLogger
}
