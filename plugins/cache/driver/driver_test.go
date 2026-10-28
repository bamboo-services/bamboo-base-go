package driver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver"
)

type stringerKey struct {
	id int
}

func (s stringerKey) String() string {
	return fmt.Sprintf("custom:%d", s.id)
}

type sampleData struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestJSONCodec(t *testing.T) {
	codec := driver.JSONCodec{}
	orig := sampleData{Name: "alice", Age: 18}

	encodedBytes, err := codec.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	expectedJSON, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if !bytes.Equal(encodedBytes, expectedJSON) {
		t.Fatalf("expected marshaled JSON %s, got %s", expectedJSON, encodedBytes)
	}

	var decoded sampleData
	if err := codec.Unmarshal(encodedBytes, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if decoded != orig {
		t.Fatalf("expected %+v, got %+v", orig, decoded)
	}

	var invalid sampleData
	if err := codec.Unmarshal([]byte("invalid json"), &invalid); err == nil {
		t.Fatalf("expected error on invalid JSON, got nil")
	}
}

func TestDefaultKeyEncoder(t *testing.T) {
	enc := driver.DefaultKeyEncoder{}

	tests := []struct {
		name string
		key  any
		want string
	}{
		{"nil", nil, ""},
		{"string", "my-key", "my-key"},
		{"bytes", []byte("byte-key"), "byte-key"},
		{"stringer", stringerKey{id: 42}, "custom:42"},
		{"number", 10086, "10086"},
		{"bool", true, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enc.String(tt.key)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

type prefixEncoder struct {
	prefix string
}

func (p prefixEncoder) String(key any) string {
	return p.prefix + fmt.Sprint(key)
}

func TestCustomKeyEncoderAndEncodeKey(t *testing.T) {
	// Fallback to DefaultKeyEncoder when enc is nil
	if got := driver.EncodeKey(nil, "foo"); got != "foo" {
		t.Fatalf("expected 'foo', got %q", got)
	}
	if got := driver.EncodeKey(nil, 123); got != "123" {
		t.Fatalf("expected '123', got %q", got)
	}

	// Use custom encoder
	custom := prefixEncoder{prefix: "cache:"}
	if got := driver.EncodeKey(custom, "bar"); got != "cache:bar" {
		t.Fatalf("expected 'cache:bar', got %q", got)
	}
}

func TestApplySetAndOptionCombinations(t *testing.T) {
	baseTTL := 10 * time.Second

	// Default
	cfg := driver.ApplySet(baseTTL, nil)
	if cfg.TTL != baseTTL || cfg.NX || cfg.XX || cfg.KeepTTL || cfg.NoSlide {
		t.Fatalf("unexpected default config: %+v", cfg)
	}

	// WithTTL positive
	cfg = driver.ApplySet(baseTTL, []driver.SetOption{driver.WithTTL(5 * time.Minute)})
	if cfg.TTL != 5*time.Minute {
		t.Fatalf("expected 5m TTL, got %v", cfg.TTL)
	}

	// WithTTL <= 0 (permanent)
	cfg = driver.ApplySet(baseTTL, []driver.SetOption{driver.WithTTL(0)})
	if cfg.TTL != 0 {
		t.Fatalf("expected 0 TTL, got %v", cfg.TTL)
	}

	cfg = driver.ApplySet(baseTTL, []driver.SetOption{driver.WithTTL(-1 * time.Second)})
	if cfg.TTL != 0 {
		t.Fatalf("expected 0 TTL, got %v", cfg.TTL)
	}

	// NX overrides XX when both specified
	cfg = driver.ApplySet(baseTTL, []driver.SetOption{driver.WithXX(), driver.WithNX()})
	if !cfg.NX || cfg.XX {
		t.Fatalf("expected NX=true, XX=false, got NX=%v, XX=%v", cfg.NX, cfg.XX)
	}

	cfg = driver.ApplySet(baseTTL, []driver.SetOption{driver.WithNX(), driver.WithXX()})
	if !cfg.NX || cfg.XX {
		t.Fatalf("expected NX=true, XX=false, got NX=%v, XX=%v", cfg.NX, cfg.XX)
	}

	// KeepTTL & NoSlide
	cfg = driver.ApplySet(baseTTL, []driver.SetOption{driver.WithKeepTTL(), driver.WithNoSlide()})
	if !cfg.KeepTTL || !cfg.NoSlide {
		t.Fatalf("expected KeepTTL=true, NoSlide=true, got %+v", cfg)
	}

	// Nil option in slice should be skipped safely
	cfg = driver.ApplySet(baseTTL, []driver.SetOption{nil, driver.WithNX(), nil})
	if !cfg.NX {
		t.Fatalf("expected NX=true, got %+v", cfg)
	}
}

func TestCacheType(t *testing.T) {
	if driver.CacheTypeRedis.String() != "redis" {
		t.Fatalf("expected redis, got %s", driver.CacheTypeRedis.String())
	}
	if !driver.CacheTypeRedis.IsRedis() || driver.CacheTypeRedis.IsMemory() {
		t.Fatalf("invalid redis type checks")
	}
	if !driver.CacheTypeMemory.IsMemory() || driver.CacheTypeMemory.IsRedis() {
		t.Fatalf("invalid memory type checks")
	}
	if !driver.CacheTypeRedis.Enabled() || !driver.CacheTypeMemory.Enabled() {
		t.Fatalf("expected enabled for redis/memory")
	}
	if driver.CacheTypeNone.Enabled() || driver.CacheType("").Enabled() {
		t.Fatalf("expected disabled for none/empty")
	}
}
