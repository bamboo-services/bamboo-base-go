package grpcresult_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/generate"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/grpcresult"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/grpcresult/internaltest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// nativeStatusFixture 构造带显式文案与两个 Details 的原生状态，
// 用于验证归类过程既不改写原状态，也不重排或重编码原有 Details。
func nativeStatusFixture(t *testing.T, code codes.Code, message string) *status.Status {
	t.Helper()

	detail, err := status.New(code, message).WithDetails(
		&generate.BaseResponse{Code: 200, Output: "Success", Message: "原生信封"},
		&internaltest.ValidBetaResponse{Data: "原始载荷", CustomBase: &generate.BaseResponse{Code: 500, Output: "GRPC_INTERNAL_ERROR"}},
	)
	if err != nil {
		t.Fatalf("构造带 Details 的原生状态失败: %v", err)
	}
	return detail
}

// statusBytes 返回原生状态的序列化字节，用于逐字节比对。
func statusBytes(t *testing.T, st *status.Status) []byte {
	t.Helper()

	raw, err := proto.Marshal(st.Proto())
	if err != nil {
		t.Fatalf("序列化原生状态失败: %v", err)
	}
	return raw
}

// nativeStatusFromError 从统一错误的根因中取回原生状态。
//
// 不能使用 [status.FromError]：它在命中被包装的状态时会把外层错误文本
// 写回状态文案，从而破坏原生显式文案的保真断言。
func nativeStatusFromError(t *testing.T, err error) *status.Status {
	t.Helper()

	var carrier interface{ GRPCStatus() *status.Status }
	if !errors.As(err, &carrier) {
		t.Fatal("统一错误根因中未找到原生状态")
	}
	recovered := carrier.GRPCStatus()
	if recovered == nil {
		t.Fatal("统一错误根因中的原生状态为空")
	}
	return recovered
}

// TestClassifyStatus_SixteenStandardStates 逐项验证 16 种标准非 OK 状态的兼容归类，
// 并断言归类结果不复制未知包装文本、不伪造 context 原因、原生状态保持不变。
func TestClassifyStatus_SixteenStandardStates(t *testing.T) {
	tests := []struct {
		name       string
		native     codes.Code
		wantCode   *apperr.ErrorCode
		wantOutput string
	}{
		{"Canceled", codes.Canceled, control.Canceled, "CONTROL_CANCELED"},
		{"Unknown", codes.Unknown, grpcresult.UnknownError, "GRPC_UNKNOWN_ERROR"},
		{"InvalidArgument", codes.InvalidArgument, grpcresult.ParameterError, "GRPC_PARAMETER_ERROR"},
		{"DeadlineExceeded", codes.DeadlineExceeded, control.DeadlineExceeded, "CONTROL_DEADLINE_EXCEEDED"},
		{"NotFound", codes.NotFound, grpcresult.NotFound, "GRPC_NOT_FOUND"},
		{"AlreadyExists", codes.AlreadyExists, grpcresult.Conflict, "GRPC_CONFLICT"},
		{"PermissionDenied", codes.PermissionDenied, grpcresult.PermissionDenied, "GRPC_PERMISSION_DENIED"},
		{"ResourceExhausted", codes.ResourceExhausted, grpcresult.ResourceExhausted, "GRPC_RESOURCE_EXHAUSTED"},
		{"FailedPrecondition", codes.FailedPrecondition, grpcresult.PreconditionFailed, "GRPC_PRECONDITION_FAILED"},
		{"Aborted", codes.Aborted, grpcresult.Aborted, "GRPC_ABORTED"},
		{"OutOfRange", codes.OutOfRange, grpcresult.OutOfRange, "GRPC_OUT_OF_RANGE"},
		{"Unimplemented", codes.Unimplemented, grpcresult.Unimplemented, "GRPC_UNIMPLEMENTED"},
		{"Internal", codes.Internal, grpcresult.InternalError, "GRPC_INTERNAL_ERROR"},
		{"Unavailable", codes.Unavailable, grpcresult.ServiceUnavailable, "GRPC_SERVICE_UNAVAILABLE"},
		{"DataLoss", codes.DataLoss, grpcresult.DataLoss, "GRPC_DATA_LOSS"},
		{"Unauthenticated", codes.Unauthenticated, grpcresult.Unauthorized, "GRPC_UNAUTHORIZED"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			native := nativeStatusFixture(t, tc.native, "原生显式文案")
			rawBefore := statusBytes(t, native)

			classified := grpcresult.ClassifyStatus(native, control.PhaseOperation)
			if classified == nil {
				t.Fatal("非 OK 状态必须归类为非 nil 统一错误")
			}
			if classified.Code != *tc.wantCode {
				t.Fatalf("兼容错误码不符: got %+v, want %+v", classified.Code, *tc.wantCode)
			}
			if string(classified.Code.Output) != tc.wantOutput {
				t.Fatalf("兼容 Output 不符: got %q, want %q", classified.Code.Output, tc.wantOutput)
			}
			if classified.ErrorMessage != "" {
				t.Fatalf("归类结果不得携带调用方文本: %q", classified.ErrorMessage)
			}
			if errors.Is(classified, context.Canceled) || errors.Is(classified, context.DeadlineExceeded) {
				t.Fatal("归类结果不得伪造 context.Canceled 或 context.DeadlineExceeded 原因")
			}

			// 原生状态本身必须逐字节保持不变。
			if rawAfter := statusBytes(t, native); !bytes.Equal(rawBefore, rawAfter) {
				t.Fatal("归类过程改写了原生状态或其 Details")
			}

			// 根因必须能恢复原状态：文案、状态码与 Details 顺序及字节全部一致。
			recovered := nativeStatusFromError(t, classified)
			if recovered.Code() != tc.native {
				t.Fatalf("恢复出的原生状态码不符: got %v, want %v", recovered.Code(), tc.native)
			}
			if recovered.Message() != "原生显式文案" {
				t.Fatalf("恢复出的原生文案不符: %q", recovered.Message())
			}
			if rawRecovered := statusBytes(t, recovered); !bytes.Equal(rawBefore, rawRecovered) {
				t.Fatal("恢复出的原生状态与原状态字节不一致，Details 顺序或编码被改变")
			}
			if details := recovered.Details(); len(details) != 2 {
				t.Fatalf("原生 Details 数量不符: got %d, want 2", len(details))
			}
		})
	}
}

// TestClassifyStatus_NilAndOK 验证没有失败时不制造任何错误信封。
func TestClassifyStatus_NilAndOK(t *testing.T) {
	if classified := grpcresult.ClassifyStatus(nil, control.PhaseOperation); classified != nil {
		t.Fatalf("nil 状态必须归类为 nil: %+v", classified)
	}
	if classified := grpcresult.ClassifyStatus(status.New(codes.OK, ""), control.PhaseOperation); classified != nil {
		t.Fatalf("OK 状态必须归类为 nil: %+v", classified)
	}
}

// TestClassifyStatus_DeadlinePhase 验证超时只依据显式阶段取码：
// 默认阶段与未登记阶段归 504，明确请求接收/读取阶段才归 408。
func TestClassifyStatus_DeadlinePhase(t *testing.T) {
	tests := []struct {
		name     string
		phase    control.Phase
		wantCode *apperr.ErrorCode
	}{
		{"默认阶段", control.PhaseOperation, control.DeadlineExceeded},
		{"请求读取阶段", control.PhaseRequestRead, control.RequestTimeout},
		{"未登记阶段", control.Phase(200), control.DeadlineExceeded},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			classified := grpcresult.ClassifyStatus(status.New(codes.DeadlineExceeded, "超时"), tc.phase)
			if classified == nil {
				t.Fatal("超时状态必须归类为非 nil 统一错误")
			}
			if classified.Code != *tc.wantCode {
				t.Fatalf("超时归类不符: got %+v, want %+v", classified.Code, *tc.wantCode)
			}
		})
	}
}

// TestClassifyStatus_NonStandardCode 验证非标准状态码保守归类为未知错误，
// 且原状态仍作为根因保留；该结果不在 16 态承诺内。
func TestClassifyStatus_NonStandardCode(t *testing.T) {
	native := status.New(codes.Code(99), "非标准状态")
	classified := grpcresult.ClassifyStatus(native, control.PhaseOperation)
	if classified == nil {
		t.Fatal("非标准状态必须保守归类为非 nil 统一错误")
	}
	if classified.Code != *grpcresult.UnknownError {
		t.Fatalf("非标准状态归类不符: got %+v, want %+v", classified.Code, *grpcresult.UnknownError)
	}
	recovered := nativeStatusFromError(t, classified)
	if recovered.Code() != codes.Code(99) {
		t.Fatalf("保留的原状态码不符: got %v, want 99", recovered.Code())
	}
}

// TestClassifyStatus_DoesNotCollapseDistinctStates 验证语义相近但不同的原生状态
// 不会被折叠成同一个兼容码：资源耗尽不等于请求过多，前置条件未满足不等于无效状态。
func TestClassifyStatus_DoesNotCollapseDistinctStates(t *testing.T) {
	resourceExhausted := grpcresult.ClassifyStatus(status.New(codes.ResourceExhausted, ""), control.PhaseOperation)
	failedPrecondition := grpcresult.ClassifyStatus(status.New(codes.FailedPrecondition, ""), control.PhaseOperation)
	if resourceExhausted.Code == *grpcresult.TooManyRequests {
		t.Fatal("资源耗尽不得等同请求过多")
	}
	if failedPrecondition.Code == *grpcresult.InvalidState {
		t.Fatal("前置条件未满足不得等同无效状态")
	}
	if resourceExhausted.Code == failedPrecondition.Code {
		t.Fatal("资源耗尽与前置条件未满足必须使用不同兼容码")
	}
}

// TestClassifyStatus_KeepsAbortedOutOfRangeDataLoss 验证 Aborted、OutOfRange、DataLoss
// 保留各自语义，不会被折叠成已存在、参数错误或内部错误。
func TestClassifyStatus_KeepsAbortedOutOfRangeDataLoss(t *testing.T) {
	aborted := grpcresult.ClassifyStatus(status.New(codes.Aborted, ""), control.PhaseOperation)
	outOfRange := grpcresult.ClassifyStatus(status.New(codes.OutOfRange, ""), control.PhaseOperation)
	dataLoss := grpcresult.ClassifyStatus(status.New(codes.DataLoss, ""), control.PhaseOperation)

	if aborted.Code != *grpcresult.Aborted || aborted.Code == *grpcresult.Conflict {
		t.Fatalf("Aborted 语义被改变: %+v", aborted.Code)
	}
	if outOfRange.Code != *grpcresult.OutOfRange || outOfRange.Code == *grpcresult.ParameterError {
		t.Fatalf("OutOfRange 语义被改变: %+v", outOfRange.Code)
	}
	if dataLoss.Code != *grpcresult.DataLoss || dataLoss.Code == *grpcresult.InternalError {
		t.Fatalf("DataLoss 语义被改变: %+v", dataLoss.Code)
	}
}

// TestClassifyStatus_ResultMustNotReplaceNativeStatus 验证兼容码按 HTTP 分组正向映射，
// 因此归类结果不得回转替换原生状态：Aborted、OutOfRange、DataLoss 都会因此变义。
// 本测试固化该边界，防止后续误用。
func TestClassifyStatus_ResultMustNotReplaceNativeStatus(t *testing.T) {
	aborted := grpcresult.ClassifyStatus(status.New(codes.Aborted, ""), control.PhaseOperation)
	replaced := status.Convert(grpcresult.ToStatusError(context.Background(), aborted))
	if replaced.Code() != codes.AlreadyExists {
		t.Fatalf("兼容码正向映射预期为 AlreadyExists: got %v", replaced.Code())
	}
	if replaced.Code() == codes.Aborted {
		t.Fatal("归类结果经 ToStatusError 回转后不得保留 Aborted 语义")
	}
}
