package memory_test

import (
	"context"
	"fmt"
	"time"

	"github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory"
)

func ExampleStore() {
	// 创建拥有 4 分片、单分片上限 100 条、默认 10 分钟过期的内存存储
	store := memory.NewStore(4, 100, 10*time.Minute)
	defer store.Close()

	store.Set("app:version", "v2.0.0", 0)

	if val, ok := store.Get("app:version"); ok {
		fmt.Println("version:", val)
	}

	fmt.Println("exists:", store.Exists("app:version"))
	store.Delete("app:version")
	fmt.Println("exists after delete:", store.Exists("app:version"))

	// Output:
	// version: v2.0.0
	// exists: true
	// exists after delete: false
}

func ExampleNewKeyCache() {
	ctx := context.Background()
	store := memory.NewStore(4, 100, 5*time.Minute)
	defer store.Close()

	// 泛型键值缓存：string -> string
	keyCache := memory.NewKeyCache[string, string](store, nil, nil, 0)

	val := "bamboo-service"
	if err := keyCache.Set(ctx, "service:name", &val); err != nil {
		panic(err)
	}

	got, ok, err := keyCache.Get(ctx, "service:name")
	if err != nil {
		panic(err)
	}
	if ok && got != nil {
		fmt.Println("found:", *got)
	}

	// Output:
	// found: bamboo-service
}

func ExampleNewHashCache() {
	ctx := context.Background()
	store := memory.NewStore(4, 100, 5*time.Minute)
	defer store.Close()

	type DummyStruct struct{}
	hashCache := memory.NewHashCache[string, string, string, DummyStruct](store, nil, nil, 0)

	score := "98"
	if err := hashCache.Set(ctx, "user:101", "math", &score); err != nil {
		panic(err)
	}

	got, ok, err := hashCache.Get(ctx, "user:101", "math")
	if err != nil {
		panic(err)
	}
	if ok && got != nil {
		fmt.Println("score:", *got)
	}

	// Output:
	// score: 98
}

func ExampleNewSetCache() {
	ctx := context.Background()
	store := memory.NewStore(4, 100, 5*time.Minute)
	defer store.Close()

	setCache := memory.NewSetCache[string, string](store, nil, nil, 0)

	tags := []string{"golang", "cache", "v2"}
	if err := setCache.Add(ctx, "topic:tags", tags); err != nil {
		panic(err)
	}

	isMember, err := setCache.IsMember(ctx, "topic:tags", "golang")
	if err != nil {
		panic(err)
	}
	count, err := setCache.Count(ctx, "topic:tags")
	if err != nil {
		panic(err)
	}

	fmt.Printf("isMember=%v, count=%d\n", isMember, count)

	// Output:
	// isMember=true, count=3
}

func ExampleNewListCache() {
	ctx := context.Background()
	store := memory.NewStore(4, 100, 5*time.Minute)
	defer store.Close()

	listCache := memory.NewListCache[string, string](store, nil, nil, 0)

	// 追加元素
	if err := listCache.Append(ctx, "task:queue", []string{"job1", "job2", "job3"}); err != nil {
		panic(err)
	}

	// 头部弹出一个元素
	first, err := listCache.Pop(ctx, "task:queue")
	if err != nil {
		panic(err)
	}
	length, err := listCache.Len(ctx, "task:queue")
	if err != nil {
		panic(err)
	}

	if first != nil {
		fmt.Printf("popped=%s, remaining=%d\n", *first, length)
	}

	// 范围查询剩余元素
	items, err := listCache.Range(ctx, "task:queue", 0, -1)
	if err != nil {
		panic(err)
	}
	fmt.Printf("items=%v\n", items)

	// Output:
	// popped=job1, remaining=2
	// items=[job2 job3]
}

func ExampleStore_Update() {
	store := memory.NewStore(4, 100, 5*time.Minute)
	defer store.Close()

	// 原子操作累加计数器
	key := "counter:hits"
	store.Set(key, 10, 0)

	store.Update(key, 0, func(old any) any {
		if old == nil {
			return 1
		}
		return old.(int) + 1
	})

	if val, ok := store.Get(key); ok {
		fmt.Println("hits:", val)
	}

	// 若无需变更，返回 UpdateNoChange
	store.Update(key, 0, func(old any) any {
		return memory.UpdateNoChange
	})

	if val, ok := store.Get(key); ok {
		fmt.Println("hits unchanged:", val)
	}

	// Output:
	// hits: 11
	// hits unchanged: 11
}
