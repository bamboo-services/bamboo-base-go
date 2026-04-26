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
