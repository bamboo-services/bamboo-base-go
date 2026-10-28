// Package oracle 提供 Oracle 数据库驱动适配器实现与自动注册。
//
// 本包实现了 [dialect.Adapter] 接口，为数据库核心插件 [plugins/database] 提供
// Oracle GORM 方言构造能力。底层基于官方 [github.com/oracle-samples/gorm-oracle] v1.1.3
// 及 godror 驱动实现，编译期需要启用 CGO，运行期需要宿主环境安装配置 Oracle Instant Client。
//
// # 核心 API 与构造方式
//
// 本包提供两种适配器构造方式：
//
//   - 显式 DSN 构造：通过 [New] 传入显式连接字符串构造适配器。支持标准 URL 格式
//     （oracle:// 或 oracledb://）、godror logfmt 键值对格式以及标准 Easy Connect 格式。
//     若 DSN 为空或格式非法，错误将延迟推迟至 [dialect.Adapter.Open] 方法调用时返回，
//     且返回的错误信息已进行凭据脱敏处理。
//   - 环境变量构造：通过 [FromEnv] 从标准环境变量中解析连接参数。优先读取环境变量
//     DATABASE_DSN；若未设置，则自动读取 DATABASE_USER (默认 "system")、
//     DATABASE_PASS (默认 "")、DATABASE_HOST (默认 "localhost")、DATABASE_PORT (默认 "1521")、
//     DATABASE_SERVICE_NAME (默认 "ORCLPDB1")，以及可选的客户端驱动库路径
//     DATABASE_LIB_DIR（指定时追加 libDir="..."）组装为标准 godror logfmt 连接串。
//
// # 自动注册与匿名导入
//
// 本包在 init 函数中自动向全局方言注册表注册 [dialect.DriverOracle] 与 [FromEnv] 工厂。
// 业务应用可通过匿名导入本包，并在数据库插件中启用 FromEnv 模式实现驱动动态装载：
//
//	import _ "github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/oracle/v2"
//
// # 典型用法
//
// 显式指定 DSN 并注入数据库插件：
//
//	func setupApp() (*application.App, error) {
//		app := application.New()
//		adapter := oracle.New("oracle://scott:tiger@localhost:1521/ORCLPDB1")
//		dbInfo, dbFactory := database.New(database.WithAdapter(adapter))
//		if err := app.RegisterTool(dbInfo, dbFactory); err != nil {
//			return nil, err
//		}
//		return app, nil
//	}
//
// # 环境依赖与运行说明
//
// 本驱动底层基于 CGO 调用 Oracle 客户端 SDK：
//
//   - 编译要求：必须开启 CGO (CGO_ENABLED=1) 进行编译。
//   - 运行环境：运行环境必须安装对应操作系统的 Oracle Instant Client，
//     并正确配置动态库搜索路径（如 LD_LIBRARY_PATH、DYLD_LIBRARY_PATH 或系统 PATH），
//     或者通过 DATABASE_LIB_DIR 环境变量显式指定 Instant Client 的本地目录。
package oracle
