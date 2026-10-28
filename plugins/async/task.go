package async

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
)

// Task 表示一个已提交异步任务的执行句柄。
// 提供状态查询、并发安全的同步等待、Panic 观测及协同取消能力。
// [Task] 的所有公开方法均为并发安全。
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

// Ctx 返回任务绑定的独立执行上下文。
// 该上下文已从父请求脱钩超时与取消，但保留了容器依赖与追踪元数据。
// 任务函数正常结束、panic 或被 [Task.Cancel] 显式取消后，该上下文的 Done 信号将被触发。
func (t *Task) Ctx() context.Context {
	return t.ctx
}

// Name 返回任务设置的人类可读名称。
// 未显式指定时返回默认名称 "unnamed_task"。
func (t *Task) Name() string {
	return t.name
}

// IsDone 检查任务是否已经执行完成（包含正常返回与 panic 恢复两种结束情形）。
// 本方法非阻塞，并发安全。
func (t *Task) IsDone() bool {
	select {
	case <-t.done:
		return true
	default:
		return false
	}
}

// Done 返回等待任务完成的只读通知通道。
// 当任务函数退出或 panic 恢复后，该通道被关闭。本方法并发安全。
func (t *Task) Done() <-chan struct{} {
	return t.done
}

// Cancel 触发任务上下文的取消信号。
// 正在运行的任务可通过读取 [Task.Ctx] 的 Done 信号感知并实施协同退出。
// 本方法并发安全且多次调用幂等。
func (t *Task) Cancel() {
	if t.cancel != nil {
		t.cancel()
	}
}

// Panicked 返回任务执行过程中是否捕获到未处理的 panic。
// 任务未完成或正常结束时返回 false。本方法并发安全。
func (t *Task) Panicked() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.panicked
}

// PanicValue 返回捕获到的 panic 原始值。
// 若任务未发生 panic 或尚未执行完成，返回 nil。本方法并发安全。
func (t *Task) PanicValue() any {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.panicVal
}

// PanicStack 返回捕获 panic 时的调用栈字节切片。
// 若任务未发生 panic，返回 nil。本方法并发安全。
func (t *Task) PanicStack() []byte {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.panicStk
}

// Wait 阻塞等待任务完成。
// 若传入了可选的 ctx 且在等待期间该上下文超时或被取消，立即返回对应错误（如 [context.DeadlineExceeded]）；
// 超时仅代表调用方放弃继续等待，绝不会中断或取消后台正在运行的任务。
// 若未传入 ctx 或传入 nil，则无期限阻塞直至任务结束。本方法并发安全。
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

// WaitFor 等待任务完成，最多等待指定的超时时长 timeout。
// 若等待超时，返回 [context.DeadlineExceeded]；任务仍在后台继续执行，不被取消或强杀。
// 本方法并发安全。
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
