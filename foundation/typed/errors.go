package typed

import (
	"errors"
	"strconv"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/valueerr"
)

// 本文件集中本包自有失败出口的错误构造规则。
//
// 分类只依据实际校验分支显式给出，不解析错误文本、不按错误来源推断：
//
//   - 格式非法（受控键名语法不合法、布尔词非法、时长写法非法、解析语法错误）：InvalidFormat（40013）；
//   - 数值超出目标类型位宽（strconv 报告 ErrRange）：OutOfRange（40014）；
//   - 输入源类型不合法（JSON null 与非字符串、XML 嵌套子元素、SQL NULL 与非 string/[]byte）：InvalidType（40015）；
//   - 接收器为 nil：InvalidReceiver（50019）。
//
// 安全描述一律由本包给出，不复制原始输入内容；原生错误只作为根因保留，
// 供 errors.Is 与 errors.As 继续识别。

// errInvalidFormat 构造无根因的格式非法错误（40013）。
//
// 用于没有原生 cause 可保留的校验分支，例如受控键名语法不合法。
func errInvalidFormat(msg apperr.ErrMessage) *apperr.Error {
	return apperr.New(valueerr.InvalidFormat, msg)
}

// errInvalidType 构造无根因的输入类型非法错误（40015）。
//
// 用于没有原生 cause 可保留的类型分支，例如 JSON null、SQL NULL 与不支持的 SQL 列类型。
func errInvalidType(msg apperr.ErrMessage) *apperr.Error {
	return apperr.New(valueerr.InvalidType, msg)
}

// errNilReceiver 构造接收器为 nil 的调用契约错误（50019）。
//
// [ErrNilReceiver] 作为根因保留，调用方的 errors.Is 判定仍然可达。
func errNilReceiver() *apperr.Error {
	return valueerr.From(valueerr.KindInvalidReceiver, "typed.String 接收器为 nil", ErrNilReceiver)
}

// errParse 按 strconv 的失败分支把原生解析错误转接为值错误。
//
// 语法非法归 InvalidFormat（40013），超出目标位宽范围归 OutOfRange（40014）；
// 原生错误原样作为根因保留。
func errParse(cause error, msg apperr.ErrMessage) *apperr.Error {
	kind := valueerr.KindInvalidFormat
	if errors.Is(cause, strconv.ErrRange) {
		kind = valueerr.KindOutOfRange
	}
	return valueerr.From(kind, msg, cause)
}
