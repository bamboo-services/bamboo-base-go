package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
)

// 编译期契约断言：缓存插件必须满足强类型生命周期契约。
var (
	_ application.Plugin = (*pluginImpl)(nil)
)

// TestPlugin_LifecycleContract_MemoryStartStopReturnsTrueNil 断言内存后端启动与停止成功时返回真正的 nil 指针。
func TestPlugin_LifecycleContract_MemoryStartStopReturnsTrueNil(t *testing.T) {
	_, factory := New(WithMemory())
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

// TestPlugin_LifecycleContract_ConfigFailureUsesCacheError 断言后端配置非法时按所属域错误码归一。
func TestPlugin_LifecycleContract_ConfigFailureUsesCacheError(t *testing.T) {
	_, factory := New()
	p := factory()

	startErr := p.Start(context.Background())
	if startErr == nil {
		t.Fatal("期望 Start 因未选择缓存后端而失败，实际得到 nil")
	}
	if startErr.Code.Code != CacheError.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", startErr.Code.Code, CacheError.Code)
	}
}

// TestPlugin_LifecycleContract_RedisUnreachableKeepsRootCause 断言 Redis 不可达时底层原生根因不丢失。
//
// go-redis 在拨号超时场景下向上抛出 context.DeadlineExceeded，
// 因此该失败经控制类错误码（50401）归一，同时底层 Redis 原生根因仍然保留。
func TestPlugin_LifecycleContract_RedisUnreachableKeepsRootCause(t *testing.T) {
	_, factory := New(WithRedis("127.0.0.1:54321", WithRedisDialTimeout(50*time.Millisecond)))
	p := factory()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	startErr := p.Start(ctx)
	if startErr == nil {
		t.Fatal("期望 Start 因 Redis 不可达而失败，实际得到 nil")
	}
	if startErr.Code.Code != control.DeadlineExceeded.Code && startErr.Code.Code != CacheError.Code {
		t.Fatalf("错误码不匹配: got %d, want %d 或 %d", startErr.Code.Code, control.DeadlineExceeded.Code, CacheError.Code)
	}
	if startErr.Err == nil {
		t.Fatal("期望底层 Redis 原生错误作为根因保留，实际根因为空")
	}
}

// TestPlugin_LifecycleContract_StartCancelledUsesControlCode 断言 ctx 取消经控制类错误码归一并保留原生哨兵。
func TestPlugin_LifecycleContract_StartCancelledUsesControlCode(t *testing.T) {
	_, factory := New(WithRedis("127.0.0.1:54321", WithRedisDialTimeout(50*time.Millisecond)))
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
