package consumer_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/plugins/async/v2"
	sqliteadapter "github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/sqlite/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2"
	"gorm.io/gorm"
)

type lifecycleTestWorker struct {
	runFn func(ctx context.Context) error
}

func (w *lifecycleTestWorker) Start(_ context.Context) error { return nil }
func (w *lifecycleTestWorker) Stop(_ context.Context) error  { return nil }
func (w *lifecycleTestWorker) Run(ctx context.Context) error {
	if w.runFn != nil {
		return w.runFn(ctx)
	}
	<-ctx.Done()
	return nil
}

func TestSQLiteAsyncLifecycle_CaseA_AppForceUnblocksTask_OrderlyShutdown(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_case_a.db")
	dbInfo, dbFactory := database.New(database.WithAdapter(sqliteadapter.New(dbPath)))
	asyncInfo, asyncFactory := async.New(async.WithRequires(ctxkey.DatabaseKey))

	app := application.New(
		application.WithGraceTimeout(20*time.Millisecond),
		application.WithCleanupTimeout(100*time.Millisecond),
		application.WithShutdownTimeout(150*time.Millisecond),
	)

	if err := app.RegisterTool(dbInfo, dbFactory); err != nil {
		t.Fatalf("RegisterTool database: %v", err)
	}
	if err := app.RegisterTool(asyncInfo, asyncFactory); err != nil {
		t.Fatalf("RegisterTool async: %v", err)
	}

	taskStarted := make(chan struct{})
	appForceReceived := make(chan struct{})
	sqlCompleted := make(chan struct{})
	sqlDBCh := make(chan *sql.DB, 1)

	workerKey := ctxkey.Key("worker_case_a")
	worker := &lifecycleTestWorker{
		runFn: func(lifetimeCtx context.Context) error {
			db, getDBErr := application.Get[*gorm.DB](lifetimeCtx, ctxkey.DatabaseKey)
			if getDBErr != nil {
				return getDBErr
			}
			rawSqlDB, err := db.DB()
			if err != nil {
				return err
			}
			sqlDBCh <- rawSqlDB

			pool, getPoolErr := application.Get[async.Pool](lifetimeCtx, async.PoolKey)
			if getPoolErr != nil {
				return getPoolErr
			}
			if err := db.Exec("CREATE TABLE test_logs (id INTEGER PRIMARY KEY, msg TEXT);").Error; err != nil {
				return err
			}

			task, err := pool.Submit(lifetimeCtx, func(taskCtx context.Context) {
				close(taskStarted)
				<-taskCtx.Done()
				close(appForceReceived)

				safeCtx := context.Background()
				if err := db.WithContext(safeCtx).Exec("INSERT INTO test_logs(msg) VALUES(?);", "normal_after_force").Error; err != nil {
					t.Errorf("db.Exec failed after taskCtx cancel: %v", err)
				}
				close(sqlCompleted)
			})
			if err != nil {
				return err
			}
			return task.Wait()
		},
	}

	if err := app.RegisterWorker(application.PluginInfo{
		Key:      workerKey,
		Requires: []ctxkey.Key{async.PoolKey, ctxkey.DatabaseKey},
		Mode:     application.ModeWorker,
	}, func() application.RunnablePlugin {
		return worker
	}); err != nil {
		t.Fatalf("RegisterWorker: %v", err)
	}

	runDone := make(chan struct{})
	var runErr error
	t.Cleanup(func() {
		_ = app.Shutdown()
		select {
		case <-runDone:
		case <-time.After(1 * time.Second):
			t.Error("cleanup: timed out waiting for app.Run to terminate")
		}
	})

	go func() {
		defer close(runDone)
		runErr = app.Run()
	}()

	select {
	case <-taskStarted:
	case <-time.After(1 * time.Second):
		t.Fatal("task did not start in time")
	}

	var originalSQLDB *sql.DB
	select {
	case originalSQLDB = <-sqlDBCh:
	case <-time.After(1 * time.Second):
		t.Fatal("worker did not publish sql.DB in time")
	}

	if err := app.Shutdown(); err != nil {
		t.Fatalf("expected clean Shutdown, got: %v", err)
	}

	select {
	case <-appForceReceived:
	default:
		t.Fatal("expected task to receive App force cancellation")
	}

	select {
	case <-sqlCompleted:
	default:
		t.Fatal("expected sql record to complete before app fully finishes")
	}

	select {
	case <-runDone:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for app.Run to exit")
	}
	if runErr != nil {
		t.Fatalf("app.Run failed: %v", runErr)
	}

	dupErr := app.Shutdown()
	if dupErr != nil {
		t.Fatalf("duplicate Shutdown expected nil, got: %v", dupErr)
	}

	if err := originalSQLDB.Ping(); err == nil {
		t.Fatal("expected original sql.DB to be closed after orderly Shutdown, Ping succeeded")
	}

	reopenDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to reopen sqlite database: %v", err)
	}
	defer reopenDB.Close()

	var count int
	if err := reopenDB.QueryRow("SELECT COUNT(*) FROM test_logs WHERE msg = ?", "normal_after_force").Scan(&count); err != nil {
		t.Fatalf("failed to query records from reopened SQLite: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 record in SQLite, got %d", count)
	}
}

func TestSQLiteAsyncLifecycle_CaseB_TaskIgnoresCancel_RetainsDatabase_Incomplete(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_case_b.db")
	dbInfo, dbFactory := database.New(database.WithAdapter(sqliteadapter.New(dbPath)))
	asyncInfo, asyncFactory := async.New(async.WithRequires(ctxkey.DatabaseKey))

	app := application.New(
		application.WithGraceTimeout(20*time.Millisecond),
		application.WithCleanupTimeout(100*time.Millisecond),
		application.WithShutdownTimeout(150*time.Millisecond),
	)

	if err := app.RegisterTool(dbInfo, dbFactory); err != nil {
		t.Fatalf("RegisterTool database: %v", err)
	}
	if err := app.RegisterTool(asyncInfo, asyncFactory); err != nil {
		t.Fatalf("RegisterTool async: %v", err)
	}

	taskStarted := make(chan struct{})
	taskReady := make(chan struct{})
	releaseTask := make(chan struct{})
	var releaseOnce sync.Once

	var capturedTask *async.Task
	var capturedSqlDB *sql.DB
	var taskDoneFlag atomic.Bool

	runDone := make(chan struct{})
	var runErr error

	t.Cleanup(func() {
		releaseOnce.Do(func() { close(releaseTask) })
		_ = app.Shutdown()
		select {
		case <-taskReady:
			if capturedTask != nil {
				_ = capturedTask.Wait()
			}
		case <-time.After(500 * time.Millisecond):
		}
		select {
		case <-runDone:
		case <-time.After(1 * time.Second):
			t.Error("cleanup: timed out waiting for app.Run to terminate")
		}
		if capturedSqlDB != nil {
			_ = capturedSqlDB.Close()
		}
	})

	workerKey := ctxkey.Key("worker_case_b")
	worker := &lifecycleTestWorker{
		runFn: func(lifetimeCtx context.Context) error {
			db, getDBErr := application.Get[*gorm.DB](lifetimeCtx, ctxkey.DatabaseKey)
			if getDBErr != nil {
				return getDBErr
			}
			sDB, err := db.DB()
			if err != nil {
				return err
			}
			capturedSqlDB = sDB

			pool, getPoolErr := application.Get[async.Pool](lifetimeCtx, async.PoolKey)
			if getPoolErr != nil {
				return getPoolErr
			}
			if err := db.Exec("CREATE TABLE test_logs (id INTEGER PRIMARY KEY, msg TEXT);").Error; err != nil {
				return err
			}

			tSub, err := pool.Submit(lifetimeCtx, func(taskCtx context.Context) {
				close(taskStarted)
				<-releaseTask
				safeCtx := context.Background()
				if err := db.WithContext(safeCtx).Exec("INSERT INTO test_logs(msg) VALUES(?);", "after_release").Error; err != nil {
					t.Errorf("Exec after release: %v", err)
				}
				taskDoneFlag.Store(true)
			})
			if err != nil {
				return err
			}
			capturedTask = tSub
			close(taskReady)
			<-lifetimeCtx.Done()
			return nil
		},
	}

	if err := app.RegisterWorker(application.PluginInfo{
		Key:      workerKey,
		Requires: []ctxkey.Key{async.PoolKey, ctxkey.DatabaseKey},
		Mode:     application.ModeWorker,
	}, func() application.RunnablePlugin {
		return worker
	}); err != nil {
		t.Fatalf("RegisterWorker: %v", err)
	}

	go func() {
		defer close(runDone)
		runErr = app.Run()
	}()

	select {
	case <-taskStarted:
	case <-time.After(1 * time.Second):
		t.Fatal("task did not start in time")
	}

	select {
	case <-taskReady:
	case <-time.After(1 * time.Second):
		t.Fatal("task was not ready in time")
	}

	shutErr := app.Shutdown()
	if shutErr == nil {
		t.Fatal("expected StopIncompleteError, got nil")
	}
	var incErr *application.StopIncompleteError
	if !errors.As(shutErr, &incErr) {
		t.Fatalf("expected StopIncompleteError, got: %v", shutErr)
	}

	if st := app.State(); st != application.StateIncomplete {
		t.Fatalf("expected StateIncomplete, got %v", st)
	}

	var foundAsyncStuck bool
	for _, stuck := range incErr.StuckInstances {
		if stuck.PluginKey == async.PluginKey {
			foundAsyncStuck = true
			var hasDB bool
			for _, dep := range stuck.RetainedDeps {
				if dep == ctxkey.DatabaseKey {
					hasDB = true
					break
				}
			}
			if !hasDB {
				t.Fatalf("expected retained deps to contain DatabaseKey, got: %v", stuck.RetainedDeps)
			}
		}
	}
	if !foundAsyncStuck {
		t.Fatalf("expected async stuck instance in %v", incErr.StuckInstances)
	}

	if err := capturedSqlDB.Ping(); err != nil {
		t.Fatalf("sqlDB must NOT be closed while retained: %v", err)
	}

	dupShutErr := app.Shutdown()
	if dupShutErr == nil {
		t.Fatal("expected duplicate Shutdown to return frozen non-nil StopIncompleteError, got nil")
	}
	if !errors.Is(dupShutErr, shutErr) && dupShutErr.Error() != shutErr.Error() {
		t.Fatalf("expected duplicate Shutdown to return frozen result, got: %v", dupShutErr)
	}

	releaseOnce.Do(func() { close(releaseTask) })

	if err := capturedTask.Wait(); err != nil {
		t.Fatalf("task.Wait: %v", err)
	}
	if !taskDoneFlag.Load() {
		t.Fatal("expected task to be marked done after release")
	}

	select {
	case <-runDone:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for app.Run to terminate after release")
	}
	_ = runErr

	reopenDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to reopen sqlite db: %v", err)
	}
	defer reopenDB.Close()

	var count int
	if err := reopenDB.QueryRow("SELECT COUNT(*) FROM test_logs WHERE msg = ?", "after_release").Scan(&count); err != nil {
		t.Fatalf("failed to query records from reopened SQLite: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row recorded after release, got %d", count)
	}
}
