package httpresult_test

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/envelope"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/httpresult"
	"github.com/gin-gonic/gin"
)

func TestProtocolErrorCodes(t *testing.T) {
	tests := []struct {
		name string
		code *apperr.ErrorCode
		want apperr.ErrorCode
	}{
		{"BadRequest", httpresult.BadRequest, apperr.ErrorCode{Code: 40000, Output: "HTTP_BAD_REQUEST", Message: "错误请求"}},
		{"Disabled", httpresult.Disabled, apperr.ErrorCode{Code: 40001, Output: "HTTP_DISABLED", Message: "内容已禁用"}},
		{"InvalidState", httpresult.InvalidState, apperr.ErrorCode{Code: 40002, Output: "HTTP_INVALID_STATE", Message: "无效状态"}},
		{"ParameterEmpty", httpresult.ParameterEmpty, apperr.ErrorCode{Code: 40003, Output: "HTTP_PARAMETER_EMPTY", Message: "参数缺失"}},
		{"ParameterType", httpresult.ParameterType, apperr.ErrorCode{Code: 40004, Output: "HTTP_PARAMETER_TYPE", Message: "参数类型错误"}},
		{"Unauthorized", httpresult.Unauthorized, apperr.ErrorCode{Code: 40100, Output: "HTTP_UNAUTHORIZED", Message: "未授权"}},
		{"Forbidden", httpresult.Forbidden, apperr.ErrorCode{Code: 40300, Output: "HTTP_FORBIDDEN", Message: "禁止访问"}},
		{"PermissionDenied", httpresult.PermissionDenied, apperr.ErrorCode{Code: 40301, Output: "HTTP_PERMISSION_DENIED", Message: "权限不足"}},
		{"OperationDenied", httpresult.OperationDenied, apperr.ErrorCode{Code: 40302, Output: "HTTP_OPERATION_DENIED", Message: "操作被拒绝"}},
		{"NotFound", httpresult.NotFound, apperr.ErrorCode{Code: 40400, Output: "HTTP_NOT_FOUND", Message: "未找到"}},
		{"MethodNotAllowed", httpresult.MethodNotAllowed, apperr.ErrorCode{Code: 40500, Output: "HTTP_METHOD_NOT_ALLOWED", Message: "方法不允许"}},
		{"NotAcceptable", httpresult.NotAcceptable, apperr.ErrorCode{Code: 40600, Output: "HTTP_NOT_ACCEPTABLE", Message: "不可接受"}},
		{"Timeout", httpresult.Timeout, apperr.ErrorCode{Code: 40800, Output: "HTTP_TIMEOUT", Message: "请求超时"}},
		{"Conflict", httpresult.Conflict, apperr.ErrorCode{Code: 40900, Output: "HTTP_CONFLICT", Message: "冲突"}},
		{"Gone", httpresult.Gone, apperr.ErrorCode{Code: 41000, Output: "HTTP_GONE", Message: "资源已删除"}},
		{"PayloadTooLarge", httpresult.PayloadTooLarge, apperr.ErrorCode{Code: 41300, Output: "HTTP_PAYLOAD_TOO_LARGE", Message: "请求体过大"}},
		{"UnsupportedMedia", httpresult.UnsupportedMedia, apperr.ErrorCode{Code: 41500, Output: "HTTP_UNSUPPORTED_MEDIA", Message: "不支持的媒体类型"}},
		{"UnprocessableEntity", httpresult.UnprocessableEntity, apperr.ErrorCode{Code: 42200, Output: "HTTP_UNPROCESSABLE_ENTITY", Message: "无法处理的实体"}},
		{"TooManyRequests", httpresult.TooManyRequests, apperr.ErrorCode{Code: 42900, Output: "HTTP_TOO_MANY_REQUESTS", Message: "请求过多"}},
		{"InternalError", httpresult.InternalError, apperr.ErrorCode{Code: 50008, Output: "HTTP_INTERNAL_ERROR", Message: "服务器内部错误"}},
		{"ResponseMissing", httpresult.ResponseMissing, apperr.ErrorCode{Code: 50009, Output: "HTTP_RESPONSE_MISSING", Message: "处理器未输出响应"}},
		{"UnknownError", httpresult.UnknownError, apperr.ErrorCode{Code: 50010, Output: "HTTP_UNKNOWN_ERROR", Message: "未知错误"}},
		{"GatewayError", httpresult.GatewayError, apperr.ErrorCode{Code: 50200, Output: "HTTP_GATEWAY_ERROR", Message: "网关错误"}},
		{"ServiceUnavailable", httpresult.ServiceUnavailable, apperr.ErrorCode{Code: 50300, Output: "HTTP_SERVICE_UNAVAILABLE", Message: "服务不可用"}},
		{"ResourceExhausted", httpresult.ResourceExhausted, apperr.ErrorCode{Code: 50301, Output: "HTTP_RESOURCE_EXHAUSTED", Message: "资源耗尽"}},
		{"GatewayTimeout", httpresult.GatewayTimeout, apperr.ErrorCode{Code: 50400, Output: "HTTP_GATEWAY_TIMEOUT", Message: "网关超时"}},
		{"FrameworkInternalError", apperr.FrameworkInternalError, apperr.ErrorCode{Code: 50000, Output: "FRAMEWORK_INTERNAL_ERROR", Message: "框架内部错误"}},
		{"FrameworkContainerNotMounted", apperr.FrameworkContainerNotMounted, apperr.ErrorCode{Code: 50001, Output: "FRAMEWORK_CONTAINER_NOT_MOUNTED", Message: "应用容器未挂载"}},
		{"FrameworkDependencyUndeclared", apperr.FrameworkDependencyUndeclared, apperr.ErrorCode{Code: 50002, Output: "FRAMEWORK_DEPENDENCY_UNDECLARED", Message: "组件依赖未声明"}},
		{"FrameworkComponentNotFound", apperr.FrameworkComponentNotFound, apperr.ErrorCode{Code: 50003, Output: "FRAMEWORK_COMPONENT_NOT_FOUND", Message: "组件未找到"}},
		{"FrameworkComponentTypeMismatch", apperr.FrameworkComponentTypeMismatch, apperr.ErrorCode{Code: 50004, Output: "FRAMEWORK_COMPONENT_TYPE_MISMATCH", Message: "组件类型不匹配"}},
		{"FrameworkComponentStopped", apperr.FrameworkComponentStopped, apperr.ErrorCode{Code: 50005, Output: "FRAMEWORK_COMPONENT_STOPPED", Message: "组件已停止"}},
		{"FrameworkComponentUnavailable", apperr.FrameworkComponentUnavailable, apperr.ErrorCode{Code: 50006, Output: "FRAMEWORK_COMPONENT_UNAVAILABLE", Message: "组件不可用"}},
		{"FrameworkComponentInvalidState", apperr.FrameworkComponentInvalidState, apperr.ErrorCode{Code: 50007, Output: "FRAMEWORK_COMPONENT_INVALID_STATE", Message: "组件状态无效"}},
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

			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			httpresult.Error(ctx, appErr)
			if w.Code != int(tc.want.Code/100) {
				t.Fatalf("HTTP status changed: %d", w.Code)
			}
			var resp envelope.Response
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if resp.Code != tc.want.Code || resp.Output != tc.want.Output || resp.Message != tc.want.Message {
				t.Fatalf("HTTP envelope changed: %+v", resp)
			}
		})
	}
}
