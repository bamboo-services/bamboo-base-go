package xCacheMemory

import (
	"context"
	"testing"
	"time"

	xCacheDriver "github.com/bamboo-services/bamboo-base-go/major/cache/driver"
)

func TestMemoryKeyCache(t *testing.T) {
	store := NewStore(4, 100, 100*time.Millisecond)
	defer store.Close()

	kc := NewKeyCache[string, string](store, xCacheDriver.JSONCodec{}, xCacheDriver.DefaultKeyEncoder{}, 100*time.Millisecond)
	ctx := context.Background()

	if err := kc.Set(ctx, "name", strPtr("筱锋")); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	v, ok, err := kc.Get(ctx, "name")
	if err != nil || !ok || v == nil || *v != "筱锋" {
		t.Fatalf("Get want 筱锋, got v=%v ok=%v err=%v", v, ok, err)
	}

	exists, _ := kc.Exists(ctx, "name")
	if !exists {
		t.Fatal("Exists should be true")
	}

	if err := kc.Delete(ctx, "name"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, ok, _ = kc.Get(ctx, "name")
	if ok {
		t.Fatal("after Delete, Get should miss")
	}

	// TTL 过期
	_ = kc.Set(ctx, "ephemeral", strPtr("gone"))
	time.Sleep(150 * time.Millisecond)
	_, ok, _ = kc.Get(ctx, "ephemeral")
	if ok {
		t.Fatal("TTL expired entry should miss")
	}
}

func TestMemoryHashCache(t *testing.T) {
	store := NewStore(0, 0, 0)
	defer store.Close()

	hc := NewHashCache[string, string, string, map[string]string](store, xCacheDriver.JSONCodec{}, xCacheDriver.DefaultKeyEncoder{}, 0)
	ctx := context.Background()

	_ = hc.Set(ctx, "user:1", "name", strPtr("筱锋"))
	_ = hc.Set(ctx, "user:1", "role", strPtr("admin"))

	v, ok, _ := hc.Get(ctx, "user:1", "name")
	if !ok || v == nil || *v != "筱锋" {
		t.Fatalf("Get field name want 筱锋, got %v ok=%v", v, ok)
	}

	all, _ := hc.GetAll(ctx, "user:1")
	if all["name"] != "筱锋" || all["role"] != "admin" {
		t.Fatalf("GetAll mismatch: %+v", all)
	}

	exists, _ := hc.Exists(ctx, "user:1", "role")
	if !exists {
		t.Fatal("Exists field role should be true")
	}

	_ = hc.Remove(ctx, "user:1", "role")
	_, ok, _ = hc.Get(ctx, "user:1", "role")
	if ok {
		t.Fatal("after Remove, field should miss")
	}

	_ = hc.Delete(ctx, "user:1")
	all, _ = hc.GetAll(ctx, "user:1")
	if len(all) != 0 {
		t.Fatalf("after Delete, GetAll should be empty, got %+v", all)
	}
}

func TestMemorySetCache(t *testing.T) {
	store := NewStore(0, 0, 0)
	defer store.Close()

	sc := NewSetCache[string, string](store, xCacheDriver.JSONCodec{}, xCacheDriver.DefaultKeyEncoder{}, 0)
	ctx := context.Background()

	_ = sc.Add(ctx, "tags", "go", "cache", "go") // 去重
	count, _ := sc.Count(ctx, "tags")
	if count != 2 {
		t.Fatalf("Count want 2 (dedup), got %d", count)
	}

	isMem, _ := sc.IsMember(ctx, "tags", "cache")
	if !isMem {
		t.Fatal("IsMember cache should be true")
	}

	members, _ := sc.Members(ctx, "tags")
	if len(members) != 2 {
		t.Fatalf("Members len want 2, got %d", len(members))
	}

	_ = sc.Remove(ctx, "tags", "go")
	count, _ = sc.Count(ctx, "tags")
	if count != 1 {
		t.Fatalf("after Remove, Count want 1, got %d", count)
	}
}

func TestMemoryListCache(t *testing.T) {
	store := NewStore(0, 0, 0)
	defer store.Close()

	lc := NewListCache[string, string](store, xCacheDriver.JSONCodec{}, xCacheDriver.DefaultKeyEncoder{}, 0)
	ctx := context.Background()

	_ = lc.Append(ctx, "queue", "a", "b", "c")
	length, _ := lc.Len(ctx, "queue")
	if length != 3 {
		t.Fatalf("Len want 3, got %d", length)
	}

	ranged, _ := lc.Range(ctx, "queue", 0, -1)
	if len(ranged) != 3 || ranged[0] != "a" || ranged[2] != "c" {
		t.Fatalf("Range mismatch: %+v", ranged)
	}

	// Prepend 语义：Prepend(k, x, y) 后头部为 [x, y, ...]
	_ = lc.Prepend(ctx, "queue", "x", "y")
	ranged, _ = lc.Range(ctx, "queue", 0, 1)
	if ranged[0] != "x" || ranged[1] != "y" {
		t.Fatalf("Prepend order mismatch: %+v", ranged)
	}

	v, _ := lc.Pop(ctx, "queue")
	if v == nil || *v != "x" {
		t.Fatalf("Pop want x, got %v", v)
	}

	v, _ = lc.PopLast(ctx, "queue")
	if v == nil || *v != "c" {
		t.Fatalf("PopLast want c, got %v", v)
	}

	_ = lc.Remove(ctx, "queue", 0, "b")
	length, _ = lc.Len(ctx, "queue")
	if length != 2 {
		t.Fatalf("after Remove b, Len want 2 (y,a remain), got %d", length)
	}
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
