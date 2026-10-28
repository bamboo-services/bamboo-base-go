package bind

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/httpresult"
	"github.com/gin-gonic/gin"
)

// Binding 支持泛型目标结构的链式绑定器。
type Binding[T any] struct {
	Context *gin.Context
	GetData *T
	binder  *Binder
}

// Data 显式读取并反序列化 JSON 请求体，并由独立 Engine 校验。
// 绝不调用 ShouldBindBodyWithJSON 或 Gin 全局 validator。
func (b *Binding[T]) Data() *T {
	if !b.ensureReady() {
		return nil
	}

	bodyBytes, err := b.readBodyBytes()
	if err != nil {
		b.handleBindError(err)
		return nil
	}

	if len(bodyBytes) == 0 {
		b.handleBindError(errors.New("request body is empty"))
		return nil
	}

	if err := json.Unmarshal(bodyBytes, b.GetData); err != nil {
		b.handleBindError(err)
		return nil
	}

	if err := b.binder.engine.Validate(b.GetData); err != nil {
		b.binder.engine.HandleError(b.Context, err)
		b.Context.Abort()
		return nil
	}

	return b.GetData
}

// XML 显式读取并反序列化 XML 请求体并校验。
func (b *Binding[T]) XML() *T {
	if !b.ensureReady() {
		return nil
	}

	bodyBytes, err := b.readBodyBytes()
	if err != nil {
		b.handleBindError(err)
		return nil
	}

	if err := xml.Unmarshal(bodyBytes, b.GetData); err != nil {
		b.handleBindError(err)
		return nil
	}

	if err := b.binder.engine.Validate(b.GetData); err != nil {
		b.binder.engine.HandleError(b.Context, err)
		b.Context.Abort()
		return nil
	}

	return b.GetData
}

// Query 从 URL Query 参数映射到目标结构体并校验。
// 遵循 form 标签或字段名，支持 slice、指针、基础类型及 time.Time (time_format 标签)。
func (b *Binding[T]) Query() *T {
	if !b.ensureReady() {
		return nil
	}

	values := b.Context.Request.URL.Query()
	if err := mapValuesToStruct(values, b.GetData, false, "form"); err != nil {
		b.handleBindError(err)
		return nil
	}

	if err := b.binder.engine.Validate(b.GetData); err != nil {
		b.binder.engine.HandleError(b.Context, err)
		b.Context.Abort()
		return nil
	}

	return b.GetData
}

// URI 从 Gin Params (路径参数) 映射到目标结构体并校验。
// 遵循 uri 标签或字段名。
func (b *Binding[T]) URI() *T {
	if !b.ensureReady() {
		return nil
	}

	params := b.Context.Params
	m := make(map[string][]string, len(params))
	for _, p := range params {
		m[p.Key] = []string{p.Value}
	}

	if err := mapValuesToStruct(m, b.GetData, false, "uri"); err != nil {
		b.handleBindError(err)
		return nil
	}

	if err := b.binder.engine.Validate(b.GetData); err != nil {
		b.binder.engine.HandleError(b.Context, err)
		b.Context.Abort()
		return nil
	}

	return b.GetData
}

// Header 从 HTTP 请求头映射到目标结构体并校验。
// 遵循 header 标签或字段名，支持 HTTP 头部规范及大小写不敏感匹配。
func (b *Binding[T]) Header() *T {
	if !b.ensureReady() {
		return nil
	}

	headers := b.Context.Request.Header
	if err := mapValuesToStruct(headers, b.GetData, true, "header"); err != nil {
		b.handleBindError(err)
		return nil
	}

	if err := b.binder.engine.Validate(b.GetData); err != nil {
		b.binder.engine.HandleError(b.Context, err)
		b.Context.Abort()
		return nil
	}

	return b.GetData
}

// Form 从 PostForm / MultipartForm 映射到目标结构体并校验。
// 遵循 form 标签或字段名。
func (b *Binding[T]) Form() *T {
	if !b.ensureReady() {
		return nil
	}

	if err := b.Context.Request.ParseForm(); err != nil && !errors.Is(err, http.ErrNotMultipart) {
		b.handleBindError(err)
		return nil
	}

	values := b.Context.Request.PostForm
	if len(values) == 0 && b.Context.Request.Form != nil {
		values = b.Context.Request.Form
	}

	if err := mapValuesToStruct(values, b.GetData, false, "form"); err != nil {
		b.handleBindError(err)
		return nil
	}

	if err := b.binder.engine.Validate(b.GetData); err != nil {
		b.binder.engine.HandleError(b.Context, err)
		b.Context.Abort()
		return nil
	}

	return b.GetData
}

func (b *Binding[T]) ensureReady() bool {
	if b.Context == nil {
		return false
	}
	if b.GetData == nil {
		b.handleBindError(ErrNilTarget)
		return false
	}
	if b.binder == nil || b.binder.engine == nil {
		b.handleBindError(ErrNoBinderInContext)
		return false
	}
	return true
}

func (b *Binding[T]) handleBindError(err error) {
	if b.Context == nil {
		return
	}
	if b.binder != nil && b.binder.engine != nil {
		b.binder.engine.HandleError(b.Context, err)
		b.Context.Abort()
		return
	}
	// 缺失 Binder 明确报错，绝不偷建全局引擎
	appErr := apperr.New(apperr.ParameterError, apperr.ErrMessage("请求绑定失败: "+err.Error()), err)
	httpresult.AbortError(b.Context, appErr)
}

func (b *Binding[T]) readBodyBytes() ([]byte, error) {
	if b.Context.Request == nil || b.Context.Request.Body == nil {
		return nil, errors.New("request body is nil")
	}
	body, err := io.ReadAll(b.Context.Request.Body)
	if err != nil {
		return nil, err
	}
	b.Context.Request.Body = io.NopCloser(strings.NewReader(string(body)))
	return body, nil
}

// mapValuesToStruct 使用反射将 map[string][]string 映射到结构体，支持嵌套、slice、指针与 time.Time。
func mapValuesToStruct(values map[string][]string, target any, headerMode bool, tagNames ...string) error {
	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Pointer || val.IsNil() {
		return errors.New("target must be a non-nil pointer")
	}
	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return errors.New("target element must be a struct")
	}
	return fillStructFields(values, val, headerMode, tagNames...)
}

func fillStructFields(values map[string][]string, structVal reflect.Value, headerMode bool, tagNames ...string) error {
	typ := structVal.Type()
	for i := 0; i < typ.NumField(); i++ {
		fieldTyp := typ.Field(i)
		fieldVal := structVal.Field(i)

		if !fieldVal.CanSet() {
			continue
		}

		// 处理内嵌未命名结构体
		if fieldTyp.Anonymous && fieldVal.Kind() == reflect.Struct {
			if err := fillStructFields(values, fieldVal, headerMode, tagNames...); err != nil {
				return err
			}
			continue
		}

		key := resolveFieldKey(fieldTyp, tagNames...)
		if key == "" || key == "-" {
			continue
		}

		rawVals, found := lookupValues(values, key, headerMode)
		if !found || len(rawVals) == 0 {
			continue
		}

		timeFormat := fieldTyp.Tag.Get("time_format")
		timeLoc := fieldTyp.Tag.Get("time_utc")
		loc := time.Local
		if timeLoc == "1" || strings.ToLower(timeLoc) == "true" {
			loc = time.UTC
		}

		if err := setFieldValue(fieldVal, rawVals, timeFormat, loc); err != nil {
			return &json.UnmarshalTypeError{
				Field: key,
				Type:  fieldVal.Type(),
			}
		}
	}
	return nil
}

func resolveFieldKey(f reflect.StructField, tagNames ...string) string {
	for _, tag := range tagNames {
		if v := f.Tag.Get(tag); v != "" {
			parts := strings.Split(v, ",")
			name := strings.TrimSpace(parts[0])
			if name != "" {
				return name
			}
		}
	}
	return f.Name
}

func lookupValues(values map[string][]string, key string, headerMode bool) ([]string, bool) {
	if v, ok := values[key]; ok {
		return v, true
	}
	if !headerMode {
		return nil, false
	}
	// Header 模式：优先标准 canonical key，其次大小写不敏感匹配
	if v, ok := values[http.CanonicalHeaderKey(key)]; ok {
		return v, true
	}
	for k, v := range values {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return nil, false
}

func setFieldValue(fieldVal reflect.Value, rawVals []string, timeFormat string, loc *time.Location) error {
	if fieldVal.Kind() == reflect.Pointer {
		if fieldVal.IsNil() {
			fieldVal.Set(reflect.New(fieldVal.Type().Elem()))
		}
		fieldVal = fieldVal.Elem()
	}

	// 特殊处理 time.Time
	if fieldVal.Type() == reflect.TypeOf(time.Time{}) {
		raw := strings.TrimSpace(rawVals[0])
		t, err := parseTimeValue(raw, timeFormat, loc)
		if err != nil {
			return err
		}
		fieldVal.Set(reflect.ValueOf(t))
		return nil
	}

	switch fieldVal.Kind() {
	case reflect.Slice:
		sliceType := fieldVal.Type().Elem()
		isPtr := sliceType.Kind() == reflect.Pointer
		elemType := sliceType
		if isPtr {
			elemType = sliceType.Elem()
		}

		slice := reflect.MakeSlice(fieldVal.Type(), len(rawVals), len(rawVals))
		for i, r := range rawVals {
			itemVal := reflect.New(elemType).Elem()
			if err := setSingleScalar(itemVal, strings.TrimSpace(r), timeFormat, loc); err != nil {
				return err
			}
			if isPtr {
				slice.Index(i).Set(itemVal.Addr())
			} else {
				slice.Index(i).Set(itemVal)
			}
		}
		fieldVal.Set(slice)
		return nil

	default:
		return setSingleScalar(fieldVal, strings.TrimSpace(rawVals[0]), timeFormat, loc)
	}
}

func setSingleScalar(v reflect.Value, raw string, timeFormat string, loc *time.Location) error {
	if v.Type() == reflect.TypeOf(time.Time{}) {
		t, err := parseTimeValue(raw, timeFormat, loc)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(t))
		return nil
	}

	switch v.Kind() {
	case reflect.String:
		v.SetString(raw)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return err
		}
		v.SetInt(val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return err
		}
		v.SetUint(val)
	case reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return err
		}
		v.SetFloat(val)
	case reflect.Bool:
		val, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		v.SetBool(val)
	default:
		return errors.New("unsupported scalar type: " + v.Kind().String())
	}
	return nil
}

func parseTimeValue(raw string, format string, loc *time.Location) (time.Time, error) {
	if format != "" {
		return time.ParseInLocation(format, raw, loc)
	}
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
		time.RFC3339Nano,
	}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, raw, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("unable to parse time string: " + raw)
}
