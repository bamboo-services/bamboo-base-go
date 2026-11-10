package application

import (
	"sync"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

// StuckInstance 记录关机协调过程中无法确认完成清理或在途的实例残留快照。
//
// 当某个实例在关机总截止时仍未退出，或在 Stop 阶段发生 panic 时，
// 内核将其包装为 StuckInstance 记录在关机账本中，并作为 [StopIncompleteError]
// 的组成部分返回。
type StuckInstance struct {
	PluginKey    ctxkey.Key    // 发生残留的插件主键 [ctxkey.Key]。
	InstanceID   string        // 发生残留的实例唯一标识。
	Stage        string        // 残留发生时的生命周期阶段，如 "run"、"start"、"quiesce"、"stop" 或 "stop-panic"。
	RetainedDeps []ctxkey.Key  // 为保护该残留实例而特意保留、未执行 Stop 的底层依赖键列表。
	Err          *apperr.Error // 导致残留的原始结构化错误（如超时归因错误或捕获的 panic 统一错误）。
}

// StopIncompleteError 是应用平滑关机时存在残留实例或无法确认清理完成时返回的结构化错误。
//
// 此时应用状态将被置为 [StateIncomplete]，表明并非所有资源都已干净卸载。
// 错误内部包含全部残留实例的快照清单 [StopIncompleteError.StuckInstances]，供排障诊断。
type StopIncompleteError struct {
	StuckInstances []StuckInstance // 本轮关机中所有未确认停止的实例残留列表。
}

// Error 返回结构化关机残留错误的文本描述。
func (e *StopIncompleteError) Error() string {
	if e == nil || len(e.StuckInstances) == 0 {
		return "shutdown incomplete: instances retained"
	}
	return "shutdown incomplete: some instances could not be confirmed stopped and dependencies were retained"
}

// Unwrap 暴露全部残留成员的结构化错误，供 errors.Is 与 errors.As 逐个遍历。
//
// 每个成员的结构化错误都保留其原始错误码与根因链；
// 因此单个成员的失败原因不会因为被聚合进关机残留错误而丢失。
func (e *StopIncompleteError) Unwrap() []error {
	if e == nil {
		return nil
	}
	causes := make([]error, 0, len(e.StuckInstances))
	for _, stuck := range e.StuckInstances {
		if stuck.Err != nil {
			causes = append(causes, stuck.Err)
		}
	}
	return causes
}

// newStopIncompleteErr 构造关机未完成的统一错误。
//
// 诊断对象 [StopIncompleteError] 作为根因保留：
// 调用方既可以经 errors.As 命中诊断对象读取残留清单，
// 也可以经 errors.As / errors.Is 命中每个残留成员自身的结构化错误。
func newStopIncompleteErr(stuck []StuckInstance) *apperr.Error {
	diag := &StopIncompleteError{StuckInstances: stuck}
	return apperr.New(apperr.FrameworkShutdownIncomplete, "关机未完整完成，存在未确认停止的实例", diag)
}

// shutdownResult 存储单次协调的唯一关机结果
type shutdownResult struct {
	err *apperr.Error
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
