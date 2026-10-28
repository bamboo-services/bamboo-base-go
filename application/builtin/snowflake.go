package builtin

import (
	"context"
	"errors"
	"sync"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake"
)

// SnowflakePluginKey 声明雪花算法插件全局唯一标识。
const SnowflakePluginKey ctxkey.Key = "plugin_snowflake"

type snowflakePlugin struct {
	datacenterID int64
	nodeID       int64

	mu      sync.Mutex
	started bool
	node    *snowflake.Node
}

// NewSnowflake 构造雪花算法插件元信息与工厂函数。
// factory 纯构造，不进行任何 IO 或参数校验。
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
