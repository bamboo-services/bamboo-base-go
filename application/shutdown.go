package application

import (
	"sync"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

// StuckInstance 记录无法确认完成清理或在途的实例残留快照。
type StuckInstance struct {
	PluginKey    ctxkey.Key
	InstanceID   string
	Stage        string // run / start / quiesce / stop / stop-panic 等
	RetainedDeps []ctxkey.Key
	Err          error
}

// StopIncompleteError 是 Shutdown 发生残留或无法确认清理时返回的结构化错误。
type StopIncompleteError struct {
	StuckInstances []StuckInstance
}

func (e *StopIncompleteError) Error() string {
	if e == nil || len(e.StuckInstances) == 0 {
		return "shutdown incomplete: instances retained"
	}
	return "shutdown incomplete: some instances could not be confirmed stopped and dependencies were retained"
}

// shutdownResult 存储单次协调的唯一关机结果
type shutdownResult struct {
	err error
}

type shutdownCoordinator struct {
	app *App

	mu      sync.Mutex
	started bool
	done    chan struct{}
	res     shutdownResult

	// 追踪残留实例
	stuckMu        sync.Mutex
	stuckInstances []StuckInstance
}

func newShutdownCoordinator(app *App) *shutdownCoordinator {
	return &shutdownCoordinator{
		app:  app,
		done: make(chan struct{}),
	}
}

// recordStuck 记录残留实例
func (c *shutdownCoordinator) recordStuck(stuck StuckInstance) {
	c.stuckMu.Lock()
	defer c.stuckMu.Unlock()
	c.stuckInstances = append(c.stuckInstances, stuck)
}

func (c *shutdownCoordinator) getStuckInstances() []StuckInstance {
	c.stuckMu.Lock()
	defer c.stuckMu.Unlock()
	if len(c.stuckInstances) == 0 {
		return nil
	}
	copied := make([]StuckInstance, len(c.stuckInstances))
	copy(copied, c.stuckInstances)
	return copied
}
