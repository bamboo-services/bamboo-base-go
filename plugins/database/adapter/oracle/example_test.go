package oracle_test

import (
	"fmt"

	"github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/oracle/v2"
)

func ExampleNew() {
	// 使用显式 URL 格式 DSN 构造 Oracle 驱动适配器
	adapter := oracle.New("oracle://scott:tiger@localhost:1521/ORCLPDB1")

	// 打印适配器名称
	fmt.Println("Driver:", adapter.Name())

	// Output:
	// Driver: oracle
}

func ExampleFromEnv() {
	// 从环境变量构造 Oracle 驱动适配器（未配置环境变量时采用内置默认值）
	adapter := oracle.FromEnv()

	// 打印适配器名称
	fmt.Println("Driver:", adapter.Name())

	// Output:
	// Driver: oracle
}
