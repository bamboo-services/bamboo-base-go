// Package database 提供基于 GORM 的关系型数据库核心受管插件与上下文提取原语。
//
// 本包为应用提供数据库连接全生命周期管理、连接池优化配置、多方言适配器接入、
// 结构化日志追踪以及类型安全的上下文提取能力。数据库插件以 [application.ModeTool]
// 工具模式注册至 [application.App]，向容器交付 [*gorm.DB] 句柄，导出键为 [ctxkey.DatabaseKey]。
//
// # 核心 API 与装配入口
//
// 数据库插件的典型装配与使用路径如下：
//
//   - 创建插件：通过 [New] 传入配置选项（[Option]）构造 [application.PluginInfo] 与工厂函数。
//     该过程为纯构造，不产生任何网络 I/O、不解析环境变量且不建立实际数据库连接。
//   - 适配器配置：通过 [WithAdapter] 显式接入指定数据库驱动适配器
//     （如 MySQL、PostgreSQL、SQLite、Oracle、SQL Server），
//     或通过 [FromEnv] 声明在启动期从环境变量 DATABASE_DRIVER 动态查找已加载的驱动。
//   - 连接池配置：通过 [WithMaxOpenConns]、[WithMaxIdleConns]、[WithConnMaxLifetime]
//     及 [WithConnMaxIdleTime] 精细控制底层 [sql.DB] 的连接池行为。
//   - 日志与方言定制：支持通过 [WithLogger] 与 [WithLogConfig] 接入受管日志追踪，
//     或通过 [WithGormConfig] 注入自定义的 [gorm.Config]。
//
// # 上下文提取与请求绑定
//
// 下游业务服务、Worker 或中间件在处理具体数据操作时，推荐通过本包提取数据库句柄：
//
//   - [Get]：从传入的 [context.Context] 中安全提取 [*gorm.DB] 句柄。
//     若提取成功且 ctx 非空，方法内部自动执行 db.WithContext(ctx)，
//     使后续的 SQL 查询具备上下文取消与超时感知能力；若提取失败或句柄为 nil，
//     返回包含 [apperr.DatabaseError] 的业务级错误。
//   - [MustGet]：从上下文中提取已绑定请求上下文的 [*gorm.DB]，提取失败时直接触发 panic，
//     适合在确定数据库插件已成功启动且强依赖数据库的场景下使用。
//
// # 典型用法
//
// 以下代码演示如何在应用中装配数据库插件并从上下文中提取使用：
//
//	func setupDatabase(app *application.App, adapter dialect.Adapter) error {
//		// 1. 构造数据库插件信息与工厂
//		dbInfo, dbFactory := database.New(
//			database.WithAdapter(adapter),
//			database.WithMaxOpenConns(100),
//			database.WithMaxIdleConns(10),
//			database.WithConnMaxLifetime(time.Hour),
//		)
//
//		// 2. 注册为受管工具组件
//		return app.RegisterTool(dbInfo, dbFactory)
//	}
//
// 业务请求处理函数中提取并使用：
//
//	func HandleQuery(ctx context.Context) error {
//		db, err := database.Get(ctx)
//		if err != nil {
//			return err
//		}
//		// 后续查询自动绑定 ctx 超时与链路追踪
//		return db.Find(&users).Error
//	}
//
// # 生命周期与资源归属
//
// 数据库插件的生命周期受 [application.App] 统筹管控：
//
//   - 启动阶段（Start）：在应用启动拓扑中被调用。插件根据配置获取 [dialect.Adapter]，
//     调用其 Open 方法构造方言实例并初始化 [gorm.DB]。随后获取底层 [*sql.DB]，
//     设置最大打开连接数、空闲数与生命周期，并执行 PingContext 真实验证数据库连通性。
//     若连通性检查失败，Start 会在退出前主动调用 Close 关闭已创建的底层连接并返回错误，
//     防止在应用启动失败时残留无效连接。
//   - 停止阶段（Stop）：在应用逆拓扑平滑关闭时被调用。插件安全调用底层 [*sql.DB] 的
//     Close 方法释放物理连接，该操作具备互斥保护与幂等性。
//
// # 并发与使用边界
//
// [Plugin] 实例内部通过互斥锁 [sync.Mutex] 保护内部状态。
// [Get] 返回的 [*gorm.DB] 实例由于通过 db.WithContext(ctx) 创建了克隆会话，
// 各 goroutine 或事务可并发独立使用返回的会话句柄。
// 禁止直接使用裸 ctx.Value 断言数据库连接，必须统一使用 [Get] 或 [MustGet]。
package database
