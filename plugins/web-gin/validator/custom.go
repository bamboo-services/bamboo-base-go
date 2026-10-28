package validator

import (
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	ut "github.com/go-playground/universal-translator"
	v10 "github.com/go-playground/validator/v10"
)

var (
	urlRegex                = regexp.MustCompile(`^https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(/.*)?$`)
	uuidRegex               = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	alphanumUnderscoreRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

// ValidateStrictURL 执行严格的 URL 格式验证。
//
// 仅允许以 "http://" 或 "https://" 开头，且后跟有效域名或路径的 URL 格式。
func ValidateStrictURL(fl v10.FieldLevel) bool {
	return urlRegex.MatchString(fl.Field().String())
}

// ValidateStrictUUID 执行严格的 UUID 格式验证。
//
// 仅匹配符合 RFC 4122 标准的 8-4-4-4-12 格式 UUID 字符串。
func ValidateStrictUUID(fl v10.FieldLevel) bool {
	return uuidRegex.MatchString(fl.Field().String())
}

// ValidateAlphanumUnderscore 验证字段值仅包含字母、数字与下划线。
func ValidateAlphanumUnderscore(fl v10.FieldLevel) bool {
	return alphanumUnderscoreRegex.MatchString(fl.Field().String())
}

// ValidateRegexp 执行基于参数传参的正则表达式验证。
//
// 参数 fl.Param() 必须为合法的正则表达式模式；若模式为空或无法编译，返回 false。
func ValidateRegexp(fl v10.FieldLevel) bool {
	pattern := fl.Param()
	if pattern == "" {
		return false
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(fl.Field().String())
}

// ValidateEnumInt 验证整型字段值是否属于预设的枚举集合。
//
// 验证规则参数为以空格分隔的整数列表（例如 "1 2 3"）。
// 支持各类有符号与无符号整数类型（超出 int64 范围的大无符号数返回 false）。
func ValidateEnumInt(fl v10.FieldLevel) bool {
	param := fl.Param()
	if param == "" {
		return false
	}
	fields := strings.Fields(param)
	if len(fields) == 0 {
		return false
	}

	allowed := make(map[int64]struct{}, len(fields))
	for _, valStr := range fields {
		v, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return false
		}
		allowed[v] = struct{}{}
	}

	field := fl.Field()
	var val int64
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val = field.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u := field.Uint()
		if u > math.MaxInt64 {
			return false
		}
		val = int64(u)
	default:
		return false
	}

	_, ok := allowed[val]
	return ok
}

// ValidateEnumString 验证字符串字段值是否属于预设的枚举集合。
//
// 验证规则参数为以空格分隔的候选字符串列表（例如 "active pending disabled"）。
func ValidateEnumString(fl v10.FieldLevel) bool {
	param := fl.Param()
	if param == "" {
		return false
	}
	fields := strings.Fields(param)
	if len(fields) == 0 {
		return false
	}

	field := fl.Field()
	if field.Kind() != reflect.String {
		return false
	}
	val := field.String()

	for _, item := range fields {
		if item == val {
			return true
		}
	}
	return false
}

// ValidateEnumFloat 验证浮点数字段值是否属于预设的枚举集合。
//
// 验证规则参数为以空格分隔的浮点数字符串列表（例如 "1.5 2.0 3.14"）。
// 比较时采用 1e-9 容差 epsilon 进行绝对差比较。
func ValidateEnumFloat(fl v10.FieldLevel) bool {
	param := fl.Param()
	if param == "" {
		return false
	}
	fields := strings.Fields(param)
	if len(fields) == 0 {
		return false
	}

	allowed := make([]float64, 0, len(fields))
	for _, valStr := range fields {
		v, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			return false
		}
		allowed = append(allowed, v)
	}

	field := fl.Field()
	var val float64
	switch field.Kind() {
	case reflect.Float32, reflect.Float64:
		val = field.Float()
	default:
		return false
	}

	const epsilon = 1e-9
	for _, item := range allowed {
		if math.Abs(val-item) < epsilon {
			return true
		}
	}
	return false
}

// ValidateSnowflake 验证字符串是否符合 Snowflake 雪花算法 ID 格式。
//
// 仅允许长度为 1 至 20 个字符的纯 ASCII 数字字符串，空字符串或包含非数字字符返回 false。
func ValidateSnowflake(fl v10.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() != reflect.String {
		return false
	}
	val := field.String()
	if val == "" || len(val) > 20 {
		return false
	}
	for _, c := range val {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// registerBuiltinCustomRules 注册 8 个内置自定义规则与其对应中文翻译
func (e *Engine) registerBuiltinCustomRules() error {
	rules := map[string]v10.Func{
		"strict_url":          ValidateStrictURL,
		"strict_uuid":         ValidateStrictUUID,
		"alphanum_underscore": ValidateAlphanumUnderscore,
		"regexp":              ValidateRegexp,
		"enum_int":            ValidateEnumInt,
		"enum_string":         ValidateEnumString,
		"enum_float":          ValidateEnumFloat,
		"snowflake":           ValidateSnowflake,
	}

	for tag, fn := range rules {
		if err := e.validate.RegisterValidation(tag, fn); err != nil {
			return fmt.Errorf("validator: 注册规则 %s 失败: %w", tag, err)
		}
	}

	e.registerBuiltinTranslations()
	return nil
}

// registerBuiltinTranslations 为当前实例注册自定义规则与标准规则补充中文翻译
func (e *Engine) registerBuiltinTranslations() {
	// oneof 补充翻译
	_ = e.validate.RegisterTranslation("oneof", e.trans,
		func(ut ut.Translator) error {
			return ut.Add("oneof", "{0}必须是以下值之一: {1}", true)
		},
		func(ut ut.Translator, fe v10.FieldError) string {
			t, _ := ut.T("oneof", fe.Field(), fe.Param())
			return t
		},
	)

	// strict_url
	_ = e.validate.RegisterTranslation("strict_url", e.trans,
		func(ut ut.Translator) error {
			return ut.Add("strict_url", "{0}必须是有效的 HTTP 或 HTTPS URL", true)
		},
		func(ut ut.Translator, fe v10.FieldError) string {
			t, _ := ut.T("strict_url", fe.Field())
			return t
		},
	)

	// strict_uuid
	_ = e.validate.RegisterTranslation("strict_uuid", e.trans,
		func(ut ut.Translator) error {
			return ut.Add("strict_uuid", "{0}必须是标准的 UUID 格式", true)
		},
		func(ut ut.Translator, fe v10.FieldError) string {
			t, _ := ut.T("strict_uuid", fe.Field())
			return t
		},
	)

	// alphanum_underscore
	_ = e.validate.RegisterTranslation("alphanum_underscore", e.trans,
		func(ut ut.Translator) error {
			return ut.Add("alphanum_underscore", "{0}只能包含字母、数字和下划线", true)
		},
		func(ut ut.Translator, fe v10.FieldError) string {
			t, _ := ut.T("alphanum_underscore", fe.Field())
			return t
		},
	)

	// regexp
	_ = e.validate.RegisterTranslation("regexp", e.trans,
		func(ut ut.Translator) error {
			return ut.Add("regexp", "{0}格式不正确", true)
		},
		func(ut ut.Translator, fe v10.FieldError) string {
			t, _ := ut.T("regexp", fe.Field())
			return t
		},
	)

	// enum_int
	_ = e.validate.RegisterTranslation("enum_int", e.trans,
		func(ut ut.Translator) error {
			return ut.Add("enum_int", "{0}必须是以下值之一: {1}", true)
		},
		func(ut ut.Translator, fe v10.FieldError) string {
			t, _ := ut.T("enum_int", fe.Field(), fe.Param())
			return t
		},
	)

	// enum_string
	_ = e.validate.RegisterTranslation("enum_string", e.trans,
		func(ut ut.Translator) error {
			return ut.Add("enum_string", "{0}必须是以下值之一: {1}", true)
		},
		func(ut ut.Translator, fe v10.FieldError) string {
			t, _ := ut.T("enum_string", fe.Field(), fe.Param())
			return t
		},
	)

	// enum_float
	_ = e.validate.RegisterTranslation("enum_float", e.trans,
		func(ut ut.Translator) error {
			return ut.Add("enum_float", "{0}必须是以下值之一: {1}", true)
		},
		func(ut ut.Translator, fe v10.FieldError) string {
			t, _ := ut.T("enum_float", fe.Field(), fe.Param())
			return t
		},
	)

	// snowflake
	_ = e.validate.RegisterTranslation("snowflake", e.trans,
		func(ut ut.Translator) error {
			return ut.Add("snowflake", "{0} 必须是有效的 Snowflake ID", true)
		},
		func(ut ut.Translator, fe v10.FieldError) string {
			t, _ := ut.T("snowflake", fe.Field())
			return t
		},
	)
}
