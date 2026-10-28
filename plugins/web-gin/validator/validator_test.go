package validator_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/envelope"
	validatorPkg "github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/validator"
	"github.com/gin-gonic/gin"
	v10 "github.com/go-playground/validator/v10"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// 验证两实例同 tag 注册相反规则的真实隔离
func TestDualInstanceOppositeRulesIsolation(t *testing.T) {
	e1, err := validatorPkg.New()
	if err != nil {
		t.Fatalf("创建 e1 失败: %v", err)
	}
	e2, err := validatorPkg.New()
	if err != nil {
		t.Fatalf("创建 e2 失败: %v", err)
	}

	// 在两实例分别注册相反的 is_active 规则：e1 要求 == "active"，e2 要求 == "inactive"
	err = e1.RegisterValidation("is_active", func(fl v10.FieldLevel) bool {
		return fl.Field().String() == "active"
	})
	if err != nil {
		t.Fatalf("e1 RegisterValidation 失败: %v", err)
	}

	err = e2.RegisterValidation("is_active", func(fl v10.FieldLevel) bool {
		return fl.Field().String() == "inactive"
	})
	if err != nil {
		t.Fatalf("e2 RegisterValidation 失败: %v", err)
	}

	type Payload struct {
		Status string `binding:"is_active" label:"用户状态"`
	}

	itemActive := Payload{Status: "active"}
	itemInactive := Payload{Status: "inactive"}

	// e1 检验：active 成功，inactive 失败
	if err := e1.Validate(itemActive); err != nil {
		t.Fatalf("e1 应通过 active，实际报错: %v", err)
	}
	if err := e1.Validate(itemInactive); err == nil {
		t.Fatalf("e1 应拒绝 inactive")
	}

	// e2 检验：inactive 成功，active 失败
	if err := e2.Validate(itemInactive); err != nil {
		t.Fatalf("e2 应通过 inactive，实际报错: %v", err)
	}
	if err := e2.Validate(itemActive); err == nil {
		t.Fatalf("e2 应拒绝 active")
	}
}

// 验证内置 8 个规则及 label/json 中文错误转换
func TestBuiltinRulesAndChineseTranslations(t *testing.T) {
	engine, err := validatorPkg.New()
	if err != nil {
		t.Fatalf("创建 engine 失败: %v", err)
	}

	type UserReq struct {
		Homepage    string  `binding:"strict_url" label:"个人主页"`
		TraceID     string  `binding:"strict_uuid" json:"trace_id"`
		Username    string  `binding:"alphanum_underscore" label:"用户名"`
		Code        string  `binding:"regexp=^[A-Z]{3}[0-9]{3}$" label:"授权码"`
		Level       int     `binding:"enum_int=1 2 3" label:"用户等级"`
		Role        string  `binding:"enum_string=admin user guest" label:"用户身份"`
		Score       float64 `binding:"enum_float=1.5 2.5 3.5" label:"评测分数"`
		SnowflakeID string  `binding:"snowflake" label:"雪花编号"`
	}

	// 合法数据
	validReq := UserReq{
		Homepage:    "https://example.com/test",
		TraceID:     "123e4567-e89b-12d3-a456-426614000000",
		Username:    "user_123",
		Code:        "ABC123",
		Level:       2,
		Role:        "admin",
		Score:       2.5,
		SnowflakeID: "123456789012345678",
	}
	if err := engine.Validate(validReq); err != nil {
		t.Fatalf("合法请求校验失败: %v", err)
	}

	// 全部非法数据
	invalidReq := UserReq{
		Homepage:    "ftp://example.com",
		TraceID:     "not-a-uuid",
		Username:    "bad!user",
		Code:        "abc_12",
		Level:       99,
		Role:        "hacker",
		Score:       9.9,
		SnowflakeID: "not_a_snowflake_id_123456",
	}

	valErr := engine.Validate(invalidReq)
	if valErr == nil {
		t.Fatalf("非法请求预期报错但通过")
	}

	translated := engine.TranslateError(valErr)

	// 验证 label 优先于 json，且没有 label 时降级 json
	if msg, ok := translated["个人主页"]; !ok || msg == "" {
		t.Errorf("缺少 '个人主页' 翻译: %v", translated)
	}
	if msg, ok := translated["trace_id"]; !ok || msg == "" {
		t.Errorf("缺少 'trace_id' 翻译: %v", translated)
	}
	if msg, ok := translated["用户名"]; !ok || msg == "" {
		t.Errorf("缺少 '用户名' 翻译: %v", translated)
	}
	if msg, ok := translated["授权码"]; !ok || msg == "" {
		t.Errorf("缺少 '授权码' 翻译: %v", translated)
	}
	if msg, ok := translated["用户等级"]; !ok || msg == "" {
		t.Errorf("缺少 '用户等级' 翻译: %v", translated)
	}
	if msg, ok := translated["用户身份"]; !ok || msg == "" {
		t.Errorf("缺少 '用户身份' 翻译: %v", translated)
	}
	if msg, ok := translated["评测分数"]; !ok || msg == "" {
		t.Errorf("缺少 '评测分数' 翻译: %v", translated)
	}
	if msg, ok := translated["雪花编号"]; !ok || msg == "" {
		t.Errorf("缺少 '雪花编号' 翻译: %v", translated)
	}
}

// 验证 HandleError 安全响应且不暴露底层 err
func TestHandleErrorSafety(t *testing.T) {
	engine, err := validatorPkg.New()
	if err != nil {
		t.Fatalf("创建 engine 失败: %v", err)
	}

	type DemoReq struct {
		Name string `binding:"required" label:"姓名"`
	}

	valErr := engine.Validate(DemoReq{})
	if valErr == nil {
		t.Fatalf("必须产生校验错误")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	c.Request = req

	engine.HandleError(c, valErr)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("预期 400 状态码，实际: %d", w.Code)
	}

	var resp envelope.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应反序列化失败: %v, body: %s", err, w.Body.String())
	}

	if resp.Code != 40014 {
		t.Fatalf("预期错误码 40014，实际: %d", resp.Code)
	}
	if resp.ErrorMessage == "" {
		t.Fatalf("预期包含 ErrorMessage")
	}

	// 确认底层内部细节不泄露
	if resp.ErrorMessage == "EOF" || resp.ErrorMessage == "demo error" {
		t.Fatalf("泄露了底层错误")
	}

	// 验证反序列化绑定非 ValidationErrors 错误
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	req2, _ := http.NewRequest(http.MethodPost, "/test", nil)
	c2.Request = req2

	engine.HandleError(c2, errors.New("unexpected end of JSON input"))
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("绑定错误预期 400 状态码，实际: %d", w2.Code)
	}
}

// 验证并发 Validate 时的安全无 race
func TestConcurrentValidateRace(t *testing.T) {
	engine, err := validatorPkg.New()
	if err != nil {
		t.Fatalf("创建 engine 失败: %v", err)
	}

	type ConcurrentReq struct {
		Site string `binding:"strict_url" label:"站点"`
		UID  string `binding:"strict_uuid" label:"唯一标识"`
		Num  int    `binding:"enum_int=10 20 30" label:"编号"`
	}

	const goroutines = 50
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				var req ConcurrentReq
				if (idx+j)%2 == 0 {
					req = ConcurrentReq{
						Site: "https://example.com/site",
						UID:  "123e4567-e89b-12d3-a456-426614000000",
						Num:  20,
					}
					if err := engine.Validate(req); err != nil {
						t.Errorf("应校验通过: %v", err)
						return
					}
				} else {
					req = ConcurrentReq{
						Site: "not-url",
						UID:  "bad-uuid",
						Num:  99,
					}
					if err := engine.Validate(req); err == nil {
						t.Errorf("应校验失败")
						return
					}
					_ = engine.TranslateError(errors.New("custom"))
				}
			}
		}(i)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("并发测试超时")
	}
}
