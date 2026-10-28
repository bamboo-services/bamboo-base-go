package redis

import (
	"context"
	"os"
	"testing"
	"time"

	driver "github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver"
	goredis "github.com/redis/go-redis/v9"
)

// 断言各实现满足 driver 接口契约
var (
	_ driver.KeyCache[string, string]                    = (*KeyCache[string, string])(nil)
	_ driver.HashCache[string, string, string, struct{}] = (*HashCache[string, string, string, struct{}])(nil)
	_ driver.SetCache[string, string]                    = (*SetCache[string, string])(nil)
	_ driver.ListCache[string, string]                   = (*ListCache[string, string])(nil)
)

type sampleUser struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestInterfaceAssertionsAndConstructors(t *testing.T) {
	// 使用假 client 仅用于结构体构造测试，不发起真实网络连接
	rdb := goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:0"})
	defer func() {
		_ = rdb.Close()
	}()

	keyCache := NewKeyCache[string, sampleUser](rdb, nil, nil, time.Minute)
	if keyCache == nil {
		t.Fatal("expected non-nil KeyCache")
	}

	hashCache := NewHashCache[string, string, sampleUser, struct{}](rdb, nil, nil, time.Minute)
	if hashCache == nil {
		t.Fatal("expected non-nil HashCache")
	}

	setCache := NewSetCache[string, string](rdb, nil, nil, time.Minute)
	if setCache == nil {
		t.Fatal("expected non-nil SetCache")
	}

	listCache := NewListCache[string, sampleUser](rdb, nil, nil, time.Minute)
	if listCache == nil {
		t.Fatal("expected non-nil ListCache")
	}
}

func TestHashCacheGetAllTypeCheck(t *testing.T) {
	rdb := goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:0"})
	defer func() {
		_ = rdb.Close()
	}()

	// F 为 int（非 string kind）时，GetAll 应直接报错
	hc := NewHashCache[string, int, string, struct{}](rdb, nil, nil, time.Minute)
	ctx := context.Background()
	_, err := hc.GetAll(ctx, "test-key")
	if err == nil {
		t.Fatal("expected error when F is not string kind")
	}
}

func TestRedisIntegrationFixture(t *testing.T) {
	// 严格遵循：需要专用环境变量 BAMBOO_TEST_REDIS_ADDR 与显式开关 BAMBOO_TEST_REDIS_ENABLED
	// 不能用普通 NOSQL_* 连接业务或默认 127.0.0.1:6379 误连现有服务
	enabled := os.Getenv("BAMBOO_TEST_REDIS_ENABLED")
	addr := os.Getenv("BAMBOO_TEST_REDIS_ADDR")

	if enabled != "true" || addr == "" {
		t.Skip("skipping real redis integration test: set BAMBOO_TEST_REDIS_ENABLED=true and BAMBOO_TEST_REDIS_ADDR to run")
	}

	// 预留真实夹具逻辑，未配置环境变量时跳过
	rdb := goredis.NewClient(&goredis.Options{Addr: addr})
	defer func() {
		_ = rdb.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("real redis ping failed: %v", err)
	}
}
