package log

import (
	"context"

	"github.com/gin-gonic/gin"

	xConsts "github.com/bamboo-services/bamboo-base-go/defined/context"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
)

// GinLogExtractor 基于 gin.Context 的日志追踪 ID 提取器
//
// 实现 [xLog.LogContextExtractor] 接口，从 gin.Context 中提取请求追踪 ID。
// 提取顺序：
//  1. gin.Context.Get(string(xConsts.RequestKey)) — 由 RequestContext 中间件注入
//  2. gin.Context.Request.Context().Value(xConsts.RequestKey) — 标准请求 context
//  3. ctx.Value(xConsts.RequestKey) — 通用 context 回退
type GinLogExtractor struct{}

// 编译期接口断言，确保 GinLogExtractor 实现 LogContextExtractor
var _ xLog.LogContextExtractor = (*GinLogExtractor)(nil)

// ExtractTraceID 从 context 中提取 trace ID
//
// 参数说明:
//   - ctx: 请求上下文（可能是 *gin.Context 或标准 context.Context）
//
// 返回值:
//   - string: 提取到的 trace ID，未找到时返回空字符串
func (g *GinLogExtractor) ExtractTraceID(ctx context.Context) string {
	// 1. 尝试作为 gin.Context 提取
	if ginCtx, ok := ctx.(*gin.Context); ok {
		// 1.1 从 gin.Context 的 keys map 中提取（由 RequestContext 中间件通过 c.Set 注入）
		if v, exists := ginCtx.Get(string(xConsts.RequestKey)); exists {
			if str, ok := v.(string); ok {
				return str
			}
		}
		// 1.2 从 gin.Context 的 Request.Context() 中提取
		if v := ginCtx.Request.Context().Value(xConsts.RequestKey); v != nil {
			if str, ok := v.(string); ok {
				return str
			}
		}
	}

	// 2. 回退到标准 context.Context（GORM 数据库操作等场景）
	if v := ctx.Value(xConsts.RequestKey); v != nil {
		if str, ok := v.(string); ok {
			return str
		}
	}

	return ""
}
