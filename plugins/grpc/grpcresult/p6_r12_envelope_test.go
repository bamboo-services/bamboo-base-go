package grpcresult

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/generate"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestP6_NormalizeSDKError_And_NilOK(t *testing.T) {
	// 1. 真正的 nil 与 typed-nil 均返回 nil
	if got := NormalizeSDKError(nil, control.PhaseOperation); got != nil {
		t.Fatalf("nil 输入期望 nil，实际: %+v", got)
	}
	var typedNil *apperr.Error
	if got := NormalizeSDKError(typedNil, control.PhaseOperation); got != nil {
		t.Fatalf("typed-nil 输入期望 nil，实际: %+v", got)
	}
	if got := NormalizeToStatusError(context.Background(), typedNil, control.PhaseOperation, nil, nil); got != nil {
		t.Fatalf("NormalizeToStatusError(typed-nil) 期望 nil，实际: %v", got)
	}

	// 2. codes.OK 原生状态不制造错误
	if got := NormalizeToStatusError(context.Background(), status.Error(codes.OK, "ok"), control.PhaseOperation, nil, nil); got != nil {
		t.Fatalf("codes.OK 期望 nil，实际: %v", got)
	}
}

func TestP6_R12_All16NonOKStatus_ComplementAndPreserveCode(t *testing.T) {
	t.Setenv(string(env.Debug), "true")

	reqCtx := context.WithValue(context.Background(), ctxkey.RequestKey, "req-r12-trace")
	reqCtx = context.WithValue(reqCtx, ctxkey.UserStartTimeKey, time.Now().Add(-2*time.Millisecond))

	cases := []struct {
		grpcCode   codes.Code
		wantCode   apperr.Code
		wantOutput apperr.Output
	}{
		{codes.Canceled, control.Canceled.Code, control.Canceled.Output},
		{codes.Unknown, UnknownError.Code, UnknownError.Output},
		{codes.InvalidArgument, ParameterError.Code, ParameterError.Output},
		{codes.DeadlineExceeded, control.DeadlineExceeded.Code, control.DeadlineExceeded.Output},
		{codes.NotFound, NotFound.Code, NotFound.Output},
		{codes.AlreadyExists, Conflict.Code, Conflict.Output},
		{codes.PermissionDenied, PermissionDenied.Code, PermissionDenied.Output},
		{codes.ResourceExhausted, ResourceExhausted.Code, ResourceExhausted.Output},
		{codes.FailedPrecondition, PreconditionFailed.Code, PreconditionFailed.Output},
		{codes.Aborted, Aborted.Code, Aborted.Output},
		{codes.OutOfRange, OutOfRange.Code, OutOfRange.Output},
		{codes.Unimplemented, Unimplemented.Code, Unimplemented.Output},
		{codes.Internal, InternalError.Code, InternalError.Output},
		{codes.Unavailable, ServiceUnavailable.Code, ServiceUnavailable.Output},
		{codes.DataLoss, DataLoss.Code, DataLoss.Output},
		{codes.Unauthenticated, Unauthorized.Code, Unauthorized.Output},
	}

	for _, tc := range cases {
		t.Run(tc.grpcCode.String(), func(t *testing.T) {
			explicitMsg := "caller explicit " + tc.grpcCode.String()
			// 模拟外层 fmt.Errorf 包装了敏感文本，不得抄入显式状态文案
			wrapped := fmt.Errorf("wrapper secret_token=abc: %w", status.Error(tc.grpcCode, explicitMsg))

			outErr := NormalizeToStatusError(reqCtx, wrapped, control.PhaseOperation, nil, nil)
			if outErr == nil {
				t.Fatal("预期非 nil status error")
			}

			// 原生 status.Canceled/DeadlineExceeded 不得伪造 context 哨兵
			if tc.grpcCode == codes.Canceled && errors.Is(outErr, context.Canceled) {
				t.Fatal("原生 codes.Canceled 不得伪造 context.Canceled 哨兵")
			}
			if tc.grpcCode == codes.DeadlineExceeded && errors.Is(outErr, context.DeadlineExceeded) {
				t.Fatal("原生 codes.DeadlineExceeded 不得伪造 context.DeadlineExceeded 哨兵")
			}

			st, ok := status.FromError(outErr)
			if !ok {
				t.Fatalf("期望 gRPC status 错误，实际 %T", outErr)
			}
			// 原生 codes.Code 必须严格保持（Aborted/DataLoss/OutOfRange 不得改写）
			if st.Code() != tc.grpcCode {
				t.Fatalf("状态码被改写：期望 %v，实际 %v", tc.grpcCode, st.Code())
			}
			if st.Message() != explicitMsg {
				t.Fatalf("显式文案被改写或外泄：期望 %q，实际 %q", explicitMsg, st.Message())
			}

			details := st.Proto().GetDetails()
			if len(details) != 1 {
				t.Fatalf("期望 1 条 BaseResponse detail，实际 %d", len(details))
			}
			var br generate.BaseResponse
			if err := details[0].UnmarshalTo(&br); err != nil {
				t.Fatalf("BaseResponse 解码失败: %v", err)
			}
			if br.Code != uint64(tc.wantCode) || br.Output != string(tc.wantOutput) {
				t.Fatalf("兼容信封编码错误：got (%d, %q), want (%d, %q)", br.Code, br.Output, tc.wantCode, tc.wantOutput)
			}
			if br.Context != "req-r12-trace" || br.Overhead == nil || *br.Overhead <= 0 {
				t.Fatalf("追踪或耗时缺失: context=%q overhead=%v", br.Context, br.Overhead)
			}
			if br.Message != explicitMsg || br.ErrorMessage != nil {
				t.Fatalf("BaseResponse 文案或 ErrorMessage 不符合预期: msg=%q errMsg=%v", br.Message, br.ErrorMessage)
			}
		})
	}
}

func TestP6_R12_UnknownAnyOrder_ExistingDedup_Conflict_Undecodable_And_Degrade(t *testing.T) {
	reqCtx := context.WithValue(context.Background(), ctxkey.RequestKey, "server-req-id")

	// 1. 保留任意未知 Any 类型顺序并仅在尾部追加一个 BaseResponse
	t.Run("未知 Any 保序保留并追加单个 BaseResponse", func(t *testing.T) {
		unknown1 := &anypb.Any{TypeUrl: "type.googleapis.com/thirdparty.Alpha", Value: []byte{0x08, 0x01}}
		unknown2 := &anypb.Any{TypeUrl: "type.googleapis.com/thirdparty.Beta", Value: []byte{0x12, 0x02, 'h', 'i'}}
		rawProto := &spb.Status{
			Code:    int32(codes.Aborted),
			Message: "optimistic abort",
			Details: []*anypb.Any{unknown1, unknown2},
		}
		outErr := NormalizeToStatusError(reqCtx, status.FromProto(rawProto).Err(), control.PhaseOperation, nil, nil)
		st := status.Convert(outErr)
		if st.Code() != codes.Aborted || st.Message() != "optimistic abort" {
			t.Fatalf("原状态或文案被改写: code=%v msg=%q", st.Code(), st.Message())
		}
		gotDetails := st.Proto().GetDetails()
		if len(gotDetails) != 3 {
			t.Fatalf("期望保留 2 个未知 Any + 1 个 BaseResponse，共 3 条，实际 %d", len(gotDetails))
		}
		if !proto.Equal(gotDetails[0], unknown1) || !proto.Equal(gotDetails[1], unknown2) {
			t.Fatalf("未知 Any 内容或顺序发生改变")
		}
		var br generate.BaseResponse
		if err := gotDetails[2].UnmarshalTo(&br); err != nil {
			t.Fatalf("追加 BaseResponse 解码失败: %v", err)
		}
		if br.Code != uint64(Aborted.Code) || br.Context != "server-req-id" {
			t.Fatalf("兼容信封内容错误: %+v", &br)
		}
	})

	// 2. 已携带 BaseResponse -> 不覆盖、不改 Context、不重复追加
	t.Run("已有 BaseResponse 原样保序不覆盖不追加", func(t *testing.T) {
		existingBR, err := anypb.New(&generate.BaseResponse{
			Context: "caller-original-ctx",
			Output:  string(NotFound.Output),
			Code:    uint64(NotFound.Code),
			Message: "caller msg",
		})
		if err != nil {
			t.Fatal(err)
		}
		unknownTail, _ := anypb.New(wrapperspb.String("tail-meta"))
		rawProto := &spb.Status{
			Code:    int32(codes.NotFound),
			Message: "resource missing",
			Details: []*anypb.Any{existingBR, unknownTail},
		}
		outErr := NormalizeToStatusError(reqCtx, status.FromProto(rawProto).Err(), control.PhaseOperation, nil, nil)
		st := status.Convert(outErr)
		gotDetails := st.Proto().GetDetails()
		if len(gotDetails) != 2 {
			t.Fatalf("不得重复追加 BaseResponse，实际 len=%d", len(gotDetails))
		}
		var br generate.BaseResponse
		if err := gotDetails[0].UnmarshalTo(&br); err != nil {
			t.Fatal(err)
		}
		if br.Context != "caller-original-ctx" {
			t.Fatalf("已有 BaseResponse Context 被改写: %q", br.Context)
		}
		if !proto.Equal(gotDetails[1], unknownTail) {
			t.Fatalf("尾部 Any 被篡改")
		}
	})

	// 3. 多条冲突或不可解码 BaseResponse -> 原样保留原载荷并记录受控诊断
	for _, tc := range []struct {
		name    string
		details []*anypb.Any
	}{
		{
			name: "多条冲突 BaseResponse",
			details: func() []*anypb.Any {
				a1, _ := anypb.New(&generate.BaseResponse{Code: 40401, Output: "A", Context: "c1"})
				a2, _ := anypb.New(&generate.BaseResponse{Code: 50011, Output: "B", Context: "c2"})
				return []*anypb.Any{a1, a2}
			}(),
		},
		{
			name: "不可解码 BaseResponse",
			details: []*anypb.Any{
				{TypeUrl: "type.googleapis.com/" + string(baseResponseFullName), Value: []byte{0xff, 0xff, 0xff}},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rawProto := &spb.Status{
				Code:    int32(codes.FailedPrecondition),
				Message: "precondition",
				Details: tc.details,
			}
			var diag string
			outErr := NormalizeToStatusError(reqCtx, status.FromProto(rawProto).Err(), control.PhaseOperation, nil, func(reason string) {
				diag = reason
			})
			if diag == "" {
				t.Fatal("预期触发受控诊断回调")
			}
			st := status.Convert(outErr)
			if len(st.Proto().GetDetails()) != len(tc.details) {
				t.Fatalf("原载荷条目数被改变: got %d, want %d", len(st.Proto().GetDetails()), len(tc.details))
			}
			for i := range tc.details {
				if !proto.Equal(st.Proto().GetDetails()[i], tc.details[i]) {
					t.Fatalf("第 %d 条原载荷被篡改", i)
				}
			}
		})
	}

	// 4. 非标准 status code 保守兜底 50024 且不改原状态码
	t.Run("非标准状态码保留原码并补齐 50024", func(t *testing.T) {
		rawProto := &spb.Status{Code: 99, Message: "custom code"}
		outErr := NormalizeToStatusError(reqCtx, status.FromProto(rawProto).Err(), control.PhaseOperation, nil, nil)
		st := status.Convert(outErr)
		if st.Code() != codes.Code(99) {
			t.Fatalf("期望原状态码 99，实际 %v", st.Code())
		}
		var br generate.BaseResponse
		if err := st.Proto().GetDetails()[0].UnmarshalTo(&br); err != nil {
			t.Fatal(err)
		}
		if br.Code != uint64(UnknownError.Code) {
			t.Fatalf("期望 50024，实际 %d", br.Code)
		}
	})

	// 5. 补充编码失败时受控降级，保留原 status/Details，不改状态且不泄内部编码错误
	t.Run("补齐编码失败受控降级", func(t *testing.T) {
		origMarshal := marshalStatusDetail
		marshalStatusDetail = func(_ proto.Message) (*anypb.Any, error) {
			return nil, errors.New("marshal secret internal failure")
		}
		defer func() { marshalStatusDetail = origMarshal }()

		unknown := &anypb.Any{TypeUrl: "type.googleapis.com/thirdparty.Keep", Value: []byte{0x01}}
		rawProto := &spb.Status{
			Code:    int32(codes.DataLoss),
			Message: "disk corrupted",
			Details: []*anypb.Any{unknown},
		}
		var diag string
		outErr := NormalizeToStatusError(reqCtx, status.FromProto(rawProto).Err(), control.PhaseOperation, nil, func(reason string) {
			diag = reason
		})
		if diag == "" {
			t.Fatal("预期触发降级诊断")
		}
		st := status.Convert(outErr)
		if st.Code() != codes.DataLoss || st.Message() != "disk corrupted" {
			t.Fatalf("降级后原状态或文案被改写: code=%v msg=%q", st.Code(), st.Message())
		}
		if strings.Contains(st.Message(), "marshal secret") {
			t.Fatalf("降级路径泄漏了内部编码失败文本")
		}
		if len(st.Proto().GetDetails()) != 1 || !proto.Equal(st.Proto().GetDetails()[0], unknown) {
			t.Fatalf("降级路径改写了原 Details")
		}
	})
}

func TestP6_R12_ContextControl_And_UnknownLeakGuard(t *testing.T) {
	reqCtx := context.WithValue(context.Background(), ctxkey.RequestKey, "ctx-trace")

	for _, tc := range []struct {
		name      string
		err       error
		phase     control.Phase
		wantGRPC  codes.Code
		wantCode  apperr.Code
		forbidStr string
	}{
		{
			name:      "context.Canceled 包装脱敏与 49900 信封",
			err:       fmt.Errorf("secret_wrapper_cancel: %w", context.Canceled),
			phase:     control.PhaseOperation,
			wantGRPC:  codes.Canceled,
			wantCode:  control.Canceled.Code,
			forbidStr: "secret_wrapper_cancel",
		},
		{
			name:      "context.DeadlineExceeded 内部操作归 50401",
			err:       fmt.Errorf("secret_wrapper_deadline: %w", context.DeadlineExceeded),
			phase:     control.PhaseOperation,
			wantGRPC:  codes.DeadlineExceeded,
			wantCode:  control.DeadlineExceeded.Code,
			forbidStr: "secret_wrapper_deadline",
		},
		{
			name:      "context.DeadlineExceeded 请求读取阶段归 40802",
			err:       fmt.Errorf("secret_read_timeout: %w", context.DeadlineExceeded),
			phase:     control.PhaseRequestRead,
			wantGRPC:  codes.DeadlineExceeded,
			wantCode:  control.RequestTimeout.Code,
			forbidStr: "secret_read_timeout",
		},
		{
			name:      "未知普通 error 归 50011 且不外泄",
			err:       errors.New("postgres://root:supersecret@127.0.0.1/db"),
			phase:     control.PhaseOperation,
			wantGRPC:  codes.Internal,
			wantCode:  InternalError.Code,
			forbidStr: "supersecret",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outErr := NormalizeToStatusError(reqCtx, tc.err, tc.phase, nil, nil)
			st := status.Convert(outErr)
			if st.Code() != tc.wantGRPC {
				t.Fatalf("期望 %v，实际 %v", tc.wantGRPC, st.Code())
			}
			if strings.Contains(st.Message(), tc.forbidStr) {
				t.Fatalf("status.Message 泄露了禁止文本 %q: %q", tc.forbidStr, st.Message())
			}
			var br generate.BaseResponse
			if err := st.Proto().GetDetails()[0].UnmarshalTo(&br); err != nil {
				t.Fatalf("解码 BaseResponse 失败: %v", err)
			}
			if br.Code != uint64(tc.wantCode) || br.Context != "ctx-trace" {
				t.Fatalf("信封错误: got code=%d ctx=%q, want code=%d", br.Code, br.Context, tc.wantCode)
			}
			if strings.Contains(br.Message, tc.forbidStr) || (br.ErrorMessage != nil && strings.Contains(*br.ErrorMessage, tc.forbidStr)) {
				t.Fatalf("BaseResponse 泄露了禁止文本 %q: %+v", tc.forbidStr, &br)
			}
		})
	}
}
