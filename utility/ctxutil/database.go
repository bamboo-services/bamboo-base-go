package xCtxUtil

import (
	"context"

	xCtx "github.com/bamboo-services/bamboo-base-go/context"
	xError "github.com/bamboo-services/bamboo-base-go/error"
	xLog "github.com/bamboo-services/bamboo-base-go/log"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// MustGetDB 从上下文中获取数据库连接实例（panic 版本）。
//
// 该函数尝试从上下文中检索数据库连接实例，如果成功则返回该实例；
// 如果未找到，则记录错误并触发 panic。
//
// 注意: 在使用此函数之前，请确保数据库连接已正确注入到上下文中，
// 通常通过中间件或其他初始化逻辑完成。
//
// 参数说明:
//   - ctx: context.Context 上下文
//
// 返回值:
//   - *gorm.DB: 数据库连接实例
func MustGetDB(ctx context.Context) *gorm.DB {
	if ginCtx, ok := ctx.(*gin.Context); ok {
		ctx = ginCtx.Request.Context()
	}
	if value := ctx.Value(xCtx.RegNodeKey); value != nil {
		if nodeList, ok := value.(xCtx.ContextNodeList); ok {
			if component := nodeList.Get(xCtx.DatabaseKey); component != nil {
				if db, ok := component.(*gorm.DB); ok {
					return db.WithContext(ctx)
				}
			}
		}
	}
	if value := ctx.Value(xCtx.DatabaseKey); value != nil {
		if db, ok := value.(*gorm.DB); ok {
			return db.WithContext(ctx)
		}
	}
	xLog.WithName(xLog.NamedUTIL).Error(ctx, "在上下文中找不到数据库，真的注入成功了吗？")
	panic("在上下文中找不到数据库，真的注入成功了吗？")
}

// GetDB 从上下文中获取数据库连接实例（错误返回版本）。
//
// 该函数尝试从上下文中检索数据库连接实例，如果成功则返回该实例；
// 如果未找到，则返回错误而不是 panic。
//
// 参数说明:
//   - ctx: context.Context 上下文
//
// 返回值:
//   - *gorm.DB: 数据库连接实例
//   - *xError.Error: 错误信息，成功时为 nil
func GetDB(ctx context.Context) (*gorm.DB, *xError.Error) {
	if ginCtx, ok := ctx.(*gin.Context); ok {
		ctx = ginCtx.Request.Context()
	}
	if value := ctx.Value(xCtx.RegNodeKey); value != nil {
		if nodeList, ok := value.(xCtx.ContextNodeList); ok {
			if component := nodeList.Get(xCtx.DatabaseKey); component != nil {
				if db, ok := component.(*gorm.DB); ok {
					return db.WithContext(ctx), nil
				}
			}
		}
	}

	value := ctx.Value(xCtx.DatabaseKey)
	if value != nil {
		if db, ok := value.(*gorm.DB); ok {
			return db.WithContext(ctx), nil
		}
	}
	return nil, &xError.Error{
		ErrorCode:    xError.DatabaseError,
		ErrorMessage: "在上下文中找不到数据库",
	}
}
