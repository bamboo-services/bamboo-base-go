package database_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
	database "github.com/bamboo-services/bamboo-base-go/plugins/database/v2"
)

// 编译期契约断言：数据库插件必须满足强类型生命周期契约。
var (
	_ application.Plugin = (*database.Plugin)(nil)
)

// TestPlugin_LifecycleContract_StartSuccessReturnsTrueNil 断言启动成功时返回真正的 nil 指针。
//
// 契约要求成功必须返回真正的 nil 指针，禁止把 nil 指针装入 error 接口冒充成功。
func TestPlugin_LifecycleContract_StartSuccessReturnsTrueNil(t *testing.T) {
	adp, _ := newFakeAdapter("mock")
	_, factory := database.New(database.WithAdapter(adp))
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

// TestPlugin_LifecycleContract_StartFailureUsesDatabaseError 断言启动内部失败按所属域错误码归一并保留根因。
func TestPlugin_LifecycleContract_StartFailureUsesDatabaseError(t *testing.T) {
	adp, drv := newFakeAdapter("mock")
	pingErr := errors.New("ping boom")
	drv.mu.Lock()
	drv.pingErr = pingErr
	drv.mu.Unlock()

	_, factory := database.New(database.WithAdapter(adp))
	p := factory()

	startErr := p.Start(context.Background())
	if startErr == nil {
		t.Fatal("期望 Start 失败，实际得到 nil")
	}
	if startErr.Code.Code != database.DatabaseError.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", startErr.Code.Code, database.DatabaseError.Code)
	}
	if !errors.Is(startErr, pingErr) {
		t.Fatalf("期望底层 ping 错误作为根因保留，实际: %v", startErr)
	}
}

// TestPlugin_LifecycleContract_StartCancelledUsesControlCode 断言 ctx 取消经控制类错误码归一并保留原生哨兵。
func TestPlugin_LifecycleContract_StartCancelledUsesControlCode(t *testing.T) {
	adp, _ := newFakeAdapter("mock")
	_, factory := database.New(database.WithAdapter(adp))
	p := factory()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	startErr := p.Start(ctx)
	if startErr == nil {
		t.Fatal("期望 Start 因 ctx 取消而失败，实际得到 nil")
	}
	if startErr.Code.Code != control.Canceled.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", startErr.Code.Code, control.Canceled.Code)
	}
	if !errors.Is(startErr, context.Canceled) {
		t.Fatalf("期望 context.Canceled 作为根因保留，实际: %v", startErr)
	}
}
