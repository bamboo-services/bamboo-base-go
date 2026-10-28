package async

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
)

// Task 表示一个已提交异步任务的句柄。
type Task struct {
	name   string
	ctx    context.Context
	cancel context.CancelFunc

	done      chan struct{}
	closeOnce sync.Once

	mu       sync.RWMutex
	panicked bool
	panicVal any
	panicStk []byte
}

func newTask(ctx context.Context, cancel context.CancelFunc, name string) *Task {
	return &Task{
		name:   name,
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
}

// Ctx 返回任务绑定的 Detach 独立上下文。
func (t *Task) Ctx() context.Context {
	return t.ctx
}

// Name 返回任务设置的人类可读名称。
func (t *Task) Name() string {
	return t.name
}

// IsDone 检查任务是否已经执行完成（无论是正常返回还是从 panic 恢复）。
func (t *Task) IsDone() bool {
	select {
	case <-t.done:
		return true
	default:
		return false
	}
}

// Done 返回等待任务完成的只读通道。
func (t *Task) Done() <-chan struct{} {
	return t.done
}

// Cancel 触发任务上下文取消信号。
func (t *Task) Cancel() {
	if t.cancel != nil {
		t.cancel()
	}
}

// Panicked 返回任务执行过程中是否捕获到 panic。
func (t *Task) Panicked() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.panicked
}

// PanicValue 返回捕获到的 panic 原始值（若未 panic 则返回 nil）。
func (t *Task) PanicValue() any {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.panicVal
}

// PanicStack 返回捕获 panic 时的调用栈。
func (t *Task) PanicStack() []byte {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.panicStk
}

// Wait 阻塞等待任务完成。若指定了超时时间且超时到期，返回 context.DeadlineExceeded，但绝不取消运行中的任务。
func (t *Task) Wait(ctx ...context.Context) error {
	if len(ctx) > 0 && ctx[0] != nil {
		select {
		case <-t.done:
			return nil
		case <-ctx[0].Done():
			return ctx[0].Err()
		}
	}
	<-t.done
	return nil
}

// WaitFor 等待任务完成，最多等待指定超时时间 timeout。
// 若超时，返回 context.DeadlineExceeded，任务仍在后台继续执行，不被强杀或取消。
func (t *Task) WaitFor(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return t.Wait(ctx)
}

func (t *Task) execute(fn func(context.Context), logger *logging.LogNamedLogger, debugEnabled bool) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			t.mu.Lock()
			t.panicked = true
			t.panicVal = r
			t.panicStk = stack
			t.mu.Unlock()

			if logger != nil {
				logger.SugarError(t.ctx, "async: 任务执行发生 panic",
					"task", t.name,
					"panic", fmt.Sprint(r),
					"stack", string(stack),
				)
			}
		}

		if debugEnabled && logger != nil {
			logger.SugarInfo(t.ctx, "async: 任务执行完成", "task", t.name)
		}

		if t.cancel != nil {
			t.cancel()
		}

		t.closeOnce.Do(func() {
			close(t.done)
		})
	}()

	if debugEnabled && logger != nil {
		logger.SugarInfo(t.ctx, "async: 任务开始执行", "task", t.name)
	}

	fn(t.ctx)
}
