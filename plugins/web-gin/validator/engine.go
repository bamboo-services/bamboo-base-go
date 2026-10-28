package validator

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	v10 "github.com/go-playground/validator/v10"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
)

// Engine 封装独立的 validator.Validate 实例与其私有中文翻译器。
// 每实例完全独立，不持有任何全局状态或 global validator/extractor。
type Engine struct {
	validate *v10.Validate
	trans    ut.Translator
}

// New 构造新的独立验证器引擎，初始化私有 v10 与中文 translator，并注册内置规则与翻译。
func New() (*Engine, error) {
	v := v10.New()
	v.SetTagName("binding")

	zhLocale := zh.New()
	uni := ut.New(zhLocale, zhLocale)
	translator, found := uni.GetTranslator("zh")
	if !found {
		return nil, errors.New("validator: 未找到 zh 语言包翻译器")
	}

	if err := zh_translations.RegisterDefaultTranslations(v, translator); err != nil {
		return nil, fmt.Errorf("validator: 注册默认中文翻译失败: %w", err)
	}

	// 注册字段名提取：优先使用 label tag，其次使用 json tag，若为 "-" 则忽略
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := fld.Tag.Get("label")
		if name == "" {
			name = fld.Tag.Get("json")
		}
		if name == "-" {
			return ""
		}
		return name
	})

	engine := &Engine{
		validate: v,
		trans:    translator,
	}

	// 注册 8 个内置自定义规则与其对应中文翻译
	if err := engine.registerBuiltinCustomRules(); err != nil {
		return nil, err
	}

	return engine, nil
}

// Validate 对目标结构体或数据进行校验。
func (e *Engine) Validate(obj any) error {
	if e == nil || e.validate == nil {
		return errors.New("validator: 验证器未初始化")
	}
	return e.validate.Struct(obj)
}

// RegisterValidation 为当前实例注册自定义验证规则。
// 规则在使用前注册，不承诺运行中改规则的并发安全性。
func (e *Engine) RegisterValidation(tag string, fn v10.Func) error {
	if e == nil || e.validate == nil {
		return errors.New("validator: 验证器未初始化")
	}
	return e.validate.RegisterValidation(tag, fn)
}

// TranslateError 将验证错误转换为字段到中文错误文案的映射。
func (e *Engine) TranslateError(err error) map[string]string {
	result := make(map[string]string)
	if err == nil || e == nil || e.trans == nil {
		return result
	}

	var valErrs v10.ValidationErrors
	if !errors.As(err, &valErrs) {
		return result
	}

	for _, fe := range valErrs {
		result[fe.Field()] = fe.Translate(e.trans)
	}
	return result
}

// HandleError 将验证或绑定错误安全转换为 HTTP 响应写回客户端。
// 详细实现见 error.go。
func (e *Engine) HandleError(ctx *gin.Context, err error) {
	e.handleErrorInternal(ctx, err)
}
