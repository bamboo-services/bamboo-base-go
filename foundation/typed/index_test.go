package typed

import (
	"bytes"
	"database/sql/driver"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"testing"
)

// 验证常量定义与类型契约断言。
func TestIndex_ConstantsAndInterfaces(t *testing.T) {
	if MaxKeyBytes != 128 {
		t.Fatalf("expected MaxKeyBytes to be 128, got %d", MaxKeyBytes)
	}

	var _ Operable = String("")
	var _ Convertible = (*String)(nil)
}

// 模拟派生领域词汇（参考设计稿中 ctxkey.Key 与 env.Key 范式）。
type mockDomainKey String

var (
	_ Operable    = mockDomainKey("")
	_ Convertible = (*mockDomainKey)(nil)
)

func (k mockDomainKey) String() string {
	return string(k)
}

func (k mockDomainKey) TypedString() String {
	return String(k)
}

func (k mockDomainKey) IsEmpty() bool {
	return len(k) == 0
}

func (k mockDomainKey) IsValid() bool {
	return k.TypedString().IsValid()
}

func (k mockDomainKey) MarshalText() ([]byte, error) {
	return k.TypedString().MarshalText()
}

func (k *mockDomainKey) UnmarshalText(text []byte) error {
	if k == nil {
		return fmt.Errorf("mockDomainKey: nil receiver")
	}
	var candidate String
	if err := candidate.UnmarshalText(text); err != nil {
		return err
	}
	*k = mockDomainKey(candidate)
	return nil
}

func (k mockDomainKey) MarshalBinary() ([]byte, error) {
	return k.TypedString().MarshalBinary()
}

func (k *mockDomainKey) UnmarshalBinary(data []byte) error {
	if k == nil {
		return fmt.Errorf("mockDomainKey: nil receiver")
	}
	var candidate String
	if err := candidate.UnmarshalBinary(data); err != nil {
		return err
	}
	*k = mockDomainKey(candidate)
	return nil
}

func (k mockDomainKey) MarshalJSON() ([]byte, error) {
	return k.TypedString().MarshalJSON()
}

func (k *mockDomainKey) UnmarshalJSON(raw []byte) error {
	if k == nil {
		return fmt.Errorf("mockDomainKey: nil receiver")
	}
	var candidate String
	if err := candidate.UnmarshalJSON(raw); err != nil {
		return err
	}
	*k = mockDomainKey(candidate)
	return nil
}

func (k mockDomainKey) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return k.TypedString().MarshalXML(e, start)
}

func (k *mockDomainKey) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if k == nil {
		return fmt.Errorf("mockDomainKey: nil receiver")
	}
	var candidate String
	if err := candidate.UnmarshalXML(d, start); err != nil {
		return err
	}
	*k = mockDomainKey(candidate)
	return nil
}

func (k mockDomainKey) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return k.TypedString().MarshalXMLAttr(name)
}

func (k *mockDomainKey) UnmarshalXMLAttr(attr xml.Attr) error {
	if k == nil {
		return fmt.Errorf("mockDomainKey: nil receiver")
	}
	var candidate String
	if err := candidate.UnmarshalXMLAttr(attr); err != nil {
		return err
	}
	*k = mockDomainKey(candidate)
	return nil
}

func (k mockDomainKey) Value() (driver.Value, error) {
	return k.TypedString().Value()
}

func (k *mockDomainKey) Scan(src any) error {
	if k == nil {
		return fmt.Errorf("mockDomainKey: nil receiver")
	}
	var candidate String
	if err := candidate.Scan(src); err != nil {
		return err
	}
	*k = mockDomainKey(candidate)
	return nil
}

func (k *mockDomainKey) Set(raw string) error {
	if k == nil {
		return fmt.Errorf("mockDomainKey: nil receiver")
	}
	var candidate String
	if err := candidate.Set(raw); err != nil {
		return err
	}
	*k = mockDomainKey(candidate)
	return nil
}

// 验证派生词汇的 Operable 与 Convertible 显式委托、nil 安全及原子回滚。
func TestDerivedKey_OperableAndConvertible(t *testing.T) {
	k := mockDomainKey("context_database")
	if k.String() != "context_database" {
		t.Fatalf("unexpected String(): %s", k.String())
	}
	if k.TypedString() != String("context_database") {
		t.Fatalf("unexpected TypedString(): %s", k.TypedString())
	}
	if k.IsEmpty() {
		t.Fatalf("expected non-empty")
	}
	if !k.IsValid() {
		t.Fatalf("expected valid key")
	}

	// 1. Text 通道
	text, err := k.MarshalText()
	if err != nil || string(text) != "context_database" {
		t.Fatalf("MarshalText failed: %v, text: %s", err, text)
	}
	var kText mockDomainKey = "initial"
	if err := kText.UnmarshalText([]byte("new_key")); err != nil || kText != "new_key" {
		t.Fatalf("UnmarshalText failed: %v, got %s", err, kText)
	}
	if err := kText.UnmarshalText([]byte("bad__key")); err == nil || kText != "new_key" {
		t.Fatalf("UnmarshalText invalid should fail and preserve old value")
	}

	// 2. Binary 通道及 gob 真实编解码路径
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(k); err != nil {
		t.Fatalf("gob encode failed: %v", err)
	}
	var kGob mockDomainKey
	dec := gob.NewDecoder(&buf)
	if err := dec.Decode(&kGob); err != nil || kGob != k {
		t.Fatalf("gob decode failed: %v, got %s", err, kGob)
	}

	// 3. JSON 通道
	jsonData, err := json.Marshal(k)
	if err != nil || string(jsonData) != `"context_database"` {
		t.Fatalf("json marshal failed: %v, data: %s", err, jsonData)
	}
	var kJSON mockDomainKey = "initial"
	if err := json.Unmarshal(jsonData, &kJSON); err != nil || kJSON != k {
		t.Fatalf("json unmarshal failed: %v, got %s", err, kJSON)
	}
	if err := json.Unmarshal([]byte("null"), &kJSON); err == nil || kJSON != k {
		t.Fatalf("json unmarshal null should fail and preserve old value")
	}

	// 4. XML 元素与属性通道
	type xmlContainer struct {
		XMLName xml.Name      `xml:"root"`
		Elem    mockDomainKey `xml:"elem"`
		Attr    mockDomainKey `xml:"attr,attr"`
	}
	xmlInput := xmlContainer{Elem: "elem_key", Attr: "attr_key"}
	xmlData, err := xml.Marshal(xmlInput)
	if err != nil {
		t.Fatalf("xml marshal failed: %v", err)
	}
	var xmlOutput xmlContainer
	if err := xml.Unmarshal(xmlData, &xmlOutput); err != nil || xmlOutput.Elem != "elem_key" || xmlOutput.Attr != "attr_key" {
		t.Fatalf("xml unmarshal failed: %v, got %+v", err, xmlOutput)
	}

	// 5. SQL 通道
	val, err := k.Value()
	if err != nil || val != "context_database" {
		t.Fatalf("SQL Value failed: %v, got %v", err, val)
	}
	var kSQL mockDomainKey = "initial"
	if err := kSQL.Scan("new_sql_key"); err != nil || kSQL != "new_sql_key" {
		t.Fatalf("SQL Scan failed: %v", err)
	}
	if err := kSQL.Scan(nil); err == nil || kSQL != "new_sql_key" {
		t.Fatalf("SQL Scan nil should fail and preserve old value")
	}

	// 6. CLI Flag 通道
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	var kFlag mockDomainKey = "initial"
	fs.Var(&kFlag, "flag_key", "usage")
	if err := fs.Parse([]string{"-flag_key", "valid_flag"}); err != nil || kFlag != "valid_flag" {
		t.Fatalf("flag parse failed: %v, got %s", err, kFlag)
	}
	if err := fs.Parse([]string{"-flag_key", "bad__flag"}); err == nil || kFlag != "valid_flag" {
		t.Fatalf("flag parse invalid should fail and preserve old value")
	}

	// 7. Nil 接收器安全性
	var nilKey *mockDomainKey
	if err := nilKey.UnmarshalText([]byte("a")); err == nil {
		t.Fatalf("expected error on nil receiver UnmarshalText")
	}
	if err := nilKey.UnmarshalBinary([]byte("a")); err == nil {
		t.Fatalf("expected error on nil receiver UnmarshalBinary")
	}
	if err := nilKey.UnmarshalJSON([]byte(`"a"`)); err == nil {
		t.Fatalf("expected error on nil receiver UnmarshalJSON")
	}
	if err := nilKey.Scan("a"); err == nil {
		t.Fatalf("expected error on nil receiver Scan")
	}
	if err := nilKey.Set("a"); err == nil {
		t.Fatalf("expected error on nil receiver Set")
	}
	if err := nilKey.UnmarshalXMLAttr(xml.Attr{Value: "a"}); err == nil {
		t.Fatalf("expected error on nil receiver UnmarshalXMLAttr")
	}
}
