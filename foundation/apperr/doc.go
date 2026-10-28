// Package apperr 定义结构化业务错误码目录与统一的框架错误对象，
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
// error_code.go 预置了 189 个覆盖常用 HTTP 语义的错误码变量，
// 例如 [BadRequest]、[Unauthorized]、[NotFound] 与 [ServerInternalError]。
//
// 业务层通过 [New] 选定错误码并构造 [*Error]。[*Error] 实现
// error 接口，携带可选的补充描述、业务数据与底层根因，
// 可直接作为普通错误在层间传递，由协议层读取字段组装响应。
//
// # 错误码目录
//
// 错误码 [Code] 为 5 位整数，前 3 位对齐 HTTP 状态分组：
// 400xx 表示请求与业务语义类错误，500xx 表示服务端内部错误。
// 同一前 3 位内按主题划分区间（如 4001x 请求参数、4011x 用户账户、
// 409xx 状态冲突），完整目录见 error_code.go 中的预置变量。
//
// 目录变量均为 *ErrorCode 指针，可以在多个错误间共享只读引用；
// [New] 会在构造时复制值快照，调用方后续修改原变量不影响已构造的错误。
//
// # 错误组合与识别
//
// [New] 的 errs 参数收纳底层根因：过滤 nil 后单个根因直接写入
// [Error.Err] 字段，多个根因经 errors.Join 合并。
// [*Error.Unwrap] 暴露根因，因此调用方可以用 errors.Is 与
// errors.As 直接判定 [*Error] 内包装的标准错误或自定义错误。
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
//	err := apperr.New(apperr.DatabaseError, "查询用户失败", connErr)
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
