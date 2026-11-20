package database

import (
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
)

// lifecycleErr 将生命周期阶段的失败原因归一为框架统一错误。
//
// 归一顺序与内核保持一致：
//   - 原因链中已经存在 [*apperr.Error] 时沿用其错误码与描述，并保留整条原始原因链；
//   - 取消与超时经 [control.Classify] 归一为控制类错误码（49900 / 50401），
//     原生 context 哨兵继续作为根因保留，供 errors.Is 精确判定；
//   - 其余普通原因按所属域错误码 [DatabaseError] 兜底，原始原因链完整保留。
//
// 返回值在存在真实原因时始终非 nil；成功路径必须直接返回真正的 nil 指针，
// 禁止把 nil 指针装入 error 接口冒充失败。
func lifecycleErr(msg apperr.ErrMessage, cause error) *apperr.Error {
	if apperr.IsNil(cause) {
		return nil
	}
	if structured := apperr.Find(cause); structured != nil {
		return apperr.From(&structured.Code, structured.ErrorMessage, cause)
	}
	if classified := control.Classify(cause, control.PhaseOperation); classified != nil {
		return apperr.New(&classified.Code, msg, cause)
	}
	return apperr.From(DatabaseError, msg, cause)
}
