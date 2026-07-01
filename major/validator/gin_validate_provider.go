// validator 包提供请求验证错误处理
package validator

import (
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// GinValidateProvider 基于 gin/binding 的验证器提供者
//
// 实现 common/validator.ValidateProvider 接口，从 gin 的 binding.Validator 中
// 提取 *validator.Validate 引擎实例，供 TranslateError 在翻译器未初始化时使用。
type GinValidateProvider struct{}

// GetValidate 返回 gin/binding 内部的 validator.Validate 引擎
//
// 如果 binding.Validator 的 Engine 不是 *validator.Validate 类型，返回 nil。
func (g *GinValidateProvider) GetValidate() *validator.Validate {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		return v
	}
	return nil
}
