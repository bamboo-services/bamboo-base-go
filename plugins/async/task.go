package xAsync

import "context"

// Task 代表一个异步任务的句柄，支持强制终止和等待完成两种关闭策略。
type Task struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// Ctx 返回任务的独立上下文，可用于检查取消状态或传递给下游函数。
func (t *Task) Ctx() context.Context {
	return t.ctx
}

// IsDone 检查任务是否已完成（正常结束或 Panic 恢复后均返回 true）。
func (t *Task) IsDone() bool {
	select {
	case <-t.done:
		return true
	default:
		return false
	}
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
