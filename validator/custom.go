package xVaild

import (
	"github.com/go-playground/validator/v10"
)

// RegisterCustomValidators 注册自定义验证器
func RegisterCustomValidators(validate *validator.Validate) {
	_ = validate.RegisterValidation("strict_url", ValidateURL)
	_ = validate.RegisterValidation("strict_uuid", ValidateUUID)
	_ = validate.RegisterValidation("alphanum_underscore", ValidateAlphanumUnderscore)
}
