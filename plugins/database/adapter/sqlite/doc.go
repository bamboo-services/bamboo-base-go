// Package sqlite 提供纯 Go 实现的 SQLite 数据库驱动适配器与自动注册。
//
// 本包实现了 [dialect.Adapter] 接口，为数据库核心插件 [plugins/database] 提供
// SQLite GORM 方言 ([gorm.Dialector]) 构造能力。
// 底层基于纯 Go 驱动 [github.com/libtnb/sqlite] 实现，不依赖 CGO 或外部 C 库编译器，
// 支持在任意架构与交叉编译环境下直接运行。
//
// # 核心 API 与构造方式
//
// 本包提供两种适配器构造方式：
//
//   - 显式 DSN 构造：通过 [New] 传入显式 SQLite DSN 或数据库文件路径
//     （例如 "test.db" 或 ":memory:"）构造适配器。若 DSN 为空，
//     函数不会 panic，而是将错误延迟推迟至 [dialect.Adapter.Open] 方法调用时返回。
//   - 环境变量构造：通过 [FromEnv] 从标准环境变量中解析连接参数。
//     优先读取环境变量 DATABASE_DSN；若未配置或为空，则读取 DATABASE_PATH；
//     若两者均未设置，默认采用内存数据库 ":memory:"。
//     若显式设置的路径仅包含空白字符，将在 Open 时返回明确的配置错误。
//
// # 自动注册与匿名导入
//
// 本包在 init 函数中自动向全局方言注册表注册 [dialect.DriverSQLite] 与 [FromEnv] 工厂。
// 业务应用可通过匿名导入本包，并在数据库插件中启用 FromEnv 模式实现驱动动态装载：
//
//	import _ "github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/sqlite/v2"
//
// # 典型用法
//
// 显式指定本地文件路径或内存数据库并注入数据库插件：
//
//	func setupApp() (*application.App, error) {
//		app := application.New()
//		adapter := sqlite.New("data.db")
//		dbInfo, dbFactory := database.New(database.WithAdapter(adapter))
//		if err := app.RegisterTool(dbInfo, dbFactory); err != nil {
//			return nil, err
//		}
//		return app, nil
//	}
//
// # 纯 Go 特性与资源归属
//
// 本适配器完全摒弃了对 CGO 编译工具链与 SQLite 本地动态库的依赖：
//
//   - 纯 Go 驱动：支持一键交叉编译与容器轻量部署，杜绝 CGO 相关的跨平台兼容痛点。
//   - 生命周期与连接：[New] 与 [FromEnv] 纯在内存中保存路径，不执行文件打开或锁检查。
//     底层文件创建、锁机制与连接池均由数据库插件在 Start 阶段统一管理与释放。
package sqlite
