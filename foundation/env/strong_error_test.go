package env

import (
	"encoding/xml"
	"errors"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// TestKeyNilReceiverErrorCode 断言 nil 接收器路径归调用契约错误 50019，根因哨兵仍可达。
func TestKeyNilReceiverErrorCode(t *testing.T) {
	var nilPtr *Key

	cases := []struct {
		name string
		call func() error
	}{
		{name: "UnmarshalText", call: func() error { return nilPtr.UnmarshalText([]byte("APP_NAME")) }},
		{name: "UnmarshalBinary", call: func() error { return nilPtr.UnmarshalBinary([]byte("APP_NAME")) }},
		{name: "UnmarshalJSON", call: func() error { return nilPtr.UnmarshalJSON([]byte(`"APP_NAME"`)) }},
		{name: "UnmarshalXML", call: func() error {
			return nilPtr.UnmarshalXML(xml.NewDecoder(strings.NewReader("<tag>APP_NAME</tag>")), xml.StartElement{})
		}},
		{name: "UnmarshalXMLAttr", call: func() error {
			return nilPtr.UnmarshalXMLAttr(xml.Attr{Value: "APP_NAME"})
		}},
		{name: "Scan", call: func() error { return nilPtr.Scan("APP_NAME") }},
		{name: "Set", call: func() error { return nilPtr.Set("APP_NAME") }},
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
			if !errors.Is(err, errNilReceiver) {
				t.Errorf("根因哨兵不可达: %v", err)
			}
		})
	}
}

// TestKeyFormatAndTypeErrorCodes 断言键名格式非法归 40013、输入源类型非法归 40015。
func TestKeyFormatAndTypeErrorCodes(t *testing.T) {
	formatCases := []struct {
		name string
		call func() error
	}{
		{name: "MarshalText非法键", call: func() error {
			_, err := Key("bad key").MarshalText()
			return err
		}},
		{name: "UnmarshalText非法键", call: func() error {
			var k Key
			return k.UnmarshalText([]byte("bad key"))
		}},
		{name: "Set非法键", call: func() error {
			var k Key
			return k.Set("bad key")
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
			var k Key
			return k.UnmarshalJSON([]byte("null"))
		}},
		{name: "Scan拒绝SQL NULL", call: func() error {
			var k Key
			return k.Scan(nil)
		}},
		{name: "Scan拒绝不支持的列类型", call: func() error {
			var k Key
			return k.Scan(123)
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

// TestKeySuccessReturnsNil 断言标准方法在成功路径返回真正的 nil。
func TestKeySuccessReturnsNil(t *testing.T) {
	key := Key("APP_NAME")

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

	var text Key
	if err := text.UnmarshalText([]byte("APP_NAME")); err != nil {
		t.Errorf("UnmarshalText() 期望 nil，实际 %v", err)
	}
	var binary Key
	if err := binary.UnmarshalBinary([]byte("APP_NAME")); err != nil {
		t.Errorf("UnmarshalBinary() 期望 nil，实际 %v", err)
	}
	var jsonValue Key
	if err := jsonValue.UnmarshalJSON([]byte(`"APP_NAME"`)); err != nil {
		t.Errorf("UnmarshalJSON() 期望 nil，实际 %v", err)
	}
	var attrValue Key
	if err := attrValue.UnmarshalXMLAttr(xml.Attr{Value: "APP_NAME"}); err != nil {
		t.Errorf("UnmarshalXMLAttr() 期望 nil，实际 %v", err)
	}
	var scanned Key
	if err := scanned.Scan("APP_NAME"); err != nil {
		t.Errorf("Scan() 期望 nil，实际 %v", err)
	}
	var flagValue Key
	if err := flagValue.Set("APP_NAME"); err != nil {
		t.Errorf("Set() 期望 nil，实际 %v", err)
	}
}

// TestKeyFailureDoesNotWriteBack 断言失败不写回原键值。
func TestKeyFailureDoesNotWriteBack(t *testing.T) {
	original := Key("APP_NAME")

	cases := []struct {
		name string
		call func(k *Key)
	}{
		{name: "UnmarshalText失败", call: func(k *Key) { _ = k.UnmarshalText([]byte("bad key")) }},
		{name: "UnmarshalJSON失败", call: func(k *Key) { _ = k.UnmarshalJSON([]byte("null")) }},
		{name: "UnmarshalXMLAttr失败", call: func(k *Key) { _ = k.UnmarshalXMLAttr(xml.Attr{Value: "bad key"}) }},
		{name: "Scan失败", call: func(k *Key) { _ = k.Scan(nil) }},
		{name: "Set失败", call: func(k *Key) { _ = k.Set("bad key") }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key := original
			tc.call(&key)
			if key != original {
				t.Errorf("失败后原值被改写: %q", key)
			}
		})
	}
}
