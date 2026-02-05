package xReg

import (
	"context"

	xHelper "github.com/bamboo-services/bamboo-base-go/helper"
	xLog "github.com/bamboo-services/bamboo-base-go/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/utility/ctxutil"
	xVaild "github.com/bamboo-services/bamboo-base-go/validator"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// engineInit 启动并返回一个默认的 Gin 引擎实例。
//
// 该方法创建并返回一个使用默认配置初始化的 `gin.Engine` 实例
// 通常用于构建基础的 HTTP 服务器。
//
// 返回值:
//   - `*gin.Engine`: 成功初始化的默认 Gin 引擎实例。
func (r *Reg) engineInit() {
	log := xLog.WithName(xLog.NamedINIT)

	log.Debug(r.Init.Ctx, "初始化 GIN 引擎")
	if !xCtxUtil.IsDebugMode() {
		gin.SetMode(gin.ReleaseMode)
	}

	// 注册自定义验证器和翻译器
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// 注册翻译器（必须在自定义验证器之前）
		if err := xVaild.RegisterTranslator(v); err != nil {
			log.Error(r.Init.Ctx, "翻译器注册失败: "+err.Error())
		} else {
			log.Info(r.Init.Ctx, "翻译器注册成功")
		}

		// 注册自定义验证器
		if err := xVaild.RegisterCustomValidators(v); err != nil {
			log.Error(r.Init.Ctx, "验证器注册失败: "+err.Error())
		} else {
			log.Info(r.Init.Ctx, "预制内部验证器注册成功")
		}
	}

	r.Serve = gin.New(func(engine *gin.Engine) {
		engine.Use(xHelper.RequestContext())
		engine.Use(xHelper.PanicRecovery())
		engine.Use(xHelper.HttpLogger())
		engine.Use(injectContext(r.Init.Ctx))
	})
}

// injectContext 返回一个 Gin 中间件，用于将外部上下文注入到请求的上下文中。
//
// 该中间件将传入的 context.Context 设置为请求的上下文，确保在后续的处理流程中，
// 可以访问该上下文中携带的值（如初始化配置、请求追踪信息等）。调用 c.Next() 继续执行后续中间件。
func injectContext(ctx context.Context) func(c *gin.Context) {
	return func(c *gin.Context) {
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
