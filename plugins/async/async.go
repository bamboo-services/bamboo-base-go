package xAsync

import (
	"context"
	"runtime/debug"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
)

// Async 从父上下文中提取组件引用，创建独立的上下文后在新的 goroutine 中异步执行 fn。
//
// 异步任务的上下文不受父上下文取消的影响，可以通过 xCtxUtil 系列函数访问 DB、Redis 等组件。
// 返回的 *Task 可通过 Cancel 强制终止或 Wait 等待完成。
func Async(parentCtx context.Context, fn func(ctx context.Context)) *Task {
	ctx, cancel := detachContext(parentCtx)
	task := &Task{
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				xLog.SugarError(ctx, "async task panicked",
					"error", r,
					"stack", string(debug.Stack()),
				)
			}
			close(task.done)
		}()
		fn(ctx)
	}()

	return task
}

// Cancel 强制终止异步任务，调用后异步任务的 ctx.Done() 将被触发。
//
// Cancel 不会阻塞等待任务退出，仅发送取消信号。
func Cancel(task *Task) {
	if task == nil {
		return
	}
	task.cancel()
}

// Wait 阻塞等待异步任务执行完成（正常结束或 Panic 恢复后均会返回）。
func Wait(task *Task) {
	if task == nil {
		return
	}
	<-task.done
}
