package xVaild

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

// ValidateRegexp 验证正则表达式格式
//
// 该函数通过自定义正则表达式验证字符串是否符合指定的模式。
// 正则表达式从 binding tag 的 param 参数中获取。
//
// 使用示例：
//
//	type Request struct {
//	    Username string `binding:"regexp=^[a-zA-Z0-9_]{6,64}$"`
//	    Password string `binding:"regexp=^(?=.*[a-zA-Z])(?=.*[0-9]).{6,}$"`
//	}
//
// 支持的场景：
//   - 用户名格式验证
//   - 密码复杂度验证
//   - 手机号、邮箱等自定义格式验证
//
// 注意：
//   - 正则表达式必须是有效的 Go 正则语法
//   - 如果正则表达式编译失败，验证将返回 false
//   - 空字符串将被视为验证失败（除非正则明确允许）
//   - 正则表达式应在服务端定义，避免性能问题
func ValidateRegexp(fl validator.FieldLevel) bool {
	// 获取正则表达式参数
	regexPattern := fl.Param()
	if regexPattern == "" {
		return false
	}

	// 编译正则表达式
	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		// 正则表达式无效，验证失败
		return false
	}

	// 验证字段值
	fieldValue := fl.Field().String()
	return regex.MatchString(fieldValue)
}
