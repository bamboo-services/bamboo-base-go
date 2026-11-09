// Package control 归一取消与超时等控制语义，产出统一的强类型错误。
//
// 本包属于 foundation 领域原语基座，只依赖 [apperr]，不依赖 HTTP、gRPC
// 或 application 层，因此可以被任意插件与协议层复用。
//
// # 职责边界
//
// 本包只回答"这次失败是不是取消或超时"，不负责其它失败原因：
//
//   - 取消（context.Canceled）归一为 [Canceled]（49900，HTTP 499）；
//   - 超时（context.DeadlineExceeded）按阶段归一为 [RequestTimeout]
//     （40802，HTTP 408）或 [DeadlineExceeded]（50401，HTTP 504）。
//
// 判定只依据真实原因链（errors.Is）与调用方显式给出的阶段 [Phase]：
// 不解析错误文本，不从 gRPC status 或其它协议状态反推原因，
// 也不会把普通服务故障、网络失败或 5xx 推断成超时。
//
// # 不可识别的普通原因
//
// [Classify] 不是所有原因的通用归一器。传入与取消、超时无关的普通错误时，
// 它返回 nil：既不猜测超时，也不把普通故障包装成通用 500。
// 调用方必须在自己的所属域显式兜底，并把原始原因链保留下来：
//
//	err := control.Classify(cause, control.PhaseOperation)
//	if err == nil {
//		// 不是控制原因：由所属域用自己的错误码显式兜底。
//		err = apperr.From(apperr.FrameworkInternalError, "读取配置失败", cause)
//	}
//
// 返回 nil 只表示"不是控制错误"，绝不表示失败已经消失：
// 调用方不得因为 Classify 返回 nil 就丢弃 cause 或直接返回成功。
//
// # 与已有结构化错误的关系
//
// 原因链中已经存在 [apperr.Error] 时，[Classify] 沿用既有错误码与安全描述，
// 并保留整条原因链：既有错误不会因为链内深层出现 context.Canceled
// 或 context.DeadlineExceeded 就被重新分类。
//
// # 阶段与错误码的对应
//
// | 阶段                    | 超时错误码                     |
// | ----------------------- | ------------------------------ |
// | [PhaseRequestRead]      | [RequestTimeout]（40802，408） |
// | [PhaseOperation] 及默认 | [DeadlineExceeded]（50401，504） |
//
// 未登记的阶段值按内部操作处理，因此 408 只会出现在有明确请求阶段证据的路径上。
package control
