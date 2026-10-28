// Package logging 提供 Bamboo Base 框架的结构化日志系统，
// 覆盖日志器装配、命名派生、控制台与文件双写输出、文件轮转与
// 按天归档能力。
//
// 本包的核心设计原则：
//
//   - 显式对象与实例隔离：基于显式 [Logger] 实例，不替换
//     slog.Default，不做进程级全局日志器，不同 App 之间日志器
//     相互隔离；
//   - 命名日志器：通过 [Logger.WithName] 派生具有统一命名规范与
//     可选业务标签的 [LogNamedLogger]；
//   - 链路追踪解耦：Handler 直接从 context 的 [ctxkey.RequestKey]
//     读取 TraceID，无需全局提取器接口；
//   - 双写与错误可见：[NewLogHandler] 支持彩色控制台输出与 JSON
//     文件输出，保留并返回底层 Writer 的写入错误。
//
// # 核心类型与调用路径
//
// [New] 从任意 slog.Handler 构造 [Logger]，[Logger.WithName] 派生
// [LogNamedLogger] 并执行各级别日志输出。[NewLogHandler] 构造本包的
// 双写 Handler，[HandlerConfig] 决定控制台与文件输出目标及最低级别。
// 文件侧通常接入 [NewRotatingWriter] 返回的 [RotatingWriter]，
// 获得按大小与日期切割、按天归档的能力。
//
// # 典型用法
//
// 以下函数装配一个控制台与轮转文件双写的日志器：
//
//	func setupLogger() (*logging.Logger, *logging.RotatingWriter, error) {
//		writer, err := logging.NewRotatingWriter(logging.RotatorConfig{
//			Dir:      ".logs",
//			BaseName: "app",
//			Ext:      ".log",
//			MaxSize:  10 * 1024 * 1024,
//		})
//		if err != nil {
//			return nil, nil, err
//		}
//		handler := logging.NewLogHandler(logging.HandlerConfig{
//			Console: os.Stdout,
//			File:    writer,
//			Level:   slog.LevelInfo,
//		})
//		logger, err := logging.New(handler)
//		if err != nil {
//			_ = writer.Close()
//			return nil, nil, err
//		}
//		return logger, writer, nil
//	}
//
// 使用结束后调用 [RotatingWriter.Close] 释放文件句柄与后台调度器。
//
// # 命名规范
//
// Named* 常量定义统一的项目日志命名（如 [NamedCORE]、[NamedHTTP]），
// 名称固定 4 个字符，控制台输出按职责类别着色；推荐作为
// [Logger.WithName] 的名称入参。
//
// # 双写输出
//
// [LogHandler] 的控制台输出为彩色文本，文件输出为单行 JSON，
// 两路输出均携带从 [ctxkey.RequestKey] 提取的 TraceID。级别低于
// [HandlerConfig.Level] 的记录被过滤；ERROR 级别的控制台输出附加
// 当前 goroutine 堆栈。任一路底层写入失败时，错误以 errors.Join
// 汇总返回，不静默吞错。
//
// # 文件轮转与归档
//
// [NewRotatingWriter] 按 [RotatorConfig] 创建轮转写入器：文件超过
// MaxSize 时按索引切割，跨天时当日文件按日期重命名保留，历史文件
// 每日归档为 logger-<日期>.tar.gz（前缀固定为 "logger"）并删除
// 原文件。[RotatingWriter.Close] 幂等，等待全部在途归档完成后
// 返回汇总错误。
//
// # 并发与资源
//
// [RotatingWriter] 与 [LogHandler] 均可被多个 goroutine 并发使用；
// [LogHandler] 经 [LogHandler.WithAttrs] 与 [LogHandler.WithGroup]
// 派生的 Handler 与原 Handler 共享互斥锁与底层 Writer。
// [Logger] 与 [LogNamedLogger] 自身无状态，并发安全由底层 Handler
// 决定。资源归属：[RotatingWriter] 持有文件句柄与后台调度协程，
// 须由调用方通过 Close 释放；[HandlerConfig] 中的 Writer 由调用方
// 管理生命周期。
package logging
