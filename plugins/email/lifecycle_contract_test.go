package email_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	email "github.com/bamboo-services/bamboo-base-go/plugins/email/v2"
)

// 编译期契约断言：邮件插件必须满足强类型生命周期契约。
var (
	_ application.Plugin = (*email.Plugin)(nil)
)

// TestPlugin_LifecycleContract_StartStopReturnsTrueNil 断言启动与停止成功时返回真正的 nil 指针。
func TestPlugin_LifecycleContract_StartStopReturnsTrueNil(t *testing.T) {
	_, fac := email.New(email.WithConfig(email.Config{
		Host:     "127.0.0.1",
		Port:     25,
		Username: "u",
		Password: "p",
		FromAddr: "f@a.com",
		TLSType:  "none",
	}))
	plugin := fac()

	startErr := plugin.Start(context.Background())
	if startErr != nil {
		t.Fatalf("Start 应返回真正的 nil 指针，实际得到: %v", startErr)
	}
	if !apperr.IsNil(startErr) {
		t.Fatalf("Start 返回值应被 apperr.IsNil 判定为空，实际: %v", startErr)
	}

	stopErr := plugin.Stop(context.Background())
	if stopErr != nil {
		t.Fatalf("Stop 应返回真正的 nil 指针，实际得到: %v", stopErr)
	}
	if !apperr.IsNil(stopErr) {
		t.Fatalf("Stop 返回值应被 apperr.IsNil 判定为空，实际: %v", stopErr)
	}
}

// TestPlugin_LifecycleContract_StartConfigFailureUsesEmailError 断言配置缺失时按所属域错误码归一并保留配置哨兵。
func TestPlugin_LifecycleContract_StartConfigFailureUsesEmailError(t *testing.T) {
	_, fac := email.New(email.WithConfig(email.Config{
		Host:     "127.0.0.1",
		Password: "p",
		FromAddr: "f@a.com",
	}))
	plugin := fac()

	startErr := plugin.Start(context.Background())
	if startErr == nil {
		t.Fatal("期望 Start 因缺少 SMTP 用户名而失败，实际得到 nil")
	}
	if startErr.Code.Code != email.EmailError.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", startErr.Code.Code, email.EmailError.Code)
	}
	if !errors.Is(startErr, email.ErrMissingUser) {
		t.Fatalf("期望 ErrMissingUser 作为根因保留，实际: %v", startErr)
	}
}

// TestPlugin_LifecycleContract_StopIsIdempotent 断言重复 Stop 均返回真正的 nil 指针。
func TestPlugin_LifecycleContract_StopIsIdempotent(t *testing.T) {
	_, fac := email.New(email.WithConfig(email.Config{
		Host:     "127.0.0.1",
		Port:     25,
		Username: "u",
		Password: "p",
		FromAddr: "f@a.com",
		TLSType:  "none",
	}))
	plugin := fac()

	if startErr := plugin.Start(context.Background()); startErr != nil {
		t.Fatalf("Start 失败: %v", startErr)
	}
	for i := 0; i < 3; i++ {
		if stopErr := plugin.Stop(context.Background()); stopErr != nil {
			t.Fatalf("第 %d 次 Stop 应返回真正的 nil 指针，实际得到: %v", i+1, stopErr)
		}
	}
}
