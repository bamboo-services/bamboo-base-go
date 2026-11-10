// Package negativeplugininterface 验证原生 error 签名的插件无法满足内核契约。
package negativeplugininterface

import (
	"context"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
)

// NativePlugin 用原生 error 表达生命周期钩子的失败结果。
type NativePlugin struct{}

// Start 使用原生 error 签名。
func (p *NativePlugin) Start(ctx context.Context) error { return nil }

// Stop 使用原生 error 签名。
func (p *NativePlugin) Stop(ctx context.Context) error { return nil }

// Run 使用原生 error 签名。
func (p *NativePlugin) Run(ctx context.Context) error { return nil }

// 编译期断言：原生签名的插件必须因方法签名不匹配而编译失败。
var _ application.Plugin = (*NativePlugin)(nil)
