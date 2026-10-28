package ctxkey_test

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"flag"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/typed"
)

// 静态接口断言
var (
	_ typed.Operable    = ctxkey.Key("")
	_ typed.Convertible = (*ctxkey.Key)(nil)
	_ driver.Valuer     = ctxkey.Key("")
	_ sql.Scanner       = (*ctxkey.Key)(nil)
	_ flag.Value        = (*ctxkey.Key)(nil)
)

type dummyExposer struct {
	m map[ctxkey.Key]any
}

func (d dummyExposer) Expose() map[ctxkey.Key]any {
	return d.m
}

func TestExposer(t *testing.T) {
	var exp ctxkey.Exposer = dummyExposer{
		m: map[ctxkey.Key]any{
			ctxkey.DatabaseKey: "db-instance",
		},
	}
	res := exp.Expose()
	if res[ctxkey.DatabaseKey] != "db-instance" {
		t.Fatalf("unexpected Expose result: %v", res)
	}
}

func TestKey_Operable(t *testing.T) {
	k := ctxkey.RequestKey
	if k.String() != "context_request_key" {
		t.Errorf("expected string context_request_key, got %s", k.String())
	}
	if k.TypedString() != typed.String("context_request_key") {
		t.Errorf("expected typed.String context_request_key, got %s", k.TypedString())
	}
	if k.IsEmpty() {
		t.Errorf("expected non-empty for RequestKey")
	}
	if !ctxkey.Nil.IsEmpty() {
		t.Errorf("expected empty for Nil")
	}
	if !k.IsValid() {
		t.Errorf("expected valid for RequestKey")
	}
	if ctxkey.Key("1invalid").IsValid() {
		t.Errorf("expected invalid for 1invalid")
	}
}

func TestKey_SixChannels_RoundTrip(t *testing.T) {
	valid := ctxkey.DatabaseKey

	// 1. Text
	textBytes, err := valid.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText failed: %v", err)
	}
	var textDec ctxkey.Key
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
	var jsonDec ctxkey.Key
	if err := json.Unmarshal(jsonBytes, &jsonDec); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if jsonDec != valid {
		t.Errorf("JSON roundtrip mismatch: got %v, want %v", jsonDec, valid)
	}

	// 3. XML (Element & Attribute)
	type XMLStruct struct {
		XMLName xml.Name   `xml:"root"`
		Elem    ctxkey.Key `xml:"elem"`
		Attr    ctxkey.Key `xml:"attr,attr"`
	}
	xmlIn := XMLStruct{Elem: valid, Attr: ctxkey.RequestKey}
	xmlBytes, err := xml.Marshal(xmlIn)
	if err != nil {
		t.Fatalf("xml.Marshal failed: %v", err)
	}
	var xmlOut XMLStruct
	if err := xml.Unmarshal(xmlBytes, &xmlOut); err != nil {
		t.Fatalf("xml.Unmarshal failed: %v", err)
	}
	if xmlOut.Elem != valid || xmlOut.Attr != ctxkey.RequestKey {
		t.Errorf("XML roundtrip mismatch: got elem=%v attr=%v", xmlOut.Elem, xmlOut.Attr)
	}

	// 4. Binary (gob roundtrip)
	var gobBuf bytes.Buffer
	enc := gob.NewEncoder(&gobBuf)
	if err := enc.Encode(valid); err != nil {
		t.Fatalf("gob.Encode failed: %v", err)
	}
	var gobDec ctxkey.Key
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
	var sqlDec ctxkey.Key
	if err := sqlDec.Scan(val); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if sqlDec != valid {
		t.Errorf("SQL roundtrip mismatch: got %v, want %v", sqlDec, valid)
	}

	// 6. CLI Flag
	var flagKey ctxkey.Key
	if err := flagKey.Set(string(valid)); err != nil {
		t.Fatalf("flag.Set failed: %v", err)
	}
	if flagKey != valid {
		t.Errorf("Flag Set mismatch: got %v, want %v", flagKey, valid)
	}
}

func TestKey_NilReceiver_Safety(t *testing.T) {
	var k *ctxkey.Key

	if err := k.UnmarshalText([]byte("valid_key")); err == nil {
		t.Error("expected error on nil UnmarshalText")
	}
	if err := k.UnmarshalBinary([]byte("valid_key")); err == nil {
		t.Error("expected error on nil UnmarshalBinary")
	}
	if err := k.UnmarshalJSON([]byte(`"valid_key"`)); err == nil {
		t.Error("expected error on nil UnmarshalJSON")
	}
	if err := k.UnmarshalXML(nil, xml.StartElement{}); err == nil {
		t.Error("expected error on nil UnmarshalXML")
	}
	if err := k.UnmarshalXMLAttr(xml.Attr{Value: "valid_key"}); err == nil {
		t.Error("expected error on nil UnmarshalXMLAttr")
	}
	if err := k.Scan("valid_key"); err == nil {
		t.Error("expected error on nil Scan")
	}
	if err := k.Set("valid_key"); err == nil {
		t.Error("expected error on nil Set")
	}
}

func TestKey_InvalidInput_PreservesOriginal(t *testing.T) {
	const original = ctxkey.DatabaseKey

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
		Key ctxkey.Key `xml:"k"`
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
		Key ctxkey.Key `xml:"k,attr"`
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
	invalid := ctxkey.Key("1invalid")

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
