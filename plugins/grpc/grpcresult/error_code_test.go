package grpcresult_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/generate"
	"github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/grpcresult"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestProtocolErrorCodes(t *testing.T) {
	tests := []struct {
		name   string
		code   *apperr.ErrorCode
		want   apperr.ErrorCode
		status codes.Code
	}{
		{"ParameterError", grpcresult.ParameterError, apperr.ErrorCode{Code: 40008, Output: "GRPC_PARAMETER_ERROR", Message: "参数错误"}, codes.InvalidArgument},
		{"ParameterEmpty", grpcresult.ParameterEmpty, apperr.ErrorCode{Code: 40009, Output: "GRPC_PARAMETER_EMPTY", Message: "参数缺失"}, codes.InvalidArgument},
		{"ParameterType", grpcresult.ParameterType, apperr.ErrorCode{Code: 40010, Output: "GRPC_PARAMETER_TYPE", Message: "参数类型错误"}, codes.InvalidArgument},
		{"Disabled", grpcresult.Disabled, apperr.ErrorCode{Code: 40011, Output: "GRPC_DISABLED", Message: "内容已禁用"}, codes.InvalidArgument},
		{"MetadataMissing", grpcresult.MetadataMissing, apperr.ErrorCode{Code: 40012, Output: "GRPC_METADATA_MISSING", Message: "请求元数据缺失"}, codes.InvalidArgument},
		{"Unauthorized", grpcresult.Unauthorized, apperr.ErrorCode{Code: 40101, Output: "GRPC_UNAUTHORIZED", Message: "未授权"}, codes.Unauthenticated},
		{"Forbidden", grpcresult.Forbidden, apperr.ErrorCode{Code: 40303, Output: "GRPC_FORBIDDEN", Message: "禁止访问"}, codes.PermissionDenied},
		{"PermissionDenied", grpcresult.PermissionDenied, apperr.ErrorCode{Code: 40304, Output: "GRPC_PERMISSION_DENIED", Message: "权限不足"}, codes.PermissionDenied},
		{"OperationDenied", grpcresult.OperationDenied, apperr.ErrorCode{Code: 40305, Output: "GRPC_OPERATION_DENIED", Message: "操作被拒绝"}, codes.PermissionDenied},
		{"NotFound", grpcresult.NotFound, apperr.ErrorCode{Code: 40401, Output: "GRPC_NOT_FOUND", Message: "未找到"}, codes.NotFound},
		{"Timeout", grpcresult.Timeout, apperr.ErrorCode{Code: 40801, Output: "GRPC_TIMEOUT", Message: "请求超时"}, codes.DeadlineExceeded},
		{"Conflict", grpcresult.Conflict, apperr.ErrorCode{Code: 40901, Output: "GRPC_CONFLICT", Message: "冲突"}, codes.AlreadyExists},
		{"InvalidState", grpcresult.InvalidState, apperr.ErrorCode{Code: 41200, Output: "GRPC_INVALID_STATE", Message: "无效状态"}, codes.FailedPrecondition},
		{"TooManyRequests", grpcresult.TooManyRequests, apperr.ErrorCode{Code: 42901, Output: "GRPC_TOO_MANY_REQUESTS", Message: "请求过多"}, codes.ResourceExhausted},
		{"InternalError", grpcresult.InternalError, apperr.ErrorCode{Code: 50011, Output: "GRPC_INTERNAL_ERROR", Message: "服务器内部错误"}, codes.Internal},
		{"ResponseMissing", grpcresult.ResponseMissing, apperr.ErrorCode{Code: 50012, Output: "GRPC_RESPONSE_MISSING", Message: "处理器未返回响应"}, codes.Internal},
		{"ContextMissing", grpcresult.ContextMissing, apperr.ErrorCode{Code: 50013, Output: "GRPC_CONTEXT_MISSING", Message: "RPC上下文缺失"}, codes.Internal},
		{"ServiceUnavailable", grpcresult.ServiceUnavailable, apperr.ErrorCode{Code: 50302, Output: "GRPC_SERVICE_UNAVAILABLE", Message: "服务不可用"}, codes.Unavailable},
		{"FrameworkInternalError", apperr.FrameworkInternalError, apperr.ErrorCode{Code: 50000, Output: "FRAMEWORK_INTERNAL_ERROR", Message: "框架内部错误"}, codes.Internal},
		{"FrameworkContainerNotMounted", apperr.FrameworkContainerNotMounted, apperr.ErrorCode{Code: 50001, Output: "FRAMEWORK_CONTAINER_NOT_MOUNTED", Message: "应用容器未挂载"}, codes.Internal},
		{"FrameworkDependencyUndeclared", apperr.FrameworkDependencyUndeclared, apperr.ErrorCode{Code: 50002, Output: "FRAMEWORK_DEPENDENCY_UNDECLARED", Message: "组件依赖未声明"}, codes.Internal},
		{"FrameworkComponentNotFound", apperr.FrameworkComponentNotFound, apperr.ErrorCode{Code: 50003, Output: "FRAMEWORK_COMPONENT_NOT_FOUND", Message: "组件未找到"}, codes.Internal},
		{"FrameworkComponentTypeMismatch", apperr.FrameworkComponentTypeMismatch, apperr.ErrorCode{Code: 50004, Output: "FRAMEWORK_COMPONENT_TYPE_MISMATCH", Message: "组件类型不匹配"}, codes.Internal},
		{"FrameworkComponentStopped", apperr.FrameworkComponentStopped, apperr.ErrorCode{Code: 50005, Output: "FRAMEWORK_COMPONENT_STOPPED", Message: "组件已停止"}, codes.Internal},
		{"FrameworkComponentUnavailable", apperr.FrameworkComponentUnavailable, apperr.ErrorCode{Code: 50006, Output: "FRAMEWORK_COMPONENT_UNAVAILABLE", Message: "组件不可用"}, codes.Internal},
		{"FrameworkComponentInvalidState", apperr.FrameworkComponentInvalidState, apperr.ErrorCode{Code: 50007, Output: "FRAMEWORK_COMPONENT_INVALID_STATE", Message: "组件状态无效"}, codes.Internal},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.code == nil || *tc.code != tc.want {
				t.Fatalf("protocol error code changed: got %+v, want %+v", tc.code, tc.want)
			}
			cause := errors.New("private underlying error")
			appErr := apperr.New(tc.code, "请求失败", cause)
			if appErr.Code != tc.want || !errors.Is(appErr, cause) {
				t.Fatalf("error code or cause was lost: %+v", appErr)
			}

			st := status.Convert(grpcresult.ToStatusError(context.Background(), appErr))
			if st.Code() != tc.status || st.Message() != string(tc.want.Message) {
				t.Fatalf("RPC status changed: %v", st)
			}
			details := st.Details()
			if len(details) != 1 {
				t.Fatalf("expected one response detail, got %d", len(details))
			}
			resp, ok := details[0].(*generate.BaseResponse)
			if !ok || resp.Code != uint64(tc.want.Code) || resp.Output != string(tc.want.Output) || resp.Message != string(tc.want.Message) {
				t.Fatalf("RPC response detail changed: %+v", details[0])
			}
		})
	}
}
