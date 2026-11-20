package typed

import (
	"encoding/xml"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// TestParseErrorCodeBranches 断言解析类自有出口按实际失败分支给出错误码：
// 格式非法 40013、数值超出范围 40014，并保留原生根因。
func TestParseErrorCodeBranches(t *testing.T) {
	tests := []struct {
		name     string
		parse    func() *apperr.Error
		wantCode apperr.Code
		wantRoot error
	}{
		{
			name:     "Int语法非法",
			parse:    func() *apperr.Error { _, err := String("abc").Int(); return err },
			wantCode: 40013,
			wantRoot: strconv.ErrSyntax,
		},
		{
			name:     "Int超出int64范围",
			parse:    func() *apperr.Error { _, err := String("9223372036854775808").Int(); return err },
			wantCode: 40014,
			wantRoot: strconv.ErrRange,
		},
		{
			name:     "Int8上溢",
			parse:    func() *apperr.Error { _, err := String("128").Int8(); return err },
			wantCode: 40014,
			wantRoot: strconv.ErrRange,
		},
		{
			name:     "Uint负数属语法非法",
			parse:    func() *apperr.Error { _, err := String("-1").Uint(); return err },
			wantCode: 40013,
			wantRoot: strconv.ErrSyntax,
		},
		{
			name:     "Uint8上溢",
			parse:    func() *apperr.Error { _, err := String("256").Uint8(); return err },
			wantCode: 40014,
			wantRoot: strconv.ErrRange,
		},
		{
			name:     "Float64语法非法",
			parse:    func() *apperr.Error { _, err := String("not-a-number").Float64(); return err },
			wantCode: 40013,
			wantRoot: strconv.ErrSyntax,
		},
		{
			name:     "Float64超出范围",
			parse:    func() *apperr.Error { _, err := String("1e309").Float64(); return err },
			wantCode: 40014,
			wantRoot: strconv.ErrRange,
		},
		{
			name:     "Float64拒绝非有限值",
			parse:    func() *apperr.Error { _, err := String("NaN").Float64(); return err },
			wantCode: 40013,
		},
		{
			name:     "Bool非布尔词",
			parse:    func() *apperr.Error { _, err := String("maybe").Bool(); return err },
			wantCode: 40013,
		},
		{
			name:     "Duration缺单位",
			parse:    func() *apperr.Error { _, err := String("30").Duration(); return err },
			wantCode: 40013,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.parse()
			if err == nil {
				t.Fatal("期望失败，但返回 nil")
			}
			if err.Code.Code != tt.wantCode {
				t.Errorf("错误码 = %d, 期望 %d", err.Code.Code, tt.wantCode)
			}
			if tt.wantRoot != nil {
				if !errors.Is(err, tt.wantRoot) {
					t.Errorf("根因不可达: %v", err)
				}
				if err.Err == nil {
					t.Error("期望保留原生根因")
				}
			}
		})
	}
}

// TestParseSuccessReturnsNil 断言解析成功时返回真正的 nil，而不是 nil 指针载体。
func TestParseSuccessReturnsNil(t *testing.T) {
	if _, err := String(" 42 ").Int(); err != nil {
		t.Errorf("Int() 期望 nil，实际 %v", err)
	}
	if _, err := String("42").Uint(); err != nil {
		t.Errorf("Uint() 期望 nil，实际 %v", err)
	}
	if _, err := String("3.5").Float64(); err != nil {
		t.Errorf("Float64() 期望 nil，实际 %v", err)
	}
	if _, err := String("enabled").Bool(); err != nil {
		t.Errorf("Bool() 期望 nil，实际 %v", err)
	}
	if _, err := String("30s").Duration(); err != nil {
		t.Errorf("Duration() 期望 nil，实际 %v", err)
	}
}

// TestStandardMethodsSuccessReturnNil 断言十三个标准接口方法在成功路径返回真正的 nil。
func TestStandardMethodsSuccessReturnNil(t *testing.T) {
	key := String("app_port")

	if _, err := key.MarshalText(); err != nil {
		t.Errorf("MarshalText() 期望 nil，实际 %v", err)
	}
	if _, err := key.MarshalBinary(); err != nil {
		t.Errorf("MarshalBinary() 期望 nil，实际 %v", err)
	}
	if _, err := key.MarshalJSON(); err != nil {
		t.Errorf("MarshalJSON() 期望 nil，实际 %v", err)
	}
	if _, err := key.MarshalXMLAttr(xml.Name{}); err != nil {
		t.Errorf("MarshalXMLAttr() 期望 nil，实际 %v", err)
	}
	if _, err := key.Value(); err != nil {
		t.Errorf("Value() 期望 nil，实际 %v", err)
	}

	encoder := xml.NewEncoder(io.Discard)
	if err := key.MarshalXML(encoder, xml.StartElement{Name: xml.Name{Local: "tag"}}); err != nil {
		t.Errorf("MarshalXML() 期望 nil，实际 %v", err)
	}

	var text String
	if err := text.UnmarshalText([]byte("app_port")); err != nil {
		t.Errorf("UnmarshalText() 期望 nil，实际 %v", err)
	}
	var binary String
	if err := binary.UnmarshalBinary([]byte("app_port")); err != nil {
		t.Errorf("UnmarshalBinary() 期望 nil，实际 %v", err)
	}
	var jsonValue String
	if err := jsonValue.UnmarshalJSON([]byte(`"app_port"`)); err != nil {
		t.Errorf("UnmarshalJSON() 期望 nil，实际 %v", err)
	}
	var xmlValue String
	xmlDecoder := xml.NewDecoder(strings.NewReader("<tag>app_port</tag>"))
	if _, err := xmlDecoder.Token(); err != nil {
		t.Fatalf("准备 XML 解码器失败: %v", err)
	}
	if err := xmlValue.UnmarshalXML(xmlDecoder, xml.StartElement{Name: xml.Name{Local: "tag"}}); err != nil {
		t.Errorf("UnmarshalXML() 期望 nil，实际 %v", err)
	}
	var attrValue String
	if err := attrValue.UnmarshalXMLAttr(xml.Attr{Value: "app_port"}); err != nil {
		t.Errorf("UnmarshalXMLAttr() 期望 nil，实际 %v", err)
	}
	var scanned String
	if err := scanned.Scan("app_port"); err != nil {
		t.Errorf("Scan() 期望 nil，实际 %v", err)
	}
	var flagValue String
	if err := flagValue.Set("app_port"); err != nil {
		t.Errorf("Set() 期望 nil，实际 %v", err)
	}
}

// TestNilReceiverErrorCode 断言 nil 接收器路径归调用契约错误 50019，且旧哨兵仍可达。
func TestNilReceiverErrorCode(t *testing.T) {
	var nilPtr *String

	cases := []struct {
		name string
		call func() error
	}{
		{name: "UnmarshalText", call: func() error { return nilPtr.UnmarshalText([]byte("app_port")) }},
		{name: "UnmarshalBinary", call: func() error { return nilPtr.UnmarshalBinary([]byte("app_port")) }},
		{name: "UnmarshalJSON", call: func() error { return nilPtr.UnmarshalJSON([]byte(`"app_port"`)) }},
		{name: "UnmarshalXML", call: func() error {
			return nilPtr.UnmarshalXML(xml.NewDecoder(strings.NewReader("<tag>app_port</tag>")), xml.StartElement{})
		}},
		{name: "UnmarshalXMLAttr", call: func() error { return nilPtr.UnmarshalXMLAttr(xml.Attr{Value: "app_port"}) }},
		{name: "Scan", call: func() error { return nilPtr.Scan("app_port") }},
		{name: "Set", call: func() error { return nilPtr.Set("app_port") }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if err == nil {
				t.Fatal("期望失败，但返回 nil")
			}
			var appErr *apperr.Error
			if !errors.As(err, &appErr) {
				t.Fatalf("期望 *apperr.Error，实际 %T", err)
			}
			if appErr.Code.Code != 50019 {
				t.Errorf("错误码 = %d, 期望 50019", appErr.Code.Code)
			}
			if !errors.Is(err, ErrNilReceiver) {
				t.Errorf("旧哨兵不可达: %v", err)
			}
		})
	}
}

// TestFormatAndTypeErrorCodes 断言格式类归 40013、输入类型类归 40015。
func TestFormatAndTypeErrorCodes(t *testing.T) {
	formatCases := []struct {
		name string
		call func() error
	}{
		{name: "MarshalText非法键", call: func() error {
			_, err := String("bad key").MarshalText()
			return err
		}},
		{name: "MarshalBinary非法键", call: func() error {
			_, err := String("bad key").MarshalBinary()
			return err
		}},
		{name: "MarshalJSON非法键", call: func() error {
			_, err := String("bad key").MarshalJSON()
			return err
		}},
		{name: "MarshalXMLAttr非法键", call: func() error {
			_, err := String("bad key").MarshalXMLAttr(xml.Name{})
			return err
		}},
		{name: "Value非法键", call: func() error {
			_, err := String("bad key").Value()
			return err
		}},
		{name: "UnmarshalText非法键", call: func() error {
			var v String
			return v.UnmarshalText([]byte("bad key"))
		}},
		{name: "UnmarshalBinary非法键", call: func() error {
			var v String
			return v.UnmarshalBinary([]byte("bad key"))
		}},
		{name: "UnmarshalJSON非法键", call: func() error {
			var v String
			return v.UnmarshalJSON([]byte(`"bad key"`))
		}},
		{name: "UnmarshalJSON语法非法", call: func() error {
			var v String
			return v.UnmarshalJSON([]byte(`{"broken`))
		}},
		{name: "UnmarshalXMLAttr非法键", call: func() error {
			var v String
			return v.UnmarshalXMLAttr(xml.Attr{Value: "bad key"})
		}},
		{name: "Set非法键", call: func() error {
			var v String
			return v.Set("bad key")
		}},
		{name: "Scan非法键", call: func() error {
			var v String
			return v.Scan("bad key")
		}},
	}

	for _, tc := range formatCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if err == nil {
				t.Fatal("期望失败，但返回 nil")
			}
			var appErr *apperr.Error
			if !errors.As(err, &appErr) {
				t.Fatalf("期望 *apperr.Error，实际 %T", err)
			}
			if appErr.Code.Code != 40013 {
				t.Errorf("错误码 = %d, 期望 40013", appErr.Code.Code)
			}
		})
	}

	typeCases := []struct {
		name string
		call func() error
	}{
		{name: "UnmarshalJSON接受null", call: func() error {
			var v String
			return v.UnmarshalJSON([]byte("null"))
		}},
		{name: "UnmarshalJSON接受非字符串", call: func() error {
			var v String
			return v.UnmarshalJSON([]byte("123"))
		}},
		{name: "Scan拒绝SQL NULL", call: func() error {
			var v String
			return v.Scan(nil)
		}},
		{name: "Scan拒绝不支持的列类型", call: func() error {
			var v String
			return v.Scan(123)
		}},
		{name: "UnmarshalXML拒绝嵌套子元素", call: func() error {
			var v String
			return v.UnmarshalXML(xml.NewDecoder(strings.NewReader("<tag><sub>app_port</sub></tag>")), xml.StartElement{})
		}},
	}

	for _, tc := range typeCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if err == nil {
				t.Fatal("期望失败，但返回 nil")
			}
			var appErr *apperr.Error
			if !errors.As(err, &appErr) {
				t.Fatalf("期望 *apperr.Error，实际 %T", err)
			}
			if appErr.Code.Code != 40015 {
				t.Errorf("错误码 = %d, 期望 40015", appErr.Code.Code)
			}
		})
	}
}

// TestFailureDoesNotWriteBack 断言任何失败都不写回接收器原值。
func TestFailureDoesNotWriteBack(t *testing.T) {
	original := String("origin_key")

	cases := []struct {
		name string
		call func(v *String)
	}{
		{name: "UnmarshalText失败", call: func(v *String) { _ = v.UnmarshalText([]byte("bad key")) }},
		{name: "UnmarshalBinary失败", call: func(v *String) { _ = v.UnmarshalBinary([]byte("bad key")) }},
		{name: "UnmarshalJSON失败", call: func(v *String) { _ = v.UnmarshalJSON([]byte("null")) }},
		{name: "UnmarshalXMLAttr失败", call: func(v *String) { _ = v.UnmarshalXMLAttr(xml.Attr{Value: "bad key"}) }},
		{name: "Scan失败", call: func(v *String) { _ = v.Scan(nil) }},
		{name: "Set失败", call: func(v *String) { _ = v.Set("bad key") }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value := original
			tc.call(&value)
			if value != original {
				t.Errorf("失败后原值被改写: %q", value)
			}
		})
	}
}

// TestSafeMessageDoesNotLeakInput 断言安全描述不复制原始输入内容。
func TestSafeMessageDoesNotLeakInput(t *testing.T) {
	const secret = "s3cret token!"

	_, err := String(secret).MarshalText()
	if err == nil {
		t.Fatal("期望失败，但返回 nil")
	}
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("期望 *apperr.Error，实际 %T", err)
	}
	if strings.Contains(string(appErr.ErrorMessage), secret) {
		t.Errorf("安全描述泄漏原始输入: %q", appErr.ErrorMessage)
	}
}
