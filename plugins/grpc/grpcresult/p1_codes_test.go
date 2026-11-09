package grpcresult_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/valueerr"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/generate"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/grpcresult"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestP1NewErrorCodesThroughStatusExit 对 P1 新增的 19 个预置码逐项验证 gRPC 出口：
// 错误码三元组、错误码分组映射出的状态码、信封内容与根因隔离都必须成立。
//
// 说明：本表断言的是"预置码自身的正向映射"，与原生 16 态兼容分类是两条独立契约。
// 例如 GRPC_ABORTED（40907）按分组映射为 AlreadyExists，而原生 Aborted 状态
// 在归类契约中仍取 GRPC_ABORTED，二者不得互相期待。
func TestP1NewErrorCodesThroughStatusExit(t *testing.T) {
	tests := []struct {
		name   string
		code   *apperr.ErrorCode
		want   apperr.ErrorCode
		status codes.Code
	}{
		{"control.Canceled", control.Canceled, apperr.ErrorCode{Code: 49900, Output: "CONTROL_CANCELED", Message: "操作已取消"}, codes.Canceled},
		{"control.RequestTimeout", control.RequestTimeout, apperr.ErrorCode{Code: 40802, Output: "CONTROL_REQUEST_TIMEOUT", Message: "请求接收超时"}, codes.DeadlineExceeded},
		{"control.DeadlineExceeded", control.DeadlineExceeded, apperr.ErrorCode{Code: 50401, Output: "CONTROL_DEADLINE_EXCEEDED", Message: "操作超过截止时间"}, codes.DeadlineExceeded},
		{"valueerr.InvalidFormat", valueerr.InvalidFormat, apperr.ErrorCode{Code: 40013, Output: "VALUE_INVALID_FORMAT", Message: "值格式非法"}, codes.InvalidArgument},
		{"valueerr.OutOfRange", valueerr.OutOfRange, apperr.ErrorCode{Code: 40014, Output: "VALUE_OUT_OF_RANGE", Message: "值超出范围"}, codes.InvalidArgument},
		{"valueerr.InvalidType", valueerr.InvalidType, apperr.ErrorCode{Code: 40015, Output: "VALUE_INVALID_TYPE", Message: "值类型非法"}, codes.InvalidArgument},
		{"valueerr.UnsupportedTarget", valueerr.UnsupportedTarget, apperr.ErrorCode{Code: 50018, Output: "VALUE_UNSUPPORTED_TARGET", Message: "不支持的转换目标"}, codes.Internal},
		{"valueerr.InvalidReceiver", valueerr.InvalidReceiver, apperr.ErrorCode{Code: 50019, Output: "VALUE_INVALID_RECEIVER", Message: "值接收器无效"}, codes.Internal},
		{"apperr.FrameworkMultipleFailures", apperr.FrameworkMultipleFailures, apperr.ErrorCode{Code: 50020, Output: "FRAMEWORK_MULTIPLE_FAILURES", Message: "多个框架阶段失败"}, codes.Internal},
		{"apperr.FrameworkHookPanic", apperr.FrameworkHookPanic, apperr.ErrorCode{Code: 50021, Output: "FRAMEWORK_HOOK_PANIC", Message: "框架生命周期钩子异常"}, codes.Internal},
		{"apperr.FrameworkShutdownIncomplete", apperr.FrameworkShutdownIncomplete, apperr.ErrorCode{Code: 50022, Output: "FRAMEWORK_SHUTDOWN_INCOMPLETE", Message: "框架关闭未完成"}, codes.Internal},
		{"apperr.FrameworkWorkerUnexpectedExit", apperr.FrameworkWorkerUnexpectedExit, apperr.ErrorCode{Code: 50023, Output: "FRAMEWORK_WORKER_UNEXPECTED_EXIT", Message: "后台服务意外退出"}, codes.Internal},
		{"grpcresult.OutOfRange", grpcresult.OutOfRange, apperr.ErrorCode{Code: 40016, Output: "GRPC_OUT_OF_RANGE", Message: "参数超出范围"}, codes.InvalidArgument},
		{"grpcresult.Aborted", grpcresult.Aborted, apperr.ErrorCode{Code: 40907, Output: "GRPC_ABORTED", Message: "操作已中止"}, codes.AlreadyExists},
		{"grpcresult.PreconditionFailed", grpcresult.PreconditionFailed, apperr.ErrorCode{Code: 41201, Output: "GRPC_PRECONDITION_FAILED", Message: "前置条件未满足"}, codes.FailedPrecondition},
		{"grpcresult.ResourceExhausted", grpcresult.ResourceExhausted, apperr.ErrorCode{Code: 42902, Output: "GRPC_RESOURCE_EXHAUSTED", Message: "资源耗尽"}, codes.ResourceExhausted},
		{"grpcresult.UnknownError", grpcresult.UnknownError, apperr.ErrorCode{Code: 50024, Output: "GRPC_UNKNOWN_ERROR", Message: "RPC未知错误"}, codes.Internal},
		{"grpcresult.DataLoss", grpcresult.DataLoss, apperr.ErrorCode{Code: 50025, Output: "GRPC_DATA_LOSS", Message: "数据丢失"}, codes.Internal},
		{"grpcresult.Unimplemented", grpcresult.Unimplemented, apperr.ErrorCode{Code: 50100, Output: "GRPC_UNIMPLEMENTED", Message: "功能未实现"}, codes.Unimplemented},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.code == nil || *tc.code != tc.want {
				t.Fatalf("预置错误码被改动: got %+v, want %+v", tc.code, tc.want)
			}

			secret := "secret-internal-detail-" + string(tc.want.Output)
			cause := errors.New(secret)
			appErr := apperr.New(tc.code, "请求失败", cause)
			if appErr.Code != tc.want || !errors.Is(appErr, cause) {
				t.Fatalf("错误码或根因丢失: %+v", appErr)
			}

			st := status.Convert(grpcresult.ToStatusError(context.Background(), appErr))
			if st.Code() != tc.status {
				t.Fatalf("状态码不符: got %v, want %v", st.Code(), tc.status)
			}
			if st.Message() != string(tc.want.Message) {
				t.Fatalf("状态文案不符: got %q, want %q", st.Message(), tc.want.Message)
			}

			details := st.Details()
			if len(details) != 1 {
				t.Fatalf("期望单个信封详情, got %d", len(details))
			}
			resp, ok := details[0].(*generate.BaseResponse)
			if !ok {
				t.Fatalf("详情类型不符: %T", details[0])
			}
			if resp.Code != uint64(tc.want.Code) || resp.Output != string(tc.want.Output) || resp.Message != string(tc.want.Message) {
				t.Fatalf("信封内容不符: %+v", resp)
			}
			if resp.ErrorMessage == nil || *resp.ErrorMessage != "请求失败" {
				t.Fatalf("安全补充描述不符: %+v", resp.ErrorMessage)
			}

			// 根因必须既不进入状态文案，也不进入信封。
			if strings.Contains(st.Message(), secret) {
				t.Fatalf("状态文案泄露底层原因: %q", st.Message())
			}
			if strings.Contains(resp.GetMessage(), secret) || (resp.ErrorMessage != nil && strings.Contains(*resp.ErrorMessage, secret)) {
				t.Fatalf("信封泄露底层原因: %+v", resp)
			}
		})
	}
}

// TestP1NewErrorCodesKeepDistinctSemantics 验证新增预置码之间不发生语义塌缩：
// 资源耗尽不等于请求过多，前置条件未满足不等于无效状态，未知错误不等于内部错误。
func TestP1NewErrorCodesKeepDistinctSemantics(t *testing.T) {
	if *grpcresult.ResourceExhausted == *grpcresult.TooManyRequests {
		t.Fatal("资源耗尽不得等同请求过多")
	}
	if *grpcresult.PreconditionFailed == *grpcresult.InvalidState {
		t.Fatal("前置条件未满足不得等同无效状态")
	}
	if *grpcresult.UnknownError == *grpcresult.InternalError {
		t.Fatal("未知错误不得等同内部错误")
	}
	if *grpcresult.Aborted == *grpcresult.Conflict {
		t.Fatal("操作已中止不得等同冲突")
	}
	if *grpcresult.OutOfRange == *grpcresult.ParameterError {
		t.Fatal("参数超出范围不得等同参数错误")
	}
	if *grpcresult.DataLoss == *grpcresult.InternalError {
		t.Fatal("数据丢失不得等同内部错误")
	}
	if *grpcresult.Unimplemented == *grpcresult.InternalError {
		t.Fatal("功能未实现不得等同内部错误")
	}
}
