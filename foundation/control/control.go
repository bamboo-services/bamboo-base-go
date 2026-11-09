package control

import (
	"context"
	"errors"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// Phase 描述控制失败所处的阶段，决定超时错误的归属。
//
// 阶段只能由调用方显式给出：本包不解析错误文本，也不从 gRPC status、
// HTTP 状态或调用栈反推阶段，因此 408（请求接收/读取超时）只可能出现在
// 有明确请求阶段证据的调用路径上。
type Phase uint8

const (
	// PhaseOperation 表示应用内部操作阶段，是零值默认阶段。
	//
	// 该阶段的超时归一为 [DeadlineExceeded]（50401，HTTP 504）。
	PhaseOperation Phase = iota

	// PhaseRequestRead 表示请求接收或请求体读取阶段。
	//
	// 该阶段的超时归一为 [RequestTimeout]（40802，HTTP 408）。
	PhaseRequestRead
)

// Classify 把取消或超时原因归一为控制类统一错误。
//
// 判定只依据真实原因链与显式阶段，不解析错误文本、不从协议状态反推原因，
// 也不会为了凑出可判定的结果去伪造 context.Canceled 根因：
//
//   - cause 为真正的 nil 或 nil 载体（装入 error 接口的 nil 指针等）时返回 nil，
//     表示没有失败；调用方应直接返回真正的 nil，不要把 nil 当作控制错误。
//   - 原因链中已经存在 [apperr.Error] 时沿用该错误的错误码与安全描述，
//     并把整条原始原因链保留为根因：既有错误不会因为链内深层出现
//     context.Canceled 或 context.DeadlineExceeded 就被重新分类。
//     结构化原因查找沿用 [apperr.Find]：尊重调用方自定义的 As 钩子，
//     并跳过 nil 载体，不会被链中的 nil 投影遮蔽。
//   - 原因链可经 errors.Is 识别为 context.Canceled 时返回 [Canceled]（49900），
//     与阶段无关。
//   - 原因链可经 errors.Is 识别为 context.DeadlineExceeded 时按阶段返回：
//     [PhaseRequestRead] 返回 [RequestTimeout]（40802），
//     其余阶段（含未登记的阶段值）返回 [DeadlineExceeded]（50401）。
//   - 其它普通错误返回 nil。本函数不是所有原因的通用归一器：
//     它不猜测超时、不把普通故障包装成通用 500，也不吞掉未知失败。
//     调用方必须在自己的所属域显式兜底，例如
//     [apperr.From] 配合所属域自己的错误码，并在兜底时保留原始原因链。
//
// 仅在识别到控制原因或既有统一错误时返回非 nil；未知原因必须由调用方兜底。
// 返回的已构造对象在不再修改字段时可被多 goroutine 并发读取。
func Classify(cause error, phase Phase) *apperr.Error {
	if apperr.IsNil(cause) {
		return nil
	}
	if structured := apperr.Find(cause); structured != nil {
		return apperr.From(&structured.Code, structured.ErrorMessage, cause)
	}
	switch {
	case errors.Is(cause, context.Canceled):
		return apperr.From(Canceled, "", cause)
	case errors.Is(cause, context.DeadlineExceeded):
		return apperr.From(deadlineCode(phase), "", cause)
	default:
		return nil
	}
}

// deadlineCode 返回超时阶段对应的错误码。
//
// 只有明确的请求接收/读取阶段才使用 408；未登记的阶段值按内部操作处理，
// 保证 408 不会出现在缺少请求阶段证据的路径上。
func deadlineCode(phase Phase) *apperr.ErrorCode {
	if phase == PhaseRequestRead {
		return RequestTimeout
	}
	return DeadlineExceeded
}
