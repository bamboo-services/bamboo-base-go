package typed

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"strings"
	"testing"
)

// 测试 Text 通道编解码、校验与原子性。
func TestConvert_Text(t *testing.T) {
	valid := String("db_host")
	data, err := valid.MarshalText()
	if err != nil || string(data) != "db_host" {
		t.Fatalf("MarshalText valid failed: %v, data: %s", err, data)
	}

	invalid := String("db__host")
	if _, err := invalid.MarshalText(); err == nil {
		t.Fatalf("MarshalText invalid expected error")
	}

	var target String = "initial_val"
	if err := target.UnmarshalText([]byte("new_db")); err != nil || target != "new_db" {
		t.Fatalf("UnmarshalText failed: %v, target: %s", err, target)
	}

	// 校验失败保留旧值
	if err := target.UnmarshalText([]byte("bad__val")); err == nil || target != "new_db" {
		t.Fatalf("UnmarshalText should fail and preserve old value, got %s", target)
	}

	// nil 接收器
	var nilPtr *String
	if err := nilPtr.UnmarshalText([]byte("valid")); !errors.Is(err, ErrNilReceiver) {
		t.Fatalf("expected ErrNilReceiver, got %v", err)
	}
}

// 测试 Binary 通道编解码与 gob 真实路径。
func TestConvert_BinaryAndGob(t *testing.T) {
	valid := String("redis_key")
	data, err := valid.MarshalBinary()
	if err != nil || string(data) != "redis_key" {
		t.Fatalf("MarshalBinary valid failed: %v, data: %s", err, data)
	}

	invalid := String("-invalid")
	if _, err := invalid.MarshalBinary(); err == nil {
		t.Fatalf("MarshalBinary invalid expected error")
	}

	var target String = "old_bin"
	if err := target.UnmarshalBinary([]byte("new_bin")); err != nil || target != "new_bin" {
		t.Fatalf("UnmarshalBinary failed: %v, got: %s", err, target)
	}

	if err := target.UnmarshalBinary([]byte("bad__bin")); err == nil || target != "new_bin" {
		t.Fatalf("UnmarshalBinary should preserve old value on error")
	}

	var nilPtr *String
	if err := nilPtr.UnmarshalBinary([]byte("valid")); !errors.Is(err, ErrNilReceiver) {
		t.Fatalf("expected ErrNilReceiver, got %v", err)
	}

	// gob 真实序列化与反序列化
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(valid); err != nil {
		t.Fatalf("gob encode failed: %v", err)
	}

	var decoded String
	dec := gob.NewDecoder(&buf)
	if err := dec.Decode(&decoded); err != nil || decoded != valid {
		t.Fatalf("gob decode failed: %v, got %s", err, decoded)
	}
}

// 测试 JSON 通道：null拒绝、非字符串拒绝、空键拒绝、有效序列化及原子写回。
func TestConvert_JSON(t *testing.T) {
	valid := String("user_name")
	data, err := json.Marshal(valid)
	if err != nil || string(data) != `"user_name"` {
		t.Fatalf("json Marshal failed: %v, data: %s", err, data)
	}

	invalid := String("user__name")
	if _, err := json.Marshal(invalid); err == nil {
		t.Fatalf("json Marshal invalid expected error")
	}

	var target String = "initial_json"
	if err := json.Unmarshal([]byte(`"new_user"`), &target); err != nil || target != "new_user" {
		t.Fatalf("json Unmarshal failed: %v, target: %s", err, target)
	}

	// 拒绝 JSON null（保持旧值）
	if err := json.Unmarshal([]byte("null"), &target); err == nil || target != "new_user" {
		t.Fatalf("json Unmarshal null must fail and preserve old value, got %s", target)
	}

	// 拒绝非字符串（数字、对象、布尔等）
	if err := json.Unmarshal([]byte("12345"), &target); err == nil || target != "new_user" {
		t.Fatalf("json Unmarshal number must fail and preserve old value")
	}

	// 拒绝空键
	if err := json.Unmarshal([]byte(`""`), &target); err == nil || target != "new_user" {
		t.Fatalf("json Unmarshal empty string must fail and preserve old value")
	}

	// nil 接收器
	var nilPtr *String
	if err := nilPtr.UnmarshalJSON([]byte(`"valid"`)); !errors.Is(err, ErrNilReceiver) {
		t.Fatalf("expected ErrNilReceiver, got %v", err)
	}
}

// 测试 XML 元素与 XML 属性通道。
func TestConvert_XML(t *testing.T) {
	type XMLStruct struct {
		XMLName xml.Name `xml:"config"`
		Elem    String   `xml:"elem_node"`
		Attr    String   `xml:"attr_node,attr"`
	}

	validObj := XMLStruct{Elem: "elem_val", Attr: "attr_val"}
	xmlData, err := xml.Marshal(validObj)
	if err != nil {
		t.Fatalf("xml Marshal failed: %v", err)
	}

	var parsed XMLStruct
	if err := xml.Unmarshal(xmlData, &parsed); err != nil || parsed.Elem != "elem_val" || parsed.Attr != "attr_val" {
		t.Fatalf("xml Unmarshal failed: %v, parsed: %+v", err, parsed)
	}

	// 元素解码错误时保持旧值
	var target String = "old_xml"
	invalidElemXML := `<String>invalid__name</String>`
	if err := xml.Unmarshal([]byte(invalidElemXML), &target); err == nil || target != "old_xml" {
		t.Fatalf("xml unmarshal invalid elem should fail and preserve old value")
	}

	// 嵌套元素拒绝
	nestedXML := `<String><nested>sub</nested></String>`
	if err := xml.Unmarshal([]byte(nestedXML), &target); err == nil || target != "old_xml" {
		t.Fatalf("xml unmarshal nested elem should fail")
	}

	// XML 属性无效报错保持旧值
	var attrTarget String = "old_attr"
	if err := attrTarget.UnmarshalXMLAttr(xml.Attr{Value: "bad__attr"}); err == nil || attrTarget != "old_attr" {
		t.Fatalf("UnmarshalXMLAttr invalid should fail and preserve old value")
	}

	// nil 接收器
	var nilPtr *String
	if err := nilPtr.UnmarshalXML(xml.NewDecoder(strings.NewReader("<tag>val</tag>")), xml.StartElement{}); !errors.Is(err, ErrNilReceiver) {
		t.Fatalf("expected ErrNilReceiver on UnmarshalXML, got %v", err)
	}
	if err := nilPtr.UnmarshalXMLAttr(xml.Attr{Value: "val"}); !errors.Is(err, ErrNilReceiver) {
		t.Fatalf("expected ErrNilReceiver on UnmarshalXMLAttr, got %v", err)
	}
}

// 测试 SQL 通道：Valuer、Scanner、NULL 拒绝与错类型拒绝。
func TestConvert_SQL(t *testing.T) {
	valid := String("db_col")
	val, err := valid.Value()
	if err != nil || val != "db_col" {
		t.Fatalf("SQL Value valid failed: %v, val: %v", err, val)
	}

	invalid := String("db__col")
	if _, err := invalid.Value(); err == nil {
		t.Fatalf("SQL Value invalid expected error")
	}

	var target String = "initial_sql"
	// 支持 string
	if err := target.Scan("new_col"); err != nil || target != "new_col" {
		t.Fatalf("SQL Scan string failed: %v, target: %s", err, target)
	}

	// 支持 []byte
	if err := target.Scan([]byte("byte_col")); err != nil || target != "byte_col" {
		t.Fatalf("SQL Scan []byte failed: %v, target: %s", err, target)
	}

	// 拒绝 SQL NULL 保持旧值
	if err := target.Scan(nil); err == nil || target != "byte_col" {
		t.Fatalf("SQL Scan nil must fail and preserve old value")
	}

	// 拒绝错误类型（例如 int）保持旧值
	if err := target.Scan(12345); err == nil || target != "byte_col" {
		t.Fatalf("SQL Scan int must fail and preserve old value")
	}

	// nil 接收器
	var nilPtr *String
	if err := nilPtr.Scan("test"); !errors.Is(err, ErrNilReceiver) {
		t.Fatalf("expected ErrNilReceiver, got %v", err)
	}
}

// 测试 flag.FlagSet CLI 通道。
func TestConvert_Flag(t *testing.T) {
	fs := flag.NewFlagSet("test_flag", flag.ContinueOnError)
	var flagVal String = "init_flag"
	fs.Var(&flagVal, "config_key", "usage")

	if err := fs.Parse([]string{"-config_key", "parsed_key"}); err != nil || flagVal != "parsed_key" {
		t.Fatalf("flag Parse failed: %v, flagVal: %s", err, flagVal)
	}

	// 非法值拒绝且保持原值
	if err := fs.Parse([]string{"-config_key", "bad__flag"}); err == nil || flagVal != "parsed_key" {
		t.Fatalf("flag Parse bad key must fail and preserve old value")
	}

	// nil 接收器
	var nilPtr *String
	if err := nilPtr.Set("valid"); !errors.Is(err, ErrNilReceiver) {
		t.Fatalf("expected ErrNilReceiver, got %v", err)
	}
}
