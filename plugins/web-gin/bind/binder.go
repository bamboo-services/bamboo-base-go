package bind

import (
	"context"
	"errors"

	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/validator"
	"github.com/gin-gonic/gin"
)

type binderCtxKey struct{}

var (
	// ErrNoBinderInContext 表示未在请求上下文中找到已通过 Inject 注入的 Binder 实例。
	ErrNoBinderInContext = errors.New("bind: context 中未找到有效的 Binder 实例，请先通过 Inject 注入")
	// ErrNilEngine 表示构造 Binder 时传入的 validator.Engine 为 nil。
	ErrNilEngine = errors.New("bind: validator.Engine 不能为 nil")
	// ErrNilTarget 表示执行绑定操作时传入的目标结构体指针为 nil。
	ErrNilTarget = errors.New("bind: 目标结构体指针不能为 nil")
)

// Binder 封装私有的参数验证引擎，为 HTTP 请求绑定流程提供结构体校验服务。
//
// 实例由 Web 插件在启动流程中创建，并在请求流转中通过上下文注入。
// 初始化后属于无状态或只读状态，可被多个请求协程安全并发共享。
type Binder struct {
	engine *validator.Engine
}

// New 使用显式传入的独立 [*validator.Engine] 实例构造全新的 [Binder]。
//
// 若传入的 engine 为 nil，返回 [ErrNilEngine]。构造过程绝不使用全局验证器单例。
func New(engine *validator.Engine) (*Binder, error) {
	if engine == nil {
		return nil, ErrNilEngine
	}
	return &Binder{engine: engine}, nil
}

// Engine 返回绑定的 [*validator.Engine] 验证引擎实例。
//
// 若 Binder 接收器本身为 nil，返回 nil。
func (b *Binder) Engine() *validator.Engine {
	if b == nil {
		return nil
	}
	return b.engine
}

// Inject 将 [Binder] 实例使用私有上下文键挂载至 [*gin.Context] 的底层请求上下文。
//
// 若 ctx、ctx.Request 或 b 任一参数为 nil，此函数安全忽略，不执行任何操作。
func Inject(ctx *gin.Context, b *Binder) {
	if ctx == nil || ctx.Request == nil || b == nil {
		return
	}
	reqCtx := context.WithValue(ctx.Request.Context(), binderCtxKey{}, b)
	ctx.Request = ctx.Request.WithContext(reqCtx)
}

// FromContext 从 [*gin.Context] 的请求上下文中提取预先注入的 [Binder] 实例。
//
// 若未找到注入的实例、类型不匹配或上下文为 nil，返回 nil，绝不隐式回退到任何全局单例。
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

// Bind 构造泛型绑定器 [Binding]，作为通用的链式参数绑定入口。
//
// 调用方需传入当前请求的 [*gin.Context] 以及目标结构体指针 data。
// 该函数内部自动通过 [FromContext] 提取请求中挂载的 [Binder] 实例。
// 随后可链式调用 [Binding.Data]、[Binding.Query] 等方法执行实际绑定与校验。
func Bind[T any](ctx *gin.Context, data *T) *Binding[T] {
	return &Binding[T]{
		Context: ctx,
		GetData: data,
		binder:  FromContext(ctx),
	}
}

// WithBinder 使用显式指定的 [Binder] 实例构造泛型绑定器 [Binding]。
//
// 适合单元测试、离线调试或未经过标准 [Inject] 中间件的场景。
func WithBinder[T any](ctx *gin.Context, b *Binder, data *T) *Binding[T] {
	return &Binding[T]{
		Context: ctx,
		GetData: data,
		binder:  b,
	}
}
