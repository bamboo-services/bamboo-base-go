package dialect_test

import (
	"errors"
	"fmt"

	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
)

func ExampleDriver() {
	// 查看框架支持的预置数据库驱动名称常量
	fmt.Println(dialect.DriverMySQL)
	fmt.Println(dialect.DriverPostgres)
	fmt.Println(dialect.DriverSQLite)
	fmt.Println(dialect.DriverOracle)
	fmt.Println(dialect.DriverSQLServer)

	// Output:
	// mysql
	// postgres
	// sqlite
	// oracle
	// sqlserver
}

func ExampleLookup() {
	// 查找未注册的驱动时返回包含 ErrDriverNotLoaded 的错误
	_, err := dialect.Lookup("unregistered_db")
	if errors.Is(err, dialect.ErrDriverNotLoaded) {
		fmt.Println("driver not loaded")
	}

	// Output:
	// driver not loaded
}
