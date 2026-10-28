// Package logging 提供 Bamboo Base 框架的结构化日志系统。
//
// 核心设计原则：
// 1. 显式对象与实例隔离：基于显式 Logger 实例，不替换 slog.Default，不做进程级全局日志器，不同 App 间日志器相互隔离。
// 2. 命名日志器：通过 Logger.WithName(name, optionName...) 派生具有统一命名规范与可选业务标签的 LogNamedLogger。
// 3. 链路追踪解耦：标准 Handler 直接从 context.Context 中的 ctxkey.RequestKey 读取 TraceID，无需全局提取器接口。
// 4. 双写与标准兼容：保留 HandlerConfig 和 NewLogHandler，支持彩色控制台输出与 JSON 文件输出，保留并返回底层 Writer 写入错误。
package logging
