package xCacheMemory

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	xCacheDriver "github.com/bamboo-services/bamboo-base-go/major/cache/driver"
)

// TestConcurrentHashCacheWrite 验证 hash cache 在高并发写同一 key 下不会 panic 或丢失字段。
//
// 修复前：loadOrCreate → mutate → Set 模式会触发 `concurrent map writes` panic。
// 修复后：所有写操作走 [Store.Update]，单把分片锁保证原子性。
func TestConcurrentHashCacheWrite(t *testing.T) {
	store := NewStore(4, 0, 0)
	defer store.Close()

	hc := NewHashCache[string, string, int, map[string]int](store, xCacheDriver.JSONCodec{}, xCacheDriver.DefaultKeyEncoder{}, 0)
	ctx := context.Background()

	const goroutines = 50
	const fieldsPerG = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < fieldsPerG; i++ {
				field := fmt.Sprintf("g%d-f%d", gid, i)
				val := gid*1000 + i
				if err := hc.Set(ctx, "hash", field, &val); err != nil {
					t.Errorf("Set failed: %v", err)
					return
				}
			}
		}(g)
	}
	wg.Wait()

	all, err := hc.GetAll(ctx, "hash")
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	expected := goroutines * fieldsPerG
	if len(all) != expected {
		t.Fatalf("field count want %d, got %d (lost updates)", expected, len(all))
	}
}

// TestConcurrentSetCacheAdd 验证 set cache 并发 Add 同一 key 不丢成员。
func TestConcurrentSetCacheAdd(t *testing.T) {
	store := NewStore(4, 0, 0)
	defer store.Close()

	sc := NewSetCache[string, int](store, xCacheDriver.JSONCodec{}, xCacheDriver.DefaultKeyEncoder{}, 0)
	ctx := context.Background()

	const goroutines = 50
	const membersPerG = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < membersPerG; i++ {
				member := gid*1000 + i
				if err := sc.Add(ctx, "set", member); err != nil {
					t.Errorf("Add failed: %v", err)
					return
				}
			}
		}(g)
	}
	wg.Wait()

	count, _ := sc.Count(ctx, "set")
	expected := int64(goroutines * membersPerG)
	if count != expected {
		t.Fatalf("member count want %d, got %d (lost members)", expected, count)
	}
}

// TestConcurrentListCacheAppend 验证 list cache 并发 Append 同一 key 不丢元素。
func TestConcurrentListCacheAppend(t *testing.T) {
	store := NewStore(4, 0, 0)
	defer store.Close()

	lc := NewListCache[string, int](store, xCacheDriver.JSONCodec{}, xCacheDriver.DefaultKeyEncoder{}, 0)
	ctx := context.Background()

	const goroutines = 50
	const itemsPerG = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < itemsPerG; i++ {
				val := gid*1000 + i
				if err := lc.Append(ctx, "list", val); err != nil {
					t.Errorf("Append failed: %v", err)
					return
				}
			}
		}(g)
	}
	wg.Wait()

	length, _ := lc.Len(ctx, "list")
	expected := int64(goroutines * itemsPerG)
	if length != expected {
		t.Fatalf("list length want %d, got %d (lost items)", expected, length)
	}
}

// TestConcurrentKeyCacheGetSet 验证 key cache 并发 Get/Set/Delete 混合操作稳定。
func TestConcurrentKeyCacheGetSet(t *testing.T) {
	store := NewStore(4, 0, 100*time.Millisecond)
	defer store.Close()

	kc := NewKeyCache[string, int](store, xCacheDriver.JSONCodec{}, xCacheDriver.DefaultKeyEncoder{}, 100*time.Millisecond)
	ctx := context.Background()

	const goroutines = 30
	const opsPerG = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < opsPerG; i++ {
				key := fmt.Sprintf("k-%d", i%10)
				if gid%2 == 0 {
					_ = kc.Set(ctx, key, &i)
				} else {
					_, _, _ = kc.Get(ctx, key)
				}
			}
		}(g)
	}
	wg.Wait()
}
