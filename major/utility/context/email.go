package xCtxUtil

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtx "github.com/bamboo-services/bamboo-base-go/defined/context"
	xEmail "github.com/bamboo-services/bamboo-base-go/plugins/email"
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
	// 使用 ContextExtractor 提取标准 context
	stdCtx := ctx
	if globalContextExtractor != nil {
		stdCtx = globalContextExtractor.ExtractRequestContext(ctx)
	}
	if value := stdCtx.Value(xCtx.RegNodeKey); value != nil {
		if nodeList, ok := value.(xCtx.ContextNodeList); ok {
			if component := nodeList.Get(xCtx.EmailClientKey); component != nil {
				if client, ok := component.(*xEmail.EmailClient); ok {
					return client
				}
			}
		}
	}

	value := stdCtx.Value(xCtx.EmailClientKey)
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
func GetEmailClient(ctx context.Context) (*xEmail.EmailClient, *xError.Error) {
	// 使用 ContextExtractor 提取标准 context
	stdCtx := ctx
	if globalContextExtractor != nil {
		stdCtx = globalContextExtractor.ExtractRequestContext(ctx)
	}
	if value := stdCtx.Value(xCtx.RegNodeKey); value != nil {
		if nodeList, ok := value.(xCtx.ContextNodeList); ok {
			if component := nodeList.Get(xCtx.EmailClientKey); component != nil {
				if client, ok := component.(*xEmail.EmailClient); ok {
					return client, nil
				}
			}
		}
	}

	value := stdCtx.Value(xCtx.EmailClientKey)
	if value != nil {
		if client, ok := value.(*xEmail.EmailClient); ok {
			return client, nil
		}
	}
	return nil, &xError.Error{
		ErrorCode:    xError.EmailError,
		ErrorMessage: "在上下文中找不到邮件客户端",
	}
}
