package bind_test

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/bind"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/validator"
	"github.com/gin-gonic/gin"
)

type KeyCaseQueryReq struct {
	Name string `form:"Name"`
}

type KeyCaseFieldReq struct {
	Age int
}

type JSONOnlyTagReq struct {
	Secret string `json:"secret"`
}

type ExactBothKeysReq struct {
	UpperName string `form:"Name"`
	LowerName string `form:"name"`
}

type KeyCaseURIReq struct {
	UserID string `uri:"userId"`
}

type KeyCaseFormReq struct {
	UserName string `form:"userName"`
}

type HeaderCaseReq struct {
	CustomHeader string `header:"x-custom-token"`
}

type ComplexTypesReq struct {
	Tags      []string  `form:"tags"`
	Count     *int      `form:"count"`
	CreatedAt time.Time `form:"created_at" time_format:"2006-01-02"`
}

func initTestBinder(t *testing.T) *bind.Binder {
	t.Helper()
	engine, err := validator.New()
	if err != nil {
		t.Fatalf("validator.New() failed: %v", err)
	}
	binder, err := bind.New(engine)
	if err != nil {
		t.Fatalf("bind.New() failed: %v", err)
	}
	return binder
}

func TestQueryKeyCaseStrictness(t *testing.T) {
	binder := initTestBinder(t)

	t.Run("Query 参数不因错误大小写命中，真正反例无精确键时不回退", func(t *testing.T) {
		// 目标为 Name，传入 ?NAME=a&naME=b，两者均不精确匹配 Name，Query 不应回退匹配
		ctx, _ := setupTestContext(http.MethodGet, "/test?NAME=a&naME=b", nil)
		bind.Inject(ctx, binder)

		var req KeyCaseQueryReq
		res := bind.Bind(ctx, &req).Query()
		if res == nil {
			t.Fatalf("expected bind call to return struct, got nil")
		}
		if res.Name != "" {
			t.Fatalf("expected Name to be empty due to strict case check, got %q", res.Name)
		}
	})

	t.Run("Query 存在精确键 Name 时应确定命中精确键", func(t *testing.T) {
		// 传入 ?Name=exact&name=other，目标 Name 必须稳定获取 exact
		ctx, _ := setupTestContext(http.MethodGet, "/test?Name=exact&name=other", nil)
		bind.Inject(ctx, binder)

		var req KeyCaseQueryReq
		res := bind.Bind(ctx, &req).Query()
		if res == nil {
			t.Fatalf("expected bind call to return struct, got nil")
		}
		if res.Name != "exact" {
			t.Fatalf("expected Name to be %q, got %q", "exact", res.Name)
		}
	})

	t.Run("Query 精确双键 UpperName 与 LowerName 均精确映射且确定", func(t *testing.T) {
		ctx, _ := setupTestContext(http.MethodGet, "/test?Name=upperVal&name=lowerVal", nil)
		bind.Inject(ctx, binder)

		var req ExactBothKeysReq
		res := bind.Bind(ctx, &req).Query()
		if res == nil {
			t.Fatalf("expected bind call to return struct, got nil")
		}
		if res.UpperName != "upperVal" || res.LowerName != "lowerVal" {
			t.Fatalf("mismatch: UpperName=%q, LowerName=%q", res.UpperName, res.LowerName)
		}
	})

	t.Run("Query 无 tag 时按结构体字段名严格匹配", func(t *testing.T) {
		ctx, _ := setupTestContext(http.MethodGet, "/test?age=18&Age=20", nil)
		bind.Inject(ctx, binder)

		var req KeyCaseFieldReq
		res := bind.Bind(ctx, &req).Query()
		if res == nil {
			t.Fatalf("expected bind call to return struct, got nil")
		}
		if res.Age != 20 {
			t.Fatalf("expected Age to strictly match 20, got %d", res.Age)
		}
	})

	t.Run("Query 不借用 json tag 扩大可绑定字段范围", func(t *testing.T) {
		ctx, _ := setupTestContext(http.MethodGet, "/test?secret=my-secret", nil)
		bind.Inject(ctx, binder)

		var req JSONOnlyTagReq
		res := bind.Bind(ctx, &req).Query()
		if res == nil {
			t.Fatalf("expected bind call to return struct, got nil")
		}
		// 只有 json tag 而无 form tag 的 Secret 字段，默认以字段名 Secret 匹配，不应按 secret 绑定
		if res.Secret != "" {
			t.Fatalf("expected Secret to be empty because ShouldBindQuery does not recognize json tag, got %q", res.Secret)
		}
	})
}

func TestURIKeyCaseStrictness(t *testing.T) {
	binder := initTestBinder(t)

	t.Run("URI 路径参数严格区分大小写", func(t *testing.T) {
		ctx, _ := setupTestContext(http.MethodGet, "/test", nil)
		ctx.Params = gin.Params{{Key: "USERID", Value: "1001"}}
		bind.Inject(ctx, binder)

		var req KeyCaseURIReq
		res := bind.Bind(ctx, &req).URI()
		if res == nil {
			t.Fatalf("expected bind call to return struct, got nil")
		}
		if res.UserID != "" {
			t.Fatalf("expected UserID to be empty on URI mismatch, got %q", res.UserID)
		}
	})
}

func TestFormKeyCaseStrictness(t *testing.T) {
	binder := initTestBinder(t)

	t.Run("Form 参数严格区分大小写且不认 json tag", func(t *testing.T) {
		form := url.Values{}
		form.Set("USERNAME", "alice")
		form.Set("secret", "hidden")

		req, _ := http.NewRequest(http.MethodPost, "/test", nil)
		req.PostForm = form

		ctx, _ := setupTestContext(http.MethodPost, "/test", nil)
		ctx.Request = req
		bind.Inject(ctx, binder)

		var reqCase KeyCaseFormReq
		resCase := bind.Bind(ctx, &reqCase).Form()
		if resCase == nil || resCase.UserName != "" {
			t.Fatalf("expected UserName to be empty on mismatch, got %+v", resCase)
		}

		var reqJSON JSONOnlyTagReq
		resJSON := bind.Bind(ctx, &reqJSON).Form()
		if resJSON == nil || resJSON.Secret != "" {
			t.Fatalf("expected Secret not to bind from json tag, got %+v", resJSON)
		}
	})
}

func TestHeaderCaseInsensitive(t *testing.T) {
	binder := initTestBinder(t)

	t.Run("Header 支持 canonical 与大小写不敏感匹配", func(t *testing.T) {
		ctx, _ := setupTestContext(http.MethodGet, "/test", nil)
		ctx.Request.Header.Set("X-Custom-Token", "token-val-123")
		bind.Inject(ctx, binder)

		var req HeaderCaseReq
		res := bind.Bind(ctx, &req).Header()
		if res == nil || res.CustomHeader != "token-val-123" {
			t.Fatalf("expected CustomHeader to match token-val-123, got %+v", res)
		}
	})
}

func TestComplexTypesBinding(t *testing.T) {
	binder := initTestBinder(t)

	t.Run("既有日期、指针与 slice 绑定正常", func(t *testing.T) {
		ctx, _ := setupTestContext(http.MethodGet, "/test?tags=gopher&tags=bamboo&count=5&created_at=2026-10-27", nil)
		bind.Inject(ctx, binder)

		var req ComplexTypesReq
		res := bind.Bind(ctx, &req).Query()
		if res == nil {
			t.Fatalf("expected bind call to return struct, got nil")
		}
		if len(res.Tags) != 2 || res.Tags[0] != "gopher" || res.Tags[1] != "bamboo" {
			t.Fatalf("slice tags mismatch: %+v", res.Tags)
		}
		if res.Count == nil || *res.Count != 5 {
			t.Fatalf("pointer count mismatch: %+v", res.Count)
		}
		expectedDate := time.Date(2026, 10, 27, 0, 0, 0, 0, time.Local)
		if !res.CreatedAt.Equal(expectedDate) {
			t.Fatalf("created_at mismatch: expected %v, got %v", expectedDate, res.CreatedAt)
		}
	})
}
