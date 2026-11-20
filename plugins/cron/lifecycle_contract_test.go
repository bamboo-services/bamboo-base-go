package cron_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	cron "github.com/bamboo-services/bamboo-base-go/plugins/cron/v2"
)

// 编译期契约断言：Cron 插件必须满足强类型生命周期契约。
var (
	_ application.RunnablePlugin = (*cron.Plugin)(nil)
	_ application.Quiescer       = (*cron.Plugin)(nil)
)

// contractActivator 是满足 [application.Activator] 的最小测试替身。
type contractActivator struct{}

func (contractActivator) Activate(context.Context, ctxkey.Key, ...application.ActivateOption) (application.RunHandle, *apperr.Error) {
	return nil, nil
}

// TestPlugin_LifecycleContract_StartWithoutActivatorFails 断言缺少激活器时启动失败且哨兵根因保留。
func TestPlugin_LifecycleContract_StartWithoutActivatorFails(t *testing.T) {
	_, factory := cron.New(cron.WithCronJob("0 * * * *", "target_comp"))
	p := factory()

	startErr := p.Start(context.Background())
	if startErr == nil {
		t.Fatal("期望 Start 因缺少激活器而失败，实际得到 nil")
	}
	if !errors.Is(startErr, cron.ErrNoActivator) {
		t.Fatalf("期望 ErrNoActivator 作为根因保留，实际: %v", startErr)
	}
	if startErr.Code.Code != cron.SchedulerError.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", startErr.Code.Code, cron.SchedulerError.Code)
	}
}

// TestPlugin_LifecycleContract_LifecycleReturnsTrueNil 断言成功路径返回真正的 nil 指针。
func TestPlugin_LifecycleContract_LifecycleReturnsTrueNil(t *testing.T) {
	_, factory := cron.New(
		cron.WithActivator(contractActivator{}),
		cron.WithIntervalJob(50*time.Millisecond, "target_comp"),
	)
	p, ok := factory().(*cron.Plugin)
	if !ok {
		t.Fatal("工厂未返回 *cron.Plugin")
	}

	startErr := p.Start(context.Background())
	if startErr != nil {
		t.Fatalf("Start 应返回真正的 nil 指针，实际得到: %v", startErr)
	}
	if !apperr.IsNil(startErr) {
		t.Fatalf("Start 返回值应被 apperr.IsNil 判定为空，实际: %v", startErr)
	}

	quiesceErr := p.Quiesce(context.Background())
	if quiesceErr != nil {
		t.Fatalf("Quiesce 应返回真正的 nil 指针，实际得到: %v", quiesceErr)
	}

	stopErr := p.Stop(context.Background())
	if stopErr != nil {
		t.Fatalf("Stop 应返回真正的 nil 指针，实际得到: %v", stopErr)
	}
	if !apperr.IsNil(stopErr) {
		t.Fatalf("Stop 返回值应被 apperr.IsNil 判定为空，实际: %v", stopErr)
	}
}

// TestPlugin_LifecycleContract_StopWithCancelledContextUsesControlCode 断言关机 ctx 取消按控制类错误码归一并保留原生哨兵。
func TestPlugin_LifecycleContract_StopWithCancelledContextUsesControlCode(t *testing.T) {
	_, factory := cron.New(
		cron.WithActivator(contractActivator{}),
		cron.WithIntervalJob(50*time.Millisecond, "target_comp"),
	)
	p := factory()

	if startErr := p.Start(context.Background()); startErr != nil {
		t.Fatalf("Start 失败: %v", startErr)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stopErr := p.Stop(ctx)
	if stopErr == nil {
		t.Fatal("期望 Stop 因 ctx 取消而返回失败结果，实际得到 nil")
	}
	if stopErr.Code.Code != 49900 {
		t.Fatalf("错误码不匹配: got %d, want %d", stopErr.Code.Code, 49900)
	}
	if !errors.Is(stopErr, context.Canceled) {
		t.Fatalf("期望 context.Canceled 作为根因保留，实际: %v", stopErr)
	}
}
