package memory

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestStoreLifecycle_DefaultShards(t *testing.T) {
	s := NewStore(0, 0, 0)
	defer s.Close()

	if len(s.shards) != 16 {
		t.Fatalf("expected 16 shards by default, got %d", len(s.shards))
	}
	if s.shardMask != 15 {
		t.Fatalf("expected shardMask 15, got %d", s.shardMask)
	}
}

func TestStoreLifecycle_ConcurrentClose(t *testing.T) {
	s := NewStore(4, 100, time.Minute)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			s.Close()
		}()
	}

	wg.Wait()

	// 再次调用 Close 仍然安全且幂等
	s.Close()

	select {
	case <-s.janitor.doneCh:
		// doneCh 已关闭说明 janitor 已退出
	default:
		t.Fatal("janitor doneCh should be closed after Close()")
	}
}

func TestStoreLifecycle_CloseWaitsForActiveCleanup(t *testing.T) {
	s := NewStore(2, 10, time.Second)

	cleanupStarted := make(chan struct{})
	allowCleanupFinish := make(chan struct{})
	var cleanupExited atomic.Bool

	s.onCleanupHook = func() {
		close(cleanupStarted)
		<-allowCleanupFinish
		cleanupExited.Store(true)
	}

	// 触发一次异步 cleanup 模拟正在执行的清理任务
	go s.cleanup()

	// 等待 cleanup 进入执行中
	<-cleanupStarted

	// 并发启动 Close()，此时 cleanup 尚未放行，Close() 不得提前返回
	closeReturned := make(chan struct{})
	go func() {
		s.Close()
		close(closeReturned)
	}()

	// 验证在 allowCleanupFinish 之前 Close 阻塞未返回
	select {
	case <-closeReturned:
		t.Fatal("Close() returned prematurely before active cleanup finished")
	case <-time.After(50 * time.Millisecond):
		// 正常阻塞
	}

	// 放行 cleanup
	close(allowCleanupFinish)

	// Close() 应该顺利返回
	select {
	case <-closeReturned:
	case <-time.After(2 * time.Second):
		t.Fatal("Close() did not return in time after cleanup finished")
	}

	if !cleanupExited.Load() {
		t.Fatal("cleanup should have exited completely before Close() returns")
	}
}

func TestStoreLifecycle_PostCloseReadWrite(t *testing.T) {
	s := NewStore(4, 50, time.Minute)
	s.Set("key1", "val1", 0)

	s.Close()

	// Close 后旧读写仍可用且正确
	val, ok := s.Get("key1")
	if !ok || val != "val1" {
		t.Fatalf("expected to get val1 after close, got val=%v ok=%v", val, ok)
	}

	s.Set("key2", "val2", 0)
	val2, ok2 := s.Get("key2")
	if !ok2 || val2 != "val2" {
		t.Fatalf("expected to get val2 after close, got val=%v ok=%v", val2, ok2)
	}

	if !s.Exists("key1") || !s.Exists("key2") {
		t.Fatal("Exists returned false after Close")
	}

	s.Update("key2", 0, func(old any) any {
		return "val2_updated"
	})
	val2Up, ok2Up := s.Get("key2")
	if !ok2Up || val2Up != "val2_updated" {
		t.Fatalf("expected val2_updated, got %v", val2Up)
	}

	if !s.Delete("key1") {
		t.Fatal("Delete returned false after Close")
	}
	if s.Exists("key1") {
		t.Fatal("key1 still exists after Delete")
	}
}
