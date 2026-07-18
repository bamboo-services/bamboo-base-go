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
