package xReg

import (
	"context"

	"github.com/bamboo-services/bamboo-base-go/env"
	"github.com/gin-gonic/gin"
)

// Reg 表示应用程序的核心注册结构，包含所有初始化后的组件实例。
type Reg struct {
	Context context.Context // 上下文，用于控制取消和超时
	Serve   *gin.Engine     // Gin 引擎实例
}

// New 创建并返回一个未初始化的 `Reg` 实例。
//
// 该函数仅分配内存并返回 `Reg` 类型的初始值，
// 需要调用者进一步初始化相关字段。
//
// 返回值:
//   - `*Reg`: 返回一个新的 `Reg` 实例。
func newReg() *Reg {
	return &Reg{
		Context: context.Background(),
	}
}

// Register 注册并初始化应用的核心组件，包括配置、日志、Gin 引擎及系统上下文。
//
// 此函数调用 `ConfigInit` 加载配置文件，`LoggerInit` 初始化日志记录器，
// `EngineInit` 启动 Gin 引擎实例，并通过 `SystemContextInit` 配置系统必要的上下文功能。
// 返回初始化完成的 `gin.Engine` 实例，用于处理 HTTP 请求。
//
// 注意: 如果在初始化过程中发生致命错误（如配置文件缺失、日志初始化失败等），
// 可能会触发 `panic`，请在部署前确认所有依赖的资源均已就绪。
//
// 返回值:
//   - `*gin.Engine`: 完成初始化的 Gin 引擎实例，可直接用于运行服务器。
func Register() *Reg {
	reg := newReg()

	reg.ConfigInit()    // 初始化配置
	reg.LoggerInit()    // 初始化日志记录器
	reg.SnowflakeInit() // 初始化雪花算法节点
	reg.EngineInit()    // 启动 Gin 引擎

	// 初始化系统上下文
	reg.SystemContextInit()

	return reg
}

// isDebugMode 判断是否处于调试模式。
func isDebugMode() bool {
	return xEnv.GetEnvBool(xEnv.Debug.String(), false)
}
