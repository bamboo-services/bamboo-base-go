package bind_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/bind"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/validator"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// UserLoginRequest 定义登录请求的参数结构与校验规则。
type UserLoginRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Age      int    `json:"age" binding:"required,min=1"`
}

// SearchQueryRequest 定义查询参数结构体。
type SearchQueryRequest struct {
	Keyword string `form:"keyword" binding:"required"`
	Page    int    `form:"page" binding:"min=1"`
}

// 本示例演示如何通过 bind.Bind 绑定并校验 JSON 请求体。
func ExampleBind() {
	vEngine, err := validator.New()
	if err != nil {
		panic(err)
	}
	binder, err := bind.New(vEngine)
	if err != nil {
		panic(err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	jsonBody := []byte(`{"username":"konata","age":18}`)
	req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewReader(jsonBody))
	c.Request = req

	// 模拟中间件注入 Binder
	bind.Inject(c, binder)

	var loginReq UserLoginRequest
	res := bind.Bind(c, &loginReq).Data()
	if res != nil {
		fmt.Printf("绑定成功: %s, 年龄: %d\n", res.Username, res.Age)
	}

	// Output:
	// 绑定成功: konata, 年龄: 18
}

// 本示例演示如何通过 bind.Bind 绑定 URL Query 查询参数。
func ExampleBind_query() {
	vEngine, err := validator.New()
	if err != nil {
		panic(err)
	}
	binder, err := bind.New(vEngine)
	if err != nil {
		panic(err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/search?keyword=bamboo&page=2", nil)
	c.Request = req

	bind.Inject(c, binder)

	var queryReq SearchQueryRequest
	res := bind.Bind(c, &queryReq).Query()
	if res != nil {
		fmt.Printf("查询关键字: %s, 页码: %d\n", res.Keyword, res.Page)
	}

	// Output:
	// 查询关键字: bamboo, 页码: 2
}
