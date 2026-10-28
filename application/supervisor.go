package application

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

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

func (h *runHandleImpl) Wait(ctx context.Context) (RunResult, error) {
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
		return RunResult{}, ctx.Err()
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
func (r *RunResult) computeWaitError() error {
	if r.FactoryErr != nil {
		return r.FactoryErr
	}
	if r.CancelErr != nil && !r.Dispatched {
		return r.CancelErr
	}

	var errs []error
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

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

type supervisor struct {
	app *App
}

func newSupervisor(app *App) *supervisor {
	return &supervisor{app: app}
}
