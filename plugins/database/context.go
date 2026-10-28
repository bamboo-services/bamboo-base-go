package database

import (
	"context"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"gorm.io/gorm"
)

// Get 从上下文中提取已绑定当前请求上下文的 [*gorm.DB] 句柄。
//
// 该函数内部调用 [application.Get] 泛型提取句柄，并在 ctx 非空时自动执行
// db.WithContext(ctx) 绑定上下文取消与超时语义。若句柄缺失或为 nil，返回包含
// [apperr.DatabaseError] 的错误。
func Get(ctx context.Context) (*gorm.DB, *apperr.Error) {
	db, err := application.Get[*gorm.DB](ctx, ctxkey.DatabaseKey)
	if err != nil {
		return nil, err
	}
	if db == nil {
		return nil, apperr.New(apperr.DatabaseError, "上下文中的数据库句柄为 nil", nil)
	}
	if ctx != nil {
		return db.WithContext(ctx), nil
	}
	return db, nil
}

// MustGet 从上下文中提取已绑定当前请求上下文的 [*gorm.DB] 句柄，提取失败时直接触发 panic。
//
// 适用于强依赖数据库组件且未就绪属于严重编程或配置错误的调用场景。
func MustGet(ctx context.Context) *gorm.DB {
	db, err := Get(ctx)
	if err != nil {
		panic(err)
	}
	return db
}
