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

// ValidateStrictURL 严格的 URL 验证（仅支持 HTTP/HTTPS）
func ValidateStrictURL(fl v10.FieldLevel) bool {
	return urlRegex.MatchString(fl.Field().String())
}

// ValidateStrictUUID 严格的 UUID 格式验证
func ValidateStrictUUID(fl v10.FieldLevel) bool {
	return uuidRegex.MatchString(fl.Field().String())
}

// ValidateAlphanumUnderscore 字母数字下划线验证
func ValidateAlphanumUnderscore(fl v10.FieldLevel) bool {
	return alphanumUnderscoreRegex.MatchString(fl.Field().String())
}

// ValidateRegexp 正则表达式验证（支持自定义正则参数）
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

// ValidateEnumInt 整数枚举值验证
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

// ValidateEnumString 字符串枚举值验证
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

// ValidateEnumFloat 浮点数枚举值验证
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

// ValidateSnowflake Snowflake ID 格式验证（纯数字、长度 1~20）
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
