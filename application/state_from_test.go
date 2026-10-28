package application

import (
	"context"
	"testing"
)

func TestStateFrom(t *testing.T) {
	// 1. nil 或空 Context 测试
	for _, tc := range []struct {
		name string
		ctx  context.Context
	}{
		{name: "nil context", ctx: nil},
		{name: "empty context", ctx: context.Background()},
	} {
		if state, ok := StateFrom(tc.ctx); ok || state != 0 {
			t.Fatalf("expected (0, false) on %s, got (%v, %v)", tc.name, state, ok)
		}
	}

	// 2. 伪造或兼容单元测试的 newSharedContainer (无 stateReader)
	unboundContainer := newSharedContainer("orphan-app")
	orphanCtx := context.WithValue(context.Background(), containerCtxKey, unboundContainer)
	if state, ok := StateFrom(orphanCtx); ok || state != 0 {
		t.Fatalf("expected (0, false) on unbound container, got (%v, %v)", state, ok)
	}

	// 3. 正常 App 状态查询与生命周期变化测试
	app := New()
	appCtx := context.WithValue(context.Background(), containerCtxKey, app.container)

	if state, ok := StateFrom(appCtx); !ok || state != StateNew {
		t.Fatalf("expected (StateNew, true), got (%v, %v)", state, ok)
	}

	// 衍生上下文 (Detach / BridgeRequest) 保持只读状态可查
	detachedCtx, cancel := Detach(appCtx)
	defer cancel()
	if state, ok := StateFrom(detachedCtx); !ok || state != StateNew {
		t.Fatalf("expected StateNew from detachedCtx, got (%v, %v)", state, ok)
	}

	type traceKey string
	const traceCtxKey traceKey = "trace"
	reqCtx := context.WithValue(context.Background(), traceCtxKey, "test")
	bridgedCtx := BridgeRequest(reqCtx, appCtx, "req-1")
	if state, ok := StateFrom(bridgedCtx); !ok || state != StateNew {
		t.Fatalf("expected StateNew from bridgedCtx, got (%v, %v)", state, ok)
	}

	// 状态变迁测试 (并发锁安全读取)
	app.stateMu.Lock()
	app.state = StateStarting
	app.stateMu.Unlock()
	if state, ok := StateFrom(appCtx); !ok || state != StateStarting {
		t.Fatalf("expected StateStarting, got (%v, %v)", state, ok)
	}

	app.stateMu.Lock()
	app.state = StateRunning
	app.stateMu.Unlock()
	if state, ok := StateFrom(appCtx); !ok || state != StateRunning {
		t.Fatalf("expected StateRunning, got (%v, %v)", state, ok)
	}

	app.stateMu.Lock()
	app.state = StateStopping
	app.stateMu.Unlock()
	if state, ok := StateFrom(appCtx); !ok || state != StateStopping {
		t.Fatalf("expected StateStopping, got (%v, %v)", state, ok)
	}

	app.stateMu.Lock()
	app.state = StateStopped
	app.stateMu.Unlock()
	if state, ok := StateFrom(appCtx); !ok || state != StateStopped {
		t.Fatalf("expected StateStopped, got (%v, %v)", state, ok)
	}

	// 4. 多 App 互不串扰
	app2 := New()
	appCtx2 := context.WithValue(context.Background(), containerCtxKey, app2.container)

	if state, ok := StateFrom(appCtx2); !ok || state != StateNew {
		t.Fatalf("expected StateNew for app2, got (%v, %v)", state, ok)
	}
	if state, ok := StateFrom(appCtx); !ok || state != StateStopped {
		t.Fatalf("expected StateStopped for app1, got (%v, %v)", state, ok)
	}
}
