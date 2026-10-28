package sqlite_test

import (
	"fmt"

	"github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/sqlite/v2"
)

func ExampleNew() {
	// 使用显式内存数据库 DSN 构造 SQLite 驱动适配器
	adapter := sqlite.New(":memory:")

	// 打印适配器名称
	fmt.Println("Driver:", adapter.Name())

	// Output:
	// Driver: sqlite
}

func ExampleFromEnv() {
	// 从环境变量构造 SQLite 驱动适配器（未配置环境变量时默认采用 :memory:）
	adapter := sqlite.FromEnv()

	// 打印适配器名称
	fmt.Println("Driver:", adapter.Name())

	// Output:
	// Driver: sqlite
}
