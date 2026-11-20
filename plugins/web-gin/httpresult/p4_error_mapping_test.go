package httpresult_test

import (
	"encoding/json"
	"errors"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/envelope"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/httpresult"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
)

// 五项协议输入夹具逐字对应受控词表；不引入跨插件运行依赖。
func TestP4DatabaseCodesOverHTTP(t *testing.T) {
	for _, code := range []apperr.ErrorCode{
		{Code: 50033, Output: "DATABASE_DIALECT_DRIVER_INVALID", Message: "数据库驱动声明非法"},
		{Code: 50034, Output: "DATABASE_DIALECT_DRIVER_NOT_LOADED", Message: "数据库驱动未加载"},
		{Code: 50035, Output: "DATABASE_DIALECT_ADAPTER_INVALID", Message: "数据库适配器无效"},
		{Code: 50036, Output: "DATABASE_DIALECT_ADAPTER_CONFIG_INVALID", Message: "数据库适配器配置非法"},
		{Code: 50037, Output: "DATABASE_MODEL_INVALID", Message: "数据库模型契约无效"},
	} {
		t.Run(string(code.Output), func(t *testing.T) {
			e := apperr.New(&code, "数据库操作失败", errors.New("dsn-secret"))
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			httpresult.Error(c, e)
			var response envelope.Response
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if w.Code != 500 || response.Code != code.Code || response.Output != code.Output || response.Message != code.Message {
				t.Fatalf("incorrect envelope: %d %+v", w.Code, response)
			}
			if strings.Contains(w.Body.String(), "dsn-secret") {
				t.Fatal("native cause leaked")
			}
		})
	}
}
