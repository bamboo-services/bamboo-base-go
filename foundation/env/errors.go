package env

import (
	"errors"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/valueerr"
)

// errNilReceiver 表示对 nil 接收器调用解码方法。
//
// 该哨兵只作为根因保留，最终对外错误为 [*apperr.Error]（错误码 50019）。
var errNilReceiver = errors.New("env.Key: nil receiver")

// nilReceiverError 构造接收器为 nil 的调用契约错误（50019）。
//
// 分类只依据实际失败分支显式给出，不解析错误文本；
// 根因保留 [errNilReceiver]，errors.Is 判定仍然可达。
func nilReceiverError() *apperr.Error {
	return valueerr.From(valueerr.KindInvalidReceiver, "env.Key 接收器为 nil", errNilReceiver)
}
