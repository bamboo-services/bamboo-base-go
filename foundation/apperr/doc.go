// Package apperr 定义结构化错误原语、内核错误码与统一的框架错误对象，
// 供业务层构造错误、协议层（HTTP 与 gRPC）映射标准错误信封。
//
// 本包属于 foundation 领域原语基座，不依赖任何上层模块，也不负责
// 日志记录与网络输出：[New] 是纯构造函数，不产生任何副作用，
// 日志由调用方或协议层兜底显式记录。
//
// # 核心类型与调用路径
//
// [ErrorCode] 以值形式保留一个稳定的错误码三元组：
// 机器码 [Code]、输出标识 [Output] 与对外文案 [Message]。
// error_code.go 仅预置内核所需的错误码，
// 全部使用 FRAMEWORK_ 前缀和500分组，例如 [FrameworkContainerNotMounted] 与 [FrameworkInternalError]。
//
// 业务层通过 [New] 选定错误码并构造 [*Error]。[*Error] 实现
// error 接口，携带可选的补充描述、业务数据与底层根因，
// 可直接作为普通错误在层间传递，由协议层读取字段组装响应。
//
// # 错误码目录
//
// 错误码 [Code] 为 5 位整数，前 3 位对齐 HTTP 状态分组：
// 内置目录只维护框架上下文、组件状态与内部错误，不维护 HTTP 或 RPC 的协议目录。
// 编号与Output由 docs/engineering/spec/0002 的统一词表分配，禁止各包自行取号。
// SDK使用分组内00..49序号，下游业务使用50..99并维护自身中央台账。
// 请求绑定与校验由 Web 插件的 bind、validator 子包维护，
// HTTP 与 RPC 响应语义分别由各自的结果包维护；业务与插件也可定义 [ErrorCode] 传给 [New]。
//
// 目录变量均为 *ErrorCode 指针，可以在多个错误间共享只读引用；
// [New] 会在构造时复制值快照，调用方后续修改原变量不影响已构造的错误。
//
// # 错误组合与识别
//
// [New] 的 errs 参数收纳底层根因：过滤 nil 与 nil 载体后单个根因直接写入
// [Error.Err] 字段，多个根因经 errors.Join 合并。
// [*Error.Unwrap] 暴露根因，因此调用方可以用 errors.Is 与
// errors.As 直接判定 [*Error] 内包装的标准错误或自定义错误。
//
// # 空值判定与安全转接
//
// [IsNil] 识别真正的 nil 与"装入 error 接口的 nil 指针"等 nil 载体，
// 判定过程不会调用 Error 方法，可用于清洗第三方原因。
//
// [Find] 在原因链中查找已经存在的统一错误：判定顺序与 errors.As 一致，
// 既认节点自身类型，也认节点实现的标准 As 钩子；
// nil 载体与命中 nil 统一错误的投影都不算命中，不会遮蔽真实原因。
// 查找不设固定深度上限，合法深层包装链会被完整展开；
// 本函数只做查找，不构造失败结果，未找到时返回 nil 指针。
//
// [From] 是 nil 安全转接入口：原因为空时返回 nil 指针（不产生假失败）；
// 原因已是统一错误时保留其分类、安全描述与业务数据；
// 原因链中已存在统一错误时（由 [Find] 判定）沿用其分类，同时保留整条原始原因链，
// 外层包装文本与全部兄弟原因都不会丢失；未知原因按调用方显式给出的
// 错误码与描述包装，不会自动复制原因文本。
// [Join] 聚合多个统一错误：没有有效失败时返回 nil 指针，单项失败原样返回，
// 多项失败由调用方显式给出总码，全部子失败按入参顺序保留为根因。
// 聚合辅助不会自动改用 [FrameworkMultipleFailures]，也不会吞掉取消等控制原因。
//
// # 框架诊断码
//
// [FrameworkMultipleFailures]、[FrameworkHookPanic]、[FrameworkShutdownIncomplete]
// 与 [FrameworkWorkerUnexpectedExit] 表达框架多阶段失败、生命周期钩子异常、
// 关闭未完成与后台服务意外退出，均处于 500 分组。
// 它们只提供错误分类与聚合契约，框架生命周期的实际接入不属于本包职责。
//
// # JSON 序列化边界
//
// [*Error] 的 [Error.Err] 字段携带 json:"-" 标签，序列化时被忽略，
// 底层根因（可能包含 SQL、连接串等敏感信息）不会泄漏到对外响应；
// [Error.Data] 与 [Error.ErrorMessage] 为空时省略，
// [Error.Code] 始终输出。
//
// # 典型用法
//
// 构造带根因的业务错误并识别根因：
//
//	err := apperr.New(apperr.FrameworkInternalError, "处理请求失败", connErr)
//	if errors.Is(err, context.DeadlineExceeded) {
//		// 按超时处理
//	}
//
// # 并发与资源边界
//
// [New] 构造后不再修改字段的 [*Error] 可被多 goroutine 并发读取。
// 本包不持有外部资源，没有后台任务；错误码目录变量在初始化后
// 应视为只读，调用方不应修改目录内容。
package apperr
