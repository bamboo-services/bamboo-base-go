package sqlserver_test

import (
	"fmt"

	"github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/sqlserver/v2"
)

func ExampleNew() {
	// 使用显式 URL 格式 DSN 构造 SQL Server 驱动适配器
	adapter := sqlserver.New("sqlserver://sa:secret@localhost:1433?database=master")

	// 打印适配器名称
	fmt.Println("Driver:", adapter.Name())

	// Output:
	// Driver: sqlserver
}

func ExampleFromEnv() {
	// 从环境变量构造 SQL Server 驱动适配器（未配置环境变量时采用内置默认值）
	adapter := sqlserver.FromEnv()

	// 打印适配器名称
	fmt.Println("Driver:", adapter.Name())

	// Output:
	// Driver: sqlserver
}
