//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/application/v2/builtin"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake"
)

// ==============================
// 业务配置工具
// ==============================

// ConfigGreetingKey 是配置工具向共享容器暴露问候语资源的上下文键。
const ConfigGreetingKey ctxkey.Key = "resource_config_greeting"

// ConfigToolKey 是配置工具插件在 App 拓扑中的唯一插件标识。
const ConfigToolKey ctxkey.Key = "plugin_tool_config"

// WorkerKey 是工作者插件在 App 拓扑中的唯一插件标识。
const WorkerKey ctxkey.Key = "plugin_worker_printer"

// ConfigTool 提供只读配置资源，实现 application.Plugin 与 ctxkey.Exposer 契约。
type ConfigTool struct {
	greeting string // 预设的问候语文本
}

// Start 验证或初始化工具内部资源。
func (t *ConfigTool) Start(_ context.Context) error {
	return nil
}

// Stop 释放工具持有的资源。
func (t *ConfigTool) Stop(_ context.Context) error {
	return nil
}

// Expose 向共享容器原子发布 Provides 声明的资源键集合。
func (t *ConfigTool) Expose() map[ctxkey.Key]any {
	return map[ctxkey.Key]any{
		ConfigGreetingKey: t.greeting,
	}
}

// NewConfigTool 返回工具元信息与无 I/O 的纯构造工厂函数。
func NewConfigTool(greeting string) (application.PluginInfo, func() application.Plugin) {
	info := application.PluginInfo{
		Key:      ConfigToolKey,
		Name:     "config-tool",
		Provides: []ctxkey.Key{ConfigGreetingKey},
	}
	factory := func() application.Plugin {
		return &ConfigTool{greeting: greeting}
	}
	return info, factory
}

// ==============================
// 业务工作者
// ==============================

// PrinterWorker 读取依赖并常驻运行，实现 application.RunnablePlugin 契约。
type PrinterWorker struct {
	message string          // 从配置工具提取的问候语
	logger  *logging.Logger // 从内置日志组件提取的受管日志器
	node    *snowflake.Node // 从内置雪花组件提取的发号节点
}

// Start 在受限启动上下文中提取 Requires 声明的资源键。
func (w *PrinterWorker) Start(ctx context.Context) error {
	msg, err := application.Get[string](ctx, ConfigGreetingKey)
	if err != nil {
		return fmt.Errorf("读取配置资源失败: %w", err)
	}
	w.message = msg

	logger, err := application.Get[*logging.Logger](ctx, builtin.LoggerKey)
	if err != nil {
		return fmt.Errorf("读取内置日志器失败: %w", err)
	}
	w.logger = logger

	node, err := application.Get[*snowflake.Node](ctx, ctxkey.SnowflakeNodeKey)
	if err != nil {
		return fmt.Errorf("读取内置雪花发号节点失败: %w", err)
	}
	w.node = node
	return nil
}

// Run 执行常驻工作并在 Context 取消时正常退出，避免因早退触发看门狗停机。
func (w *PrinterWorker) Run(ctx context.Context) error {
	id, genErr := w.node.Generate()
	if genErr != nil {
		return fmt.Errorf("生成雪花 ID 失败: %w", genErr)
	}

	w.logger.WithName("WORKER").Info(
		ctx,
		"worker started successfully",
		slog.Int64("snowflake_id", int64(id)),
		slog.String("greeting", w.message),
	)

	<-ctx.Done()
	return ctx.Err()
}

// Stop 完成工作者的清理工作。
func (w *PrinterWorker) Stop(_ context.Context) error {
	return nil
}

// NewPrinterWorker 返回工作者元信息与纯构造工厂函数。
func NewPrinterWorker() (application.PluginInfo, func() application.RunnablePlugin) {
	info := application.PluginInfo{
		Key:  WorkerKey,
		Name: "printer-worker",
		Requires: []ctxkey.Key{
			ConfigGreetingKey,
			builtin.LoggerKey,
			ctxkey.SnowflakeNodeKey,
		},
	}
	factory := func() application.RunnablePlugin {
		return &PrinterWorker{}
	}
	return info, factory
}

// ==============================
// 入口函数
// ==============================

func main() {
	app := application.New()

	// 1. 注册受管内置日志组件
	logInfo, logFactory := builtin.NewLogger(builtin.LoggerConfig{
		Level: slog.LevelInfo,
	})
	if err := app.RegisterTool(logInfo, logFactory); err != nil {
		log.Fatalf("注册 Builtin Logger 失败: %v", err)
	}

	// 2. 注册受管内置雪花算法发号器组件
	sfInfo, sfFactory := builtin.NewSnowflake(1, 1)
	if err := app.RegisterTool(sfInfo, sfFactory); err != nil {
		log.Fatalf("注册 Builtin Snowflake 失败: %v", err)
	}

	// 3. 注册自定义业务配置工具
	toolInfo, toolFactory := NewConfigTool("Hello, Bamboo Base v2!")
	if err := app.RegisterTool(toolInfo, toolFactory); err != nil {
		log.Fatalf("注册 ConfigTool 失败: %v", err)
	}

	// 4. 注册业务工作者
	workerInfo, workerFactory := NewPrinterWorker()
	if err := app.RegisterWorker(workerInfo, workerFactory); err != nil {
		log.Fatalf("注册 PrinterWorker 失败: %v", err)
	}

	// 5. 启动应用并进入看门狗监督运行
	if err := app.Run(); err != nil {
		log.Fatalf("应用运行异常退出: %v", err)
	}
}
