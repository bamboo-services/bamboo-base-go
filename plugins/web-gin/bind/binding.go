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

// Binding 提供面向具体泛型目标结构体 T 的链式参数绑定器。
//
// 该结构体通常由 [Bind] 或 [WithBinder] 创建，属于请求内部的短生命周期临时对象，
// 不应在多个并发请求或协程间共享。
type Binding[T any] struct {
	// Context 保存当前处理请求的 [*gin.Context] 实例。
	Context *gin.Context
	// GetData 保存接收反序列化与字段校验结果的目标结构体指针。
	GetData *T
	binder  *Binder
}

// Data 读取并反序列化 JSON 请求体数据到目标结构体，并自动触发实例级校验。
//
// 该方法内部读取并缓冲请求体字节，支持在后续流程中再次读取请求体。
// 若请求体为空、JSON 反序列化失败或结构体验证未通过，将自动写回统一错误响应、
// 调用 [gin.Context.Abort] 并返回 nil；校验完全通过时返回 GetData 指针。
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

// XML 读取并反序列化 XML 请求体数据到目标结构体，并自动触发实例级校验。
//
// 该方法内部读取并重置请求体数据。若 XML 反序列化失败或字段校验未通过，
// 将自动写回统一错误响应并调用 [gin.Context.Abort]，最终返回 nil；通过时返回 GetData。
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

// Query 从 HTTP 请求的 URL Query 参数映射到目标结构体，并自动触发实例级校验。
//
// 映射过程优先遵循结构体字段的 form 标签，未标注时使用字段名。
// 支持基础标量类型、指针、切片以及带有 time_format 与 time_utc 标签的 [time.Time]。
// 若类型转换失败或字段校验未通过，将写回统一错误响应并调用 [gin.Context.Abort]，返回 nil。
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

// URI 从 Gin 路由路径参数（[gin.Context.Params]）映射到目标结构体，并自动触发实例级校验。
//
// 映射过程遵循结构体字段的 uri 标签，未标注时使用字段名。
// 若参数解析失败或字段校验未通过，将写回统一错误响应并调用 [gin.Context.Abort]，返回 nil。
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

// Header 从 HTTP 请求标头中提取数据并映射到目标结构体，并自动触发实例级校验。
//
// 映射过程遵循结构体字段的 header 标签，支持标准 Canonical 标头名以及大小写不敏感匹配。
// 若标头解析失败或字段校验未通过，将写回统一错误响应并调用 [gin.Context.Abort]，返回 nil。
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

// Form 从 HTTP 请求的 PostForm 或 MultipartForm 表单中映射数据并执行校验。
//
// 映射过程遵循结构体字段的 form 标签。
// 若表单解析失败或字段校验未通过，将写回统一错误响应并调用 [gin.Context.Abort]，返回 nil。
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
