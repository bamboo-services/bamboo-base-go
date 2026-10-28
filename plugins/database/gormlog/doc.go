// Package gormlog 提供基于 [logging.Logger] 的 GORM 日志适配器。
//
// 本包实现了 GORM 官方的 [gormlogger.Interface] 接口，将 GORM 内部的 SQL 追踪、
// 慢查询告警与错误输出无缝接入框架的受管日志体系 [logging.Logger]，
// 并统一将日志归类至 [logging.NamedREPO] 命名空间。
//
// # 核心类型与配置
//
// 本包的核心类型包括：
//
//   - [Logger]：实现了 [gormlogger.Interface] 接口的适配器结构体。
//   - [Config]：日志行为配置，包含慢查询阈值 [Config.SlowThreshold]、
//     日志级别 [Config.LogLevel] 以及是否忽略记录未找到错误的
//     [Config.IgnoreRecordNotFoundError]。
//   - [LogLevel]：日志输出级别枚举（[LevelSilent]、[LevelError]、[LevelWarn]、[LevelInfo]）。
//
// # 实例构造与默认行为
//
// 通过 [New] 构造日志适配器时遵循以下约定：
//
//   - 显式依赖：必须显式传入有效的 [*logging.Logger]；若传入为 nil，
//     函数返回 [gormlogger.Discard]，绝不隐式暗造日志器，也不依赖全局 [slog.Default]。
//   - 默认阈值：若 [Config.SlowThreshold] <= 0，默认采用 200ms 作为慢查询阈值。
//   - 默认级别：若 [Config.LogLevel] 为零值，默认采用 [LevelWarn] 级别，
//     仅在发生错误或慢查询时输出日志。
//
// # SQL 追踪与日志结构化
//
// 在 [Logger.Trace] 方法中，执行详情以结构化属性（[slog.Attr]）输出：
//
//   - "elapsed_ms"：执行耗时（毫秒，float64）。
//   - "rows"：影响或返回的行数（int64）。
//   - "sql"：实际执行的格式化 SQL 语句。
//   - "error"：执行失败时的错误信息。
//   - "threshold"：慢查询告警时附带的阈值。
//
// 当错误为 [gorm.ErrRecordNotFound] 且启用了 [Config.IgnoreRecordNotFoundError] 时，
// 错误将被静默忽略，不记录 ERROR 日志。
//
// # 并发安全
//
// [Logger] 内部持有的 [*logging.Logger] 为并发安全实例，配置字段在构造后不可变。
// [Logger.LogMode] 方法返回浅拷贝后的新实例，多个 goroutine 可并发调用各级别日志记录方法。
package gormlog
