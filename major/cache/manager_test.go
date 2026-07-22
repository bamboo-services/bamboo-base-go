package xCache_test

import (
	"context"
	"testing"
	"time"

	xCache "github.com/bamboo-services/bamboo-base-go/major/cache"
	xCacheMemory "github.com/bamboo-services/bamboo-base-go/major/cache/memory"
)

func TestManagerDispatch(t *testing.T) {
	store := xCacheMemory.NewStore(0, 0, 0)
	defer store.Close()

	m := xCache.NewManager(xCache.CacheTypeMemory,
		xCache.WithMemoryStore(store),
		xCache.WithManagerTTL(50*time.Millisecond),
	)
	if m.Type() != xCache.CacheTypeMemory {
		t.Fatalf("Type want memory, got %s", m.Type())
	}
	if m.Memory() == nil {
		t.Fatal("Memory() should not be nil for memory backend")
	}
	if m.Redis() != nil {
		t.Fatal("Redis() should be nil for memory backend")
	}

	kc := xCache.KeyCacheOf[string, int](m)
	if kc == nil {
		t.Fatal("KeyCacheOf returned nil")
	}
	ctx := context.Background()
	_ = kc.Set(ctx, "count", intPtr(42))
	v, ok, _ := kc.Get(ctx, "count")
	if !ok || v == nil || *v != 42 {
		t.Fatalf("Get want 42, got %v ok=%v", v, ok)
	}
}

func intPtr(i int) *int { return &i }

func TestManagerSetWithTTL(t *testing.T) {
	store := xCacheMemory.NewStore(0, 0, 0)
	defer store.Close()

	// 默认 TTL 50ms，但用 WithTTL 覆盖为 500ms
	m := xCache.NewManager(xCache.CacheTypeMemory,
		xCache.WithMemoryStore(store),
		xCache.WithManagerTTL(50*time.Millisecond),
	)
	kc := xCache.KeyCacheOf[string, int](m)
	ctx := context.Background()

	_ = kc.Set(ctx, "short", intPtr(1))                          // 用默认 50ms
	_ = kc.Set(ctx, "long", intPtr(2), xCache.WithTTL(500*time.Millisecond)) // 覆盖为 500ms

	if _, ok, _ := kc.Get(ctx, "short"); !ok {
		t.Fatal("short should exist immediately after Set")
	}

	time.Sleep(80 * time.Millisecond)
	if _, ok, _ := kc.Get(ctx, "short"); ok {
		t.Fatal("short (default TTL 50ms) should have expired")
	}
	if _, ok, _ := kc.Get(ctx, "long"); !ok {
		t.Fatal("long (WithTTL 500ms) should still exist")
	}
}

func TestManagerSetWithTTLNoExpire(t *testing.T) {
	store := xCacheMemory.NewStore(0, 0, 0)
	defer store.Close()

	// 默认 TTL 50ms，用 WithTTL(0) 覆盖为永不过期
	m := xCache.NewManager(xCache.CacheTypeMemory,
		xCache.WithMemoryStore(store),
		xCache.WithManagerTTL(50*time.Millisecond),
	)
	kc := xCache.KeyCacheOf[string, int](m)
	ctx := context.Background()

	_ = kc.Set(ctx, "perm", intPtr(1), xCache.WithTTL(0))

	time.Sleep(80 * time.Millisecond)
	if _, ok, _ := kc.Get(ctx, "perm"); !ok {
		t.Fatal("perm (WithTTL 0 = no expire) should still exist")
	}
}

// TestWithNX 验证 WithNX 仅当键不存在时写入。
func TestWithNX(t *testing.T) {
	store := xCacheMemory.NewStore(0, 0, 0)
	defer store.Close()

	m := xCache.NewManager(xCache.CacheTypeMemory,
		xCache.WithMemoryStore(store),
		xCache.WithManagerTTL(0),
	)
	kc := xCache.KeyCacheOf[string, int](m)
	ctx := context.Background()

	// 不存在的 key 用 WithNX 写入成功
	if err := kc.Set(ctx, "lock", intPtr(1), xCache.WithNX()); err != nil {
		t.Fatalf("WithNX set on absent key should succeed, got err: %v", err)
	}
	v, ok, _ := kc.Get(ctx, "lock")
	if !ok || v == nil || *v != 1 {
		t.Fatalf("Get want 1, got %v ok=%v", v, ok)
	}

	// 已存在的 key 用 WithNX 写入应被跳过（返回 nil 且原值不变）
	if err := kc.Set(ctx, "lock", intPtr(2), xCache.WithNX()); err != nil {
		t.Fatalf("WithNX set on existing key should return nil err, got: %v", err)
	}
	v, ok, _ = kc.Get(ctx, "lock")
	if !ok || v == nil || *v != 1 {
		t.Fatalf("WithNX should not overwrite existing key, want 1, got %v ok=%v", v, ok)
	}
}

// TestWithXX 验证 WithXX 仅当键已存在时写入。
func TestWithXX(t *testing.T) {
	store := xCacheMemory.NewStore(0, 0, 0)
	defer store.Close()

	m := xCache.NewManager(xCache.CacheTypeMemory,
		xCache.WithMemoryStore(store),
		xCache.WithManagerTTL(0),
	)
	kc := xCache.KeyCacheOf[string, int](m)
	ctx := context.Background()

	// 不存在的 key 用 WithXX 写入应被跳过
	if err := kc.Set(ctx, "cfg", intPtr(1), xCache.WithXX()); err != nil {
		t.Fatalf("WithXX set on absent key should return nil err, got: %v", err)
	}
	if _, ok, _ := kc.Get(ctx, "cfg"); ok {
		t.Fatal("WithXX should not create absent key")
	}

	// 先创建 key，再用 WithXX 更新成功
	_ = kc.Set(ctx, "cfg", intPtr(1))
	if err := kc.Set(ctx, "cfg", intPtr(2), xCache.WithXX()); err != nil {
		t.Fatalf("WithXX set on existing key should succeed, got err: %v", err)
	}
	v, ok, _ := kc.Get(ctx, "cfg")
	if !ok || v == nil || *v != 2 {
		t.Fatalf("Get want 2, got %v ok=%v", v, ok)
	}
}

// TestWithKeepTTL 验证 WithKeepTTL 覆盖值但保留原 TTL。
func TestWithKeepTTL(t *testing.T) {
	store := xCacheMemory.NewStore(0, 0, 0)
	defer store.Close()

	m := xCache.NewManager(xCache.CacheTypeMemory,
		xCache.WithMemoryStore(store),
		xCache.WithManagerTTL(0),
	)
	kc := xCache.KeyCacheOf[string, int](m)
	ctx := context.Background()

	// 先设 key 带 TTL 200ms
	_ = kc.Set(ctx, "k", intPtr(1), xCache.WithTTL(200*time.Millisecond))
	// 用 WithKeepTTL 覆盖值（不应重设 TTL）
	_ = kc.Set(ctx, "k", intPtr(2), xCache.WithKeepTTL())

	// Sleep 100ms 后值还在（200ms TTL 未过期）
	time.Sleep(100 * time.Millisecond)
	v, ok, _ := kc.Get(ctx, "k")
	if !ok || v == nil || *v != 2 {
		t.Fatalf("after 100ms want 2 (KeepTTL preserved), got %v ok=%v", v, ok)
	}

	// 再 Sleep 150ms（总计 250ms > 200ms），值应过期
	time.Sleep(150 * time.Millisecond)
	if _, ok, _ := kc.Get(ctx, "k"); ok {
		t.Fatal("after 250ms key should expire (original 200ms TTL preserved by KeepTTL)")
	}
}

// TestWithNoSlide 验证 WithNoSlide 追加数据但不滑动 TTL。
func TestWithNoSlide(t *testing.T) {
	store := xCacheMemory.NewStore(0, 0, 0)
	defer store.Close()

	// 默认 TTL 100ms
	m := xCache.NewManager(xCache.CacheTypeMemory,
		xCache.WithMemoryStore(store),
		xCache.WithManagerTTL(100*time.Millisecond),
	)
	sc := xCache.SetCacheOf[string, string](m)
	ctx := context.Background()

	// 先添加成员（集合创建，TTL 100ms）
	_ = sc.Add(ctx, "set", []string{"a"})
	// 再用 WithNoSlide 追加成员（不应续期）
	_ = sc.Add(ctx, "set", []string{"b"}, xCache.WithNoSlide())

	// Sleep 120ms 后集合因原 TTL 过期而消失
	time.Sleep(120 * time.Millisecond)
	count, _ := sc.Count(ctx, "set")
	if count != 0 {
		t.Fatalf("set should expire after 120ms (NoSlide did not extend TTL), got count=%d", count)
	}
}

// TestNXXXMutualExclusion 验证 WithNX+WithXX 同传时 NX 优先。
func TestNXXXMutualExclusion(t *testing.T) {
	store := xCacheMemory.NewStore(0, 0, 0)
	defer store.Close()

	m := xCache.NewManager(xCache.CacheTypeMemory,
		xCache.WithMemoryStore(store),
		xCache.WithManagerTTL(0),
	)
	kc := xCache.KeyCacheOf[string, int](m)
	ctx := context.Background()

	// 先创建 key=1
	_ = kc.Set(ctx, "k", intPtr(1))

	// 对已存在的 key 用 WithNX()+WithXX() 写入 → NX 优先，key 已存在，不写入
	_ = kc.Set(ctx, "k", intPtr(2), xCache.WithNX(), xCache.WithXX())
	v, ok, _ := kc.Get(ctx, "k")
	if !ok || v == nil || *v != 1 {
		t.Fatalf("NX+XX mutual exclusion: NX should win, want 1, got %v ok=%v", v, ok)
	}
}

// TestHashWithNX 验证 HashCache.Set 对 field 的 WithNX 语义。
func TestHashWithNX(t *testing.T) {
	store := xCacheMemory.NewStore(0, 0, 0)
	defer store.Close()

	m := xCache.NewManager(xCache.CacheTypeMemory,
		xCache.WithMemoryStore(store),
		xCache.WithManagerTTL(0),
	)
	hc := xCache.HashCacheOf[string, string, string, map[string]string](m)
	ctx := context.Background()

	// 不存在的 field 用 WithNX 写入成功
	if err := hc.Set(ctx, "h", "f1", strPtr("v1"), xCache.WithNX()); err != nil {
		t.Fatalf("WithNX set on absent field should succeed, got err: %v", err)
	}
	v, ok, _ := hc.Get(ctx, "h", "f1")
	if !ok || v == nil || *v != "v1" {
		t.Fatalf("Get want v1, got %v ok=%v", v, ok)
	}

	// 已存在的 field 用 WithNX 写入应被跳过，原值不变
	if err := hc.Set(ctx, "h", "f1", strPtr("v2"), xCache.WithNX()); err != nil {
		t.Fatalf("WithNX set on existing field should return nil err, got: %v", err)
	}
	v, ok, _ = hc.Get(ctx, "h", "f1")
	if !ok || v == nil || *v != "v1" {
		t.Fatalf("WithNX should not overwrite existing field, want v1, got %v ok=%v", v, ok)
	}
}

// TestListWithNoSlide 验证 ListCache.Append 用 WithNoSlide 不滑动 TTL。
func TestListWithNoSlide(t *testing.T) {
	store := xCacheMemory.NewStore(0, 0, 0)
	defer store.Close()

	// 默认 TTL 100ms
	m := xCache.NewManager(xCache.CacheTypeMemory,
		xCache.WithMemoryStore(store),
		xCache.WithManagerTTL(100*time.Millisecond),
	)
	lc := xCache.ListCacheOf[string, string](m)
	ctx := context.Background()

	// 先追加元素（列表创建，TTL 100ms）
	_ = lc.Append(ctx, "list", []string{"a"})
	// 再用 WithNoSlide 追加元素（不应续期）
	_ = lc.Append(ctx, "list", []string{"b"}, xCache.WithNoSlide())

	// Sleep 120ms 后列表因原 TTL 过期而消失
	time.Sleep(120 * time.Millisecond)
	length, _ := lc.Len(ctx, "list")
	if length != 0 {
		t.Fatalf("list should expire after 120ms (NoSlide did not extend TTL), got len=%d", length)
	}
}

func strPtr(s string) *string { return &s }
