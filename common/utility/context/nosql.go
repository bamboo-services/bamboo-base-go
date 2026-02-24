package xCtxUtil

import (
	"context"

	error2 "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtx2 "github.com/bamboo-services/bamboo-base-go/defined/context"
	"github.com/gin-gonic/gin"
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
	if ginCtx, ok := ctx.(*gin.Context); ok {
		ctx = ginCtx.Request.Context()
	}
	if value := ctx.Value(xCtx2.RegNodeKey); value != nil {
		if nodeList, ok := value.(xCtx2.ContextNodeList); ok {
			if component := nodeList.Get(xCtx2.RedisClientKey); component != nil {
				if rdb, ok := component.(*redis.Client); ok {
					return rdb
				}
			}
		}
	}

	value := ctx.Value(xCtx2.RedisClientKey)
	if value != nil {
		if rdb, ok := value.(*redis.Client); ok {
			return rdb
		}
	}
	xLog.WithName(xLog.NamedUTIL).Error(ctx, "在上下文中找不到 Redis 客户端，真的注入成功了吗？")
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
func GetRDB(ctx context.Context) (*redis.Client, *error2.Error) {
	if ginCtx, ok := ctx.(*gin.Context); ok {
		ctx = ginCtx.Request.Context()
	}
	if value := ctx.Value(xCtx2.RegNodeKey); value != nil {
		if nodeList, ok := value.(xCtx2.ContextNodeList); ok {
			if component := nodeList.Get(xCtx2.RedisClientKey); component != nil {
				if rdb, ok := component.(*redis.Client); ok {
					return rdb, nil
				}
			}
		}
	}

	value := ctx.Value(xCtx2.RedisClientKey)
	if value != nil {
		if rdb, ok := value.(*redis.Client); ok {
			return rdb, nil
		}
	}
	return nil, &error2.Error{
		ErrorCode:    error2.CacheError,
		ErrorMessage: "在上下文中找不到 Redis 客户端",
	}
}
