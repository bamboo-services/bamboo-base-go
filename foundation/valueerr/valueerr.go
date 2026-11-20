package valueerr

import "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"

// Kind 表示值错误的分类，决定错误码归属。
//
// 分类只能由调用方按实际校验分支显式给出：本包不解析错误文本，
// 也不按错误来源（请求输入还是服务自身配置）或调用时机推断分类。
type Kind uint8

const (
	// KindInvalidFormat 表示值内容格式非法，对应 400 组的 [InvalidFormat]。
	KindInvalidFormat Kind = iota

	// KindOutOfRange 表示数值超出约定范围，对应 400 组的 [OutOfRange]。
	KindOutOfRange

	// KindInvalidType 表示值的类型不可接受，对应 400 组的 [InvalidType]。
	KindInvalidType

	// KindUnsupportedTarget 表示不支持的转换目标类型，
	// 属于调用契约错误，对应 500 组的 [UnsupportedTarget]。
	KindUnsupportedTarget

	// KindInvalidReceiver 表示值接收器无效（例如向 nil 接收器写入），
	// 属于调用契约错误，对应 500 组的 [InvalidReceiver]。
	KindInvalidReceiver
)

// From 按显式分类把值校验原因转接为统一错误。
//
// 分类与归属规则：
//
//   - cause 为真正的 nil 或 nil 载体（装入 error 接口的 nil 指针等）时返回 nil，
//     表示没有失败；需要无原因的失败时请显式使用 [apperr.New]。
//   - 原因链中已经存在 [apperr.Error] 时沿用该错误的错误码与安全描述，
//     并把整条原始原因链保留为根因：既有错误不会因为链内深层出现
//     其它类型的错误而被改判为值错误，也不会丢失兄弟原因。
//     结构化原因查找沿用 [apperr.Find]：尊重调用方自定义的 As 钩子，
//     并跳过 nil 载体，不会被链中的 nil 投影遮蔽。
//   - 其余情况按 kind 映射到明确错误码：
//     [KindInvalidFormat]、[KindOutOfRange]、[KindInvalidType] 归 400 组，
//     [KindUnsupportedTarget]、[KindInvalidReceiver] 归 500 组。
//   - 未登记的 kind 属于调用契约错误，统一按 [UnsupportedTarget]（50018）返回，
//     并在补充描述中标注"未登记的值错误分类"；本包不会按错误文本猜测分类。
//
// msg 是调用方提供的安全补充描述，必须由调用方保证不复制可能包含凭据的
// 原始输入。本包不判断错误来自请求还是服务自身配置：
// 读取服务自身配置的入口应由所属域按配置责任使用自己的 500 码重新包装，
// 并把本包产出的值错误保留为原因。该重包装必须用 [apperr.New] 显式构造：
// [apperr.From] 会沿用已有结构化错误的分类，不会覆盖错误码。
//
// 返回值在存在真实原因时非 nil，可被多 goroutine 并发读取。
func From(kind Kind, msg apperr.ErrMessage, cause error) *apperr.Error {
	if apperr.IsNil(cause) {
		return nil
	}
	if structured := apperr.Find(cause); structured != nil {
		return apperr.From(&structured.Code, structured.ErrorMessage, cause)
	}
	code, ok := codeOf(kind)
	if !ok {
		return apperr.From(UnsupportedTarget, unknownKindMessage(msg), cause)
	}
	return apperr.From(code, msg, cause)
}

// codeOf 返回分类对应的错误码；未登记的分类返回 false。
func codeOf(kind Kind) (*apperr.ErrorCode, bool) {
	switch kind {
	case KindInvalidFormat:
		return InvalidFormat, true
	case KindOutOfRange:
		return OutOfRange, true
	case KindInvalidType:
		return InvalidType, true
	case KindUnsupportedTarget:
		return UnsupportedTarget, true
	case KindInvalidReceiver:
		return InvalidReceiver, true
	default:
		return nil, false
	}
}

// unknownKindMessage 生成未登记分类的可读说明，并保留调用方已给出的安全描述。
func unknownKindMessage(msg apperr.ErrMessage) apperr.ErrMessage {
	const note = "未登记的值错误分类"
	if msg == "" {
		return note
	}
	return msg + "；" + note
}
