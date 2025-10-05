package xRoute

import (
	"fmt"

	xConsts "github.com/bamboo-services/bamboo-base-go/constants"
	xError "github.com/bamboo-services/bamboo-base-go/error"
	xResult "github.com/bamboo-services/bamboo-base-go/result"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/utility/ctxutil"
	"github.com/gin-gonic/gin"
)

// NoRoute 处理未定义路由的请求。
//
// 当访问的路由未被定义时，该方法会返回一个标准化的 404 响应，包含详细的错误信息。
//
// 参数说明:
//   - ctx: `*gin.Context` 上下文对象，包含请求和响应的信息。
//
// 注意: 此方法用于全局未匹配路由的处理，需通过 `router.NoRoute` 绑定使用。
func NoRoute(ctx *gin.Context) {
	getLogger := xCtxUtil.GetLogger(ctx, xConsts.LogROUT)

	getLogger.Warn("未找到路由")
	xResult.Error(
		ctx, xError.PageNotFound,
		xError.ErrMessage(fmt.Sprintf(
			"页面「 [%s]%s 」不存在，请检查 <路由> 或 <静态资源> 是否正确配置",
			ctx.Request.Method,
			ctx.Request.URL.Path,
		)),
		nil,
	)
}
