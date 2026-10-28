package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
)

type failWriter struct {
	errMsg string
}

func (f *failWriter) Write(_ []byte) (int, error) {
	return 0, errors.New(f.errMsg)
}

func TestNew_RejectsNilHandler(t *testing.T) {
	logger, err := logging.New(nil)
	if err == nil {
		t.Fatal("expected error when handler is nil, got nil")
	}
	if logger != nil {
		t.Fatalf("expected logger to be nil, got %v", logger)
	}
}

func TestSlogDefault_RemainsUnchanged(t *testing.T) {
	origDefault := slog.Default()

	var buf bytes.Buffer
	handler := logging.NewLogHandler(logging.HandlerConfig{
		Console: &buf,
		Level:   slog.LevelDebug,
	})
	logger, err := logging.New(handler)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	named := logger.WithName("TEST")
	named.Info(context.Background(), "isolated message")

	if slog.Default() != origDefault {
		t.Fatal("slog.Default was modified, expected it to remain unchanged")
	}
}

func TestTwoLoggers_OutputIsolation(t *testing.T) {
	var buf1, buf2 bytes.Buffer

	h1 := logging.NewLogHandler(logging.HandlerConfig{
		Console: &buf1,
		Level:   slog.LevelDebug,
	})
	h2 := logging.NewLogHandler(logging.HandlerConfig{
		Console: &buf2,
		Level:   slog.LevelDebug,
	})

	log1, err := logging.New(h1)
	if err != nil {
		t.Fatalf("failed to create log1: %v", err)
	}
	log2, err := logging.New(h2)
	if err != nil {
		t.Fatalf("failed to create log2: %v", err)
	}

	n1 := log1.WithName("APP1")
	n2 := log2.WithName("APP2")

	ctx := context.Background()
	n1.Info(ctx, "hello from app 1")
	n2.Info(ctx, "hello from app 2")

	out1 := buf1.String()
	out2 := buf2.String()

	if !strings.Contains(out1, "hello from app 1") || strings.Contains(out1, "hello from app 2") {
		t.Fatalf("unexpected content in log1: %s", out1)
	}
	if !strings.Contains(out2, "hello from app 2") || strings.Contains(out2, "hello from app 1") {
		t.Fatalf("unexpected content in log2: %s", out2)
	}
}

func TestTraceID_Extraction(t *testing.T) {
	var consoleBuf, fileBuf bytes.Buffer
	handler := logging.NewLogHandler(logging.HandlerConfig{
		Console: &consoleBuf,
		File:    &fileBuf,
		Level:   slog.LevelDebug,
	})
	logger, err := logging.New(handler)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	traceVal := "req-trace-123456"
	ctx := context.WithValue(context.Background(), ctxkey.RequestKey, traceVal)

	named := logger.WithName(logging.NamedCORE)
	named.Info(ctx, "trace test message")

	// 检查控制台输出
	consoleOut := consoleBuf.String()
	if !strings.Contains(consoleOut, traceVal) {
		t.Fatalf("console output does not contain trace id %q: %s", traceVal, consoleOut)
	}

	// 检查文件 JSON 输出
	var entry map[string]any
	if err := json.Unmarshal(fileBuf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse json file output: %v, raw: %s", err, fileBuf.String())
	}
	if entry["trace"] != traceVal {
		t.Fatalf("expected trace %q, got %v", traceVal, entry["trace"])
	}
	if entry["logger"] != logging.NamedCORE {
		t.Fatalf("expected logger %q, got %v", logging.NamedCORE, entry["logger"])
	}
	if entry["message"] != "trace test message" {
		t.Fatalf("expected message 'trace test message', got %v", entry["message"])
	}
}

func TestDualWrite_AndLevels(t *testing.T) {
	var consoleBuf, fileBuf bytes.Buffer
	handler := logging.NewLogHandler(logging.HandlerConfig{
		Console: &consoleBuf,
		File:    &fileBuf,
		Level:   slog.LevelWarn,
	})
	logger, err := logging.New(handler)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	named := logger.WithName(logging.NamedBUSI, "OPT1", "OPT2")
	ctx := context.Background()

	// Debug 和 Info 应被过滤（低于 Warn）
	named.Debug(ctx, "debug should be skipped")
	named.Info(ctx, "info should be skipped")
	named.SugarInfo(ctx, "sugar info should be skipped")

	if consoleBuf.Len() > 0 || fileBuf.Len() > 0 {
		t.Fatalf("expected no output for Debug/Info under Warn level, got console: %s, file: %s", consoleBuf.String(), fileBuf.String())
	}

	// Warn 应该输出
	named.Warn(ctx, "warn msg", slog.String("k1", "v1"))
	if !strings.Contains(consoleBuf.String(), "warn msg") {
		t.Fatalf("expected warn msg in console output: %s", consoleBuf.String())
	}
	if !strings.Contains(fileBuf.String(), "warn msg") {
		t.Fatalf("expected warn msg in file output: %s", fileBuf.String())
	}

	// Error 应该输出
	named.Error(ctx, "error msg")
	if !strings.Contains(consoleBuf.String(), "error msg") {
		t.Fatalf("expected error msg in console output: %s", consoleBuf.String())
	}
	if !strings.Contains(fileBuf.String(), "error msg") {
		t.Fatalf("expected error msg in file output: %s", fileBuf.String())
	}

	// 验证 options
	if !strings.Contains(consoleBuf.String(), "OPT1") || !strings.Contains(consoleBuf.String(), "OPT2") {
		t.Fatalf("console output missing option names: %s", consoleBuf.String())
	}
}

func TestSugarMethods_AndNotice(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewLogHandler(logging.HandlerConfig{
		Console: &buf,
		Level:   slog.LevelDebug,
	})
	logger, err := logging.New(handler)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	named := logger.WithName("SUGAR")
	ctx := context.Background()

	named.SugarDebug(ctx, "msg debug", "extra", 123)
	named.SugarWarn(ctx, "msg warn", "reason", "bad")
	named.SugarError(ctx, "msg error", "code", 500)
	named.Notice(ctx, "msg notice", slog.Int("n", 1))
	named.SugarNotice(ctx, "msg sugar notice", "n", 2)

	out := buf.String()
	for _, m := range []string{"msg debug", "msg warn", "msg error", "msg notice", "msg sugar notice"} {
		if !strings.Contains(out, m) {
			t.Fatalf("expected output to contain %q, full: %s", m, out)
		}
	}
}

func TestPanicMethods_TriggerPanic(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewLogHandler(logging.HandlerConfig{
		Console: &buf,
		Level:   slog.LevelDebug,
	})
	logger, err := logging.New(handler)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	named := logger.WithName("PANIC")

	func() {
		defer func() {
			r := recover()
			if r != "panic message" {
				t.Fatalf("expected recovered 'panic message', got %v", r)
			}
		}()
		named.Panic(context.Background(), "panic message")
	}()

	func() {
		defer func() {
			r := recover()
			if r != "sugar panic message" {
				t.Fatalf("expected recovered 'sugar panic message', got %v", r)
			}
		}()
		named.SugarPanic(context.Background(), "sugar panic message", "key", "val")
	}()
}

func TestWriterError_PreservedAndNotSwallowed(t *testing.T) {
	// 1. Console fail
	hConsoleFail := logging.NewLogHandler(logging.HandlerConfig{
		Console: &failWriter{errMsg: "disk full on stdout"},
		Level:   slog.LevelInfo,
	})
	err := hConsoleFail.Handle(context.Background(), slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "test console fail",
	})
	if err == nil || !strings.Contains(err.Error(), "disk full on stdout") {
		t.Fatalf("expected error containing 'disk full on stdout', got %v", err)
	}

	// 2. File fail
	var consoleBuf bytes.Buffer
	hFileFail := logging.NewLogHandler(logging.HandlerConfig{
		Console: &consoleBuf,
		File:    &failWriter{errMsg: "file write permission denied"},
		Level:   slog.LevelInfo,
	})
	err = hFileFail.Handle(context.Background(), slog.Record{
		Level:   slog.LevelInfo,
		Message: "test file fail",
	})
	if err == nil || !strings.Contains(err.Error(), "file write permission denied") {
		t.Fatalf("expected error containing 'file write permission denied', got %v", err)
	}

	// 3. Both fail
	hBothFail := logging.NewLogHandler(logging.HandlerConfig{
		Console: &failWriter{errMsg: "console err"},
		File:    &failWriter{errMsg: "file err"},
		Level:   slog.LevelInfo,
	})
	err = hBothFail.Handle(context.Background(), slog.Record{
		Level:   slog.LevelInfo,
		Message: "test both fail",
	})
	if err == nil || !strings.Contains(err.Error(), "console err") || !strings.Contains(err.Error(), "file err") {
		t.Fatalf("expected error containing both failures, got %v", err)
	}
}

func TestWithAttrs_AndWithGroup(t *testing.T) {
	var fileBuf bytes.Buffer
	handler := logging.NewLogHandler(logging.HandlerConfig{
		File:  &fileBuf,
		Level: slog.LevelInfo,
	})

	hWithGroup := handler.WithGroup("MYGROUP")
	hWithAttrs := hWithGroup.WithAttrs([]slog.Attr{slog.String("env", "prod")})

	rec := slog.Record{
		Level:   slog.LevelInfo,
		Message: "grouped message",
	}
	if err := hWithAttrs.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(fileBuf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}
	if entry["logger"] != "MYGROUP" {
		t.Fatalf("expected logger 'MYGROUP', got %v", entry["logger"])
	}
	if entry["env"] != "prod" {
		t.Fatalf("expected env 'prod', got %v", entry["env"])
	}
}
