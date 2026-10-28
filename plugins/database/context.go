package database

import (
	"context"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"gorm.io/gorm"
)

// Get 从上下文中提取已绑定当前请求上下文的 *gorm.DB。
// 内部执行 db.WithContext(ctx)，返回 application.Get 的安全应用级错误。
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

// MustGet 从上下文中提取已绑定当前请求上下文的 *gorm.DB，失败触发 panic。
func MustGet(ctx context.Context) *gorm.DB {
	db, err := Get(ctx)
	if err != nil {
		panic(err)
	}
	return db
}
