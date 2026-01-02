package xVaild

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
)

var trans ut.Translator

// RegisterTranslator 注册中文翻译器
//
// 该函数为 validator 注册中文翻译支持，使验证错误消息能够显示为中文。
// 同时注册自定义的字段名称翻译和特定验证规则的翻译。
func RegisterTranslator(validate *validator.Validate) error {
	// 创建中文翻译器
	zhLocale := zh.New()
	uni := ut.New(zhLocale, zhLocale)
	trans, _ = uni.GetTranslator("zh")

	// 注册默认的中文翻译
	if err := zh_translations.RegisterDefaultTranslations(validate, trans); err != nil {
		return fmt.Errorf("注册默认中文翻译失败: %w", err)
	}

	// 注册自定义验证规则的翻译
	registerCustomTranslations(validate)

	// 注册字段名称翻译函数
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		// 优先使用 label tag，如果没有则使用 json tag
		name := fld.Tag.Get("label")
		if name == "" {
			name = fld.Tag.Get("json")
		}
		if name == "-" {
			return ""
		}
		return name
	})

	return nil
}

// registerCustomTranslations 注册自定义翻译
func registerCustomTranslations(validate *validator.Validate) {
	// 自定义 oneof 的翻译
	_ = validate.RegisterTranslation("oneof", trans,
		func(ut ut.Translator) error {
			return ut.Add("oneof", "{0}必须是以下值之一: {1}", true)
		},
		func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T("oneof", fe.Field(), fe.Param())
			return t
		},
	)

	// 自定义 strict_url 的翻译
	_ = validate.RegisterTranslation("strict_url", trans,
		func(ut ut.Translator) error {
			return ut.Add("strict_url", "{0}必须是有效的 HTTP 或 HTTPS URL", true)
		},
		func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T("strict_url", fe.Field())
			return t
		},
	)

	// 自定义 strict_uuid 的翻译
	_ = validate.RegisterTranslation("strict_uuid", trans,
		func(ut ut.Translator) error {
			return ut.Add("strict_uuid", "{0}必须是标准的 UUID 格式", true)
		},
		func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T("strict_uuid", fe.Field())
			return t
		},
	)

	// 自定义 alphanum_underscore 的翻译
	_ = validate.RegisterTranslation("alphanum_underscore", trans,
		func(ut ut.Translator) error {
			return ut.Add("alphanum_underscore", "{0}只能包含字母、数字和下划线", true)
		},
		func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T("alphanum_underscore", fe.Field())
			return t
		},
	)
}

// GetTranslator 获取翻译器实例
//
// 返回已初始化的翻译器，用于在其他地方使用验证错误的翻译功能。
func GetTranslator() ut.Translator {
	return trans
}

// TranslateError 翻译验证错误
//
// 将 validator 的验证错误翻译为中文消息。
// 如果翻译器未初始化或翻译失败，则返回默认的错误消息。
func TranslateError(err error) map[string]string {
	result := make(map[string]string)

	// 如果翻译器未初始化，尝试从 binding 获取
	if trans == nil {
		if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
			if err := RegisterTranslator(v); err != nil {
				// 翻译器初始化失败，返回原始错误
				var validationErrors validator.ValidationErrors
				if errors.As(err, &validationErrors) {
					for _, fe := range validationErrors {
						result[fe.Field()] = fe.Error()
					}
				}
				return result
			}
		}
	}

	var validationErrors validator.ValidationErrors
	ok := errors.As(err, &validationErrors)
	if !ok {
		return result
	}

	for _, fe := range validationErrors {
		result[fe.Field()] = fe.Translate(trans)
	}

	return result
}
