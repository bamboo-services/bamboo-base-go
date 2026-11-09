package grpcresult

import (
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ClassifyStatus 把原生 gRPC 状态归类为框架统一错误 [*apperr.Error]。
//
// # 归类契约
//
// 本函数只做"原生 RPC 状态 → 兼容信封错误码"的纯归类，不重建原生状态、
// 不追加 Details、不改写原生状态携带的任何信息：
//
//   - st 为 nil 或状态码为 [codes.OK] 时返回 nil，表示没有失败；
//     OK 不制造失败信封，成功路径不得因为归类而产生错误。
//   - 16 种标准非 OK 状态按 spec/0002 的兼容分类表逐项取码：
//     Canceled→CONTROL_CANCELED、Unknown→GRPC_UNKNOWN_ERROR、
//     InvalidArgument→GRPC_PARAMETER_ERROR、DeadlineExceeded→CONTROL_DEADLINE_EXCEEDED
//     （明确请求接收/读取阶段时为 CONTROL_REQUEST_TIMEOUT）、NotFound→GRPC_NOT_FOUND、
//     AlreadyExists→GRPC_CONFLICT、PermissionDenied→GRPC_PERMISSION_DENIED、
//     ResourceExhausted→GRPC_RESOURCE_EXHAUSTED、FailedPrecondition→GRPC_PRECONDITION_FAILED、
//     Aborted→GRPC_ABORTED、OutOfRange→GRPC_OUT_OF_RANGE、Unimplemented→GRPC_UNIMPLEMENTED、
//     Internal→GRPC_INTERNAL_ERROR、Unavailable→GRPC_SERVICE_UNAVAILABLE、
//     DataLoss→GRPC_DATA_LOSS、Unauthenticated→GRPC_UNAUTHORIZED。
//   - 原生状态只证明 RPC 状态类别，不证明发生的是数据库、缓存、认证逻辑或某个业务资源，
//     因此不得从状态反推领域原因：ResourceExhausted 不等于请求过多，
//     FailedPrecondition 不等于无效状态，Aborted 不等于已存在，OutOfRange 不等于参数错误。
//   - 非标准或非法状态码不在上述 16 态承诺内，本函数保守归类为 [UnknownError]
//     （GRPC_UNKNOWN_ERROR），并保留原状态作为根因供诊断；不得据此宣称已完成协议链路验收。
//
// # 根因与文案边界
//
//   - 归类结果的根因固定为原生状态的 st.Err()，可经 errors.As 恢复原状态，
//     原生状态自带的显式文案与全部 Details 原样保留，不复制未知包装层文本。
//   - 原生状态不等于 context.Canceled 或 context.DeadlineExceeded 原因，
//     因此不会为了凑出可判定结果往原因链里制造这两个哨兵。
//   - 归类结果的安全描述只来自错误码预置文案，不携带调用方文本；
//     原始文案仍保留在根因中，需要对外时由协议层按既有脱敏规则显式取用。
//
// # 阶段与超时
//
// phase 只影响 DeadlineExceeded 的归属：默认阶段与未登记阶段一律归入
// [control.DeadlineExceeded]（50401），只有调用方显式给出 [control.PhaseRequestRead]
// 时才归入 [control.RequestTimeout]（40802）。其余状态与阶段无关。
//
// # 禁止替换原生状态
//
// 归类结果不得再传给 [ToStatusError] 去替换原生状态：兼容码按 HTTP 分组正向映射，
// Aborted（40907）会被改成 AlreadyExists、OutOfRange（40016）会被改成 InvalidArgument、
// DataLoss（50025）会被改成 Internal，原生语义因此变义。
// 原生状态始终是权威结果，补齐信封与 Details 接入由协议层在后续阶段实施。
//
// 返回值在不再修改字段时可被多 goroutine 并发读取。
func ClassifyStatus(st *status.Status, phase control.Phase) *apperr.Error {
	if st == nil {
		return nil
	}
	switch st.Code() {
	case codes.OK:
		return nil
	case codes.Canceled:
		return apperr.New(control.Canceled, "", st.Err())
	case codes.Unknown:
		return apperr.New(UnknownError, "", st.Err())
	case codes.InvalidArgument:
		return apperr.New(ParameterError, "", st.Err())
	case codes.DeadlineExceeded:
		return apperr.New(deadlineCode(phase), "", st.Err())
	case codes.NotFound:
		return apperr.New(NotFound, "", st.Err())
	case codes.AlreadyExists:
		return apperr.New(Conflict, "", st.Err())
	case codes.PermissionDenied:
		return apperr.New(PermissionDenied, "", st.Err())
	case codes.ResourceExhausted:
		return apperr.New(ResourceExhausted, "", st.Err())
	case codes.FailedPrecondition:
		return apperr.New(PreconditionFailed, "", st.Err())
	case codes.Aborted:
		return apperr.New(Aborted, "", st.Err())
	case codes.OutOfRange:
		return apperr.New(OutOfRange, "", st.Err())
	case codes.Unimplemented:
		return apperr.New(Unimplemented, "", st.Err())
	case codes.Internal:
		return apperr.New(InternalError, "", st.Err())
	case codes.Unavailable:
		return apperr.New(ServiceUnavailable, "", st.Err())
	case codes.DataLoss:
		return apperr.New(DataLoss, "", st.Err())
	case codes.Unauthenticated:
		return apperr.New(Unauthorized, "", st.Err())
	default:
		// 非标准或非法状态码不在 16 态承诺内，保守归类为未知错误并保留原状态根因。
		return apperr.New(UnknownError, "", st.Err())
	}
}

// deadlineCode 返回超时阶段对应的兼容错误码。
//
// 只有明确的请求接收/读取阶段才使用 408；默认阶段与未登记阶段一律按
// 内部操作处理为 504，保证 408 不会出现在缺少请求阶段证据的路径上。
func deadlineCode(phase control.Phase) *apperr.ErrorCode {
	if phase == control.PhaseRequestRead {
		return control.RequestTimeout
	}
	return control.DeadlineExceeded
}
