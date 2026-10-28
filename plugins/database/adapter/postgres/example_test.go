package postgres_test

import (
	"fmt"

	"github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/postgres/v2"
)

func ExampleNew() {
	// 使用显式 key=value 格式 DSN 构造 PostgreSQL 驱动适配器
	adapter := postgres.New("host=localhost user=postgres password=secret dbname=bamboo port=5432 sslmode=disable TimeZone=Asia/Shanghai")

	// 打印适配器名称
	fmt.Println("Driver:", adapter.Name())

	// Output:
	// Driver: postgres
}

func ExampleFromEnv() {
	// 从环境变量构造 PostgreSQL 驱动适配器（未配置环境变量时采用内置默认值）
	adapter := postgres.FromEnv()

	// 打印适配器名称
	fmt.Println("Driver:", adapter.Name())

	// Output:
	// Driver: postgres
}
