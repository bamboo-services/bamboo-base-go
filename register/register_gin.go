package xReg

import (
	xHelper "github.com/bamboo-services/bamboo-base-go/helper"
	xLog "github.com/bamboo-services/bamboo-base-go/log"
	xVaild "github.com/bamboo-services/bamboo-base-go/validator"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// EngineInit 启动并返回一个默认的 Gin 引擎实例。
//
// 该方法创建并返回一个使用默认配置初始化的 `gin.Engine` 实例，
// 通常用于构建基础的 HTTP 服务器。
//
// 返回值:
//   - `*gin.Engine`: 成功初始化的默认 Gin 引擎实例。
func (r *Reg) EngineInit() {
	log := xLog.WithName(xLog.NamedINIT)

	log.Debug(r.Context, "初始化 GIN 引擎")
	if !isDebugMode() {
		gin.SetMode(gin.ReleaseMode)
	}
	r.Serve = gin.New(func(engine *gin.Engine) {
		engine.Use(xHelper.PanicRecovery())
		engine.Use(gin.Logger())
	})

	// 注册自定义验证器
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		xVaild.RegisterCustomValidators(v)
		log.Debug(r.Context, "预制内部验证器注册成功")
	}
}
