package httpresult_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/httpresult"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// 本示例演示在处理函数中写回标准 200 成功响应。
func ExampleSuccess() {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/ping", nil)
	c.Request = req

	httpresult.Success(c, "pong")

	fmt.Printf("HTTP 状态码: %d\n", w.Code)
	fmt.Printf("响应体: %s\n", w.Body.String())

	// Output:
	// HTTP 状态码: 200
	// 响应体: {"code":200,"output":"Success","message":"pong"}
}

// 本示例演示在处理函数中写回带数据载荷的成功响应。
func ExampleSuccessWithData() {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/profile", nil)
	c.Request = req

	data := map[string]string{"name": "konata"}
	httpresult.SuccessWithData(c, "获取成功", data)

	fmt.Printf("HTTP 状态码: %d\n", w.Code)
	fmt.Printf("响应体: %s\n", w.Body.String())

	// Output:
	// HTTP 状态码: 200
	// 响应体: {"code":200,"output":"Success","message":"获取成功","data":{"name":"konata"}}
}

// 本示例演示在遇到业务错误时写回映射状态码与结构化信封。
func ExampleError() {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/item", nil)
	c.Request = req

	err := apperr.New(apperr.RecordNotFound, "指定条目不存在")
	httpresult.Error(c, err)

	fmt.Printf("HTTP 状态码: %d\n", w.Code)
	fmt.Printf("响应体: %s\n", w.Body.String())

	// Output:
	// HTTP 状态码: 404
	// 响应体: {"code":40406,"output":"RECORD_NOT_FOUND","message":"记录未找到","error_message":"指定条目不存在"}
}
