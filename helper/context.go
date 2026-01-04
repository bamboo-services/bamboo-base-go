package xHelper

import (
	"context"
	"time"

	xConsts "github.com/bamboo-services/bamboo-base-go/context"
	"github.com/bamboo-services/bamboo-base-go/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestContext 是一个 Gin 中间件，用于为每个请求生成唯一 ID 和记录请求的开始时间。
//
// - 请求唯一 ID 会通过 UUID 生成，并存储在响应头字段 `X-Request-UUID`，用于请求溯源。
// - 请求的开始时间会被存储到上下文中，以实现请求生命周期的时间追踪。
//
// 上下文中设置的关键值：
// - `context_request_key`: 表示请求的唯一标识符。
// - `context_user_start_time`: 表示请求开始处理的时间。
func RequestContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成请求唯一 ID 「用于溯源」
		requestID := uuid.NewString()
		c.Writer.Header().Set(http.HeaderRequestUUID.String(), requestID)

		c.Set(xConsts.RequestKey.String(), requestID)        // 上下文请求记录
		c.Set(xConsts.UserStartTimeKey.String(), time.Now()) // 请求开始时间记录

		// 将 RequestID 注入到标准 context 中（供 slog 使用）
		ctx := context.WithValue(c.Request.Context(), xConsts.RequestKey, requestID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
