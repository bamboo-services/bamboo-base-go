package env_test

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/typed"
)

// 静态接口断言
var (
	_ typed.Operable    = env.Key("")
	_ typed.Convertible = (*env.Key)(nil)
	_ driver.Valuer     = env.Key("")
	_ sql.Scanner       = (*env.Key)(nil)
	_ flag.Value        = (*env.Key)(nil)
)

func TestLookup(t *testing.T) {
	const testKey env.Key = "TEST_BAMBOO_ENV_LOOKUP_KEY"
	_ = os.Unsetenv(string(testKey))

	// 1. 缺失状态
	val, ok := env.Lookup(testKey)
	if ok {
		t.Fatalf("expected not found for unset key, got %v", val)
	}
	if val != "" {
		t.Fatalf("expected empty string for unset key, got %v", val)
	}

	// 2. 空值状态
	_ = os.Setenv(string(testKey), "")
	defer func() { _ = os.Unsetenv(string(testKey)) }()

	val, ok = env.Lookup(testKey)
	if !ok {
		t.Fatalf("expected found for empty env key")
	}
	if val != "" {
		t.Fatalf("expected empty typed.String, got %v", val)
	}
	if !val.IsEmpty() {
		t.Fatalf("expected IsEmpty() to be true for empty env key")
	}

	// 3. 原始值不进行键语法过滤（例如包含连续分隔符、数字开头、甚至特殊字符）
	rawVal := "123__raw-value!@#"
	_ = os.Setenv(string(testKey), rawVal)
	val, ok = env.Lookup(testKey)
	if !ok {
		t.Fatalf("expected found for set env key")
	}
	if val.String() != rawVal {
		t.Fatalf("expected %q, got %q", rawVal, val.String())
	}
	// 原始值不强制合规受控键名语法
	if val.IsValid() {
		t.Fatalf("expected rawVal to fail key syntax IsValid")
	}

	// 4. 数值与时长解析测试
	_ = os.Setenv(string(testKey), "5000")
	val, ok = env.Lookup(testKey)
	if !ok {
		t.Fatalf("expected found")
	}
	n, err := val.Int()
	if err != nil || n != 5000 {
		t.Fatalf("expected int 5000, got %d, err: %v", n, err)
	}

	_ = os.Setenv(string(testKey), "15s")
	val, ok = env.Lookup(testKey)
	if !ok {
		t.Fatalf("expected found")
	}
	d, err := val.Duration()
	if err != nil || d != 15*time.Second {
		t.Fatalf("expected duration 15s, got %v, err: %v", d, err)
	}

	_ = os.Setenv(string(testKey), "true")
	val, ok = env.Lookup(testKey)
	if !ok {
		t.Fatalf("expected found")
	}
	b, err := val.Bool()
	if err != nil || !b {
		t.Fatalf("expected bool true, got %v, err: %v", b, err)
	}
}

func TestKey_Operable(t *testing.T) {
	// 验证新增系统键
	sysKeys := map[env.Key]string{
		env.Env:        "APP_ENV",
		env.AppName:    "APP_NAME",
		env.AppVersion: "APP_VERSION",
		env.Timezone:   "APP_TIMEZONE",
	}
	for k, expected := range sysKeys {
		if k.String() != expected {
			t.Errorf("expected string %s, got %s", expected, k.String())
		}
		if !k.IsValid() {
			t.Errorf("expected valid syntax for %s", expected)
		}
	}

	k := env.StartTimeout
	if k.String() != "APP_START_TIMEOUT" {
		t.Errorf("expected string APP_START_TIMEOUT, got %s", k.String())
	}
	if k.TypedString() != typed.String("APP_START_TIMEOUT") {
		t.Errorf("expected typed.String APP_START_TIMEOUT, got %s", k.TypedString())
	}
	if k.IsEmpty() {
		t.Errorf("expected non-empty for StartTimeout")
	}
	if !env.Key("").IsEmpty() {
		t.Errorf("expected empty for Key(\"\")")
	}
	if !k.IsValid() {
		t.Errorf("expected valid for StartTimeout")
	}
	if env.Key("1invalid").IsValid() {
		t.Errorf("expected invalid for 1invalid")
	}
}

func TestKey_SixChannels_RoundTrip(t *testing.T) {
	valid := env.DatabaseDSN

	// 1. Text
	textBytes, err := valid.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText failed: %v", err)
	}
	var textDec env.Key
	if err := textDec.UnmarshalText(textBytes); err != nil {
		t.Fatalf("UnmarshalText failed: %v", err)
	}
	if textDec != valid {
		t.Errorf("Text roundtrip mismatch: got %v, want %v", textDec, valid)
	}

	// 2. JSON
	jsonBytes, err := json.Marshal(valid)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	var jsonDec env.Key
	if err := json.Unmarshal(jsonBytes, &jsonDec); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if jsonDec != valid {
		t.Errorf("JSON roundtrip mismatch: got %v, want %v", jsonDec, valid)
	}

	// 3. XML (Element & Attribute)
	type XMLStruct struct {
		XMLName xml.Name `xml:"root"`
		Elem    env.Key  `xml:"elem"`
		Attr    env.Key  `xml:"attr,attr"`
	}
	xmlIn := XMLStruct{Elem: valid, Attr: env.Port}
	xmlBytes, err := xml.Marshal(xmlIn)
	if err != nil {
		t.Fatalf("xml.Marshal failed: %v", err)
	}
	var xmlOut XMLStruct
	if err := xml.Unmarshal(xmlBytes, &xmlOut); err != nil {
		t.Fatalf("xml.Unmarshal failed: %v", err)
	}
	if xmlOut.Elem != valid || xmlOut.Attr != env.Port {
		t.Errorf("XML roundtrip mismatch: got elem=%v attr=%v", xmlOut.Elem, xmlOut.Attr)
	}

	// 4. Binary (gob roundtrip)
	var gobBuf bytes.Buffer
	enc := gob.NewEncoder(&gobBuf)
	if err := enc.Encode(valid); err != nil {
		t.Fatalf("gob.Encode failed: %v", err)
	}
	var gobDec env.Key
	dec := gob.NewDecoder(&gobBuf)
	if err := dec.Decode(&gobDec); err != nil {
		t.Fatalf("gob.Decode failed: %v", err)
	}
	if gobDec != valid {
		t.Errorf("gob roundtrip mismatch: got %v, want %v", gobDec, valid)
	}

	// 5. SQL
	val, err := valid.Value()
	if err != nil {
		t.Fatalf("Value failed: %v", err)
	}
	var sqlDec env.Key
	if err := sqlDec.Scan(val); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if sqlDec != valid {
		t.Errorf("SQL roundtrip mismatch: got %v, want %v", sqlDec, valid)
	}

	// 6. CLI Flag
	var flagKey env.Key
	if err := flagKey.Set(string(valid)); err != nil {
		t.Fatalf("flag.Set failed: %v", err)
	}
	if flagKey != valid {
		t.Errorf("Flag Set mismatch: got %v, want %v", flagKey, valid)
	}
}

func TestKey_NilReceiver_Safety(t *testing.T) {
	var k *env.Key

	if err := k.UnmarshalText([]byte("VALID_KEY")); err == nil {
		t.Error("expected error on nil UnmarshalText")
	}
	if err := k.UnmarshalBinary([]byte("VALID_KEY")); err == nil {
		t.Error("expected error on nil UnmarshalBinary")
	}
	if err := k.UnmarshalJSON([]byte(`"VALID_KEY"`)); err == nil {
		t.Error("expected error on nil UnmarshalJSON")
	}
	if err := k.UnmarshalXML(nil, xml.StartElement{}); err == nil {
		t.Error("expected error on nil UnmarshalXML")
	}
	if err := k.UnmarshalXMLAttr(xml.Attr{Value: "VALID_KEY"}); err == nil {
		t.Error("expected error on nil UnmarshalXMLAttr")
	}
	if err := k.Scan("VALID_KEY"); err == nil {
		t.Error("expected error on nil Scan")
	}
	if err := k.Set("VALID_KEY"); err == nil {
		t.Error("expected error on nil Set")
	}
}

func TestKey_InvalidInput_PreservesOriginal(t *testing.T) {
	const original = env.DatabaseDSN

	// 1. Text
	k1 := original
	if err := k1.UnmarshalText([]byte("1invalid")); err == nil {
		t.Error("expected error on invalid text")
	}
	if k1 != original {
		t.Errorf("original modified on failed UnmarshalText: got %v", k1)
	}

	// 2. Binary
	k2 := original
	if err := k2.UnmarshalBinary([]byte("1invalid")); err == nil {
		t.Error("expected error on invalid binary")
	}
	if k2 != original {
		t.Errorf("original modified on failed UnmarshalBinary: got %v", k2)
	}

	// 3. JSON
	k3 := original
	if err := json.Unmarshal([]byte(`"1invalid"`), &k3); err == nil {
		t.Error("expected error on invalid JSON string")
	}
	if k3 != original {
		t.Errorf("original modified on failed UnmarshalJSON: got %v", k3)
	}
	if err := json.Unmarshal([]byte(`null`), &k3); err == nil {
		t.Error("expected error on JSON null")
	}
	if k3 != original {
		t.Errorf("original modified on JSON null: got %v", k3)
	}

	// 4. XML element
	type ElemWrapper struct {
		Key env.Key `xml:"k"`
	}
	wrapper := ElemWrapper{Key: original}
	if err := xml.Unmarshal([]byte(`<ElemWrapper><k>1invalid</k></ElemWrapper>`), &wrapper); err == nil {
		t.Error("expected error on invalid XML element")
	}
	if wrapper.Key != original {
		t.Errorf("original modified on invalid XML element: got %v", wrapper.Key)
	}

	// XML attribute
	type AttrWrapper struct {
		Key env.Key `xml:"k,attr"`
	}
	attrWrapper := AttrWrapper{Key: original}
	if err := xml.Unmarshal([]byte(`<AttrWrapper k="1invalid"/>`), &attrWrapper); err == nil {
		t.Error("expected error on invalid XML attribute")
	}
	if attrWrapper.Key != original {
		t.Errorf("original modified on invalid XML attr: got %v", attrWrapper.Key)
	}

	// 5. SQL
	k4 := original
	if err := k4.Scan("1invalid"); err == nil {
		t.Error("expected error on invalid SQL string")
	}
	if k4 != original {
		t.Errorf("original modified on failed Scan: got %v", k4)
	}
	if err := k4.Scan(nil); err == nil {
		t.Error("expected error on SQL NULL")
	}
	if k4 != original {
		t.Errorf("original modified on SQL NULL: got %v", k4)
	}

	// 6. CLI Flag
	k5 := original
	if err := k5.Set("1invalid"); err == nil {
		t.Error("expected error on invalid Flag")
	}
	if k5 != original {
		t.Errorf("original modified on failed Set: got %v", k5)
	}
}

func TestKey_MarshalInvalid_ReturnsError(t *testing.T) {
	invalid := env.Key("1invalid")

	if _, err := invalid.MarshalText(); err == nil {
		t.Error("expected error on MarshalText of invalid key")
	}
	if _, err := invalid.MarshalBinary(); err == nil {
		t.Error("expected error on MarshalBinary of invalid key")
	}
	if _, err := json.Marshal(invalid); err == nil {
		t.Error("expected error on json.Marshal of invalid key")
	}
	if _, err := invalid.MarshalXMLAttr(xml.Name{Local: "attr"}); err == nil {
		t.Error("expected error on MarshalXMLAttr of invalid key")
	}
	if _, err := invalid.Value(); err == nil {
		t.Error("expected error on Value of invalid key")
	}
}
