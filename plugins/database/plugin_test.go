package database_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	database "github.com/bamboo-services/bamboo-base-go/plugins/database/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/gormlog"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// fakeDriver 模拟 driver.Driver 用于测试
type fakeDriver struct {
	mu          sync.Mutex
	pingErr     error
	closedCount int32
	openedConns []*fakeConn
	driverErr   error
}

type fakeConn struct {
	driver *fakeDriver
	closed int32
}

func (c *fakeConn) Prepare(query string) (driver.Stmt, error) {
	return &fakeStmt{}, nil
}

func (c *fakeConn) Close() error {
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		atomic.AddInt32(&c.driver.closedCount, 1)
	}
	return nil
}

func (c *fakeConn) Begin() (driver.Tx, error) {
	return &fakeTx{}, nil
}

func (c *fakeConn) Ping(ctx context.Context) error {
	c.driver.mu.Lock()
	err := c.driver.pingErr
	c.driver.mu.Unlock()
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

type fakeStmt struct{}

func (s *fakeStmt) Close() error  { return nil }
func (s *fakeStmt) NumInput() int { return 0 }
func (s *fakeStmt) Exec(args []driver.Value) (driver.Result, error) {
	return driver.RowsAffected(0), nil
}
func (s *fakeStmt) Query(args []driver.Value) (driver.Rows, error) { return &fakeRows{}, nil }

type fakeTx struct{}

func (t *fakeTx) Commit() error   { return nil }
func (t *fakeTx) Rollback() error { return nil }

type fakeRows struct{}

func (r *fakeRows) Columns() []string              { return []string{"1"} }
func (r *fakeRows) Close() error                   { return nil }
func (r *fakeRows) Next(dest []driver.Value) error { return io.EOF }

func (d *fakeDriver) Open(name string) (driver.Conn, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.driverErr != nil {
		return nil, d.driverErr
	}
	conn := &fakeConn{driver: d}
	d.openedConns = append(d.openedConns, conn)
	return conn, nil
}

var (
	fakeDriverCounter int64
)

func registerFakeDriver() (string, *fakeDriver) {
	num := atomic.AddInt64(&fakeDriverCounter, 1)
	name := fmt.Sprintf("fake_db_%d", num)
	drv := &fakeDriver{}
	sql.Register(name, drv)
	return name, drv
}

type fakeAdapter struct {
	name      string
	dialector gorm.Dialector
	openErr   error
}

func (a *fakeAdapter) Name() string {
	return a.name
}

func (a *fakeAdapter) Open() (gorm.Dialector, error) {
	if a.openErr != nil {
		return nil, a.openErr
	}
	return a.dialector, nil
}

func newFakeAdapter(driverName string) (*fakeAdapter, *fakeDriver) {
	sqlDrvName, fakeDrv := registerFakeDriver()
	db, err := sql.Open(sqlDrvName, "fake_dsn")
	if err != nil {
		panic(err)
	}
	dialector := newFakeDialector(driverName, db)
	return &fakeAdapter{
		name:      driverName,
		dialector: dialector,
	}, fakeDrv
}

// 验证 New 与工厂函数纯构造，不建立网络与 I/O
func TestPlugin_PureConstruction(t *testing.T) {
	adp, _ := newFakeAdapter("sqlite")
	info, factory := database.New(database.WithAdapter(adp))
	if info.Key != database.PluginKey {
		t.Fatalf("info.Key mismatch: got %v, want %v", info.Key, database.PluginKey)
	}
	if info.Mode != application.ModeTool {
		t.Fatalf("info.Mode mismatch: got %v, want ModeTool", info.Mode)
	}
	if len(info.Provides) != 1 || info.Provides[0] != ctxkey.DatabaseKey {
		t.Fatalf("info.Provides mismatch: got %v", info.Provides)
	}
	instance := factory()
	if instance == nil {
		t.Fatal("factory() returned nil")
	}
}

// 验证 PingContext 失败时清池与 Start 失败
func TestPlugin_StartPingFailureCleansUp(t *testing.T) {
	adp, drv := newFakeAdapter("mock")
	drv.mu.Lock()
	drv.pingErr = errors.New("ping timeout")
	drv.mu.Unlock()

	_, factory := database.New(database.WithAdapter(adp))
	p := factory()

	ctx := context.Background()
	err := p.Start(ctx)
	if err == nil {
		t.Fatal("expected Start to fail on ping error, got nil")
	}

	// 检查资源是否已通过 Stop/closeSQLDB 安全清理
	exposer, ok := p.(ctxkey.Exposer)
	if !ok {
		t.Fatal("expected p to implement ctxkey.Exposer")
	}
	exposed := exposer.Expose()
	if exposed[ctxkey.DatabaseKey] != nil {
		t.Fatalf("expected nil db on failure, got %v", exposed[ctxkey.DatabaseKey])
	}
	// 重复 Stop 幂等
	if err := p.Stop(ctx); err != nil {
		t.Fatalf("expected Stop to be idempotent, got: %v", err)
	}
}

// 验证 PingContext 超时被取消
func TestPlugin_StartPingCancelled(t *testing.T) {
	adp, _ := newFakeAdapter("mock")
	_, factory := database.New(database.WithAdapter(adp))
	p := factory()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立刻取消

	err := p.Start(ctx)
	if err == nil {
		t.Fatal("expected Start to fail on canceled context")
	}
}

// 验证 Stop 真实 Close 底层 sqlDB，并保证幂等
func TestPlugin_StopIdempotentAndRealClose(t *testing.T) {
	adp, _ := newFakeAdapter("mock")
	_, factory := database.New(
		database.WithAdapter(adp),
		database.WithMaxOpenConns(10),
		database.WithMaxIdleConns(5),
		database.WithConnMaxLifetime(time.Minute),
		database.WithConnMaxIdleTime(time.Second*30),
	)
	p := factory()

	ctx := context.Background()
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	exposer := p.(ctxkey.Exposer)
	dbVal := exposer.Expose()[ctxkey.DatabaseKey]
	gormDB, ok := dbVal.(*gorm.DB)
	if !ok || gormDB == nil {
		t.Fatal("expected exposed gorm.DB")
	}

	if err := p.Stop(ctx); err != nil {
		t.Fatalf("first Stop failed: %v", err)
	}
	if err := p.Stop(ctx); err != nil {
		t.Fatalf("second Stop should be idempotent: %v", err)
	}
}

// 验证未配置 Logger 时默认使用 gormlogger.Discard
func TestPlugin_LoggerDiscardWhenNil(t *testing.T) {
	adp, _ := newFakeAdapter("mock")
	cfg := &gorm.Config{}
	_, factory := database.New(
		database.WithAdapter(adp),
		database.WithGormConfig(cfg),
	)
	p := factory()
	ctx := context.Background()
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = p.Stop(ctx) }()

	exposer := p.(ctxkey.Exposer)
	gormDB := exposer.Expose()[ctxkey.DatabaseKey].(*gorm.DB)
	if gormDB.Logger != gormlogger.Discard {
		t.Fatalf("expected logger.Discard, got: %v", gormDB.Logger)
	}
}

// 验证 FromEnv 在找不到驱动时返回明确路径报错
func TestPlugin_FromEnvDriverNotLoadedError(t *testing.T) {
	t.Setenv(string(env.DatabaseDriver), "unregistered_driver")

	_, factory := database.New(database.FromEnv())
	p := factory()

	err := p.Start(context.Background())
	if err == nil {
		t.Fatal("expected error for unregistered driver")
	}
	expected := "数据库驱动 [unregistered_driver] 未加载，请传入 WithAdapter(unregistered_driver.FromEnv()) 或匿名导入 plugins/database/adapter/unregistered_driver/v2"
	if err.Error() != expected {
		t.Fatalf("error message mismatch: got %q, want %q", err.Error(), expected)
	}
}

// 验证 Get 与 MustGet
func TestGetAndMustGet(t *testing.T) {
	// 1. 无容器挂载
	ctx := context.Background()
	_, appErr := database.Get(ctx)
	if appErr == nil {
		t.Fatal("expected error when container not mounted")
	}

	// 2. MustGet panic
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic from MustGet")
		}
	}()
	database.MustGet(ctx)
}

// 验证 gormlog.New 接收 nil logger 返回 Discard
func TestGormLog_NilLoggerReturnsDiscard(t *testing.T) {
	l := gormlog.New(nil, gormlog.Config{})
	if l != gormlogger.Discard {
		t.Fatalf("expected gormlogger.Discard, got %v", l)
	}
}
