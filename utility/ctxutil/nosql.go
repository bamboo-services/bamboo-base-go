package xCtxUtil

import (
	xConsts "github.com/bamboo-services/bamboo-base-go/constants"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// GetRDB 从 gin.Context 中获取 Redis 客户端实例。
// 如果上下文中未找到 Redis 客户端，则记录错误日志并触发 panic。
// 返回值为 *redis.Client 类型，获取成功时返回对应的 Redis 客户端。
func GetRDB(c *gin.Context) *redis.Client {
	value, exists := c.Get(xConsts.ContextRedisClient.String())
	if exists {
		return value.(*redis.Client)
	}
	GetLogger(c, xConsts.LogUTIL).Panic("在上下文中找不到数据库，真的注入成功了吗？")
	return nil
}
