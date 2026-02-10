package xReg

import (
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
		engine.Use(r.Init.InjectContext())
	})
}
