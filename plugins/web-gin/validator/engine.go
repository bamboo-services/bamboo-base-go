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

// Engine 封装独立的 [*v10.Validate] 校验实例与其私有的中文翻译器。
//
// 每一个 Engine 实例互相完全隔离，不持有任何全局状态或全局单例引用，
// 并自动内置了中文语言环境翻译与 8 项领域自定义规则。
type Engine struct {
	validate *v10.Validate
	trans    ut.Translator
}

// New 构造全新的独立参数验证引擎实例。
//
// 构造过程初始化私有的 [*v10.Validate] 实例与 "zh" 中文翻译器，
// 将结构体验证标签名设为 "binding"，配置字段名提取规则（优先 label，其次 json），
// 并自动注册内置的 8 项领域扩展规则及其中文翻译。
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

// Validate 对传入的目标结构体指针或数据对象执行字段校验。
//
// 若验证器尚未初始化（接收器或内部引擎为 nil），返回错误；
// 若数据校验失败，返回包含违规详情的 [v10.ValidationErrors] 错误；校验通过返回 nil。
// 该方法可被多个 goroutine 并发安全调用。
func (e *Engine) Validate(obj any) error {
	if e == nil || e.validate == nil {
		return errors.New("validator: 验证器未初始化")
	}
	return e.validate.Struct(obj)
}

// RegisterValidation 为当前验证引擎实例注册自定义验证规则函数。
//
// 参数 tag 为验证标签名称，fn 为具体的校验逻辑回调。
// 该方法仅对当前 Engine 实例生效，不影响任何其他引擎。
// 必须在服务初始化单协程阶段完成注册，不承诺运行期并发修改规则的线程安全。
func (e *Engine) RegisterValidation(tag string, fn v10.Func) error {
	if e == nil || e.validate == nil {
		return errors.New("validator: 验证器未初始化")
	}
	return e.validate.RegisterValidation(tag, fn)
}

// TranslateError 将验证失败返回的错误转换为以字段名为键、本地化中文提示为值的键值映射表。
//
// 若传入的 err 为 nil 或非 [v10.ValidationErrors] 类型，返回空 map。
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

// HandleError 将参数绑定或校验错误安全转换为统一 HTTP 错误响应并写回客户端。
//
// 该方法执行本地化翻译、敏感字段值脱敏、构造 [ValidationErrorDetail] 明细列表，
// 并通过 [httpresult.Error] 输出安全的 JSON 信封响应。
func (e *Engine) HandleError(ctx *gin.Context, err error) {
	e.handleErrorInternal(ctx, err)
}
