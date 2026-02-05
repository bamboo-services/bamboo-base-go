package xRegNode

import (
	"context"
	"fmt"

	xConsts "github.com/bamboo-services/bamboo-base-go/context"
)

// Node 定义了组件初始化函数的签名。
//
// 该函数类型用于封装各类组件的初始化逻辑（如配置加载、日志器创建、数据库连接等），
// 接收上下文参数以便访问已初始化的依赖，返回初始化后的组件实例或错误。
//
// 参数:
//   - ctx: 包含已注册依赖的上下文，可通过 context.Value 获取其他组件实例
//
// 返回值:
//   - any: 初始化成功的组件实例，将被存储到上下文中供后续节点使用
//   - error: 初始化失败时返回的错误信息，非 nil 时会导致整个注册流程中断
type Node func(ctx context.Context) (any, error)

// RegNodeList 存储组件的上下文键及其初始化函数。
type RegNodeList struct {
	Key  xConsts.ContextKey
	Node Node
}

// RegNode 是应用程序组件注册和初始化的管理器。
type RegNode struct {
	list []RegNodeList
	Ctx  context.Context
}

// NewRegNode 创建并初始化 RegNode 实例。
//
// 该函数初始化用于注册组件函数的内部列表。
//
// 参数 ctx 必须非空，否则将使用 context.Background()。
//
// 返回初始化好的 RegNode 指针。
func NewRegNode(ctx context.Context) *RegNode {
	if ctx == nil {
		ctx = context.Background()
	}
	regNode := &RegNode{
		list: make([]RegNodeList, 0),
		Ctx:  ctx,
	}
	return regNode
}

// Use 注册一个组件初始化函数到执行队列中。
//
// 该方法将指定的初始化函数与上下文键关联，并添加到待执行列表中。
// 在调用 Exec() 时，这些函数会按注册顺序依次执行，每个函数的返回值
// 会通过对应的 ContextKey 存储到上下文中，供后续节点访问。
//
// 参数:
//   - ctxKey: 上下文键，用于在上下文中唯一标识该组件，不能为空或重复
//   - registerFunc: 组件初始化函数，接收当前上下文并返回组件实例或错误
//
// 注册顺序:
//   - 应按照组件的依赖关系顺序注册（被依赖的组件先注册）
//   - 例如：配置 -> 日志器 -> 数据库 -> 业务服务
//
// Panic 条件:
//   - 在 Exec() 执行后再次调用（list 已被清空）
//   - registerFunc 为 nil
//   - ctxKey 已被注册过（重复注册）
//
// 使用示例:
//
//	rn.Use(xConsts.ConfigKey, func(ctx context.Context) (any, error) {
//	    return loadConfig(), nil
//	})
//	rn.Use(xConsts.DatabaseKey, func(ctx context.Context) (any, error) {
//	    cfg := ctx.Value(xConsts.ConfigKey).(Config)
//	    return connectDB(cfg.DSN), nil
//	})
func (rn *RegNode) Use(ctxKey xConsts.ContextKey, registerFunc Node) {
	if rn.list == nil {
		panic("初始化外部禁止二次初始化")
	}
	if ctxKey != xConsts.Exec {
		if ctxKey.IsNil() {
			return
		}
		if registerFunc == nil {
			panic("registerFunc 不能为空")
		}
		for _, it := range rn.list {
			if it.Key == ctxKey {
				panic("重复注册 ContextKey: " + ctxKey.String())
			}
		}
	}
	rn.list = append(rn.list, RegNodeList{Key: ctxKey, Node: registerFunc})
}

// Exec 按注册顺序执行所有初始化节点，并将结果存入上下文。
//
// 该方法遍历通过 Use() 注册的所有初始化函数，按注册顺序依次执行。
// 每个函数执行成功后，其返回值会通过 context.WithValue 存储到 Ctx 中，
// 键为注册时指定的 ContextKey，值为函数返回的组件实例。
//
// 执行流程:
//  1. 遍历 list 中的所有节点
//  2. 调用节点的 Node 函数，传入当前 Ctx（包含已初始化的组件）
//  3. 将返回值存入 Ctx，更新上下文
//  4. 继续执行下一个节点
//  5. 所有节点执行完成后，清空 list 释放内存
//
// Panic 条件:
//   - 任何节点函数返回非 nil 错误时，会 panic 并输出节点索引、键名和错误信息
//
// 注意事项:
//   - 该方法只能调用一次，执行后 list 会被设置为 nil
//   - 执行后不能再调用 Use() 方法注册新节点
//   - 节点函数中可以通过 ctx.Value() 访问之前已初始化的组件
//
// 使用示例:
//
//	rn := NewRegNode()
//	rn.Use(xConsts.ConfigKey, loadConfigFunc)
//	rn.Use(xConsts.LoggerKey, initLoggerFunc)
//	rn.Exec() // 按顺序执行所有初始化函数
//	// 此时 rn.Ctx 包含所有已初始化的组件
func (rn *RegNode) Exec() {
	for i, node := range rn.list {
		val, err := node.Node(rn.Ctx)
		if !node.Key.IsExec() {
			if err != nil {
				panic(fmt.Sprintf("执行注册节点失败: index=%d Key=%v err=%v", i, node.Key, err))
			}
			rn.Ctx = context.WithValue(rn.Ctx, node.Key, val)
		}
	}
	// 释放空间
	rn.list = nil
}
