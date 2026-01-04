package xCtxUtil

import (
	"log/slog"

	xConsts "github.com/bamboo-services/bamboo-base-go/context"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetDB 从 Gin 上下文中获取数据库连接实例。
//
// 该函数尝试从上下文中检索数据库连接实例，如果成功则返回该实例；
// 如果未找到，则记录错误并触发 panic。
//
// 注意: 在使用此函数之前，请确保数据库连接已正确注入到上下文中，
// 通常通过中间件或其他初始化逻辑完成。
func GetDB(c *gin.Context) *gorm.DB {
	value, exists := c.Get(xConsts.DatabaseKey.String())
	if exists {
		return value.(*gorm.DB).WithContext(c)
	}
	slog.ErrorContext(c, "在上下文中找不到数据库，真的注入成功了吗？")
	panic("在上下文中找不到数据库，真的注入成功了吗？")
}
