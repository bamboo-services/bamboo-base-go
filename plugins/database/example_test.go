package database_test

import (
	"fmt"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2"
)

func ExampleNew() {
	// 构造数据库受管插件信息与工厂函数，配置连接池参数
	info, factory := database.New(
		database.WithMaxOpenConns(50),
		database.WithMaxIdleConns(10),
		database.WithConnMaxLifetime(30*time.Minute),
	)

	// 打印插件元信息，验证其插件标识与导出的上下文键
	fmt.Println("Key:", info.Key)
	fmt.Println("IsTool:", info.Mode == application.ModeTool)
	fmt.Println("Provides:", info.Provides[0])

	// 工厂函数可在应用中通过 app.RegisterTool(info, factory) 进行注册
	_ = factory

	// Output:
	// Key: plugin_database
	// IsTool: true
	// Provides: context_database
}
