package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/middleware"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// 本示例演示 CORS 中间件如何拦截并响应 OPTIONS 预检请求。
func ExampleCORS() {
	engine := gin.New()
	engine.Use(middleware.CORS())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/api/data", nil)
	engine.ServeHTTP(w, req)

	fmt.Printf("预检状态码: %d\n", w.Code)
	fmt.Printf("Allow-Origin: %s\n", w.Header().Get("Access-Control-Allow-Origin"))

	// Output:
	// 预检状态码: 204
	// Allow-Origin: *
}

// 本示例演示 Recovery 中间件如何捕获下游 panic 并转换为安全的 500 响应。
func ExampleRecovery() {
	engine := gin.New()
	engine.Use(middleware.Recovery(nil))
	engine.GET("/panic", func(c *gin.Context) {
		panic("数据库连接突然中断")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
	engine.ServeHTTP(w, req)

	fmt.Printf("HTTP 状态码: %d\n", w.Code)

	// Output:
	// HTTP 状态码: 500
}

// 本示例演示 Response 中间件在业务处理函数遗漏响应时如何自动兜底。
func ExampleResponse() {
	engine := gin.New()
	engine.Use(middleware.Response())
	engine.GET("/empty", func(c *gin.Context) {
		// 开发者遗漏了响应写出
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/empty", nil)
	engine.ServeHTTP(w, req)

	fmt.Printf("HTTP 状态码: %d\n", w.Code)

	// Output:
	// HTTP 状态码: 400
}
