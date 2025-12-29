package xInit

import (
	"log/slog"
	"time"

	xConsts "github.com/bamboo-services/bamboo-base-go/constants"
	xHelper "github.com/bamboo-services/bamboo-base-go/helper"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type handler struct {
	Data *Reg
}

// SystemContextInit 初始化系统上下文。
//
// 该方法为系统设置必要的上下文中间件，用于扩展 Gin 的上下文功能，
// 以便在请求生命周期内共享状态或传递必要的信息。
//
// 注意: 确保在 Gin 引擎初始化后调用此方法，以正确注册中间件。
func (r *Reg) SystemContextInit() {
	slog.Info("初始化系统上下文")

	// 创建处理器实例
	handler := &handler{
		Data: r,
	}

	// 注册系统上下文处理函数
	r.Serve.Use(handler.systemContextHandlerFunc)
	r.Serve.Use(xHelper.PanicRecovery())
}

// systemContextHandlerFunc 创建和管理请求的唯一上下文标识符。
//
// 该函数为每个 HTTP 请求生成一个唯一的 `RequestID`，随后将该值存储在 Gin 上下文中，
// 通过 `consts.ContextRequestKey` 进行访问，并设置为响应头的一部分以便于请求溯源。
//
// 在生成和设置 `RequestID` 后，函数将调用 `c.Next()` 放行请求，允许后续中间件或路由处理。
//
// 注意:
//   - 生成的 `RequestID` 使用 `uuid.NewString()` 方法。
//   - 响应头中添加了 `X-Request-ID` 字段以包含该 `RequestID`。
func (h *handler) systemContextHandlerFunc(c *gin.Context) {
	// 生成请求唯一 ID 「用于溯源」
	requestID := uuid.NewString()
	c.Writer.Header().Set(xConsts.HeaderRequestUUID.String(), requestID)

	c.Set(xConsts.ContextRequestKey.String(), requestID)     // 上下文请求记录
	c.Set(xConsts.ContextUserStartTime.String(), time.Now()) // 请求开始时间记录

	// 注入雪花算法节点
	c.Set(xConsts.ContextSnowflakeNode.String(), xSnowflake.GetDefaultNode())         // 标准雪花节点
	c.Set(xConsts.ContextGeneSnowflakeNode.String(), xSnowflake.GetDefaultGeneNode()) // 基因雪花节点

	// 放行内容
	c.Next()
}
