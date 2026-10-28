package bind_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/bind"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/validator"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type evilValidator struct{}

func (e *evilValidator) ValidateStruct(any) error {
	panic("evil global validator called: ShouldBind or gin.binding.Validator must NOT be used")
}

func (e *evilValidator) Engine() any {
	return nil
}

type UserRequest struct {
	Name      string    `json:"name" form:"name" binding:"required,min=2"`
	Age       int       `json:"age" form:"age" binding:"min=1"`
	Tags      []string  `json:"tags" form:"tags"`
	CreatedAt time.Time `json:"created_at" form:"created_at" time_format:"2006-01-02"`
}

type URIParam struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type HeaderParam struct {
	Token string `header:"X-Token" binding:"required"`
}

func setupTestContext(method, target string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, target, bytes.NewBuffer(body))
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	ctx.Request = req
	return ctx, w
}

func TestBindWithNoGlobalValidatorLeak(t *testing.T) {
	// 注入恶意 Gin 全局 validator，证明完全不经过全局验证器
	oldValidator := binding.Validator
	binding.Validator = &evilValidator{}
	defer func() {
		binding.Validator = oldValidator
	}()

	engine, err := validator.New()
	if err != nil {
		t.Fatalf("validator.New() failed: %v", err)
	}

	binder, err := bind.New(engine)
	if err != nil {
		t.Fatalf("bind.New() failed: %v", err)
	}

	t.Run("JSON Body 正常解析与独立校验", func(t *testing.T) {
		expectedTime, _ := time.Parse(time.RFC3339, "2026-10-27T00:00:00Z")
		body, _ := json.Marshal(map[string]any{
			"name":       "Alice",
			"age":        18,
			"tags":       []string{"go", "gin"},
			"created_at": "2026-10-27T00:00:00Z",
		})
		ctx, w := setupTestContext(http.MethodPost, "/test", body)
		bind.Inject(ctx, binder)

		var req UserRequest
		res := bind.Bind(ctx, &req).Data()
		if res == nil {
			t.Fatalf("expected successful bind, got nil. response: %s", w.Body.String())
		}
		if res.Name != "Alice" || res.Age != 18 || len(res.Tags) != 2 || !res.CreatedAt.Equal(expectedTime) {
			t.Errorf("bound data mismatch: %+v", res)
		}
	})

	t.Run("JSON Body 字段缺失触发验证失败", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"age": 18,
		})
		ctx, w := setupTestContext(http.MethodPost, "/test", body)
		bind.Inject(ctx, binder)

		var req UserRequest
		res := bind.Bind(ctx, &req).Data()
		if res != nil {
			t.Errorf("expected validation failure, got: %+v", res)
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 status code, got %d", w.Code)
		}
	})

	t.Run("Query 参数解析支持切片与时间", func(t *testing.T) {
		ctx, _ := setupTestContext(http.MethodGet, "/test?name=Bob&age=20&tags=tag1&tags=tag2&created_at=2026-10-27", nil)
		bind.Inject(ctx, binder)

		var req UserRequest
		res := bind.Bind(ctx, &req).Query()
		if res == nil {
			t.Fatalf("expected query bind success, got nil")
		}
		if res.Name != "Bob" || len(res.Tags) != 2 {
			t.Errorf("query bind mismatch: %+v", res)
		}
	})

	t.Run("URI 路径参数绑定", func(t *testing.T) {
		ctx, _ := setupTestContext(http.MethodGet, "/users/123", nil)
		ctx.Params = gin.Params{{Key: "id", Value: "123"}}
		bind.Inject(ctx, binder)

		var uri URIParam
		res := bind.Bind(ctx, &uri).URI()
		if res == nil || res.ID != 123 {
			t.Fatalf("expected URI param id=123, got %+v", res)
		}
	})

	t.Run("Header 参数绑定", func(t *testing.T) {
		ctx, _ := setupTestContext(http.MethodGet, "/test", nil)
		ctx.Request.Header.Set("X-Token", "secret123")
		bind.Inject(ctx, binder)

		var h HeaderParam
		res := bind.Bind(ctx, &h).Header()
		if res == nil || res.Token != "secret123" {
			t.Fatalf("expected header token bound, got %+v", res)
		}
	})

	t.Run("未注入 Binder 明确失败不偷建全局引擎", func(t *testing.T) {
		ctx, w := setupTestContext(http.MethodPost, "/test", []byte(`{"name":"Nobody"}`))
		// 注意：未调用 bind.Inject(ctx, binder)
		var req UserRequest
		res := bind.Bind(ctx, &req).Data()
		if res != nil {
			t.Fatalf("expected failure when binder not injected, got %+v", res)
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 status code, got %d", w.Code)
		}
	})
}
