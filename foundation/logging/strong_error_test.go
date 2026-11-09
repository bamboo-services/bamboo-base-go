package logging_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
)

// failingWriter 是固定失败的写入器，用于验证双路写入失败的错误聚合。
type failingWriter struct {
	cause error
}

// Write 始终返回固定失败原因。
func (f *failingWriter) Write(_ []byte) (int, error) {
	return 0, f.cause
}

// TestLoggingErrorCodes 按表驱动断言日志包四个错误码的三元组与分组。
func TestLoggingErrorCodes(t *testing.T) {
	cases := []struct {
		name        string
		code        *apperr.ErrorCode
		wantCode    apperr.Code
		wantOutput  apperr.Output
		wantMessage apperr.Message
	}{
		{
			name:        "日志处理器缺失",
			code:        logging.HandlerInvalid,
			wantCode:    50028,
			wantOutput:  "LOGGING_HANDLER_INVALID",
			wantMessage: "日志处理器缺失",
		},
		{
			name:        "日志配置非法",
			code:        logging.ConfigInvalid,
			wantCode:    50029,
			wantOutput:  "LOGGING_CONFIG_INVALID",
			wantMessage: "日志配置非法",
		},
		{
			name:        "日志写入失败",
			code:        logging.WriteFailed,
			wantCode:    50030,
			wantOutput:  "LOGGING_WRITE_FAILED",
			wantMessage: "日志写入失败",
		},
		{
			name:        "日志轮转失败",
			code:        logging.RotationFailed,
			wantCode:    50031,
			wantOutput:  "LOGGING_ROTATION_FAILED",
			wantMessage: "日志轮转失败",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.code == nil {
				t.Fatal("错误码目录变量不得为 nil")
			}
			want := apperr.ErrorCode{
				Code:    testCase.wantCode,
				Output:  testCase.wantOutput,
				Message: testCase.wantMessage,
			}
			if *testCase.code != want {
				t.Fatalf("错误码三元组漂移: got %+v, want %+v", *testCase.code, want)
			}
			if group := int(testCase.code.Code / 100); group != 500 {
				t.Fatalf("HTTP 分组漂移: got %d, want 500", group)
			}
		})
	}
}

// TestLoggingErrorCodesAreDistinct 断言四个日志错误码互不重复。
func TestLoggingErrorCodesAreDistinct(t *testing.T) {
	codes := []*apperr.ErrorCode{
		logging.HandlerInvalid,
		logging.ConfigInvalid,
		logging.WriteFailed,
		logging.RotationFailed,
	}
	seen := map[apperr.Code]bool{}
	for _, code := range codes {
		if seen[code.Code] {
			t.Fatalf("错误码重复: %d", code.Code)
		}
		seen[code.Code] = true
	}
}

// TestNew_NilHandler_ReturnsHandlerInvalid 断言处理器缺失时返回 50028。
func TestNew_NilHandler_ReturnsHandlerInvalid(t *testing.T) {
	logger, appErr := logging.New(nil)
	if appErr == nil {
		t.Fatal("处理器缺失时必须返回错误")
	}
	if logger != nil {
		t.Fatalf("处理器缺失时不得返回日志器实例: %v", logger)
	}
	if appErr.Code != *logging.HandlerInvalid {
		t.Fatalf("错误码不符: got %+v, want %+v", appErr.Code, *logging.HandlerInvalid)
	}
	var target *apperr.Error
	if !errors.As(appErr, &target) {
		t.Fatal("失败结果必须可经 errors.As 提取统一错误")
	}
}

// TestNew_Success_ReturnsNilError 断言成功路径返回真正的 nil 错误。
//
// 该断言针对"非 nil 指针装入 error 接口"的假失败陷阱：
// 返回值必须能以 == nil 判定为成功。
func TestNew_Success_ReturnsNilError(t *testing.T) {
	logger, appErr := logging.New(logging.NewLogHandler(logging.HandlerConfig{
		Console: &bytes.Buffer{},
		Level:   slog.LevelInfo,
	}))
	if appErr != nil {
		t.Fatalf("成功路径不得返回错误: %v", appErr)
	}
	if logger == nil {
		t.Fatal("成功路径必须返回日志器实例")
	}
}

// TestHandle_Success_ReturnsNilError 断言双写成功时返回真正的 nil 错误。
func TestHandle_Success_ReturnsNilError(t *testing.T) {
	var consoleBuf, fileBuf bytes.Buffer
	handler := logging.NewLogHandler(logging.HandlerConfig{
		Console: &consoleBuf,
		File:    &fileBuf,
		Level:   slog.LevelInfo,
	})
	err := handler.Handle(context.Background(), slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "handle success",
	})
	if err != nil {
		t.Fatalf("成功路径必须返回真正的 nil 错误: %v", err)
	}
}

// TestHandle_SingleFailure_KeepsCause 断言单路失败保留底层原因。
func TestHandle_SingleFailure_KeepsCause(t *testing.T) {
	cause := errors.New("console writer refused")
	var fileBuf bytes.Buffer
	handler := logging.NewLogHandler(logging.HandlerConfig{
		Console: &failingWriter{cause: cause},
		File:    &fileBuf,
		Level:   slog.LevelInfo,
	})
	err := handler.Handle(context.Background(), slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "single failure",
	})
	if err == nil {
		t.Fatal("单路写入失败必须返回错误")
	}
	if !errors.Is(err, cause) {
		t.Fatalf("底层原因必须经 errors.Is 可达: %v", err)
	}
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		t.Fatal("失败结果必须可经 errors.As 提取统一错误")
	}
	if appErr.Code != *logging.WriteFailed {
		t.Fatalf("错误码不符: got %+v, want %+v", appErr.Code, *logging.WriteFailed)
	}
}

// TestHandle_DualFailure_JoinsBothCauses 断言双路失败时两路原因均可达。
func TestHandle_DualFailure_JoinsBothCauses(t *testing.T) {
	consoleCause := errors.New("console writer refused")
	fileCause := errors.New("file writer refused")
	handler := logging.NewLogHandler(logging.HandlerConfig{
		Console: &failingWriter{cause: consoleCause},
		File:    &failingWriter{cause: fileCause},
		Level:   slog.LevelInfo,
	})
	err := handler.Handle(context.Background(), slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "dual failure",
	})
	if err == nil {
		t.Fatal("双路写入失败必须返回错误")
	}
	if !errors.Is(err, consoleCause) {
		t.Fatalf("控制台原因必须经 errors.Is 可达: %v", err)
	}
	if !errors.Is(err, fileCause) {
		t.Fatalf("文件原因必须经 errors.Is 可达: %v", err)
	}
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		t.Fatal("失败结果必须可经 errors.As 提取统一错误")
	}
	if appErr.Code != *logging.WriteFailed {
		t.Fatalf("错误码不符: got %+v, want %+v", appErr.Code, *logging.WriteFailed)
	}
}

// TestRotatingWriter_ConfigInvalid 断言纯配置问题归类为 ConfigInvalid。
func TestRotatingWriter_ConfigInvalid(t *testing.T) {
	dir := t.TempDir()
	writer, appErr := logging.NewRotatingWriter(logging.RotatorConfig{
		Dir:      dir,
		BaseName: "nested/app",
		Ext:      ".log",
	})
	if appErr == nil {
		_ = writer.Close()
		t.Fatal("文件名含路径分隔符必须返回配置错误")
	}
	if appErr.Code != *logging.ConfigInvalid {
		t.Fatalf("错误码不符: got %+v, want %+v", appErr.Code, *logging.ConfigInvalid)
	}
}

// TestRotatingWriter_WriteAfterClose_KeepsErrClosed 断言关闭后写入仍可识别 ErrClosed。
func TestRotatingWriter_WriteAfterClose_KeepsErrClosed(t *testing.T) {
	dir := t.TempDir()
	writer, appErr := logging.NewRotatingWriter(logging.RotatorConfig{
		Dir:      dir,
		BaseName: "app",
		Ext:      ".log",
		MaxSize:  1024,
	})
	if appErr != nil {
		t.Fatalf("构造轮转写入器失败: %v", appErr)
	}

	n, err := writer.Write([]byte("payload\n"))
	if err != nil {
		t.Fatalf("成功路径必须返回真正的 nil 错误: %v", err)
	}
	if n != len("payload\n") {
		t.Fatalf("写入字节数不符: got %d", n)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("关闭失败: %v", err)
	}

	_, err = writer.Write([]byte("after close"))
	if err == nil {
		t.Fatal("关闭后写入必须失败")
	}
	if !errors.Is(err, logging.ErrClosed) {
		t.Fatalf("ErrClosed 必须经 errors.Is 可达: %v", err)
	}
	var appErrTarget *apperr.Error
	if !errors.As(err, &appErrTarget) {
		t.Fatal("失败结果必须可经 errors.As 提取统一错误")
	}
	if appErrTarget.Code != *logging.WriteFailed {
		t.Fatalf("错误码不符: got %+v, want %+v", appErrTarget.Code, *logging.WriteFailed)
	}
}

// TestRotatingWriter_CloseIsIdempotent 断言 Close 并发幂等且返回同一结果。
func TestRotatingWriter_CloseIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	writer, appErr := logging.NewRotatingWriter(logging.RotatorConfig{
		Dir:      dir,
		BaseName: "app",
		Ext:      ".log",
		MaxSize:  1024,
	})
	if appErr != nil {
		t.Fatalf("构造轮转写入器失败: %v", appErr)
	}

	const closers = 4
	results := make(chan error, closers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < closers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- writer.Close()
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	var first error
	for err := range results {
		if first == nil {
			first = err
			continue
		}
		if err != first {
			t.Fatalf("并发关闭结果必须一致: got %v, want %v", err, first)
		}
	}
	if first != nil {
		t.Fatalf("正常关闭不得返回错误: %v", first)
	}

	// 关闭完成后再调用仍然幂等，且不产生新的失败。
	if err := writer.Close(); err != nil {
		t.Fatalf("重复关闭必须幂等且无错误: %v", err)
	}
}
