package bind

import (
	"context"
	"errors"

	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/validator"
	"github.com/gin-gonic/gin"
)

type binderCtxKey struct{}

var (
	// ErrNoBinderInContext 当从 gin 上下文找不到已注入的 Binder 实例时返回。
	ErrNoBinderInContext = errors.New("bind: context 中未找到有效的 Binder 实例，请先通过 Inject 注入")
	// ErrNilEngine 当向 New 传入空 validator.Engine 时返回。
	ErrNilEngine = errors.New("bind: validator.Engine 不能为 nil")
	// ErrNilTarget 当绑定目标结构体指针为 nil 时返回。
	ErrNilTarget = errors.New("bind: 目标结构体指针不能为 nil")
)

// Binder 封装独立的验证引擎，提供 HTTP 请求参数与 Body 的反序列化与结构体校验。
type Binder struct {
	engine *validator.Engine
}

// New 使用传入的独立 validator.Engine 实例构造 Binder。
// 绝不使用全局 validator 或偷建全局引擎。
func New(engine *validator.Engine) (*Binder, error) {
	if engine == nil {
		return nil, ErrNilEngine
	}
	return &Binder{engine: engine}, nil
}

// Engine 返回绑定的 validator.Engine 实例。
func (b *Binder) Engine() *validator.Engine {
	if b == nil {
		return nil
	}
	return b.engine
}

// Inject 将 Binder 实例以私有 context key 挂载到 ctx.Request.Context()。
func Inject(ctx *gin.Context, b *Binder) {
	if ctx == nil || ctx.Request == nil || b == nil {
		return
	}
	reqCtx := context.WithValue(ctx.Request.Context(), binderCtxKey{}, b)
	ctx.Request = ctx.Request.WithContext(reqCtx)
}

// FromContext 从 ctx.Request.Context() 提取注入的 Binder 实例。
// 不存在或类型不匹配时返回 nil，绝不退回到全局 Binder。
func FromContext(ctx *gin.Context) *Binder {
	if ctx == nil || ctx.Request == nil {
		return nil
	}
	val := ctx.Request.Context().Value(binderCtxKey{})
	if b, ok := val.(*Binder); ok {
		return b
	}
	return nil
}

// Bind 兼容入口：提供链式调用入口 Bind(ctx, &req).Data()/Query()/URI/Header()。
// 内部从 ctx.Request.Context() 解析已注入的 Binder，绝不偷建全局引擎。
func Bind[T any](ctx *gin.Context, data *T) *Binding[T] {
	return &Binding[T]{
		Context: ctx,
		GetData: data,
		binder:  FromContext(ctx),
	}
}

// WithBinder 使用显式指定的 Binder 构造链式绑定器，适合未经过 Inject 中间件的场景。
func WithBinder[T any](ctx *gin.Context, b *Binder, data *T) *Binding[T] {
	return &Binding[T]{
		Context: ctx,
		GetData: data,
		binder:  b,
	}
}
