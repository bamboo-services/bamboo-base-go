// Package bind 提供基于 Gin 框架的类型安全 HTTP 请求参数绑定与结构体校验能力。
//
// 本包为 Web 服务提供声明式泛型绑定器 [Binding]，能够从 HTTP 请求体（JSON / XML）、
// URL 查询参数、路由路径参数、HTTP 请求头以及表单等多种来源反序列化数据，
// 并自动使用绑定的独立验证器引擎 [validator.Engine] 触发字段级校验。
//
// # 核心类型与调用路径
//
// 本包由 [Binder] 与泛型绑定器 [Binding] 协同工作：
//
//   - 绑定器实例：[Binder] 封装独立的参数验证引擎。在 Web 插件启动时通过 [New] 构造，
//     并通过 [Inject] 以私有 Context Key 挂载到请求上下文中；
//   - 绑定入口：在业务 Handler 中通过 [Bind] 传入目标结构体指针，
//     内部自动通过 [FromContext] 获取预注入的 [Binder]；若在非标准生命周期或测试场景，
//     可通过 [WithBinder] 显式指定 [Binder] 实例；
//   - 链式触发：通过 [Binding.Data]、[Binding.Query]、[Binding.URI]、
//     [Binding.Header] 或 [Binding.Form] 执行对应来源的数据提取、反序列化与字段校验。
//
// # 典型用法
//
// 下面的示例展示在业务 Handler 中绑定并校验 JSON 请求体：
//
//	type CreateUserRequest struct {
//		Username string `json:"username" binding:"required,min=4"`
//		Email    string `json:"email" binding:"required,email"`
//	}
//
//	func HandleCreateUser(c *gin.Context) {
//		var req CreateUserRequest
//		if bind.Bind(c, &req).Data() == nil {
//			// 校验失败或反序列化错误时，绑定器已自动写回标准错误响应并中断请求
//			return
//		}
//		// 校验通过，安全使用 req
//	}
//
// # 数据源与映射规则
//
// 不同的链式绑定方法对应不同的请求数据源与映射规则：
//
//   - [Binding.Data]：读取并反序列化 JSON 请求体，支持多次读取（内部重置 Request.Body）；
//   - [Binding.XML]：读取并反序列化 XML 请求体；
//   - [Binding.Query]：解析 URL Query 参数，优先遵循结构体的 form 标签，
//     支持标量类型、切片、指针及带有 time_format 标签的 [time.Time]；
//   - [Binding.URI]：解析 Gin 路由路径参数（[gin.Context.Params]），遵循 uri 标签；
//   - [Binding.Header]：解析 HTTP 请求标头，遵循 header 标签，支持 HTTP 标头大小写不敏感匹配；
//   - [Binding.Form]：解析 PostForm / MultipartForm 表单数据，遵循 form 标签。
//
// # 实例级校验与错误处理
//
// 本包贯彻严格的实例隔离原则，绝不使用全局验证器或向全局单例注册规则。
// 数据反序列化成功后，绑定器立即调用独立的 [validator.Engine.Validate] 进行实例级校验：
//
//   - 校验成功：返回传入的目标结构体指针，供调用方继续执行后续业务逻辑；
//   - 校验或解析失败：自动触发统一错误写回逻辑，向客户端返回标准 JSON 错误信封
//     （底层不泄漏 SQL 或内部敏感堆栈），并调用 [gin.Context.Abort] 中断后续 Handler 执行，
//     方法最终返回 nil。
//
// # 并发安全
//
// [Binder] 实例在完成初始化后属于只读状态，可被多个请求协程并发安全共享。
// 泛型结构体 [Binding] 为单个请求内部的轻量级链式临时对象，非并发安全，
// 不应在多个 goroutine 之间跨请求共享。
package bind
