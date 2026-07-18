package xCtxUtil

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCache "github.com/bamboo-services/bamboo-base-go/major/cache"
	xCtx "github.com/bamboo-services/bamboo-base-go/defined/context"
)

// MustGetCacheManager 从上下文中获取缓存管理器实例（panic 版本）。
//
// 从 [xCtx.CacheManagerKey] 提取 [*xCache.Manager]。若未注入则记录错误并 panic。
// 业务侧拿到 Manager 后可调用 KeyCache/HashCache/SetCache/ListCache 获取泛型缓存实例，
// 或通过 Redis/Memory 直接控制底层后端。
func MustGetCacheManager(ctx context.Context) *xCache.Manager {
	stdCtx := ctx
	if globalContextExtractor != nil {
		stdCtx = globalContextExtractor.ExtractRequestContext(ctx)
	}
	if value := stdCtx.Value(xCtx.RegNodeKey); value != nil {
		if nodeList, ok := value.(xCtx.ContextNodeList); ok {
			if component := nodeList.Get(xCtx.CacheManagerKey); component != nil {
				if m, ok := component.(*xCache.Manager); ok {
					return m
				}
			}
		}
	}
	if value := stdCtx.Value(xCtx.CacheManagerKey); value != nil {
		if m, ok := value.(*xCache.Manager); ok {
			return m
		}
	}
	xLog.WithName(xLog.NamedUTIL).Error(ctx, "在上下文中找不到缓存管理器，真的注入成功了吗？")
	panic("在上下文中找不到缓存管理器，真的注入成功了吗？")
}

// GetCacheManager 从上下文中获取缓存管理器实例（错误返回版本）。
//
// 若未找到则返回错误而非 panic，便于业务侧做优雅降级。
func GetCacheManager(ctx context.Context) (*xCache.Manager, *xError.Error) {
	stdCtx := ctx
	if globalContextExtractor != nil {
		stdCtx = globalContextExtractor.ExtractRequestContext(ctx)
	}
	if value := stdCtx.Value(xCtx.RegNodeKey); value != nil {
		if nodeList, ok := value.(xCtx.ContextNodeList); ok {
			if component := nodeList.Get(xCtx.CacheManagerKey); component != nil {
				if m, ok := component.(*xCache.Manager); ok {
					return m, nil
				}
			}
		}
	}
	if value := stdCtx.Value(xCtx.CacheManagerKey); value != nil {
		if m, ok := value.(*xCache.Manager); ok {
			return m, nil
		}
	}
	return nil, &xError.Error{
		ErrorCode:    xError.CacheError,
		ErrorMessage: "在上下文中找不到缓存管理器",
	}
}
