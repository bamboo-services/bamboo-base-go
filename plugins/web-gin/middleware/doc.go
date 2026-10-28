// Package middleware 提供 Gin HTTP 服务的核心管道中间件。
//
// 本包为 Web 请求管道提供标准化处理屏障，涵盖全局异常捕获与安全恢复、
// 响应未写出时的自动兜底机制，以及通用 CORS 跨域请求控制。
//
// # 核心中间件与装配流水线
//
// Web 插件内部默认按以下顺序装配核心中间件，确保请求被完整保护：
//
//   - 异常恢复 [Recovery]：置于最外层，负责捕获下游所有处理逻辑抛出的 panic，
//     将其转化为标准内部错误并安全中断，防止服务进程崩溃与堆栈外泄；
//   - 响应兜底 [Response]：监控下游执行结果，当下游未写入响应且存在内部错误或
//     既无响应也无错误（非 301/302 重定向）时，自动组装标准错误响应兜底写出；
//   - 跨域支持 [CORS]：允许常见跨域 HTTP 方法与请求标头，自动响应预检 OPTIONS 请求；
//   - 请求桥接：挂载由宿主插件提供的上下文桥接中间件，注入追踪标识与生命周期上下文。
//
// # 典型用法
//
// 下面的示例展示在私有 Gin 引擎上挂载标准基础中间件链：
//
//	engine := gin.New()
//	engine.Use(
//		middleware.Recovery(logger),
//		middleware.Response(),
//		middleware.CORS(),
//	)
//
// # 异常恢复与敏感堆栈隔离
//
// [Recovery] 中间件执行严谨的异常处理与敏感信息防护：
//
//   - 日志记录：若传入了非 nil 的 [logging.LogNamedLogger]，异常信息将记录至应用日志；
//   - 响应写回：通过 [httpresult.AbortError] 向客户端返回 500
//     [apperr.ServerInternalError]，提示信息统一为“系统内部异常”；
//   - 安全隔离：原始 panic 对象仅作为内部 Cause 保留，绝不将其堆栈或敏感字段暴露至 HTTP 响应。
//
// # 响应兜底与开发者纠错
//
// [Response] 中间件采用后置检查机制（在 c.Next() 执行完毕后检查）：
//
//   - 未写入响应且存在 [gin.Context.Errors]：读取最后一个错误，若是 [*apperr.Error]
//     则按原错误码输出，否则包装为 500 异常写出；
//   - 未写入响应且无任何错误：若响应状态码非 301 [http.StatusMovedPermanently]
//     或 302 [http.StatusFound]，判定为业务处理逻辑遗漏了响应写出动作，
//     自动向客户端输出带有 [apperr.DeveloperError] 错误码的提示。
//
// # 并发安全
//
// 本包提供的中间件构造函数返回无内部可变状态的 [gin.HandlerFunc]，
// 可以在多 goroutine 并发处理 HTTP 请求的环境下安全复用。
package middleware
