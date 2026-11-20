package email

import (
	"context"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
)

// opErr 把邮件模块自有失败出口的原因链归一为 [*apperr.Error]。
//
// 归一顺序：
//   - 原因链（含嵌套包装）中可由 errors.Is 识别为 context.Canceled 或
//     context.DeadlineExceeded 时，经 [control.Classify] 按 [control.PhaseOperation]
//     归一为控制类错误码（49900 / 50401），原生 context 哨兵继续作为根因保留，
//     供 errors.Is 精确判定；
//   - 其余失败按所属域错误码 [EmailError]（50017）归一，原始原因链完整保留，
//     已存在的统一错误仍可经 errors.As 在链中提取。
//
// 返回值在存在真实原因时始终非 nil；成功路径必须直接返回真正的 nil 指针，
// 禁止把 nil 指针装入 error 接口冒充失败。
func opErr(msg apperr.ErrMessage, cause error) *apperr.Error {
	if apperr.IsNil(cause) {
		return nil
	}
	if classified := control.Classify(cause, control.PhaseOperation); classified != nil {
		return apperr.New(&classified.Code, msg, cause)
	}
	return apperr.New(EmailError, msg, cause)
}

// opErrWithContext 在 [opErr] 的基础上优先采用显式上下文的取消或超时信号。
//
// 当传入上下文的 ctx.Err() 非 nil 时，该信号优先于底层原因链被归一为控制类错误码
// （49900 / 50401），同时把底层原因与原生 context 哨兵一并保留为根因；
// 上下文未结束时退回 [opErr] 的通用归一。
//
// 返回值在存在真实原因时始终非 nil；成功路径必须直接返回真正的 nil 指针。
func opErrWithContext(msg apperr.ErrMessage, cause error, ctxs ...context.Context) *apperr.Error {
	for _, ctx := range ctxs {
		if ctx == nil {
			continue
		}
		ctxErr := ctx.Err()
		if ctxErr == nil {
			continue
		}
		if classified := control.Classify(ctxErr, control.PhaseOperation); classified != nil {
			return apperr.New(&classified.Code, msg, cause, ctxErr)
		}
		if classified := control.Classify(cause, control.PhaseOperation); classified != nil {
			return apperr.New(&classified.Code, msg, cause, ctxErr)
		}
		return apperr.New(EmailError, msg, cause, ctxErr)
	}
	return opErr(msg, cause)
}
