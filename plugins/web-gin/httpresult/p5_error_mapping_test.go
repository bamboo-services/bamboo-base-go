package httpresult_test

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/envelope"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/httpresult"
	"github.com/gin-gonic/gin"
)

// 五项 P5 新增协议输入夹具逐字对应受控词表；不引入跨插件运行依赖。
func TestP5PluginCodesOverHTTP(t *testing.T) {
	tests := []struct {
		code       apperr.ErrorCode
		wantStatus int
	}{
		{code: apperr.ErrorCode{Code: 42903, Output: "ASYNC_POOL_FULL", Message: "异步任务池已满"}, wantStatus: 429},
		{code: apperr.ErrorCode{Code: 50038, Output: "CACHE_DRIVER_CODEC_FAILED", Message: "缓存编解码失败"}, wantStatus: 500},
		{code: apperr.ErrorCode{Code: 50039, Output: "CACHE_DRIVER_OPERATION_FAILED", Message: "缓存操作失败"}, wantStatus: 500},
		{code: apperr.ErrorCode{Code: 50040, Output: "CRON_SCHEDULER_ERROR", Message: "定时调度器错误"}, wantStatus: 500},
		{code: apperr.ErrorCode{Code: 50041, Output: "ASYNC_POOL_ERROR", Message: "异步任务池错误"}, wantStatus: 500},
	}
	for _, tc := range tests {
		t.Run(string(tc.code.Output), func(t *testing.T) {
			e := apperr.New(&tc.code, "插件操作失败", errors.New("secret-credential-or-payload"))
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			httpresult.Error(c, e)
			var response envelope.Response
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if w.Code != tc.wantStatus || response.Code != tc.code.Code || response.Output != tc.code.Output || response.Message != tc.code.Message {
				t.Fatalf("incorrect envelope: status=%d resp=%+v", w.Code, response)
			}
			if strings.Contains(w.Body.String(), "secret-credential-or-payload") {
				t.Fatal("native cause leaked")
			}
		})
	}
}
