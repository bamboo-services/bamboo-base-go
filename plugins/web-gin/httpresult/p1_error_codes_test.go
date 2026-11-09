package httpresult_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/envelope"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/valueerr"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/httpresult"
	"github.com/gin-gonic/gin"
)

// TestP1NewErrorCodesThroughHTTPExit 对 P1 新增的 19 个预置码逐项验证 HTTP 出口：
// 错误码三元组、按错误码分组推导的 HTTP 状态、信封内容与根因隔离都必须成立。
//
// 说明：RPC 域的 7 个新码归属 plugins/grpc/grpcresult，而插件之间禁止为复用错误码
// 而横向依赖（spec/0002），因此这里按受控词表登记的三元组固化 HTTP 侧的出口契约；
// 三元组与词表、源码的一致性由中央守卫与 grpcresult 自身的测试保证。
func TestP1NewErrorCodesThroughHTTPExit(t *testing.T) {
	rpcOutOfRange := &apperr.ErrorCode{Code: 40016, Output: "GRPC_OUT_OF_RANGE", Message: "参数超出范围"}
	rpcAborted := &apperr.ErrorCode{Code: 40907, Output: "GRPC_ABORTED", Message: "操作已中止"}
	rpcPreconditionFailed := &apperr.ErrorCode{Code: 41201, Output: "GRPC_PRECONDITION_FAILED", Message: "前置条件未满足"}
	rpcResourceExhausted := &apperr.ErrorCode{Code: 42902, Output: "GRPC_RESOURCE_EXHAUSTED", Message: "资源耗尽"}
	rpcUnknownError := &apperr.ErrorCode{Code: 50024, Output: "GRPC_UNKNOWN_ERROR", Message: "RPC未知错误"}
	rpcDataLoss := &apperr.ErrorCode{Code: 50025, Output: "GRPC_DATA_LOSS", Message: "数据丢失"}
	rpcUnimplemented := &apperr.ErrorCode{Code: 50100, Output: "GRPC_UNIMPLEMENTED", Message: "功能未实现"}

	tests := []struct {
		name       string
		code       *apperr.ErrorCode
		want       apperr.ErrorCode
		wantStatus int
	}{
		{"control.Canceled", control.Canceled, apperr.ErrorCode{Code: 49900, Output: "CONTROL_CANCELED", Message: "操作已取消"}, 499},
		{"control.RequestTimeout", control.RequestTimeout, apperr.ErrorCode{Code: 40802, Output: "CONTROL_REQUEST_TIMEOUT", Message: "请求接收超时"}, http.StatusRequestTimeout},
		{"control.DeadlineExceeded", control.DeadlineExceeded, apperr.ErrorCode{Code: 50401, Output: "CONTROL_DEADLINE_EXCEEDED", Message: "操作超过截止时间"}, http.StatusGatewayTimeout},
		{"valueerr.InvalidFormat", valueerr.InvalidFormat, apperr.ErrorCode{Code: 40013, Output: "VALUE_INVALID_FORMAT", Message: "值格式非法"}, http.StatusBadRequest},
		{"valueerr.OutOfRange", valueerr.OutOfRange, apperr.ErrorCode{Code: 40014, Output: "VALUE_OUT_OF_RANGE", Message: "值超出范围"}, http.StatusBadRequest},
		{"valueerr.InvalidType", valueerr.InvalidType, apperr.ErrorCode{Code: 40015, Output: "VALUE_INVALID_TYPE", Message: "值类型非法"}, http.StatusBadRequest},
		{"valueerr.UnsupportedTarget", valueerr.UnsupportedTarget, apperr.ErrorCode{Code: 50018, Output: "VALUE_UNSUPPORTED_TARGET", Message: "不支持的转换目标"}, http.StatusInternalServerError},
		{"valueerr.InvalidReceiver", valueerr.InvalidReceiver, apperr.ErrorCode{Code: 50019, Output: "VALUE_INVALID_RECEIVER", Message: "值接收器无效"}, http.StatusInternalServerError},
		{"apperr.FrameworkMultipleFailures", apperr.FrameworkMultipleFailures, apperr.ErrorCode{Code: 50020, Output: "FRAMEWORK_MULTIPLE_FAILURES", Message: "多个框架阶段失败"}, http.StatusInternalServerError},
		{"apperr.FrameworkHookPanic", apperr.FrameworkHookPanic, apperr.ErrorCode{Code: 50021, Output: "FRAMEWORK_HOOK_PANIC", Message: "框架生命周期钩子异常"}, http.StatusInternalServerError},
		{"apperr.FrameworkShutdownIncomplete", apperr.FrameworkShutdownIncomplete, apperr.ErrorCode{Code: 50022, Output: "FRAMEWORK_SHUTDOWN_INCOMPLETE", Message: "框架关闭未完成"}, http.StatusInternalServerError},
		{"apperr.FrameworkWorkerUnexpectedExit", apperr.FrameworkWorkerUnexpectedExit, apperr.ErrorCode{Code: 50023, Output: "FRAMEWORK_WORKER_UNEXPECTED_EXIT", Message: "后台服务意外退出"}, http.StatusInternalServerError},
		{"grpc.OutOfRange", rpcOutOfRange, *rpcOutOfRange, http.StatusBadRequest},
		{"grpc.Aborted", rpcAborted, *rpcAborted, http.StatusConflict},
		{"grpc.PreconditionFailed", rpcPreconditionFailed, *rpcPreconditionFailed, http.StatusPreconditionFailed},
		{"grpc.ResourceExhausted", rpcResourceExhausted, *rpcResourceExhausted, http.StatusTooManyRequests},
		{"grpc.UnknownError", rpcUnknownError, *rpcUnknownError, http.StatusInternalServerError},
		{"grpc.DataLoss", rpcDataLoss, *rpcDataLoss, http.StatusInternalServerError},
		{"grpc.Unimplemented", rpcUnimplemented, *rpcUnimplemented, http.StatusNotImplemented},
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

			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest(http.MethodGet, "/p1-codes", nil)
			c.Request = req

			httpresult.Error(c, appErr)

			if w.Code != tc.wantStatus {
				t.Fatalf("HTTP 状态不符: got %d, want %d", w.Code, tc.wantStatus)
			}

			var resp envelope.Response
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("解析响应信封失败: %v", err)
			}
			if resp.Code != tc.want.Code {
				t.Fatalf("信封错误码不符: got %d, want %d", resp.Code, tc.want.Code)
			}
			if resp.Output != tc.want.Output {
				t.Fatalf("信封 Output 不符: got %q, want %q", resp.Output, tc.want.Output)
			}
			if resp.Message != tc.want.Message {
				t.Fatalf("信封文案不符: got %q, want %q", resp.Message, tc.want.Message)
			}
			if resp.ErrorMessage != "请求失败" {
				t.Fatalf("安全补充描述不符: %q", resp.ErrorMessage)
			}

			// 根因既不进入信封文案，也不进入原始响应体。
			if strings.Contains(w.Body.String(), secret) {
				t.Fatalf("响应体泄露底层原因: %s", w.Body.String())
			}
			if strings.Contains(string(resp.Message), secret) || strings.Contains(string(resp.ErrorMessage), secret) {
				t.Fatalf("信封泄露底层原因: %+v", resp)
			}
		})
	}
}

// TestP1NewErrorCodesKeepDistinctHTTPStatus 验证语义相近但不同的新增预置码
// 在 HTTP 出口上不会塌缩成同一个状态码与同一个标识。
func TestP1NewErrorCodesKeepDistinctHTTPStatus(t *testing.T) {
	if *control.Canceled == *control.RequestTimeout {
		t.Fatal("取消不得等同请求接收超时")
	}
	if *control.RequestTimeout == *control.DeadlineExceeded {
		t.Fatal("请求接收超时不得等同操作超时")
	}
	if *valueerr.InvalidFormat == *valueerr.InvalidType {
		t.Fatal("值格式非法不得等同值类型非法")
	}
	if *valueerr.UnsupportedTarget == *valueerr.InvalidReceiver {
		t.Fatal("不支持的转换目标不得等同值接收器无效")
	}
	if *apperr.FrameworkHookPanic == *apperr.FrameworkWorkerUnexpectedExit {
		t.Fatal("钩子异常不得等同后台服务意外退出")
	}
}
