package grpcresult

import (
	"context"
	"errors"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/generate"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// marshalStatusDetail 默认使用 anypb.New 打包详情信封，允许包内测试注入编码失败。
var marshalStatusDetail = anypb.New

func extractExplicitGRPCStatus(err error) (*status.Status, bool) {
	var se interface{ GRPCStatus() *status.Status }
	if errors.As(err, &se) {
		if st := se.GRPCStatus(); st != nil {
			return st, true
		}
	}
	return nil, false
}

// NormalizeSDKError 是 gRPC 插件的自有强类型错误归一核心，将未知异常与控制原因
// 统一收敛为 [*apperr.Error] 指针。
//
// 规则如下：
//   - 若 err 为 nil 或 typed-nil 载体，返回真正的 nil 指针；
//   - 若原因链中包含 [*apperr.Error]，按 [apperr.Find] 沿用其错误码与安全描述，保留完整原因链；
//   - 若原因链包含 context.Canceled / context.DeadlineExceeded，经 [control.Classify] 归为
//     49900 / 40802 / 50401，并保持空 ErrorMessage 以避免复制外层包装文本；
//   - 其余普通未知错误统一归为 [InternalError]（50011）并保留原始 err 作为内部根因。
//
// 本函数不处理原生 gRPC status 协议载荷的 Details 补齐；原生 status 适配由 [NormalizeToStatusError]
// 在协议边界执行，避免经 [ToStatusError] 二次映射改写原生 [codes.Code]。
func NormalizeSDKError(err error, phase control.Phase) *apperr.Error {
	if apperr.IsNil(err) {
		return nil
	}
	if structured := apperr.Find(err); structured != nil {
		return apperr.From(&structured.Code, structured.ErrorMessage, err)
	}
	if ctrlErr := control.Classify(err, phase); ctrlErr != nil {
		return ctrlErr
	}
	return apperr.New(InternalError, "系统内部异常", err)
}

// NormalizeToStatusError 按 spec/0003 R12 契约将服务端处理错误转换为对外 gRPC status 错误：
//
//   - err 为 nil、typed-nil 或携带 [codes.OK] 状态时返回 nil；
//   - 链内可提取 [*apperr.Error] 时交由 [ToStatusError] 输出，保留 Code/Output 与安全文案；
//   - 链内可识别 context.Canceled / context.DeadlineExceeded 时输出对应控制码与固定安全文案，
//     并携带带有请求追踪与调试耗时的 [generate.BaseResponse]，绝不复制包装层文本；
//   - 合法非 OK 原生 gRPC status：
//     1. 若已携带 [generate.BaseResponse]（含多条、冲突或不可解码情形），原样保留其
//     codes.Code、显式 Message 及全部原 Details 顺序，不覆盖、不补字段、不追加；
//     对多条/不可解码异常触发 onDegrade 受控诊断回调；
//     2. 若缺少 [generate.BaseResponse]，通过 [ClassifyStatus] 选取兼容业务码并克隆
//     原 status.Proto 追加单个兼容 [generate.BaseResponse]，保留原 codes.Code、显式
//     Message 与全部原 Details（含未知 Any）顺序；若编码失败触发 onDegrade 并保留原 status；
//   - 其余未知普通错误先触发 onUnknown（供内部安全日志记录），再返回固定安全文案的
//     [InternalError]（50011）status 错误，绝不外泄 err.Error()。
func NormalizeToStatusError(
	ctx context.Context,
	err error,
	phase control.Phase,
	onUnknown func(error),
	onDegrade func(string),
) error {
	if apperr.IsNil(err) {
		return nil
	}

	if structured := apperr.Find(err); structured != nil {
		return ToStatusError(ctx, structured)
	}

	if ctrlErr := control.Classify(err, phase); ctrlErr != nil {
		return ToStatusError(ctx, ctrlErr)
	}

	if st, ok := extractExplicitGRPCStatus(err); ok {
		return ensureNativeStatusEnvelope(ctx, st, phase, onDegrade)
	}

	if onUnknown != nil {
		onUnknown(err)
	}
	return ToStatusError(ctx, NormalizeSDKError(err, phase))
}

func ensureNativeStatusEnvelope(
	ctx context.Context,
	st *status.Status,
	phase control.Phase,
	onDegrade func(string),
) error {
	if st == nil || st.Code() == codes.OK {
		return nil
	}

	stProto := st.Proto()
	rawDetails := stProto.GetDetails()

	var (
		baseCount int
		decodeErr error
	)
	for _, detail := range rawDetails {
		if detail == nil || detail.MessageName() != baseResponseFullName {
			continue
		}
		baseCount++
		var br generate.BaseResponse
		if uErr := detail.UnmarshalTo(&br); uErr != nil && decodeErr == nil {
			decodeErr = uErr
		}
	}

	if baseCount > 0 {
		if decodeErr != nil && onDegrade != nil {
			onDegrade("原生 gRPC status 携带不可解码的 BaseResponse，已原样保留原载荷")
		} else if baseCount > 1 && onDegrade != nil {
			onDegrade("原生 gRPC status 携带多个 BaseResponse，已原样保留原载荷")
		}
		return st.Err()
	}

	classified := ClassifyStatus(st, phase)
	safeMsg := st.Message()
	if safeMsg == "" {
		safeMsg = string(classified.Code.Message)
	}

	compatResp := &generate.BaseResponse{
		Context:  resolveRequestKey(ctx),
		Output:   string(classified.Code.Output),
		Code:     uint64(classified.Code.Code),
		Message:  safeMsg,
		Overhead: resolveOverhead(ctx),
	}

	packedAny, marshalErr := marshalStatusDetail(compatResp)
	if marshalErr != nil {
		if onDegrade != nil {
			onDegrade("原生 gRPC status 补齐 BaseResponse 编码失败，已受控降级保留原状态")
		}
		return st.Err()
	}

	cloned, ok := proto.Clone(stProto).(*spb.Status)
	if !ok || cloned == nil {
		cloned = &spb.Status{
			Code:    int32(st.Code()),
			Message: st.Message(),
		}
	}
	cloned.Details = append(cloned.Details, packedAny)
	return status.FromProto(cloned).Err()
}
