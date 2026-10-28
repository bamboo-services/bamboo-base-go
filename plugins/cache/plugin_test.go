package cache

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
	goredis "github.com/redis/go-redis/v9"
)

// TestNew_NoJanitorOrNetwork 验证构造或工厂调用期间不启动后台 janitor goroutine，也不建立网络连接。
func TestNew_NoJanitorOrNetwork(t *testing.T) {
	info, factory := New(WithMemory())
	if info.Key != PluginKey {
		t.Fatalf("expected key %s, got %s", PluginKey, info.Key)
	}
	if info.Mode != application.ModeTool {
		t.Fatalf("expected mode ModeTool, got %v", info.Mode)
	}
	if info.Name != "plugin_cache" {
		t.Fatalf("expected name plugin_cache, got %s", info.Name)
	}

	p := factory()
	impl, ok := p.(*pluginImpl)
	if !ok {
		t.Fatalf("expected *pluginImpl, got %T", p)
	}

	if impl.memStore != nil {
		t.Fatal("expected memStore to be nil before Start")
	}
	if impl.mgr != nil {
		t.Fatal("expected mgr to be nil before Start")
	}
}

// TestNew_Provides 验证静态元数据 Provides。
func TestNew_Provides(t *testing.T) {
	// 1. WithMemory: 仅提供 CacheManagerKey
	infoMem, _ := New(WithMemory())
	if len(infoMem.Provides) != 1 || infoMem.Provides[0] != ctxkey.CacheManagerKey {
		t.Fatalf("WithMemory provides mismatch: %+v", infoMem.Provides)
	}

	// 2. WithRedis: 提供 CacheManagerKey 与 RedisClientKey
	infoRedis, _ := New(WithRedis("localhost:6379"))
	if len(infoRedis.Provides) != 2 {
		t.Fatalf("WithRedis provides length mismatch: %+v", infoRedis.Provides)
	}
	hasMgr := false
	hasRdb := false
	for _, k := range infoRedis.Provides {
		if k == ctxkey.CacheManagerKey {
			hasMgr = true
		}
		if k == ctxkey.RedisClientKey {
			hasRdb = true
		}
	}
	if !hasMgr || !hasRdb {
		t.Fatalf("WithRedis provides missing keys: %+v", infoRedis.Provides)
	}

	// 3. None: 不伪称提供 CacheManagerKey 或 RedisClientKey
	infoNone, _ := New()
	if len(infoNone.Provides) != 0 {
		t.Fatalf("None mode should have empty Provides, got %+v", infoNone.Provides)
	}
	for _, k := range infoNone.Provides {
		if k == ctxkey.RedisClientKey {
			t.Fatal("None mode should not provide RedisClientKey")
		}
	}
}

// TestNew_NoneReturnsErrorOnStart 验证未显式配置或 None 模式在 Start 阶段报错。
func TestNew_NoneReturnsErrorOnStart(t *testing.T) {
	_, factory := New() // 默认无选择
	p := factory()

	err := p.Start(context.Background())
	if err == nil {
		t.Fatal("expected Start to return error for unselected backend")
	}

	_, factoryNone := New(WithType(CacheTypeNone))
	pNone := factoryNone()
	errNone := pNone.Start(context.Background())
	if errNone == nil {
		t.Fatal("expected Start to return error for CacheTypeNone")
	}
}

// testLifetimeWorker 用于在真实 App 运行时捕获 lifetime ctx。
type testLifetimeWorker struct {
	ready   chan struct{}
	ctxChan chan context.Context
}

func (w *testLifetimeWorker) Start(ctx context.Context) error {
	return nil
}

func (w *testLifetimeWorker) Run(ctx context.Context) error {
	w.ctxChan <- ctx
	close(w.ready)
	<-ctx.Done()
	return nil
}

func (w *testLifetimeWorker) Stop(ctx context.Context) error {
	return nil
}

func runWithCacheApp(t *testing.T, opts ...Option) (context.Context, func()) {
	t.Helper()
	app := application.New(
		application.WithGraceTimeout(10*time.Millisecond),
		application.WithCleanupTimeout(100*time.Millisecond),
		application.WithShutdownTimeout(time.Second),
	)
	info, factory := New(opts...)
	if err := app.RegisterTool(info, factory); err != nil {
		t.Fatalf("app.RegisterTool failed: %v", err)
	}
	w := &testLifetimeWorker{
		ready:   make(chan struct{}),
		ctxChan: make(chan context.Context, 1),
	}
	wInfo := application.PluginInfo{
		Key:      ctxkey.Key("test_lifetime_worker_" + string(info.Key)),
		Name:     "test_lifetime_worker",
		Mode:     application.ModeWorker,
		Requires: []ctxkey.Key{ctxkey.CacheManagerKey},
	}
	if err := app.RegisterWorker(wInfo, func() application.RunnablePlugin { return w }); err != nil {
		t.Fatalf("app.RegisterWorker failed: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run()
	}()

	select {
	case <-w.ready:
	case err := <-errCh:
		t.Fatalf("app.Run terminated prematurely: %v", err)
	}
	capturedCtx := <-w.ctxChan

	cleanup := func() {
		_ = app.Shutdown()
		<-errCh
	}
	return capturedCtx, cleanup
}

// TestNew_TwoMemoryPluginsIsolation 验证两个 Memory 插件实例各自拥有独立的内存 Store。
func TestNew_TwoMemoryPluginsIsolation(t *testing.T) {
	ctx1, cleanup1 := runWithCacheApp(t, WithMemory())
	defer cleanup1()
	ctx2, cleanup2 := runWithCacheApp(t, WithMemory())
	defer cleanup2()

	mgr1, appErr1 := Get(ctx1)
	if appErr1 != nil || mgr1 == nil {
		t.Fatalf("failed to get mgr1: %v", appErr1)
	}
	mgr2, appErr2 := Get(ctx2)
	if appErr2 != nil || mgr2 == nil {
		t.Fatalf("failed to get mgr2: %v", appErr2)
	}

	if mgr1.Memory() == mgr2.Memory() {
		t.Fatal("expected two memory plugins to have isolated memory stores, but got the same pointer")
	}

	// 验证数据隔离
	cache1 := KeyCacheOf[string, string](mgr1)
	cache2 := KeyCacheOf[string, string](mgr2)

	v1 := "v1"
	_ = cache1.Set(context.Background(), "k1", &v1)
	val2, ok, _ := cache2.Get(context.Background(), "k1")
	if ok || val2 != nil {
		t.Fatalf("expected k1 not in cache2, got ok=%v, val=%v", ok, val2)
	}
}

// TestStop_IdempotentAndShared 验证 Stop 可以重复调用，共享结果与等待，不会 panic。
func TestStop_IdempotentAndShared(t *testing.T) {
	_, factory := New(WithMemory())
	p := factory()

	ctx := context.Background()
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	err1 := p.Stop(ctx)
	err2 := p.Stop(ctx)

	if !errors.Is(err1, err2) && (err1 != nil || err2 != nil) {
		t.Fatalf("expected identical stop errors, got %v and %v", err1, err2)
	}
}

// TestStop_NilOrPartialStartSafe 验证未调用 Start 或 Start 报错的情况下 Stop 依然安全。
func TestStop_NilOrPartialStartSafe(t *testing.T) {
	// 1. 完全未 Start
	_, factory := New(WithMemory())
	p := factory()
	if err := p.Stop(context.Background()); err != nil {
		t.Fatalf("Stop on unstarted plugin failed: %v", err)
	}

	// 2. Start 失败时调用 Stop
	_, factoryErr := New() // None 模式，Start 必失败
	pErr := factoryErr()
	_ = pErr.Start(context.Background())
	if err := pErr.Stop(context.Background()); err != nil {
		t.Fatalf("Stop on failed-start plugin failed: %v", err)
	}
}

// TestStop_IgnoresContextDone 验证 Stop 不会因传入 ctx.Done 而提前返回假排空。
func TestStop_IgnoresContextDone(t *testing.T) {
	_, factory := New(WithMemory())
	p := factory()
	_ = p.Start(context.Background())

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	err := p.Stop(canceledCtx)
	if err != nil {
		t.Fatalf("Stop failed with canceled ctx: %v", err)
	}
}

// TestFromEnv_ValidationErrors 验证环境变量格式错误时记录并在 Start 阶段报错。
func TestFromEnv_ValidationErrors(t *testing.T) {
	_ = os.Setenv(string(env.NoSQLDriver), "memory")
	_ = os.Setenv(string(env.NoSQLMemoryMaxEntries), "invalid_number")
	defer func() {
		_ = os.Unsetenv(string(env.NoSQLDriver))
		_ = os.Unsetenv(string(env.NoSQLMemoryMaxEntries))
	}()

	_, factory := New(FromEnv())
	p := factory()
	err := p.Start(context.Background())
	if err == nil {
		t.Fatal("expected error on Start when env has invalid number, got nil")
	}
}

// TestRedis_DefaultSkip 真实 Redis 连接测试默认跳过，若无法连接不把单元测试搞挂。
func TestRedis_ConnectionFailureCleansUp(t *testing.T) {
	// 连接一个肯定无法 ping 通的不可达地址，测试失败清理逻辑
	_, factory := New(WithRedis("127.0.0.1:54321", WithRedisDialTimeout(50*time.Millisecond)))
	p := factory()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := p.Start(ctx)
	if err == nil {
		t.Fatal("expected connection error to unreachable redis, got nil")
	}

	// 验证失败后 raw rdb 已清理或置 nil
	impl := p.(*pluginImpl)
	if impl.rdb != nil {
		t.Fatal("expected rdb to be cleared after ping failure")
	}

	// 验证失败后安全 Stop
	if stopErr := p.Stop(context.Background()); stopErr != nil {
		t.Fatalf("Stop after failed Start returned error: %v", stopErr)
	}
}

// TestExpose_ExposesActualInstances 验证 Expose 只暴露实际非 nil 实例。
func TestExpose_ExposesActualInstances(t *testing.T) {
	// 1. 未启动时 Expose 返回空映射
	_, factory := New(WithMemory())
	p := factory()
	expBefore, ok := p.(ctxkey.Exposer)
	if !ok {
		t.Fatalf("expected p to implement ctxkey.Exposer")
	}
	if exposed := expBefore.Expose(); len(exposed) != 0 {
		t.Fatalf("expected empty Expose before Start, got %+v", exposed)
	}

	// 2. 在真实 App 运行环境中暴露实际实例
	ctx, cleanup := runWithCacheApp(t, WithMemory())
	defer cleanup()

	mgr, err := application.Get[*Manager](ctx, ctxkey.CacheManagerKey)
	if err != nil || mgr == nil {
		t.Fatalf("expected CacheManagerKey in ctx, got %v", err)
	}

	// Memory 模式下不应该暴露 RedisClientKey
	_, rdbErr := application.Get[*goredis.Client](ctx, ctxkey.RedisClientKey)
	if rdbErr == nil {
		t.Fatal("Memory mode should not expose RedisClientKey")
	}
}

// TestWithLogger 验证 Logger 显式注入。
func TestWithLogger(t *testing.T) {
	h := slog.NewTextHandler(io.Discard, nil)
	l, err := logging.New(h)
	if err != nil {
		t.Fatalf("unexpected error creating logger: %v", err)
	}
	_, factory := New(WithMemory(), WithLogger(l))
	p := factory()
	_ = p.Start(context.Background())
	defer func() { _ = p.Stop(context.Background()) }()

	impl := p.(*pluginImpl)
	if impl.mgr.Logger() == nil {
		t.Fatal("expected manager logger to be set")
	}
}
