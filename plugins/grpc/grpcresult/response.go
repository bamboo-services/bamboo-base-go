package grpcresult

import (
	"context"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/generate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// DefaultSuccessOutput 是纯成功响应的默认输出标识。
	DefaultSuccessOutput = "Success"
)

// resolveRequestKey 安全提取请求追踪 ID。
func resolveRequestKey(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	return ctxkey.GetRequestKey(ctx)
}

// resolveOverhead 安全计算耗时微秒。
// 仅在调试模式下且存在起始时间时计算，其余情况稳健返回 nil。
func resolveOverhead(ctx context.Context) *int64 {
	if ctx == nil {
		return nil
	}
	overhead := ctxkey.CalcOverheadTime(ctx)
	if overhead == 0 && (!ctxkey.IsDebugMode() || ctx.Value(ctxkey.UserStartTimeKey) == nil) {
		return nil
	}
	return &overhead
}

// Success 构建符合 base.proto 定义的操作成功 BaseResponse。
// Code 填充 200，Output 填充 "Success"，Message 为业务传递的安全文案。
// Context 与 Overhead（仅调试模式）自动从 context 中提取注入。
func Success(ctx context.Context, message string) *generate.BaseResponse {
	resp := &generate.BaseResponse{
		Context:  resolveRequestKey(ctx),
		Output:   DefaultSuccessOutput,
		Code:     200,
		Message:  message,
		Overhead: resolveOverhead(ctx),
	}
	return resp
}

// httpGroupToGRPCCode 根据 HTTP 状态分组（前 3 位）映射为固定 gRPC codes.Code。
// 映射契约严格依循设计：
// 400 -> InvalidArgument
// 401 -> Unauthenticated
// 403 -> PermissionDenied
// 404 -> NotFound
// 408 -> DeadlineExceeded
// 504 -> DeadlineExceeded
// 409 -> AlreadyExists
// 412 -> FailedPrecondition
// 429 -> ResourceExhausted
// 499 -> Canceled
// 500 -> Internal
// 501 -> Unimplemented
// 502 -> Unavailable
// 503 -> Unavailable
// 其余未知分组一律兜底映射为 codes.Unknown。
func httpGroupToGRPCCode(httpGroup uint64) codes.Code {
	switch httpGroup {
	case 400:
		return codes.InvalidArgument
	case 401:
		return codes.Unauthenticated
	case 403:
		return codes.PermissionDenied
	case 404:
		return codes.NotFound
	case 408, 504:
		return codes.DeadlineExceeded
	case 409:
		return codes.AlreadyExists
	case 412:
		return codes.FailedPrecondition
	case 429:
		return codes.ResourceExhausted
	case 499:
		return codes.Canceled
	case 500:
		return codes.Internal
	case 501:
		return codes.Unimplemented
	case 502, 503:
		return codes.Unavailable
	default:
		return codes.Unknown
	}
}

// ToStatusError 将统一业务错误对象 *apperr.Error 转换为 gRPC status 错误。
// 严禁将底层内部 err.Err（如原始 SQL 错误、私密堆栈）进入 status.Message 或 Details（即使 Debug 模式也不允许）。
// status.Message 仅对外展示安全业务文案（Code.Message，空时回退至 Code.Output）。
// status.WithDetails 中附加生成的 BaseResponse，携带业务 code、output、context、安全 errorMessage 及耗时。
// 若入参 err 为 nil，安全返回 codes.Internal 状态错误。
func ToStatusError(ctx context.Context, err *apperr.Error) error {
	reqKey := resolveRequestKey(ctx)
	overhead := resolveOverhead(ctx)

	if err == nil {
		fallback := apperr.ServerInternalError
		st := status.New(codes.Internal, string(fallback.Message))
		detailResp := &generate.BaseResponse{
			Context:  reqKey,
			Output:   string(fallback.Output),
			Code:     uint64(fallback.Code),
			Message:  string(fallback.Message),
			Overhead: overhead,
		}
		if detailedSt, detailErr := st.WithDetails(detailResp); detailErr == nil {
			return detailedSt.Err()
		}
		return st.Err()
	}

	httpGroup := uint64(err.Code.Code / 100)
	grpcCode := httpGroupToGRPCCode(httpGroup)

	safeMessage := string(err.Code.Message)
	if safeMessage == "" {
		safeMessage = string(err.Code.Output)
	}

	st := status.New(grpcCode, safeMessage)

	detailResp := &generate.BaseResponse{
		Context:  reqKey,
		Output:   string(err.Code.Output),
		Code:     uint64(err.Code.Code),
		Message:  safeMessage,
		Overhead: overhead,
	}

	if err.ErrorMessage != "" {
		errMsg := string(err.ErrorMessage)
		detailResp.ErrorMessage = &errMsg
	}

	if detailedSt, detailErr := st.WithDetails(detailResp); detailErr == nil {
		return detailedSt.Err()
	}

	return st.Err()
}
