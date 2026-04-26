package xAsync

import (
	"context"
	"runtime/debug"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/gin-gonic/gin"
)

// Async 从父上下文中提取组件引用，创建独立的上下文后在新的 goroutine 中异步执行 fn。
//
// 异步任务的上下文不受父上下文取消的影响，可以通过 xCtxUtil 系列函数访问 DB、Redis 等组件。
// 返回的 *Task 可通过 Cancel 强制终止或 Wait 等待完成。
//
// 不允许传入 *gin.Context，请使用 c.Request.Context() 获取标准 context.Context。
//
// 可选配置:
//   - WithName(name): 设置异步任务名称，名称会显示在日志中
//   - WithDebug(): 启用调试日志（输出开始执行、执行完成）
//   - WithLogger(logger): 设置自定义日志器
func Async(parentCtx context.Context, fn func(ctx context.Context), options ...Option) *Task {
	if _, ok := parentCtx.(*gin.Context); ok {
		panic("async: 不允许传入 *gin.Context，请使用 c.Request.Context() 获取标准 context.Context")
	}

	config := defaultConfig()
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}

	ctx, cancel := detachContext(parentCtx)
	task := &Task{
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	log := resolveLogger(config)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.SugarError(ctx, "async task panicked",
					"error", r,
					"stack", string(debug.Stack()),
				)
			}
			if config.Debug {
				log.SugarInfo(ctx, "异步任务执行完成")
			}
			close(task.done)
		}()

		if config.Debug {
			log.SugarInfo(ctx, "异步任务开始执行")
		}
		fn(ctx)
	}()

	return task
}

// resolveLogger 根据配置解析日志器，优先使用自定义日志器，否则根据名称创建默认日志器。
func resolveLogger(config Config) *xLog.LogNamedLogger {
	if config.Logger != nil {
		return config.Logger
	}
	if config.Name != "" {
		return xLog.WithName(xLog.NamedTASK, config.Name)
	}
	return xLog.WithName(xLog.NamedTASK)
}
