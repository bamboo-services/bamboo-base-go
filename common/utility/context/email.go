package xCtxUtil

import (
	"context"

	error2 "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtx2 "github.com/bamboo-services/bamboo-base-go/defined/context"
	xEmail "github.com/bamboo-services/bamboo-base-go/plugins/email"
	"github.com/gin-gonic/gin"
)

// MustGetEmailClient 从上下文中获取邮件客户端实例（panic 版本）。
//
// 如果上下文中未找到邮件客户端，则记录错误日志并触发 panic。
//
// 参数说明:
//   - ctx: context.Context 上下文
//
// 返回值:
//   - *xEmail.EmailClient: 邮件客户端实例
func MustGetEmailClient(ctx context.Context) *xEmail.EmailClient {
	if ginCtx, ok := ctx.(*gin.Context); ok {
		ctx = ginCtx.Request.Context()
	}
	if value := ctx.Value(xCtx2.RegNodeKey); value != nil {
		if nodeList, ok := value.(xCtx2.ContextNodeList); ok {
			if component := nodeList.Get(xCtx2.EmailClientKey); component != nil {
				if client, ok := component.(*xEmail.EmailClient); ok {
					return client
				}
			}
		}
	}

	value := ctx.Value(xCtx2.EmailClientKey)
	if value != nil {
		if client, ok := value.(*xEmail.EmailClient); ok {
			return client
		}
	}
	xLog.WithName(xLog.NamedUTIL).Error(ctx, "在上下文中找不到邮件客户端，真的注入成功了吗？")
	panic("在上下文中找不到邮件客户端，真的注入成功了吗？")
}

// GetEmailClient 从上下文中获取邮件客户端实例（错误返回版本）。
//
// 如果上下文中未找到邮件客户端，则返回错误而不是 panic。
//
// 参数说明:
//   - ctx: context.Context 上下文
//
// 返回值:
//   - *xEmail.EmailClient: 邮件客户端实例
//   - *xError.Error: 错误信息，成功时为 nil
func GetEmailClient(ctx context.Context) (*xEmail.EmailClient, *error2.Error) {
	if ginCtx, ok := ctx.(*gin.Context); ok {
		ctx = ginCtx.Request.Context()
	}
	if value := ctx.Value(xCtx2.RegNodeKey); value != nil {
		if nodeList, ok := value.(xCtx2.ContextNodeList); ok {
			if component := nodeList.Get(xCtx2.EmailClientKey); component != nil {
				if client, ok := component.(*xEmail.EmailClient); ok {
					return client, nil
				}
			}
		}
	}

	value := ctx.Value(xCtx2.EmailClientKey)
	if value != nil {
		if client, ok := value.(*xEmail.EmailClient); ok {
			return client, nil
		}
	}
	return nil, &error2.Error{
		ErrorCode:    error2.EmailError,
		ErrorMessage: "在上下文中找不到邮件客户端",
	}
}
