package builtin

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake"
)

func TestBuiltinLogger_NewFactoryNoIO_StartCreatesDir(t *testing.T) {
	tempBase := t.TempDir()
	logSubDir := filepath.Join(tempBase, "nested", "logs")

	info, factory := NewLogger(LoggerConfig{
		Rotator: logging.RotatorConfig{
			Dir: logSubDir,
		},
		Console: io.Discard,
		Level:   slog.LevelInfo,
	})

	if info.Key != LoggerPluginKey {
		t.Fatalf("expected key %s, got %s", LoggerPluginKey, info.Key)
	}

	// 1. New + factory 纯构造，绝不提前创建目标目录
	p := factory()
	if _, err := os.Stat(logSubDir); !os.IsNotExist(err) {
		t.Fatalf("factory must not create dir before Start, stat err: %v", err)
	}

	ctx := context.Background()
	// 2. Start 后才真正创建目标目录并生成 writer
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() {
		_ = p.Stop(ctx)
	}()

	if _, err := os.Stat(logSubDir); err != nil {
		t.Fatalf("expected dir to exist after Start: %v", err)
	}

	exposer, ok := p.(ctxkey.Exposer)
	if !ok {
		t.Fatalf("plugin does not implement Exposer")
	}

	exposed := exposer.Expose()
	loggerVal, ok := exposed[LoggerKey]
	if !ok || loggerVal == nil {
		t.Fatalf("expected logger in exposed map")
	}

	logger, ok := loggerVal.(*logging.Logger)
	if !ok || logger == nil {
		t.Fatalf("exposed value is not *logging.Logger")
	}

	// 重复 Start 必须报错且不 panic
	if err := p.Start(ctx); err == nil {
		t.Fatalf("expected error on duplicate Start, got nil")
	}
}

func TestBuiltinLogger_IsolationAndGlobalDefaultUntouched(t *testing.T) {
	origDefault := slog.Default()

	dir1 := t.TempDir()
	dir2 := t.TempDir()

	_, factory1 := NewLogger(LoggerConfig{
		Rotator: logging.RotatorConfig{Dir: dir1},
		Console: io.Discard,
		Level:   slog.LevelInfo,
	})
	_, factory2 := NewLogger(LoggerConfig{
		Rotator: logging.RotatorConfig{Dir: dir2},
		Console: io.Discard,
		Level:   slog.LevelInfo,
	})

	p1 := factory1()
	p2 := factory2()

	ctx := context.Background()
	t.Cleanup(func() {
		_ = p1.Stop(ctx)
		_ = p2.Stop(ctx)
	})

	if err := p1.Start(ctx); err != nil {
		t.Fatalf("p1 Start failed: %v", err)
	}
	if err := p2.Start(ctx); err != nil {
		t.Fatalf("p2 Start failed: %v", err)
	}

	exp1 := p1.(ctxkey.Exposer).Expose()
	exp2 := p2.(ctxkey.Exposer).Expose()

	l1 := exp1[LoggerKey].(*logging.Logger)
	l2 := exp2[LoggerKey].(*logging.Logger)

	if l1 == l2 {
		t.Fatalf("two logger instances must not share *logging.Logger reference")
	}

	if slog.Default() != origDefault {
		t.Fatalf("global slog.Default pointer must not be modified")
	}
}

func TestBuiltinLogger_RealLogCloseAndContent(t *testing.T) {
	logDir := t.TempDir()

	_, factory := NewLogger(LoggerConfig{
		Rotator: logging.RotatorConfig{
			Dir:      logDir,
			BaseName: "app",
			Ext:      ".log",
			MaxSize:  1024 * 1024,
		},
		Console: io.Discard,
		Level:   slog.LevelInfo,
	})

	p := factory()
	ctx := context.Background()
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	exp := p.(ctxkey.Exposer).Expose()
	l := exp[LoggerKey].(*logging.Logger)
	named := l.WithName("TEST")
	named.Info(ctx, "test message")

	// 验证私有 writer 并在 Stop 之后验证真实已 close
	lp, ok := p.(*loggerPlugin)
	if !ok {
		t.Fatalf("expected *loggerPlugin instance")
	}
	rawWriter := lp.writer
	if rawWriter == nil {
		t.Fatalf("expected non-nil internal writer")
	}

	// 真实停止，并验证重复 Stop 结果共享与等待
	if err := p.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if err := p.Stop(ctx); err != nil {
		t.Fatalf("duplicate Stop failed: %v", err)
	}

	// 再次向原 writer 写入必须报错 ErrClosed
	if _, err := rawWriter.Write([]byte("after close")); !errors.Is(err, logging.ErrClosed) {
		t.Fatalf("expected ErrClosed on writer after plugin Stop, got: %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(logDir, "*.log"))
	if err != nil || len(matches) == 0 {
		t.Fatalf("expected log file to be written in %s, matches: %v, err: %v", logDir, matches, err)
	}

	content, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(content), "test message") {
		t.Fatalf("expected log content to contain 'test message', got: %s", string(content))
	}
}

func TestBuiltinSnowflake_LifecycleAndIsolation(t *testing.T) {
	// 1. 非法 ID 导致 Start 报错
	_, invalidFactory := NewSnowflake(-1, 0)
	pInvalid := invalidFactory()
	if err := pInvalid.Start(context.Background()); err == nil {
		t.Fatalf("expected Start error for negative datacenterID, got nil")
	}

	// 2. 正常初始化与隔离性
	_, factory1 := NewSnowflake(1, 1)
	_, factory2 := NewSnowflake(1, 2)

	p1 := factory1()
	p2 := factory2()

	ctx := context.Background()
	t.Cleanup(func() {
		_ = p1.Stop(ctx)
		_ = p2.Stop(ctx)
	})

	if err := p1.Start(ctx); err != nil {
		t.Fatalf("p1 Start failed: %v", err)
	}
	if err := p2.Start(ctx); err != nil {
		t.Fatalf("p2 Start failed: %v", err)
	}

	// 重复 Start 校验
	if err := p1.Start(ctx); err == nil {
		t.Fatalf("expected error on duplicate Start for snowflake, got nil")
	}

	node1 := p1.(ctxkey.Exposer).Expose()[ctxkey.SnowflakeNodeKey].(*snowflake.Node)
	node2 := p2.(ctxkey.Exposer).Expose()[ctxkey.SnowflakeNodeKey].(*snowflake.Node)

	if node1 == node2 {
		t.Fatalf("two snowflake instances must not share *snowflake.Node pointer")
	}

	id1, err := node1.Generate()
	if err != nil {
		t.Fatalf("generate id1 failed: %v", err)
	}
	id2, err := node2.Generate()
	if err != nil {
		t.Fatalf("generate id2 failed: %v", err)
	}
	if id1 == id2 {
		t.Fatalf("expected distinct IDs, got identical %d", id1)
	}

	// 3. Stop 清理
	if err := p1.Stop(ctx); err != nil {
		t.Fatalf("p1 Stop failed: %v", err)
	}
	if exp := p1.(ctxkey.Exposer).Expose(); exp[ctxkey.SnowflakeNodeKey] != nil {
		t.Fatalf("expected node to be cleared after Stop")
	}
}
