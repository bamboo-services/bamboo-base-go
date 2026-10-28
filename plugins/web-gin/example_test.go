package webgin_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	webgin "github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/httpresult"
	"github.com/gin-gonic/gin"
)

// 本示例演示如何通过 webgin.New 构造 Gin Web Worker 插件并挂载业务路由。
func ExampleNew() {
	info, factory := webgin.New(
		webgin.WithAddr("127.0.0.1:8080"),
		webgin.WithName("DemoWebService"),
		webgin.WithRoute(func(ctx context.Context, engine *gin.Engine) {
			engine.GET("/ping", func(c *gin.Context) {
				httpresult.Success(c, "pong")
			})
		}),
	)

	fmt.Printf("插件标识: %s, 模式: %d, 名称: %s\n", info.Key, info.Mode, info.Name)
	plugin := factory()
	fmt.Printf("工厂实例化成功: %t\n", plugin != nil)

	// Output:
	// 插件标识: plugin_web_gin, 模式: 2, 名称: DemoWebService
	// 工厂实例化成功: true
}

// 本示例演示 BridgeRequest 如何将普通请求上下文与应用生命周期上下文完成桥接。
func ExampleBridgeRequest() {
	lifetimeCtx := context.Background()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)

	bridged := webgin.BridgeRequest(req.Context(), lifetimeCtx, "req-example-uuid-1234")

	_, ok := bridged.Value(ctxkey.UserStartTimeKey).(time.Time)
	fmt.Printf("注入开始时间: %t\n", ok)

	// Output:
	// 注入开始时间: true
}
