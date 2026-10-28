package builtin_test

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/application/v2/builtin"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
)

func ExampleNewLogger() {
	// 配置受管日志选项，指定控制台丢弃、输出级别为 Info
	info, factory := builtin.NewLogger(builtin.LoggerConfig{
		Rotator: logging.RotatorConfig{
			Dir:      "./logs",
			BaseName: "app",
			Ext:      ".log",
		},
		Console: io.Discard,
		Level:   slog.LevelInfo,
	})

	// 打印插件元信息，验证其插件标识与导出的上下文键
	fmt.Println("Key:", info.Key)
	fmt.Println("Provides:", info.Provides[0])

	// 工厂函数可在应用中通过 app.RegisterTool(info, factory) 进行注册
	_ = factory

	// Output:
	// Key: plugin_logging
	// Provides: context_logger
}

func ExampleNewSnowflake() {
	// 配置受管雪花发号器组件，指定数据中心 ID 为 1，节点 ID 为 2
	info, factory := builtin.NewSnowflake(1, 2)

	// 打印插件元信息，验证其插件标识与导出的上下文键
	fmt.Println("Key:", info.Key)
	fmt.Println("Provides:", info.Provides[0])

	// 工厂函数可在应用中通过 app.RegisterTool(info, factory) 进行注册
	_ = factory

	// Output:
	// Key: plugin_snowflake
	// Provides: context_snowflake_node
}

func Example_registerBuiltinTools() {
	// 创建应用实例
	app := application.New()

	// 1. 构造内置日志组件并注册到 App
	logInfo, logFac := builtin.NewLogger(builtin.LoggerConfig{
		Console: io.Discard,
		Level:   slog.LevelInfo,
	})
	if err := app.RegisterTool(logInfo, logFac); err != nil {
		fmt.Println("Register logger error:", err)
		return
	}

	// 2. 构造内置雪花发号器组件（数据中心 ID 为 1，节点 ID 为 1）并注册到 App
	sfInfo, sfFac := builtin.NewSnowflake(1, 1)
	if err := app.RegisterTool(sfInfo, sfFac); err != nil {
		fmt.Println("Register snowflake error:", err)
		return
	}

	fmt.Println("Builtin tools registered successfully")

	// Output:
	// Builtin tools registered successfully
}
