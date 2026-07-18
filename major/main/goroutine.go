package xMain

import (
	"context"
	"sync"
)

// initGoroutine 启动附加后台协程并接入优雅关闭流程。
//
// 对每个非 nil 的 goroutineFunc，创建两个协程：
//   - 执行协程：调用业务函数，完成后关闭 funcDone 通道并 Done WaitGroup
//   - 监听协程：select shutdownNotify（收到信号强制 Done）与 funcDone（正常完成）
//
// WaitGroup 计数在执行协程启动前 Add，确保 Runner 的 Wait 不会提前返回。
// shutdownNotify 的优先级与 funcDone 等价，保证信号到达时能主动释放计数。
//
// 注意：goroutineFunc 的 extra 参数为预留扩展点，当前不传递任何值。
func (runner *mainRunner) initGoroutine(goroutineFunc ...func(ctx context.Context, extra ...any)) {
	for _, goroutineExec := range goroutineFunc {
		if goroutineExec == nil {
			continue
		}

		runner.sync.engineSync.Add(1)
		doneOnce := sync.Once{}
		doneFunc := func() {
			doneOnce.Do(runner.sync.engineSync.Done)
		}
		funcDone := make(chan struct{})

		go func(execFunc func(context.Context, ...any), ctx context.Context, done chan<- struct{}, finish func()) {
			defer close(done)
			defer finish()
			execFunc(ctx)
		}(goroutineExec, runner.runCtx, funcDone, doneFunc)

		go func(done <-chan struct{}, finish func()) {
			select {
			case <-runner.sync.shutdownNotify:
				finish()
			case <-done:
			}
		}(funcDone, doneFunc)
	}
}
