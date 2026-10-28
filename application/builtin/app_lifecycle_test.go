package builtin_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/application/v2/builtin"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake"
)

type testWorker struct {
	logger *logging.Logger
	node   *snowflake.Node

	readyCh chan struct{}
	idCh    chan int64
	stopCh  chan struct{}
}

func (w *testWorker) Start(ctx context.Context) error {
	var errLogger *apperr.Error
	w.logger, errLogger = application.Get[*logging.Logger](ctx, builtin.LoggerKey)
	if errLogger != nil {
		return errLogger
	}

	var errNode *apperr.Error
	w.node, errNode = application.Get[*snowflake.Node](ctx, ctxkey.SnowflakeNodeKey)
	if errNode != nil {
		return errNode
	}
	return nil
}

func (w *testWorker) Run(ctx context.Context) error {
	id, genErr := w.node.Generate()
	if genErr != nil {
		return genErr
	}
	w.logger.WithName("TEST").Info(ctx, "worker-running", slog.Int64("id", int64(id)))
	close(w.readyCh)
	w.idCh <- int64(id)

	select {
	case <-ctx.Done():
		return nil
	case <-w.stopCh:
		return nil
	}
}

func (w *testWorker) Stop(ctx context.Context) error {
	close(w.stopCh)
	if w.logger != nil {
		w.logger.WithName("TEST").Info(ctx, "worker-stop-before-logger-close")
	}
	return nil
}

func (w *testWorker) Quiesce(ctx context.Context) error {
	return nil
}

func newWorkerFactory() (application.PluginInfo, func() application.RunnablePlugin, *testWorker) {
	worker := &testWorker{
		readyCh: make(chan struct{}),
		idCh:    make(chan int64, 1),
		stopCh:  make(chan struct{}),
	}
	info := application.PluginInfo{
		Key:      ctxkey.Key("worker_lifecycle_test"),
		Name:     "worker_lifecycle_test",
		Mode:     application.ModeWorker,
		Requires: []ctxkey.Key{builtin.LoggerKey, ctxkey.SnowflakeNodeKey},
	}
	fac := func() application.RunnablePlugin {
		return worker
	}
	return info, fac, worker
}

func setupApp(t *testing.T, dir string, dcID, nodeID int64) (*application.App, *testWorker) {
	t.Helper()
	logInfo, logFac := builtin.NewLogger(builtin.LoggerConfig{
		Rotator: logging.RotatorConfig{
			Dir:      dir,
			BaseName: "app",
			Ext:      ".log",
			MaxSize:  10 * 1024 * 1024,
		},
		Console: io.Discard,
	})
	sfInfo, sfFac := builtin.NewSnowflake(dcID, nodeID)
	wInfo, wFac, worker := newWorkerFactory()

	app := application.New(
		application.WithGraceTimeout(20*time.Millisecond),
		application.WithCleanupTimeout(100*time.Millisecond),
		application.WithShutdownTimeout(1*time.Second),
	)
	if err := app.RegisterTool(logInfo, logFac); err != nil {
		t.Fatalf("RegisterTool(logger) failed: %v", err)
	}
	if err := app.RegisterTool(sfInfo, sfFac); err != nil {
		t.Fatalf("RegisterTool(snowflake) failed: %v", err)
	}
	if err := app.RegisterWorker(wInfo, wFac); err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}
	return app, worker
}

func TestAppLifecycle_BuiltinLoggingAndSnowflake(t *testing.T) {
	origDefault := slog.Default()

	dirA := t.TempDir()
	appA, workerA := setupApp(t, dirA, 1, 1)

	var runDoneA = make(chan struct{})
	var runErrA error
	t.Cleanup(func() {
		_ = appA.Shutdown()
		select {
		case <-runDoneA:
		case <-time.After(2 * time.Second):
		}
	})

	go func() {
		runErrA = appA.Run()
		close(runDoneA)
	}()

	select {
	case <-workerA.readyCh:
	case <-runDoneA:
		t.Fatalf("appA run exited early with err: %v", runErrA)
	case <-time.After(1 * time.Second):
		t.Fatal("appA worker did not ready in time")
	}

	idA := <-workerA.idCh
	if idA == 0 {
		t.Fatal("appA generated zero snowflake id")
	}

	// App B 同时启动并保持活态，证明 A 与 B 同时运行且停 A 不伤正在运行的 B
	dirB := t.TempDir()
	appB, workerB := setupApp(t, dirB, 2, 2)

	var runDoneB = make(chan struct{})
	var runErrB error
	t.Cleanup(func() {
		_ = appB.Shutdown()
		select {
		case <-runDoneB:
		case <-time.After(2 * time.Second):
		}
	})

	go func() {
		runErrB = appB.Run()
		close(runDoneB)
	}()

	select {
	case <-workerB.readyCh:
	case <-runDoneB:
		t.Fatalf("appB run exited early with err: %v", runErrB)
	case <-time.After(1 * time.Second):
		t.Fatal("appB worker did not ready in time")
	}

	idBInitial := <-workerB.idCh
	if idBInitial == 0 || idBInitial == idA {
		t.Fatalf("appB initial id invalid or colliding: %d vs %d", idBInitial, idA)
	}

	// 此时 A 与 B 均处于真实 Ready 运行态，检查两个 logger 与 node 实例独立不共享
	if workerA.logger == workerB.logger {
		t.Fatal("workerA and workerB must not share *logging.Logger pointer")
	}
	if workerA.node == workerB.node {
		t.Fatal("workerA and workerB must not share *snowflake.Node pointer")
	}

	// 停止 App A，验证重复 Shutdown 为 nil 且正常退出
	if err := appA.Shutdown(); err != nil {
		t.Fatalf("appA first shutdown failed: %v", err)
	}
	if err := appA.Shutdown(); err != nil {
		t.Fatalf("appA repeated shutdown should be nil, got: %v", err)
	}

	select {
	case <-runDoneA:
	case <-time.After(1 * time.Second):
		t.Fatal("appA run did not finish after shutdown")
	}
	if runErrA != nil {
		t.Fatalf("appA run returned error: %v", runErrA)
	}

	// 读取已停止的 App A 的 stop 日志
	logFileA := filepath.Join(dirA, "app.log")
	contentA, err := os.ReadFile(logFileA)
	if err != nil {
		t.Fatalf("read logFileA failed: %v", err)
	}
	if !strings.Contains(string(contentA), "worker-stop-before-logger-close") {
		t.Fatalf("logFileA missing stop marker: %s", string(contentA))
	}

	// App A 停止后，仍在 Run 中的 App B 生成新 id，并用 WithName 记录 B-after-A-stop
	idBAfter, genErr := workerB.node.Generate()
	if genErr != nil {
		t.Fatalf("workerB Generate after appA stop failed: %v", genErr)
	}
	if idBAfter == 0 || int64(idBAfter) == idA || int64(idBAfter) == idBInitial {
		t.Fatalf("workerB generated invalid or colliding new id: %d (idA=%d, idBInitial=%d)", idBAfter, idA, idBInitial)
	}

	workerB.logger.WithName("TEST").Info(context.Background(), "B-after-A-stop", slog.Int64("idBAfter", int64(idBAfter)))

	// 验证 B 依然健康后正常关闭 App B
	if err := appB.Shutdown(); err != nil {
		t.Fatalf("appB shutdown failed: %v", err)
	}
	if err := appB.Shutdown(); err != nil {
		t.Fatalf("appB repeated shutdown should be nil, got: %v", err)
	}

	select {
	case <-runDoneB:
	case <-time.After(1 * time.Second):
		t.Fatal("appB run did not finish after shutdown")
	}
	if runErrB != nil {
		t.Fatalf("appB run returned error: %v", runErrB)
	}

	// 校验 App B 日志必须包含 B-after-A-stop 和 stop marker
	logFileB := filepath.Join(dirB, "app.log")
	contentB, err := os.ReadFile(logFileB)
	if err != nil {
		t.Fatalf("read logFileB failed: %v", err)
	}
	strB := string(contentB)
	if !strings.Contains(strB, "B-after-A-stop") {
		t.Fatalf("logFileB missing B-after-A-stop: %s", strB)
	}
	if !strings.Contains(strB, "worker-stop-before-logger-close") {
		t.Fatalf("logFileB missing stop marker: %s", strB)
	}

	if slog.Default() != origDefault {
		t.Fatal("slog.Default() pointer changed")
	}
}

func TestAppLifecycle_IllegalSnowflakeNodeFailsRun(t *testing.T) {
	dir := t.TempDir()
	logInfo, logFac := builtin.NewLogger(builtin.LoggerConfig{
		Rotator: logging.RotatorConfig{
			Dir:      dir,
			BaseName: "app",
			Ext:      ".log",
			MaxSize:  10 * 1024 * 1024,
		},
		Console: io.Discard,
	})
	sfInfo, sfFac := builtin.NewSnowflake(-1, 0)
	wInfo, wFac, worker := newWorkerFactory()

	app := application.New(
		application.WithGraceTimeout(20*time.Millisecond),
		application.WithCleanupTimeout(100*time.Millisecond),
		application.WithShutdownTimeout(1*time.Second),
	)
	_ = app.RegisterTool(logInfo, logFac)
	_ = app.RegisterTool(sfInfo, sfFac)
	_ = app.RegisterWorker(wInfo, wFac)

	var runDone = make(chan struct{})
	var runErr error
	t.Cleanup(func() {
		_ = app.Shutdown()
		select {
		case <-runDone:
		case <-time.After(2 * time.Second):
		}
	})

	go func() {
		runErr = app.Run()
		close(runDone)
	}()

	select {
	case <-worker.readyCh:
		t.Fatal("worker should not have started on illegal snowflake node")
	case <-runDone:
	case <-time.After(2 * time.Second):
		t.Fatal("app Run did not return on illegal snowflake node start failure")
	}

	if runErr == nil {
		t.Fatal("expected Run error for illegal snowflake node (-1, 0), got nil")
	}
	select {
	case id := <-worker.idCh:
		t.Fatalf("worker generated unexpected id: %d", id)
	default:
	}
}
