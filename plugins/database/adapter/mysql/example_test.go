package mysql_test

import (
	"fmt"

	"github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/mysql/v2"
)

func ExampleNew() {
	// 使用显式 DSN 构造 MySQL 驱动适配器
	adapter := mysql.New("root:secret@tcp(localhost:3306)/bamboo?charset=utf8mb4&parseTime=True&loc=Local")

	// 打印适配器名称
	fmt.Println("Driver:", adapter.Name())

	// Output:
	// Driver: mysql
}

func ExampleFromEnv() {
	// 从环境变量构造 MySQL 驱动适配器（未配置环境变量时采用内置默认值）
	adapter := mysql.FromEnv()

	// 打印适配器名称
	fmt.Println("Driver:", adapter.Name())

	// Output:
	// Driver: mysql
}
