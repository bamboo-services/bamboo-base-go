package xReg

import (
	"context"

	xConsts "github.com/bamboo-services/bamboo-base-go/context"
	xInit "github.com/bamboo-services/bamboo-base-go/register/init"
	xRegNode "github.com/bamboo-services/bamboo-base-go/register/node"
	"github.com/gin-gonic/gin"
)

// Reg 表示应用程序的核心注册结构，包含所有初始化后的组件实例。
type Reg struct {
	Serve *gin.Engine       // Gin 引擎实例
	Init  *xRegNode.RegNode // 初始化节点
}

// New 创建并返回一个未初始化的 `Reg` 实例。
//
// 该函数仅分配内存并返回 `Reg` 类型的初始值，
// 需要调用者进一步初始化相关字段。
//
// 返回值:
//   - `*Reg`: 返回一个新的 `Reg` 实例。
func newReg(ctx context.Context) *Reg {
	return &Reg{
		Init: xRegNode.NewRegNode(ctx),
	}
}

// Register 创建并初始化应用程序核心注册中心。
//
// 该函数执行环境加载、内置组件（日志器、雪花算法）及用户自定义节点的注册与初始化，
// 随后完成系统上下文和 Gin 引擎的构建。
//
// 参数 nodeList 包含需要按顺序注册的自定义组件节点列表。
//
// 返回完全初始化的 Reg 实例。内置节点注册失败会触发 panic。</think>// Register 创建并初始化应用程序的核心注册中心，返回完全就绪的 Reg 实例。
//
// 该函数依次执行环境加载、内置组件注册、用户组件初始化及 Gin 引擎构建。
// 参数 nodeList 指定需要追加注册的自定义组件列表。
// 返回包含初始化组件和引擎实例的 Reg 对象。
func Register(ctx context.Context, nodeList []xRegNode.RegNodeList) *Reg {
	reg := newReg(ctx)

	// 初始化配置器
	reg.configInit()
	reg.loggerInit()

	// 初始化节点注册
	reg.Init.Use(xConsts.SnowflakeNodeKey, xInit.SnowflakeInit) // 雪花算法初始化配置器
	for _, node := range nodeList {
		reg.Init.Use(node.Key, node.Node)
	}
	reg.Init.Exec()

	// 初始化系统上下文
	reg.engineInit()

	return reg
}
