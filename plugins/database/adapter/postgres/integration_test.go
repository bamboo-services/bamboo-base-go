package postgres_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	postgres "github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/postgres/v2"
	rootdatabase "github.com/bamboo-services/bamboo-base-go/plugins/database/v2"
	"gorm.io/gorm"
)

type witnessWorker struct {
	ready   chan struct{}
	workCtx chan context.Context
}

func (w *witnessWorker) Start(ctx context.Context) error {
	return nil
}

func (w *witnessWorker) Run(ctx context.Context) error {
	select {
	case w.workCtx <- ctx:
	default:
	}
	close(w.ready)
	<-ctx.Done()
	return nil
}

func (w *witnessWorker) Stop(ctx context.Context) error {
	return nil
}

type testEntity struct {
	ID    int64  `gorm:"primaryKey;column:id"`
	Name  string `gorm:"column:name"`
	Value string `gorm:"column:value"`
}

func TestIntegration_PostgresReal(t *testing.T) {
	dsn := os.Getenv("BAMBOO_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("BAMBOO_TEST_POSTGRES_DSN is not set, skipping postgres integration test")
	}

	adapter := postgres.New(dsn)
	toolInfo, toolFactory := rootdatabase.New(rootdatabase.WithAdapter(adapter))

	app := application.New(
		application.WithShutdownTimeout(5*time.Second),
		application.WithGraceTimeout(2*time.Second),
		application.WithCleanupTimeout(2*time.Second),
	)

	if err := app.RegisterTool(toolInfo, toolFactory); err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	witness := &witnessWorker{
		ready:   make(chan struct{}),
		workCtx: make(chan context.Context, 1),
	}

	witnessInfo := application.PluginInfo{
		Key:      ctxkey.Key("witness_worker"),
		Mode:     application.ModeWorker,
		Requires: []ctxkey.Key{ctxkey.DatabaseKey},
	}
	if err := app.RegisterWorker(witnessInfo, func() application.RunnablePlugin {
		return witness
	}); err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run()
	}()

	select {
	case <-witness.ready:
	case err := <-runErrCh:
		t.Fatalf("app.Run exited prematurely: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for witness worker ready")
	}

	var ctx context.Context
	select {
	case ctx = <-witness.workCtx:
	default:
		t.Fatal("failed to capture witness work context")
	}

	gormDB, appErr := rootdatabase.Get(ctx)
	if appErr != nil {
		t.Fatalf("rootdatabase.Get failed: %v", appErr)
	}
	if gormDB == nil {
		t.Fatal("expected non-nil gorm.DB")
	}

	rawSQLDB, err := gormDB.DB()
	if err != nil {
		t.Fatalf("failed to retrieve underlying sql.DB: %v", err)
	}

	// 1. Ping
	if err := rawSQLDB.PingContext(ctx); err != nil {
		t.Fatalf("sql.DB PingContext failed: %v", err)
	}

	// 2. 唯一临时表
	tableName := fmt.Sprintf("tmp_integration_test_%d", time.Now().UnixNano())
	createTableSQL := fmt.Sprintf(`CREATE TABLE %s (
		id BIGINT PRIMARY KEY,
		name VARCHAR(64) NOT NULL,
		value VARCHAR(128) NOT NULL
	);`, tableName)
	dropTableSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName)

	defer func() {
		// 清理真实表，独立 context 避免受 App 关闭影响
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = rawSQLDB.ExecContext(cleanupCtx, dropTableSQL)
	}()

	if err := gormDB.Exec(createTableSQL).Error; err != nil {
		t.Fatalf("create test table failed: %v", err)
	}

	// 3. CRUD
	row1 := testEntity{ID: 1, Name: "alpha", Value: "val1"}
	if err := gormDB.Table(tableName).Create(&row1).Error; err != nil {
		t.Fatalf("insert record failed: %v", err)
	}

	var fetched testEntity
	if err := gormDB.Table(tableName).Where("id = ?", 1).First(&fetched).Error; err != nil {
		t.Fatalf("query record failed: %v", err)
	}
	if fetched.Name != "alpha" || fetched.Value != "val1" {
		t.Fatalf("fetched record mismatch: %+v", fetched)
	}

	if err := gormDB.Table(tableName).Where("id = ?", 1).Update("value", "val1_updated").Error; err != nil {
		t.Fatalf("update record failed: %v", err)
	}
	var updated testEntity
	if err := gormDB.Table(tableName).Where("id = ?", 1).First(&updated).Error; err != nil {
		t.Fatalf("query updated record failed: %v", err)
	}
	if updated.Value != "val1_updated" {
		t.Fatalf("expected updated value 'val1_updated', got %q", updated.Value)
	}

	// 4. 事务与 Rollback
	txErr := gormDB.Transaction(func(tx *gorm.DB) error {
		row2 := testEntity{ID: 2, Name: "beta", Value: "val2"}
		if err := tx.Table(tableName).Create(&row2).Error; err != nil {
			return err
		}
		return fmt.Errorf("intentional rollback error")
	})
	if txErr == nil {
		t.Fatal("expected transaction error, got nil")
	}

	var rollbackCheck testEntity
	err = gormDB.Table(tableName).Where("id = ?", 2).First(&rollbackCheck).Error
	if err == nil {
		t.Fatal("expected record with id=2 to be rolled back, but found it")
	}

	// 5. Delete
	if err := gormDB.Table(tableName).Where("id = ?", 1).Delete(&testEntity{}).Error; err != nil {
		t.Fatalf("delete record failed: %v", err)
	}
	var deleteCheck testEntity
	if err := gormDB.Table(tableName).Where("id = ?", 1).First(&deleteCheck).Error; err == nil {
		t.Fatal("expected record with id=1 to be deleted, but found it")
	}

	// 6. 显式 DROP 临时表
	if err := gormDB.Exec(dropTableSQL).Error; err != nil {
		t.Fatalf("drop test table failed: %v", err)
	}

	// 7. App 正常 Shutdown 并断言底层 sql.DB handle 真实被关闭
	shutdownErr := app.Shutdown()
	if shutdownErr != nil {
		t.Fatalf("app.Shutdown failed: %v", shutdownErr)
	}

	select {
	case runErr := <-runErrCh:
		if runErr != nil {
			t.Fatalf("app.Run returned error on shutdown: %v", runErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for app.Run to terminate")
	}

	// 8. 验证 sql.DB handle Ping 失败且底层连接已断开，而不仅是 ctx 错误
	pingErr := rawSQLDB.PingContext(context.Background())
	if pingErr == nil {
		t.Fatal("expected sql.DB PingContext to fail after App Shutdown, but succeeded")
	}
	if pingErr != sql.ErrConnDone && pingErr.Error() != "sql: database is closed" {
		t.Logf("PingContext failed as expected with: %v", pingErr)
	}

	stats := rawSQLDB.Stats()
	if stats.OpenConnections != 0 {
		t.Fatalf("expected 0 open connections after shutdown, got %d", stats.OpenConnections)
	}
}
