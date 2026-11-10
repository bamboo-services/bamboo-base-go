// Package positivepluginhook 验证强类型生命周期契约的正例可以正常编译。
//
// 正例用于排除"编译失败源自依赖或环境问题"的可能：
// 一旦正例编译失败，说明工具链不可用，负例的失败就不能作为类型契约证据。
package positivepluginhook

import (
	"context"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// StrongPlugin 的全部生命周期钩子都以强类型错误表达失败。
type StrongPlugin struct{}

// Start 启动钩子：成功返回真正的 nil 指针。
func (p *StrongPlugin) Start(ctx context.Context) *apperr.Error { return nil }

// Stop 停止钩子：成功返回真正的 nil 指针。
func (p *StrongPlugin) Stop(ctx context.Context) *apperr.Error { return nil }

// Run 执行钩子：成功返回真正的 nil 指针。
func (p *StrongPlugin) Run(ctx context.Context) *apperr.Error { return nil }

// 编译期断言：插件实现必须满足内核的强类型契约。
var (
	_ application.Plugin         = (*StrongPlugin)(nil)
	_ application.RunnablePlugin = (*StrongPlugin)(nil)
)
