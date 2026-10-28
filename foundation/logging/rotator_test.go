package logging

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestRotatingWriter_BasicWriteAndContent(t *testing.T) {
	dir := t.TempDir()
	w, err := NewRotatingWriter(RotatorConfig{
		Dir:      dir,
		BaseName: "app",
		Ext:      ".log",
		MaxSize:  1024,
	})
	if err != nil {
		t.Fatalf("NewRotatingWriter failed: %v", err)
	}

	content := []byte("hello bamboo logging\n")
	n, err := w.Write(content)
	if err != nil || n != len(content) {
		t.Fatalf("Write returned (%d, %v), want (%d, nil)", n, err, len(content))
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	readBack, err := os.ReadFile(filepath.Join(dir, "app.log"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(readBack) != string(content) {
		t.Fatalf("Content mismatch: got %q, want %q", string(readBack), string(content))
	}
}

func TestRotatingWriter_SizeRotation(t *testing.T) {
	dir := t.TempDir()
	w, err := NewRotatingWriter(RotatorConfig{
		Dir:      dir,
		BaseName: "app",
		Ext:      ".log",
		MaxSize:  50,
	})
	if err != nil {
		t.Fatalf("NewRotatingWriter failed: %v", err)
	}

	// 写入超过 50 字节
	chunk := []byte("123456789012345678901234567890\n") // 31 bytes
	if _, err := w.Write(chunk); err != nil {
		t.Fatalf("Write 1 failed: %v", err)
	}
	// 第二次写入将超限触发切割
	if _, err := w.Write(chunk); err != nil {
		t.Fatalf("Write 2 failed: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// 应该生成了 app.0.log 和新的 app.log
	if _, err := os.Stat(filepath.Join(dir, "app.0.log")); err != nil {
		t.Fatalf("app.0.log should exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "app.log")); err != nil {
		t.Fatalf("app.log should exist: %v", err)
	}
}

func TestRotatingWriter_ArchiveTarGzContent(t *testing.T) {
	dir := t.TempDir()
	// 手动创建前一天的切割文件
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	file0 := filepath.Join(dir, "app.0.log")
	fileContent := "old log content from yesterday\n"
	if err := os.WriteFile(file0, []byte(fileContent), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	w, err := NewRotatingWriter(RotatorConfig{
		Dir:      dir,
		BaseName: "app",
		Ext:      ".log",
		MaxSize:  1024,
	})
	if err != nil {
		t.Fatalf("NewRotatingWriter failed: %v", err)
	}

	// 显式触发归档
	w.archiveYesterday()

	if err := w.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	archivePath := filepath.Join(dir, fmt.Sprintf("logger-%s.tar.gz", yesterday))
	f, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("archive file not found: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Errorf("f.Close failed: %v", err)
		}
	}()

	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip reader failed: %v", err)
	}
	defer func() {
		if err := gz.Close(); err != nil {
			t.Errorf("gz.Close failed: %v", err)
		}
	}()

	tr := tar.NewReader(gz)
	hdr, err := tr.Next()
	if err != nil {
		t.Fatalf("tar next failed: %v", err)
	}
	if hdr.Name != "app.0.log" {
		t.Fatalf("tar entry name mismatch: got %s, want app.0.log", hdr.Name)
	}

	data, err := io.ReadAll(tr)
	if err != nil {
		t.Fatalf("tar read failed: %v", err)
	}
	if string(data) != fileContent {
		t.Fatalf("archived content mismatch: got %q, want %q", string(data), fileContent)
	}

	// 归档后原文件应被清理
	if _, err := os.Stat(file0); !os.IsNotExist(err) {
		t.Fatalf("original file %s should be removed", file0)
	}
}

func TestRotatingWriter_ConcurrentWriteAndClose(t *testing.T) {
	dir := t.TempDir()
	w, err := NewRotatingWriter(RotatorConfig{
		Dir:      dir,
		BaseName: "app",
		Ext:      ".log",
		MaxSize:  100,
	})
	if err != nil {
		t.Fatalf("NewRotatingWriter failed: %v", err)
	}

	var wg sync.WaitGroup
	writeDone := make(chan struct{})

	// 并发写
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			line := []byte(fmt.Sprintf("worker %d payload\n", id))
			for {
				select {
				case <-writeDone:
					return
				default:
					_, err := w.Write(line)
					if err != nil {
						if errors.Is(err, ErrClosed) {
							return
						}
					}
				}
			}
		}(i)
	}

	time.Sleep(50 * time.Millisecond)
	close(writeDone)

	// 并发重复 Close
	closeWg := sync.WaitGroup{}
	for i := 0; i < 3; i++ {
		closeWg.Add(1)
		go func() {
			defer closeWg.Done()
			_ = w.Close()
		}()
	}

	closeWg.Wait()
	wg.Wait()

	// Close 后再次写必须拒绝
	_, err = w.Write([]byte("after close"))
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("expected ErrClosed after Close, got: %v", err)
	}

	// 再次 Close 必须幂等安全
	if err := w.Close(); err != nil {
		t.Fatalf("repeat Close failed: %v", err)
	}
}

func TestRotatingWriter_CloseWaitsForArchivalBarrier(t *testing.T) {
	dir := t.TempDir()
	// 先放置一个旧文件供归档
	file0 := filepath.Join(dir, "app.0.log")
	if err := os.WriteFile(file0, []byte("content\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	w, err := NewRotatingWriter(RotatorConfig{
		Dir:      dir,
		BaseName: "app",
		Ext:      ".log",
		MaxSize:  1024,
	})
	if err != nil {
		t.Fatalf("NewRotatingWriter failed: %v", err)
	}

	barrierHold := make(chan struct{})
	barrierEntered := make(chan struct{})
	barrierDone := make(chan struct{})

	w.beforeArchiveFileHook = func(filePath string) {
		close(barrierEntered)
		<-barrierHold
		close(barrierDone)
	}

	// 异步启动归档登记
	w.triggerArchiveAsync()

	// 等待归档进入障碍点
	<-barrierEntered

	closeReturned := make(chan error, 1)
	go func() {
		closeReturned <- w.Close()
	}()

	// 确认在 release barrier 前，Close 并未返回
	select {
	case err := <-closeReturned:
		t.Fatalf("Close returned prematurely before archive finished: %v", err)
	case <-time.After(50 * time.Millisecond):
		// 符合预期，Close 正在等待
	}

	// 启动第二个并发 Close，验证其同样在 barrier release 前不能返回
	secondCloseReturned := make(chan error, 1)
	go func() {
		secondCloseReturned <- w.Close()
	}()

	select {
	case err := <-secondCloseReturned:
		t.Fatalf("Second Close returned prematurely before archive finished: %v", err)
	case <-time.After(50 * time.Millisecond):
		// 符合预期，第二个 Close 必须等待 closeDone
	}

	// 解除归档障碍
	close(barrierHold)
	<-barrierDone

	// 等待两个 Close 返回，并核对共享结果一致
	var err1, err2 error
	select {
	case err1 = <-closeReturned:
	case <-time.After(2 * time.Second):
		t.Fatalf("Close timed out waiting for archive barrier")
	}

	select {
	case err2 = <-secondCloseReturned:
	case <-time.After(2 * time.Second):
		t.Fatalf("Second Close timed out waiting for archive barrier")
	}

	if err1 != err2 {
		t.Fatalf("Close results mismatch: err1=%v, err2=%v", err1, err2)
	}
}
