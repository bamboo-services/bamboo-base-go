package mysql_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/mysql/v2"
	database "github.com/bamboo-services/bamboo-base-go/plugins/database/v2"
	"gorm.io/gorm"
)

// testRecord 仅用于集成测试真实 CRUD 与事务验证的实体结构体。
type testRecord struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	Name      string    `gorm:"size:64;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// witnessWorker 满足 application.RunnablePlugin 契约（Start/Run/Stop 俱全）。
// 作为 ModeWorker 注册并静态声明 Requires: []ctxkey.Key{ctxkey.DatabaseKey}。
// 启动后通过可信 lifetimeCtx 接收数据库测试逻辑，执行完成后通知调用方。
type witnessWorker struct {
	workFn   func(ctx context.Context) error
	doneCh   chan struct{}
	workErr  error
	startErr error
	stopErr  error
	mu       sync.Mutex
	doneOnce sync.Once
}

func newWitnessWorker(workFn func(ctx context.Context) error) *witnessWorker {
	return &witnessWorker{
		workFn: workFn,
		doneCh: make(chan struct{}),
	}
}

func (w *witnessWorker) Start(_ context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.startErr
}

func (w *witnessWorker) Run(ctx context.Context) error {
	defer w.doneOnce.Do(func() { close(w.doneCh) })
	if w.workFn != nil {
		w.workErr = w.workFn(ctx)
		w.doneOnce.Do(func() { close(w.doneCh) })
		if w.workErr != nil {
			return w.workErr
		}
	}
	<-ctx.Done()
	return ctx.Err()
}

func (w *witnessWorker) Stop(_ context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.stopErr
}

// TestIntegration_MySQL 真实 MySQL 8.4.4 集成测试。
// 必须通过 BAMBOO_TEST_MYSQL_DSN 获取环境变量，若未提供则按照契约 Skip。
// 严禁读取生产 DATABASE_* 变量或 .env 文件。
func TestIntegration_MySQL(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("BAMBOO_TEST_MYSQL_DSN"))
	if dsn == "" {
		t.Skip("skipping MySQL integration test: BAMBOO_TEST_MYSQL_DSN not set (run via .github/scripts/env/integration-env.sh env)")
	}

	// 1. 构造真实 App，配置合理的短关机超时避免默认 20s/60s 拖长用例
	app := application.New(
		application.WithGraceTimeout(100*time.Millisecond),
		application.WithCleanupTimeout(200*time.Millisecond),
		application.WithShutdownTimeout(500*time.Millisecond),
	)
	t.Cleanup(func() {
		_ = app.Shutdown()
	})

	// 2. 构造真实适配器与 Database 根插件（ModeTool）
	mysqlAdapter := mysql.New(dsn)
	dbInfo, dbFactory := database.New(
		database.WithAdapter(mysqlAdapter),
		database.WithMaxOpenConns(5),
		database.WithMaxIdleConns(2),
		database.WithConnMaxLifetime(time.Minute*5),
	)

	if err := app.RegisterTool(dbInfo, dbFactory); err != nil {
		t.Fatalf("register database tool failed: %v", err)
	}

	// 3. 构造见证 Worker（ModeWorker），严格声明 Requires DatabaseKey
	workerKey := ctxkey.Key("witness_worker_mysql")
	workerInfo := application.PluginInfo{
		Key:      workerKey,
		Name:     "witness_worker_mysql",
		Mode:     application.ModeWorker,
		Requires: []ctxkey.Key{ctxkey.DatabaseKey},
	}

	var (
		rawSQLDB     *sql.DB
		rawSQLDBLock sync.Mutex
	)

	// 生成专属且唯一的表名（限制在 bamboo_test 库内）
	tableName := fmt.Sprintf("test_integration_%d", time.Now().UnixNano())

	worker := newWitnessWorker(func(ctx context.Context) error {
		// A. 通过真实 API 从可信 ctx 获取已绑定的 *gorm.DB
		db, appErr := database.Get(ctx)
		if appErr != nil {
			return fmt.Errorf("database.Get failed: %w", appErr)
		}
		if db == nil {
			return errors.New("database.Get returned nil *gorm.DB")
		}

		// 提取并暂存底层真实 *sql.DB handle，供 App 关机后验证真实物理关闭
		sdb, err := db.DB()
		if err != nil {
			return fmt.Errorf("db.DB() failed: %w", err)
		}
		rawSQLDBLock.Lock()
		rawSQLDB = sdb
		rawSQLDBLock.Unlock()

		// B. 真实 Ping 验证
		if pingErr := sdb.PingContext(ctx); pingErr != nil {
			return fmt.Errorf("sql.DB PingContext failed: %w", pingErr)
		}

		// C. 在专属 bamboo_test 数据库中创建唯一测试表
		if err := db.Table(tableName).AutoMigrate(&testRecord{}); err != nil {
			return fmt.Errorf("auto migrate table %s failed: %w", tableName, err)
		}

		// D. 真实 CRUD 操作验证
		// Create
		record := testRecord{Name: "bamboo_v2_alpha"}
		if err := db.Table(tableName).Create(&record).Error; err != nil {
			return fmt.Errorf("insert failed: %w", err)
		}
		if record.ID == 0 {
			return errors.New("expected non-zero auto-increment ID after insert")
		}

		// Read
		var fetched testRecord
		if err := db.Table(tableName).First(&fetched, record.ID).Error; err != nil {
			return fmt.Errorf("query record failed: %w", err)
		}
		if fetched.Name != "bamboo_v2_alpha" {
			return fmt.Errorf("query record name mismatch: got %q, want %q", fetched.Name, "bamboo_v2_alpha")
		}

		// Update
		if err := db.Table(tableName).Where("id = ?", record.ID).Update("name", "bamboo_v2_updated").Error; err != nil {
			return fmt.Errorf("update failed: %w", err)
		}
		var updated testRecord
		if err := db.Table(tableName).First(&updated, record.ID).Error; err != nil {
			return fmt.Errorf("query updated record failed: %w", err)
		}
		if updated.Name != "bamboo_v2_updated" {
			return fmt.Errorf("updated name mismatch: got %q, want %q", updated.Name, "bamboo_v2_updated")
		}

		// E. 真实事务与回滚验证
		txErr := db.Transaction(func(tx *gorm.DB) error {
			txRecord := testRecord{Name: "should_be_rolled_back"}
			if err := tx.Table(tableName).Create(&txRecord).Error; err != nil {
				return err
			}
			// 主动返回错误触发事务回滚
			return errors.New("trigger rollback intentionally")
		})
		if txErr == nil || txErr.Error() != "trigger rollback intentionally" {
			return fmt.Errorf("expected intentional rollback error, got: %v", txErr)
		}

		var countAfterRollback int64
		if err := db.Table(tableName).Where("name = ?", "should_be_rolled_back").Count(&countAfterRollback).Error; err != nil {
			return fmt.Errorf("count rolled back record failed: %w", err)
		}
		if countAfterRollback != 0 {
			return fmt.Errorf("expected 0 rolled back records, got %d", countAfterRollback)
		}

		// Delete (CRUD 闭环)
		if err := db.Table(tableName).Where("id = ?", record.ID).Delete(&testRecord{}).Error; err != nil {
			return fmt.Errorf("delete record failed: %w", err)
		}
		var countAfterDelete int64
		if err := db.Table(tableName).Count(&countAfterDelete).Error; err != nil {
			return fmt.Errorf("count after delete failed: %w", err)
		}
		if countAfterDelete != 0 {
			return fmt.Errorf("expected 0 records after delete, got %d", countAfterDelete)
		}

		// F. Cleanup：严格仅 drop 本次创建的唯一临时表，不碰其它 schema
		if err := db.Migrator().DropTable(tableName); err != nil {
			return fmt.Errorf("drop table %s failed: %w", tableName, err)
		}

		return nil
	})

	if err := app.RegisterWorker(workerInfo, func() application.RunnablePlugin {
		return worker
	}); err != nil {
		t.Fatalf("register worker failed: %v", err)
	}

	// 4. 启动应用生命周期
	appRunErrCh := make(chan error, 1)
	go func() {
		appRunErrCh <- app.Run()
	}()

	// 5. 等待 Worker 业务逻辑执行完毕
	select {
	case <-worker.doneCh:
		if worker.workErr != nil {
			t.Fatalf("worker execution failed: %v", worker.workErr)
		}
	case <-time.After(time.Second * 15):
		t.Fatal("worker execution timeout (15s)")
	}

	// 6. 优雅关机 App 并确认 goroutine 回收
	shutdownErr := app.Shutdown()
	if shutdownErr != nil && !errors.Is(shutdownErr, context.Canceled) {
		t.Fatalf("app.Shutdown failed: %v", shutdownErr)
	}

	select {
	case runErr := <-appRunErrCh:
		if runErr != nil && !errors.Is(runErr, context.Canceled) {
			t.Fatalf("app.Run terminated with error: %v", runErr)
		}
	case <-time.After(time.Second * 5):
		t.Fatal("app.Run failed to exit after Shutdown")
	}

	// 7. 真实物理连接关闭验证：保留真实 sql.DB handle 验证 sqlDB.Ping() 失败
	// 不拿 Get 旧 ctx 错误当做物理关闭证明
	rawSQLDBLock.Lock()
	savedSQLDB := rawSQLDB
	rawSQLDBLock.Unlock()

	if savedSQLDB == nil {
		t.Fatal("savedSQLDB is nil, worker never extracted raw *sql.DB handle")
	}

	// 物理验证：已关闭的 sql.DB 调用 Ping 会返回 "sql: database is closed"
	pingErr := savedSQLDB.Ping()
	if pingErr == nil {
		t.Fatal("expected savedSQLDB.Ping() to fail after App.Shutdown(), but it succeeded")
	}
	if !strings.Contains(pingErr.Error(), "database is closed") {
		t.Fatalf("expected 'database is closed' error from physical sql.DB handle, got: %q", pingErr.Error())
	}
}

// TestIntegration_MySQL_StartConnectionFailure 验证真实连接失败时 Start 拒绝并安全清理回滚。
// 使用明显非法/未监听的 loopback 端口（127.0.0.1:65530），验证真实网络拒绝而非 mock。
func TestIntegration_MySQL_StartConnectionFailure(t *testing.T) {
	app := application.New()

	// 明显非法端口的本地 loopback DSN
	invalidDSN := "bamboo_user:test_user_password_123!@tcp(127.0.0.1:65530)/bamboo_test?charset=utf8mb4&parseTime=True&loc=Local&timeout=1s"
	mysqlAdapter := mysql.New(invalidDSN)

	dbInfo, dbFactory := database.New(database.WithAdapter(mysqlAdapter))
	if err := app.RegisterTool(dbInfo, dbFactory); err != nil {
		t.Fatalf("register database tool failed: %v", err)
	}

	// Worker 必须不会被拉起或在启动期即因依赖失败退出
	workerKey := ctxkey.Key("witness_worker_failure")
	workerStarted := false
	workerInfo := application.PluginInfo{
		Key:      workerKey,
		Name:     "witness_worker_failure",
		Mode:     application.ModeWorker,
		Requires: []ctxkey.Key{ctxkey.DatabaseKey},
	}

	_ = app.RegisterWorker(workerInfo, func() application.RunnablePlugin {
		return newWitnessWorker(func(ctx context.Context) error {
			workerStarted = true
			return nil
		})
	})

	err := app.Run()
	if err == nil {
		t.Fatal("expected app.Run to fail when database port is unreachable, got nil")
	}

	if workerStarted {
		t.Fatal("worker should not have run when database startup failed")
	}

	// 确认最终状态安全流转为 StateStopped，无遗留
	if app.State() != application.StateStopped {
		t.Fatalf("expected app state StateStopped, got: %v", app.State())
	}
}
