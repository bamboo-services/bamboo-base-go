package xInit

import (
	xConsts "github.com/bamboo-services/bamboo-base-go/constants"
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
	r.Logger.Named(xConsts.LogINIT).Info("初始化 GIN 引擎")
	r.Serve = gin.Default()

	// 注册自定义验证器
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		xVaild.RegisterCustomValidators(v)
		r.Logger.Named(xConsts.LogINIT).Info("预制内部验证器注册成功")
	}
}
