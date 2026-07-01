package xCtxUtil

import (
	"context"

	xCtx "github.com/bamboo-services/bamboo-base-go/defined/context"
	"github.com/gin-gonic/gin"
)

// globalContextExtractor 保存 major 层全局注册的 ContextExtractor 实例。
//
// 默认值为 nil，表示未注册，major 层上下文工具将回退到直接使用 ctx.Value() 取值。
// 与 common 层的 globalContextExtractor 独立，分别服务于各自包内的函数。
var globalContextExtractor ContextExtractor

// ContextExtractor 定义了从框架上下文提取标准 context.Context 的标准接口。
//
// 该接口用于解耦对具体 HTTP 框架（如 gin）的依赖，由上层实现并通过
// SetContextExtractor 注册。major 层的 GetDB / GetRDB 等函数优先使用
// 注册的 ContextExtractor 提取标准 context，未注册时回退到 ctx.Value()。
type ContextExtractor interface {
	// ExtractRequestContext 从框架上下文中提取标准 context.Context。
	ExtractRequestContext(ctx context.Context) context.Context
	// ExtractRequestKey 从框架上下文中提取请求唯一标识键。
	ExtractRequestKey(ctx context.Context) string
	// ExtractErrorMessage 从框架上下文中提取错误消息。
	ExtractErrorMessage(ctx context.Context) string
}

// SetContextExtractor 注册一个 ContextExtractor 实现到 major 层。
//
// 参数 ce 为上层实现的 ContextExtractor 实例；传入 nil 将清除已有注册。
// 注册后，major 层的 GetDB / GetRDB / GetEmailClient / GetSnowflakeNode /
// Get[T] / MustGet[T] 等函数将优先通过 ContextExtractor 提取标准 context。
func SetContextExtractor(ce ContextExtractor) {
	globalContextExtractor = ce
}

// ginContextExtractor 是 ContextExtractor 接口的 gin 框架实现。
//
// 该实现将 gin.Context 转换为标准 context.Context，通过 ginCtx.Request.Context()
// 提取底层 HTTP 请求上下文，从而让 major 层工具函数无需直接依赖 gin。
type ginContextExtractor struct{}

// Ensure ginContextExtractor implements ContextExtractor at compile time.
var _ ContextExtractor = (*ginContextExtractor)(nil)

// ExtractRequestContext 从 gin.Context 中提取标准 context.Context。
//
// 如果传入的 ctx 是 *gin.Context，则返回其底层 Request.Context()；
// 否则原样返回 ctx。
func (g *ginContextExtractor) ExtractRequestContext(ctx context.Context) context.Context {
	if ginCtx, ok := ctx.(*gin.Context); ok {
		return ginCtx.Request.Context()
	}
	return ctx
}

// ExtractRequestKey 从 gin.Context 中提取请求唯一标识键。
//
// 先通过 Request.Context() 提取标准 context，再从其中读取 RequestKey。
func (g *ginContextExtractor) ExtractRequestKey(ctx context.Context) string {
	stdCtx := g.ExtractRequestContext(ctx)
	if value := stdCtx.Value(xCtx.RequestKey); value != nil {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// ExtractErrorMessage 从 gin.Context 中提取错误消息。
//
// 先通过 Request.Context() 提取标准 context，再从其中读取 ErrorMessageKey。
func (g *ginContextExtractor) ExtractErrorMessage(ctx context.Context) string {
	stdCtx := g.ExtractRequestContext(ctx)
	if value := stdCtx.Value(xCtx.ErrorMessageKey); value != nil {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// NewGinContextExtractor 创建一个 gin 框架的 ContextExtractor 实例。
//
// 返回的实例可在应用启动时通过 SetContextExtractor 注册到 major 层，
// 使 GetDB / GetRDB 等函数能够正确从 gin.Context 提取标准 context。
func NewGinContextExtractor() ContextExtractor {
	return &ginContextExtractor{}
}
