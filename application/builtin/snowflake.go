package builtin

import (
	"context"
	"errors"
	"sync"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake"
)

// SnowflakePluginKey 是内置雪花算法受管插件在拓扑图中的全局唯一标识。
//
// 作为 [application.PluginInfo.Key] 传递给 [application.App.RegisterTool]，
// 用于在依赖拓扑排序与生命周期管理中唯一标识内置雪花算法发号器组件。
const SnowflakePluginKey ctxkey.Key = "plugin_snowflake"

type snowflakePlugin struct {
	datacenterID int64
	nodeID       int64

	mu      sync.Mutex
	started bool
	node    *snowflake.Node
}

// NewSnowflake 构造内置雪花算法受管插件的元信息与工厂闭包。
//
// 参数 datacenterID 与 nodeID 分别指定数据中心 ID 与工作节点 ID，
// 二者的合法取值范围均为 [0, 7]（各占 3 个二进制位）。
// 返回的 [application.PluginInfo] 声明组件为 [application.ModeTool] 工具模式，
// 插件标识为 [SnowflakePluginKey]，并对外提供 [ctxkey.SnowflakeNodeKey] 上下文服务。
//
// # 行为契约
//
//   - 纯构造语义：NewSnowflake 及其返回的工厂函数均为纯构造，不进行任何
//     节点初始化、底层参数校验或系统调用。
//   - 延迟校验与初始化：底层 [snowflake.NewNode] 的创建与参数范围检查
//     推迟至组件生命周期的 Start 阶段执行。若传入的 datacenterID 或 nodeID
//     超出范围或为负数，Start 阶段将返回明确错误并终止应用启动。
//   - 实例隔离：构造的雪花节点实例完全归属于当前 [application.App]，
//     不注册任何全局发号节点，支持多应用实例并行独立发号。
//   - 停止与注销：在组件生命周期的 Stop 阶段，将内部持有的节点引用置空，
//     不再向容器上下文发布发号器实例。
func NewSnowflake(datacenterID, nodeID int64) (application.PluginInfo, func() application.Plugin) {
	info := application.PluginInfo{
		Key:      SnowflakePluginKey,
		Name:     "plugin_snowflake",
		Mode:     application.ModeTool,
		Provides: []ctxkey.Key{ctxkey.SnowflakeNodeKey},
	}

	factory := func() application.Plugin {
		return &snowflakePlugin{
			datacenterID: datacenterID,
			nodeID:       nodeID,
		}
	}

	return info, factory
}

// Start 启动受管雪花算法组件，调用底层 NewNode 并对非法 ID 校验报错。
func (p *snowflakePlugin) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return errors.New("builtin snowflake: already started")
	}

	node, err := snowflake.NewNode(p.datacenterID, p.nodeID)
	if err != nil {
		return err
	}

	p.node = node
	p.started = true
	return nil
}

// Stop 停止雪花组件，清理实例引用并不再发布。
func (p *snowflakePlugin) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.node = nil
	return nil
}

// Expose 准确交付容器导出实例。
func (p *snowflakePlugin) Expose() map[ctxkey.Key]any {
	p.mu.Lock()
	defer p.mu.Unlock()

	out := make(map[ctxkey.Key]any)
	if p.node != nil {
		out[ctxkey.SnowflakeNodeKey] = p.node
	}
	return out
}
