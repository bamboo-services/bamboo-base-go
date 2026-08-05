// Package xDatabase 提供数据库驱动枚举与 GORM 驱动工厂注册表。
//
// 该包位于 common 层，承载「驱动枚举 → 构造 gorm.Dialector 的工厂」的全局注册表，
// 供驱动插件（plugins/database/*）在 init() 中注册自身驱动，使框架 core（major）
// 无需依赖任何具体数据库驱动。驱动枚举常量同时被 major/option/database 引用并 re-export。
package xDatabase

import (
	"fmt"
	"sync"

	"gorm.io/gorm"
)

// Driver 数据库驱动类型，标识业务端选择接入的 GORM 驱动实现。
//
// 零值空串等价于 [DriverNone]，表示不启用数据库。
type Driver string

const (
	// DriverNone 表示不启用数据库。Register 会跳过数据库节点装配。
	DriverNone Driver = "none"

	// DriverMySQL 表示使用 MySQL 驱动（github.com/go-sql-driver/mysql）。
	DriverMySQL Driver = "mysql"

	// DriverPostgres 表示使用 PostgreSQL 驱动（github.com/jackc/pgx/v5）。
	DriverPostgres Driver = "postgres"

	// DriverSQLite 表示使用 SQLite 驱动（github.com/libtnb/sqlite，纯 Go 实现）。
	DriverSQLite Driver = "sqlite"

	// DriverOracle 表示使用 Oracle 驱动（github.com/oracle-samples/gorm-oracle，底层 godror）。
	DriverOracle Driver = "oracle"

	// DriverSQLServer 表示使用 SQL Server 驱动（gorm.io/driver/sqlserver，底层 microsoft/go-mssqldb）。
	DriverSQLServer Driver = "sqlserver"
)

// DialectorFactory 构造 gorm.Dialector 的工厂函数签名。
//
// 由驱动插件在 init() 中注册，dsn 为业务端提供的数据库连接串。
type DialectorFactory func(dsn string) gorm.Dialector

// dialectFactories 全局驱动工厂注册表。
//
// key 为 [Driver] 枚举，value 为对应驱动的 Dialector 工厂。
// 由驱动插件（plugins/database/*）在包级 init() 中注册，业务端通过 import 对应插件触发注册。
// 读写受 RWMutex 保护：虽然注册通常发生在 init()，但注册表是导出的，
// 允许运行时动态注册，因此对并发读写做防护。
var (
	dialectFactories   = map[Driver]DialectorFactory{}
	dialectFactoriesMu sync.RWMutex
)

// RegisterDialector 注册「驱动枚举 → 构造 Dialector 的工厂」。
//
// 通常由驱动插件（如 plugins/database/mysql）在包级 init() 中调用，业务端无需直接调用，
// 仅需 import 对应插件即可触发注册。
//
// 注册空驱动、none 驱动、nil 工厂或重复注册同一枚举会 panic（与 database/sql.Register
// 语义一致），便于在启动早期暴露驱动拼写错误或重复 import 的问题。
func RegisterDialector(d Driver, factory DialectorFactory) {
	if d == "" || d == DriverNone {
		panic(fmt.Sprintf("bamboo: 禁止注册空驱动或 none 驱动, got %q", d))
	}
	if factory == nil {
		panic(fmt.Sprintf("bamboo: 数据库驱动 %q 注册了 nil 工厂", d))
	}
	dialectFactoriesMu.Lock()
	defer dialectFactoriesMu.Unlock()
	if _, dup := dialectFactories[d]; dup {
		panic(fmt.Sprintf("bamboo: 数据库驱动 %q 重复注册", d))
	}
	dialectFactories[d] = factory
}

// GetDialectorFactory 返回驱动枚举对应的 Dialector 工厂。
//
// 未注册（业务端未 import 对应驱动插件）时 ok=false，调用方应给出可操作的错误提示。
func GetDialectorFactory(d Driver) (DialectorFactory, bool) {
	dialectFactoriesMu.RLock()
	defer dialectFactoriesMu.RUnlock()
	factory, ok := dialectFactories[d]
	return factory, ok
}
