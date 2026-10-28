package validator_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/validator"
	"github.com/gin-gonic/gin"
)

type userReq struct {
	Password string `json:"password" binding:"required,min=8"`
	Token    string `json:"token" binding:"min=10"`
	Secret   string `json:"secret" binding:"min=6"`
}

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSecurity_ValidationValueNotLeaked(t *testing.T) {
	engine, err := validator.New()
	if err != nil {
		t.Fatalf("failed to create validator engine: %v", err)
	}
	router := gin.New()

	router.POST("/test", func(c *gin.Context) {
		var req userReq
		if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
			engine.HandleError(c, err)
			return
		}
		if err := engine.Validate(req); err != nil {
			engine.HandleError(c, err)
			return
		}
		c.Status(http.StatusOK)
	})

	body := `{"password":"pw_123","token":"short","secret":"123"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	respBody := w.Body.String()

	// 敏感原值绝不能出现在 JSON 响应中
	if strings.Contains(respBody, "pw_123") {
		t.Fatalf("password value leaked in response: %s", respBody)
	}
	if strings.Contains(respBody, "short") {
		t.Fatalf("token value leaked in response: %s", respBody)
	}
	if strings.Contains(respBody, "123") && strings.Contains(respBody, `"value":`) {
		t.Fatalf("secret value leaked in response: %s", respBody)
	}

	// 字段名与安全中文字段错误仍保留
	if !strings.Contains(respBody, "password") || !strings.Contains(respBody, "token") {
		t.Fatalf("expected fields missing in response: %s", respBody)
	}
}

func TestSecurity_UnderlyingNetworkErrorNotLeaked(t *testing.T) {
	engine, err := validator.New()
	if err != nil {
		t.Fatalf("failed to create validator engine: %v", err)
	}
	router := gin.New()

	underlyingErr := errors.New("dial tcp 10.0.0.12:5432: i/o timeout connect internal database failure")

	var capturedAppErr *apperr.Error
	router.POST("/internal-err", func(c *gin.Context) {
		engine.HandleError(c, underlyingErr)
		// 从上下文中或者如果能获取验证 err
	})

	req := httptest.NewRequest(http.MethodPost, "/internal-err", strings.NewReader(`{}`))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	respBody := w.Body.String()

	// 真实底层网络错误不能出现在客户端 JSON 响应中
	if strings.Contains(respBody, "10.0.0.12") || strings.Contains(respBody, "i/o timeout") || strings.Contains(respBody, "connect internal database failure") {
		t.Fatalf("internal network error leaked in response: %s", respBody)
	}

	// 统一安全文案
	if !strings.Contains(respBody, "请求体解析失败") {
		t.Fatalf("expected safe message '请求体解析失败', got: %s", respBody)
	}

	_ = capturedAppErr
}
