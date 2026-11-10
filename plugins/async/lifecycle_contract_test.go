package async

import (
	"context"
	"errors"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

// 编译期契约断言：异步插件必须满足强类型生命周期契约。
var (
	_ application.Plugin = (*Plugin)(nil)
)

// TestPlugin_LifecycleContract_StartStopReturnsTrueNil 断言启动与停止成功时返回真正的 nil 指针。
func TestPlugin_LifecycleContract_StartStopReturnsTrueNil(t *testing.T) {
	_, factory := New(WithCapacity(4))
	p := factory()

	startErr := p.Start(context.Background())
	if startErr != nil {
		t.Fatalf("Start 应返回真正的 nil 指针，实际得到: %v", startErr)
	}
	if !apperr.IsNil(startErr) {
		t.Fatalf("Start 返回值应被 apperr.IsNil 判定为空，实际: %v", startErr)
	}

	stopErr := p.Stop(context.Background())
	if stopErr != nil {
		t.Fatalf("Stop 应返回真正的 nil 指针，实际得到: %v", stopErr)
	}
	if !apperr.IsNil(stopErr) {
		t.Fatalf("Stop 返回值应被 apperr.IsNil 判定为空，实际: %v", stopErr)
	}
}

// TestPlugin_LifecycleContract_InvalidCapacityKeepsSentinel 断言容量非法时失败结果保留容量哨兵根因。
//
// 说明：Async 插件当前尚无专属错误码，生命周期内部失败以框架内部错误码（50000）兜底。
func TestPlugin_LifecycleContract_InvalidCapacityKeepsSentinel(t *testing.T) {
	_, factory := New(WithCapacity(0))
	p, ok := factory().(*Plugin)
	if !ok {
		t.Fatal("工厂未返回 *async.Plugin")
	}

	// WithCapacity 会忽略非正容量，这里直接构造非法容量以覆盖容量分支
	p.cfg.capacity = 0

	startErr := p.Start(context.Background())
	if startErr == nil {
		t.Fatal("期望 Start 因容量非法而失败，实际得到 nil")
	}
	if !errors.Is(startErr, ErrInvalidPoolCapacity) {
		t.Fatalf("期望 ErrInvalidPoolCapacity 作为根因保留，实际: %v", startErr)
	}
	if startErr.Code.Code != apperr.FrameworkInternalError.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", startErr.Code.Code, apperr.FrameworkInternalError.Code)
	}
}

// TestPlugin_LifecycleContract_StopIsIdempotent 断言重复 Stop 共享同一结果且成功路径返回真正的 nil 指针。
func TestPlugin_LifecycleContract_StopIsIdempotent(t *testing.T) {
	_, factory := New(WithCapacity(4))
	p := factory()

	if startErr := p.Start(context.Background()); startErr != nil {
		t.Fatalf("Start 失败: %v", startErr)
	}
	for i := 0; i < 3; i++ {
		if stopErr := p.Stop(context.Background()); stopErr != nil {
			t.Fatalf("第 %d 次 Stop 应返回真正的 nil 指针，实际得到: %v", i+1, stopErr)
		}
	}
}
