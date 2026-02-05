package xCtxUtil

import (
	"context"

	xConsts "github.com/bamboo-services/bamboo-base-go/context"
	xError "github.com/bamboo-services/bamboo-base-go/error"
	xLog "github.com/bamboo-services/bamboo-base-go/log"
	"github.com/redis/go-redis/v9"
)

// MustGetRDB 从上下文中获取 Redis 客户端实例（panic 版本）。
//
// 如果上下文中未找到 Redis 客户端，则记录错误日志并触发 panic。
//
// 参数说明:
//   - ctx: context.Context 上下文
//
// 返回值:
//   - *redis.Client: Redis 客户端实例
func MustGetRDB(ctx context.Context) *redis.Client {
	value := ctx.Value(xConsts.RedisClientKey)
	if value != nil {
		if rdb, ok := value.(*redis.Client); ok {
			return rdb
		}
	}
	xLog.Error(ctx, "在上下文中找不到 Redis 客户端，真的注入成功了吗？")
	panic("在上下文中找不到 Redis 客户端，真的注入成功了吗？")
}

// GetRDB 从上下文中获取 Redis 客户端实例（错误返回版本）。
//
// 如果上下文中未找到 Redis 客户端，则返回错误而不是 panic。
//
// 参数说明:
//   - ctx: context.Context 上下文
//
// 返回值:
//   - *redis.Client: Redis 客户端实例
//   - *xError.Error: 错误信息，成功时为 nil
func GetRDB(ctx context.Context) (*redis.Client, *xError.Error) {
	value := ctx.Value(xConsts.RedisClientKey)
	if value != nil {
		if rdb, ok := value.(*redis.Client); ok {
			return rdb, nil
		}
	}
	xLog.Error(ctx, "在上下文中找不到 Redis 客户端，真的注入成功了吗？")
	return nil, &xError.Error{
		ErrorCode:    xError.CacheError,
		ErrorMessage: "在上下文中找不到 Redis 客户端",
	}
}
