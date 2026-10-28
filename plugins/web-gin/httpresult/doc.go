// Package httpresult 提供符合框架统一协议规范的 Web HTTP 响应构造与写回函数。
//
// 本包为 Gin 路由处理函数提供标准化响应出口，将成功结果或业务错误模型 [apperr.Error]
// 统一转换为协议响应信封 [envelope.Response]，计算安全映射的 HTTP 状态码，
// 并自动注入请求追踪标识与微秒级处理耗时。
//
// # 响应信封规范与数据结构
//
// 所有由本包输出的响应均包装为统一的 JSON 结构 [envelope.Response]：
//
//   - context：服务端请求唯一追踪标识，与响应标头 header.HeaderRequestUUID 严格一致；
//   - code：业务状态码。成功响应时为 200，错误响应时对应业务错误码 [apperr.Code]；
//   - output：状态标识短语，默认成功标识为 [DefaultSuccessOutput]（"Success"）；
//   - message：框架或错误码定义的标准提示信息；
//   - error_message：业务错误发生时显式指定的安全错误提示（无敏感信息）；
//   - data：成功时返回的业务载荷，或参数验证失败时的字段错误详情列表；
//   - overhead：自请求进入框架直至响应构造所耗费的微秒整数（基于请求上下文中的起点计算）。
//
// # 典型用法
//
// 下面的示例展示在 Gin 处理函数中返回成功数据或业务错误：
//
//	func UserDetailHandler(c *gin.Context) {
//		user, err := userService.Find(c.Param("id"))
//		if err != nil {
//			// 写回业务错误响应并计算对应状态码
//			httpresult.Error(c, apperr.New(apperr.RecordNotFound, "用户不存在", err))
//			return
//		}
//		// 写回 200 成功响应并附带数据
//		httpresult.SuccessWithData(c, "获取用户成功", user)
//	}
//
// # 状态码计算与安全映射规则
//
// 对于错误响应，HTTP 状态码严格由错误码 [apperr.Code] 计算得出：
//
//   - 映射算法：状态码取业务码的前三位整数（code / 100）；
//   - 允许区间：业务错误路径仅接纳 400 至 599 范围内的状态码；
//   - 越界回退：若计算结果超出该区间（例如 <400 或 >599，包括 1xx、2xx、3xx、负数或 6xx 等），
//     一律安全回退为 500 [http.StatusInternalServerError]；
//   - nil 容错：若传入的 [*apperr.Error] 为 nil，自动以 [apperr.ServerInternalError]
//     构造 500 内部异常响应；
//   - 敏感信息隔离：底层根因错误（[apperr.Error.Err]）标记有 json:"-"，
//     仅保留在内存中供日志记录与 Unwrap 诊断，绝不序列化泄漏至客户端 HTTP 响应中。
//
// # 响应拦截与防重保护
//
// 本包各响应函数均内置重复写入防护检查：
//
//   - 防重复写入：在写入前校验 [gin.ResponseWriter.Written]，若当前连接已写入响应则直接返回；
//   - 中断执行：[AbortError] 会在写出错误信封的同时调用 [gin.Context.Abort]，
//     立即终止 Gin 中间件调用链向后传递。
//
// # 并发安全
//
// 本包内仅包含纯函数与只读常量，无任何包级可变状态。
// 同一个 [*gin.Context] 的生命周期应严格受限于其处理协程，不可在多个 goroutine 间无同步读写。
package httpresult
