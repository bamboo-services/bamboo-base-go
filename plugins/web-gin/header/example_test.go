package header_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/header"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// 本示例演示从 Gin 上下文中提取 Bearer 认证令牌。
func ExampleGetAuthorization() {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer my-secret-jwt-token")
	c.Request = req

	token, err := header.GetAuthorization(c)
	if err != nil {
		fmt.Printf("提取失败: %v\n", err)
		return
	}
	fmt.Printf("提取成功: %s\n", token)

	// Output:
	// 提取成功: my-secret-jwt-token
}

// 本示例演示提取带或不带 Bearer 前缀的 Token 字符串。
func ExampleGetToken() {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/refresh", nil)
	req.Header.Set(header.HeaderRefreshToken.String(), "Bearer refresh-token-xyz")
	c.Request = req

	refreshToken := header.GetToken(c, header.HeaderRefreshToken)
	fmt.Printf("刷新令牌: %s\n", refreshToken)

	// Output:
	// 刷新令牌: refresh-token-xyz
}

// 本示例演示判定标头是否属于自定义扩展标头。
func ExampleHeader_IsCustom() {
	fmt.Printf("HeaderRequestUUID 是否自定义: %t\n", header.HeaderRequestUUID.IsCustom())
	fmt.Printf("HeaderContentType 是否自定义: %t\n", header.HeaderContentType.IsCustom())

	// Output:
	// HeaderRequestUUID 是否自定义: true
	// HeaderContentType 是否自定义: false
}
