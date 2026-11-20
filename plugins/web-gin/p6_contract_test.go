package webgin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/envelope"
	webgin "github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/bind"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/header"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/httpresult"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/middleware"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/validator"
	"github.com/gin-gonic/gin"
	v10 "github.com/go-playground/validator/v10"
)

// 编译期强类型签名守卫，确保 P6 自有出口不再退回普通 error。
var (
	_ func(*gin.Context) (string, *apperr.Error)                = header.GetAuthorization
	_ func(*validator.Engine) (*bind.Binder, *apperr.Error)     = bind.New
	_ func() (*validator.Engine, *apperr.Error)                 = validator.New
	_ func(*validator.Engine, any) *apperr.Error                = (*validator.Engine).Validate
	_ func(*validator.Engine, string, v10.Func) *apperr.Error   = (*validator.Engine).RegisterValidation
	_ func(func(*validator.Engine) *apperr.Error) webgin.Option = webgin.WithValidatorRegistrar
)

type failingBodyReader struct {
	err error
}

func (f failingBodyReader) Read(_ []byte) (int, error) {
	return 0, f.err
}

func (f failingBodyReader) Close() error {
	return nil
}

func TestP6_WebGin_StrongErrorContracts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("bind.New(nil) 返回 50008 并保留 ErrNilEngine", func(t *testing.T) {
		b, err := bind.New(nil)
		if b != nil {
			t.Fatalf("预期 nil Binder，实际: %+v", b)
		}
		if err == nil {
			t.Fatal("预期返回 *apperr.Error，实际 nil")
		}
		if err.Code.Code != httpresult.InternalError.Code {
			t.Fatalf("期望错误码 %d，实际 %d", httpresult.InternalError.Code, err.Code.Code)
		}
		if !errors.Is(err, bind.ErrNilEngine) {
			t.Fatalf("预期 errors.Is(err, bind.ErrNilEngine) 为 true，实际 false: %v", err)
		}
	})

	t.Run("validator 零值接收器与非结构体输入归 50008，校验失败归 40007 并保留 ValidationErrors", func(t *testing.T) {
		var nilEng *validator.Engine
		if err := nilEng.Validate(struct{}{}); err == nil || err.Code.Code != httpresult.InternalError.Code {
			t.Fatalf("nil Validate 期望 50008，实际: %+v", err)
		}
		if err := nilEng.RegisterValidation("x", func(v10.FieldLevel) bool { return true }); err == nil || err.Code.Code != httpresult.InternalError.Code {
			t.Fatalf("nil RegisterValidation 期望 50008，实际: %+v", err)
		}

		eng, err := validator.New()
		if err != nil {
			t.Fatalf("validator.New 失败: %v", err)
		}
		if err := eng.Validate(123); err == nil || err.Code.Code != httpresult.InternalError.Code {
			t.Fatalf("非结构体 Validate 期望 50008，实际: %+v", err)
		}

		type req struct {
			Name string `binding:"required" label:"名称"`
		}
		valErr := eng.Validate(req{})
		if valErr == nil || valErr.Code.Code != validator.ValidationError.Code {
			t.Fatalf("校验失败期望 40007，实际: %+v", valErr)
		}
		var vErrs v10.ValidationErrors
		if !errors.As(valErr, &vErrs) {
			t.Fatalf("预期保留 v10.ValidationErrors 根因，实际: %v", valErr)
		}
	})

	t.Run("header.GetAuthorization 与 bind 在真实 Gin 链路中的协议输出与脱敏", func(t *testing.T) {
		eng, err := validator.New()
		if err != nil {
			t.Fatalf("validator.New 失败: %v", err)
		}
		b, err := bind.New(eng)
		if err != nil {
			t.Fatalf("bind.New 失败: %v", err)
		}

		router := gin.New()
		router.Use(middleware.Recovery(nil), middleware.Response())

		router.GET("/auth", func(c *gin.Context) {
			tok, appErr := header.GetAuthorization(c)
			if appErr != nil {
				httpresult.AbortError(c, appErr)
				return
			}
			httpresult.SuccessWithData(c, "ok", tok)
		})

		type payload struct {
			Age int `form:"age" json:"age" binding:"required"`
		}

		router.POST("/body", func(c *gin.Context) {
			bind.Inject(c, b)
			var p payload
			if got := bind.Bind(c, &p).Data(); got != nil {
				httpresult.SuccessWithData(c, "ok", got)
			}
		})

		router.POST("/max-bytes", func(c *gin.Context) {
			bind.Inject(c, b)
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4)
			var p payload
			if got := bind.Bind(c, &p).Data(); got != nil {
				httpresult.SuccessWithData(c, "ok", got)
			}
		})

		router.GET("/query", func(c *gin.Context) {
			bind.Inject(c, b)
			var p payload
			if got := bind.Bind(c, &p).Query(); got != nil {
				httpresult.SuccessWithData(c, "ok", got)
			}
		})

		router.GET("/ctx-error/:kind", func(c *gin.Context) {
			switch c.Param("kind") {
			case "cancel":
				_ = c.Error(context.Canceled)
			case "deadline":
				_ = c.Error(context.DeadlineExceeded)
			case "secret":
				_ = c.Error(errors.New("secret_dsn=postgres://admin:pwd@127.0.0.1/db"))
			}
		})

		// 1. GetAuthorization 缺头与格式错 -> 401 + 40100
		for _, authVal := range []string{"", "Basic abc"} {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/auth", nil)
			if authVal != "" {
				req.Header.Set("Authorization", authVal)
			}
			router.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("auth=%q 期望 HTTP 401，实际 %d", authVal, w.Code)
			}
			var resp envelope.Response
			if uErr := json.Unmarshal(w.Body.Bytes(), &resp); uErr != nil {
				t.Fatalf("Unmarshal 失败: %v", uErr)
			}
			if resp.Code != httpresult.Unauthorized.Code {
				t.Fatalf("auth=%q 期望业务码 %d，实际 %d", authVal, httpresult.Unauthorized.Code, resp.Code)
			}
		}

		// 2. readBodyBytes 遇到超时与取消 -> 40802 / 49900（不降成 40006）
		for _, tc := range []struct {
			name       string
			readErr    error
			wantStatus int
			wantCode   apperr.Code
		}{
			{"body_timeout", context.DeadlineExceeded, http.StatusRequestTimeout, control.RequestTimeout.Code},
			{"body_cancel", context.Canceled, 499, control.Canceled.Code},
		} {
			t.Run(tc.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodPost, "/body", nil)
				req.Body = failingBodyReader{err: tc.readErr}
				router.ServeHTTP(w, req)
				if w.Code != tc.wantStatus {
					t.Fatalf("期望 HTTP %d，实际 %d: %s", tc.wantStatus, w.Code, w.Body.String())
				}
				var resp envelope.Response
				if uErr := json.Unmarshal(w.Body.Bytes(), &resp); uErr != nil {
					t.Fatalf("Unmarshal 失败: %v", uErr)
				}
				if resp.Code != tc.wantCode {
					t.Fatalf("期望业务码 %d，实际 %d", tc.wantCode, resp.Code)
				}
			})
		}

		// 3. MaxBytesReader 超限 -> 41300
		wMax := httptest.NewRecorder()
		reqMax := httptest.NewRequest(http.MethodPost, "/max-bytes", bytes.NewBufferString(`{"age":123456}`))
		router.ServeHTTP(wMax, reqMax)
		if wMax.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("期望 HTTP 413，实际 %d: %s", wMax.Code, wMax.Body.String())
		}
		var respMax envelope.Response
		if uErr := json.Unmarshal(wMax.Body.Bytes(), &respMax); uErr != nil {
			t.Fatalf("Unmarshal 失败: %v", uErr)
		}
		if respMax.Code != httpresult.PayloadTooLarge.Code {
			t.Fatalf("期望业务码 %d，实际 %d", httpresult.PayloadTooLarge.Code, respMax.Code)
		}

		// 4. Query 标量类型错误 -> 40005 HTTP_BIND_PARAMETER_ERROR
		wQuery := httptest.NewRecorder()
		reqQuery := httptest.NewRequest(http.MethodGet, "/query?age=notint", nil)
		router.ServeHTTP(wQuery, reqQuery)
		if wQuery.Code != http.StatusBadRequest {
			t.Fatalf("期望 HTTP 400，实际 %d", wQuery.Code)
		}
		var respQuery envelope.Response
		if uErr := json.Unmarshal(wQuery.Body.Bytes(), &respQuery); uErr != nil {
			t.Fatalf("Unmarshal 失败: %v", uErr)
		}
		if respQuery.Code != bind.ParameterError.Code {
			t.Fatalf("期望业务码 %d，实际 %d", bind.ParameterError.Code, respQuery.Code)
		}

		// 5. middleware.Response 对 c.Errors 识别 context 控制流且不泄露原始未知错文本
		for _, tc := range []struct {
			path       string
			wantStatus int
			wantCode   apperr.Code
		}{
			{"/ctx-error/cancel", 499, control.Canceled.Code},
			{"/ctx-error/deadline", http.StatusGatewayTimeout, control.DeadlineExceeded.Code},
			{"/ctx-error/secret", http.StatusInternalServerError, httpresult.InternalError.Code},
		} {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			router.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Fatalf("%s 期望 HTTP %d，实际 %d", tc.path, tc.wantStatus, w.Code)
			}
			bodyStr := w.Body.String()
			if strings.Contains(bodyStr, "secret_dsn") || strings.Contains(bodyStr, "admin:pwd") {
				t.Fatalf("%s 响应体泄露了底层敏感信息: %s", tc.path, bodyStr)
			}
			var resp envelope.Response
			if uErr := json.Unmarshal(w.Body.Bytes(), &resp); uErr != nil {
				t.Fatalf("Unmarshal 失败: %v", uErr)
			}
			if resp.Code != tc.wantCode {
				t.Fatalf("%s 期望业务码 %d，实际 %d", tc.path, tc.wantCode, resp.Code)
			}
		}
	})

	t.Run("未注入 Binder 且无 binder 时 handleBindError 不拼接 raw err.Error()", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/no-binder", io.NopCloser(strings.NewReader("{}")))
		type dummy struct {
			A string
		}
		var d dummy
		// 缺 Binder 路径直接触发 handleBindError
		if got := bind.Bind(c, &d).Data(); got != nil {
			t.Fatalf("预期失败返回 nil，实际 %+v", got)
		}
		if w.Code != http.StatusBadRequest {
			t.Fatalf("期望 HTTP 400，实际 %d", w.Code)
		}
	})

	t.Run("WithValidatorRegistrar 返回 *apperr.Error 时直接阻断 Plugin.Start 并保码保因", func(t *testing.T) {
		sentinel := errors.New("registrar broken")
		_, factory := webgin.New(
			webgin.WithAddr("127.0.0.1:0"),
			webgin.WithValidatorRegistrar(func(_ *validator.Engine) *apperr.Error {
				return apperr.New(httpresult.InvalidState, "规则注册冲突", sentinel)
			}),
		)
		p := factory().(*webgin.Plugin)
		appErr := p.Start(context.Background())
		if appErr == nil {
			t.Fatal("预期 Start 失败")
		}
		if appErr.Code.Code != httpresult.InvalidState.Code {
			t.Fatalf("期望保留错误码 %d，实际 %d", httpresult.InvalidState.Code, appErr.Code.Code)
		}
		if !errors.Is(appErr, sentinel) {
			t.Fatalf("预期保留 sentinel 根因，实际: %v", appErr)
		}
	})
}
