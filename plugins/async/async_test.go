package xAsync

import (
	"context"
	"sync"
	"testing"
	"time"

	xCtx "github.com/bamboo-services/bamboo-base-go/defined/context"
)

// newCtxWithNodeList 创建一个携带 RegNodeKey 的测试上下文
func newCtxWithNodeList() context.Context {
	ctx := context.Background()
	list := xCtx.NewCtxNodeList()
	list.Append(xCtx.DatabaseKey, "mock_db")
	list.Append(xCtx.RedisClientKey, "mock_redis")
	return context.WithValue(ctx, xCtx.RegNodeKey, list)
}

func TestAsync_BasicExecution(t *testing.T) {
	var executed bool
	task := Async(context.Background(), func(ctx context.Context) {
		executed = true
	})

	Wait(task)
	if !executed {
		t.Error("异步任务未执行")
	}
}

func TestAsync_ContextDetached(t *testing.T) {
	parentCtx, parentCancel := context.WithCancel(context.Background())

	var taskDone bool
	task := Async(parentCtx, func(ctx context.Context) {
		time.Sleep(50 * time.Millisecond)
		taskDone = true
	})

	// 取消父上下文
	parentCancel()

	// 等待异步任务完成，应不受父上下文取消影响
	Wait(task)
	if !taskDone {
		t.Error("异步任务不应受父上下文取消影响")
	}
}

func TestAsync_ComponentAccess(t *testing.T) {
	parentCtx := newCtxWithNodeList()

	var gotDB, gotRedis any
	task := Async(parentCtx, func(ctx context.Context) {
		if nodeList, ok := ctx.Value(xCtx.RegNodeKey).(xCtx.ContextNodeList); ok {
			gotDB = nodeList.Get(xCtx.DatabaseKey)
			gotRedis = nodeList.Get(xCtx.RedisClientKey)
		}
	})

	Wait(task)
	if gotDB != "mock_db" {
		t.Errorf("期望 gotDB = 'mock_db'，实际为 %v", gotDB)
	}
	if gotRedis != "mock_redis" {
		t.Errorf("期望 gotRedis = 'mock_redis'，实际为 %v", gotRedis)
	}
}

func TestCancel(t *testing.T) {
	task := Async(context.Background(), func(ctx context.Context) {
		<-ctx.Done()
	})

	Cancel(task)

	select {
	case <-task.done:
		// 成功：任务因 cancel 而退出
	case <-time.After(time.Second):
		t.Error("Cancel 未能在 1 秒内终止异步任务")
	}
}

func TestWait(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	task := Async(context.Background(), func(ctx context.Context) {
		wg.Wait()
	})

	// Wait 应阻塞
	done := make(chan struct{})
	go func() {
		Wait(task)
		close(done)
	}()

	// 此时 Wait 应仍在阻塞
	select {
	case <-done:
		t.Error("Wait 不应在任务完成前返回")
	case <-time.After(50 * time.Millisecond):
		// 预期：Wait 仍在阻塞
	}

	wg.Done()

	select {
	case <-done:
		// 成功：Wait 在任务完成后返回
	case <-time.After(time.Second):
		t.Error("Wait 未能在任务完成后 1 秒内返回")
	}
}

func TestAsync_PanicRecovery(t *testing.T) {
	task := Async(context.Background(), func(ctx context.Context) {
		panic("test panic")
	})

	// Wait 应正常返回，不会因 Panic 而永久阻塞
	select {
	case <-task.done:
		// 成功：Panic 被恢复
	case <-time.After(time.Second):
		t.Error("Panic 后 Wait 未能在 1 秒内返回")
	}

	if !task.IsDone() {
		t.Error("Panic 后 IsDone 应返回 true")
	}
}
