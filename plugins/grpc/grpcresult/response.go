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
	// DefaultSuccessOutput 是标准成功响应信封中 Output 字段的默认语义标签。
	// 取值为 "Success"。
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

// Success 构建并返回符合框架契约的标准成功响应 [generate.BaseResponse] 实例。
//
// 字段填充规则：
//   - Context：自动从 ctx 中提取请求追踪 ID（通过 [ctxkey.GetRequestKey]）；若 ctx 为 nil 则为空字符串；
//   - Output：填充为默认的 [DefaultSuccessOutput]（即 "Success"）；
//   - Code：固定填充为 200，表示业务成功；
//   - Message：填充传入的业务安全描述文案 message；
//   - Overhead：仅当应用处于调试模式且上下文携带请求起始时间时计算微秒整数耗时；其余情况安全返回 nil；
//   - ErrorMessage：成功场景下始终保持为 nil。
//
// 该函数为纯构造函数，不执行网络 I/O，可并发安全调用。
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

// ToStatusError 将结构化业务错误 [*apperr.Error] 转换为符合 gRPC 规范的 [status.Status] 错误。
//
// # 错误安全与脱敏红线
//
// 转换严格遵守安全边界，严禁将底层内部原始错误 err.Err（如原始 SQL 错误、敏感数据库连接串、
// 堆栈追踪或系统错误）输出到 [status.Status] 的 Message 或 Details 中，即使在 Debug 模式下也绝不泄露。
//
// # 状态映射与 Details 注入
//
// 转换规则如下：
//   - 若入参 err 为 nil，安全返回包含 [codes.Internal] 状态码及兜底业务内部错误信封的错误；
//   - 提取 err.Code.Code 的前 3 位（即 HTTP 状态组），映射为对应的标准 [codes.Code]；
//   - status 的安全文案优先采用 err.Code.Message，若为空则回退使用 err.Code.Output；
//   - 通过 status.WithDetails 注入新生成的 [generate.BaseResponse]，携带业务 code、output、
//     追踪 context、安全 errorMessage（若存在）及调试耗时 overhead；
//   - 若 Details 序列化失败，安全回退至未携带 Details 的原生 status 错误。
//
// 返回的 error 可直接作为 gRPC 服务的 Handler 返回值交由框架底层传输。
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
