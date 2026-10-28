package cache_test

import (
	"context"
	"fmt"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/cache/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory"
)

func ExampleKeyCacheOf() {
	ctx := context.Background()

	// 构造内存缓存管理器实例
	store := memory.NewStore(4, 100, 10*time.Minute)
	mgr := cache.NewManager(
		cache.CacheTypeMemory,
		cache.WithMemoryStore(store),
		cache.WithManagerTTL(5*time.Minute),
	)
	defer mgr.Close()

	// 获取泛型键值缓存：string -> string
	userCache := cache.KeyCacheOf[string, string](mgr)

	name := "Konata"
	if err := userCache.Set(ctx, "user:name", &name); err != nil {
		panic(err)
	}

	got, ok, err := userCache.Get(ctx, "user:name")
	if err != nil {
		panic(err)
	}
	if ok && got != nil {
		fmt.Println("user:", *got)
	}

	// Output:
	// user: Konata
}

func ExampleHashCacheOf() {
	ctx := context.Background()

	store := memory.NewStore(4, 100, 10*time.Minute)
	mgr := cache.NewManager(
		cache.CacheTypeMemory,
		cache.WithMemoryStore(store),
	)
	defer mgr.Close()

	type DummyStruct struct{}
	hashCache := cache.HashCacheOf[string, string, int, DummyStruct](mgr)

	age := 18
	if err := hashCache.Set(ctx, "player:1", "age", &age); err != nil {
		panic(err)
	}

	got, ok, err := hashCache.Get(ctx, "player:1", "age")
	if err != nil {
		panic(err)
	}
	if ok && got != nil {
		fmt.Println("age:", *got)
	}

	// Output:
	// age: 18
}

// ExampleNew 演示将缓存模块作为受管插件注册到应用中。
func ExampleNew() {
	app := application.New()

	// 构造内存缓存受管插件
	info, factory := cache.New(
		cache.WithMemory(
			cache.WithMemoryDefaultTTL(30*time.Minute),
			cache.WithMemoryMaxEntries(5000),
			cache.WithMemoryShardCount(16),
		),
	)

	if err := app.RegisterTool(info, factory); err != nil {
		panic(err)
	}
}

// ExampleWithRedis 演示构造 Redis 后端缓存插件。
func ExampleWithRedis() {
	app := application.New()

	// 构造 Redis 缓存受管插件
	info, factory := cache.New(
		cache.WithRedis(
			"127.0.0.1:6379",
			cache.WithRedisPassword("secret"),
			cache.WithRedisDB(1),
			cache.WithRedisPoolSize(50),
			cache.WithRedisDialTimeout(2*time.Second),
		),
	)

	if err := app.RegisterTool(info, factory); err != nil {
		panic(err)
	}
}
