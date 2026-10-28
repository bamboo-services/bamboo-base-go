// Package sqlserver 提供 SQL Server 数据库驱动适配器实现与自动注册。
//
// 本包实现了 [dialect.Adapter] 接口，为数据库核心插件 [plugins/database] 提供
// SQL Server GORM 方言 ([gorm.Dialector]) 构造能力。
//
// # 核心 API 与构造方式
//
// 本包提供两种适配器构造方式：
//
//   - 显式 DSN 构造：通过 [New] 传入显式 URL 格式 DSN 字符串构造适配器。
//     要求协议头为 sqlserver:// 且包含有效的主机地址。若 DSN 为空或格式非法，
//     函数不会 panic，而是将错误延迟推迟至 [dialect.Adapter.Open] 方法调用时返回，
//     且返回的错误信息已进行凭据脱敏处理。
//   - 环境变量构造：通过 [FromEnv] 从标准环境变量中解析连接参数。优先读取环境变量
//     DATABASE_DSN；若未设置，则自动组合 DATABASE_USER (默认 "sa")、
//     DATABASE_PASS (默认 "")、DATABASE_HOST (默认 "localhost")、DATABASE_PORT (默认 "1433")
//     以及 DATABASE_NAME (默认 "master")，通过标准库 [net.JoinHostPort] 与
//     [net/url.UserPassword] 规范编码构造标准的 sqlserver:// URL 连接串。
//
// # 自动注册与匿名导入
//
// 本包在 init 函数中自动向全局方言注册表注册 [dialect.DriverSQLServer] 与 [FromEnv] 工厂。
// 业务应用可通过匿名导入本包，并在数据库插件中启用 FromEnv 模式实现驱动动态装载：
//
//	import _ "github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/sqlserver/v2"
//
// # 典型用法
//
// 显式指定 DSN 并注入数据库插件：
//
//	func setupApp() (*application.App, error) {
//		app := application.New()
//		adapter := sqlserver.New("sqlserver://sa:secret@localhost:1433?database=bamboo")
//		dbInfo, dbFactory := database.New(database.WithAdapter(adapter))
//		if err := app.RegisterTool(dbInfo, dbFactory); err != nil {
//			return nil, err
//		}
//		return app, nil
//	}
//
// # 生命周期与资源归属
//
// [New] 与 [FromEnv] 均为纯内存操作，不触发网络握手或数据库探测。
// [dialect.Adapter.Open] 方法仅构造 [gorm.Dialector] 实例，物理网络连接与连接池由
// 数据库插件在 Start 阶段统一管理与释放。
package sqlserver
