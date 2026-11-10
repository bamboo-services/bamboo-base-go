package application

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

type runHandleImpl struct {
	instanceID string
	pluginKey  ctxkey.Key
	done       chan struct{}
	once       sync.Once

	resultMu sync.RWMutex
	result   RunResult
	isDone   atomic.Bool
}

func newRunHandle(instanceID string, pluginKey ctxkey.Key) *runHandleImpl {
	return &runHandleImpl{
		instanceID: instanceID,
		pluginKey:  pluginKey,
		done:       make(chan struct{}),
	}
}

func (h *runHandleImpl) InstanceID() string {
	return h.instanceID
}

func (h *runHandleImpl) Wait(ctx context.Context) (RunResult, *apperr.Error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-h.done:
		h.resultMu.RLock()
		res := h.result
		h.resultMu.RUnlock()
		return res, res.computeWaitError()
	case <-ctx.Done():
		return RunResult{}, normalizeErr(apperr.FrameworkInternalError, "等待组件执行结果被中断", ctx.Err())
	}
}

func (h *runHandleImpl) publishResult(res RunResult) {
	h.once.Do(func() {
		h.resultMu.Lock()
		h.result = res
		h.resultMu.Unlock()
		h.isDone.Store(true)
		close(h.done)
	})
}

// computeWaitError 计算按照契约表交付给 Wait 的错误
//
// 工厂失败与未派发取消各自独占返回；多阶段失败按阶段顺序聚合，
// 全部阶段错误均保留在错误链中。
func (r *RunResult) computeWaitError() *apperr.Error {
	if r.FactoryErr != nil {
		return r.FactoryErr
	}
	if r.CancelErr != nil && !r.Dispatched {
		return r.CancelErr
	}

	var errs []*apperr.Error
	if r.StartErr != nil {
		errs = append(errs, r.StartErr)
	}
	if r.ExposeErr != nil {
		errs = append(errs, r.ExposeErr)
	}
	if r.QuiesceErr != nil {
		errs = append(errs, r.QuiesceErr)
	}
	if r.RunErr != nil {
		errs = append(errs, r.RunErr)
	}
	if r.StopErr != nil {
		errs = append(errs, r.StopErr)
	}

	return aggregateErrs(errs...)
}

type supervisor struct {
	app *App
}

func newSupervisor(app *App) *supervisor {
	return &supervisor{app: app}
}
