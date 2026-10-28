package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

// 0001 / 0002 明确命名的错误哨兵。
var (
	// 生命周期与注册
	ErrAlreadyRun           = errors.New("app has already run")
	ErrAppClosed            = errors.New("app is closed")
	ErrRegistrationClosed   = errors.New("app registration closed")
	ErrStartupInterrupted   = errors.New("app startup interrupted")
	ErrWorkerUnexpectedExit = errors.New("worker unexpected exit")

	// 工厂与插件
	ErrNilFactory = errors.New("plugin factory cannot be nil")
	ErrNilPlugin  = errors.New("plugin instance cannot be nil")

	// 激活与队列
	ErrInvalidActivationParent  = errors.New("invalid activation parent context")
	ErrComponentSkipped         = errors.New("component activation skipped due to overlap policy")
	ErrConcurrencyLimitExceeded = errors.New("component concurrency limit exceeded")
	ErrQueueFull                = errors.New("component queue is full")
	ErrNestedQueueWouldBlock    = errors.New("nested queue activation would block")
	ErrActivationCycle          = errors.New("activation dependency cycle detected")

	// 依赖与共享容器 (0002)
	ErrContainerNotMounted    = errors.New("shared container not mounted in context")
	ErrUndeclaredDependency   = errors.New("undeclared dependency access")
	ErrComponentStopped       = errors.New("component stopped and retired")
	ErrComponentTypeMismatch  = errors.New("component type mismatch")
	ErrComponentNotFound      = errors.New("component not found")
	ErrComponentUnavailable   = errors.New("component provider unavailable")
	ErrDuplicateExposedKey    = errors.New("duplicate exposed key")
	ErrExposedKeysMismatch    = errors.New("exposed keys mismatch")
	ErrInvalidDependencyScope = errors.New("invalid dependency scope")
)

// ErrParentClosing 与 ErrShuttingDown 特殊哨兵：支持 errors.Is(err, context.Canceled)。
var (
	ErrParentClosing error = &canceledSentinelError{msg: "parent component closing"}
	ErrShuttingDown  error = &canceledSentinelError{msg: "application shutting down"}
)

type canceledSentinelError struct {
	msg string
}

func (e *canceledSentinelError) Error() string {
	return e.msg
}

func (e *canceledSentinelError) Is(target error) bool {
	return target == context.Canceled || target == e
}

// HookPanicError 保留内核调用边界上的 panic，堆栈仅供内部诊断。
type HookPanicError struct {
	PluginKey  ctxkey.Key
	InstanceID string // 尚未分配实例时允许为空
	Stage      string // factory/start/expose/run/quiesce/stop
	Value      any
	Stack      []byte
}

func (e *HookPanicError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.InstanceID != "" {
		return fmt.Sprintf("hook panic on plugin %q (instance %q) at stage %q: %v", e.PluginKey, e.InstanceID, e.Stage, e.Value)
	}
	return fmt.Sprintf("hook panic on plugin %q at stage %q: %v", e.PluginKey, e.Stage, e.Value)
}

// Unwrap 当 Value 为 error 时返回该错误，否则 nil。
func (e *HookPanicError) Unwrap() error {
	if e == nil {
		return nil
	}
	if err, ok := e.Value.(error); ok {
		return err
	}
	return nil
}
