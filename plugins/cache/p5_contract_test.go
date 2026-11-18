package cache

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/valueerr"
)

// 编译期断言：配置校验错误与插件初始化错误均为强类型错误指针。
var (
	_ *apperr.Error = Config{}.validationErr
	_ *apperr.Error = pluginImpl{}.initErr
)

// TestP5ApplyFromEnvKeepsValueCause 断言配置项解析失败归一为 CacheError（50016），
// 且底层 VALUE_* 原因作为根因完整保留。
func TestP5ApplyFromEnvKeepsValueCause(t *testing.T) {
	t.Setenv(string(env.NoSQLDriver), "redis")
	t.Setenv(string(env.NoSQLDatabase), "not-a-number")
	t.Setenv(string(env.NoSQLPoolSize), "also-bad")

	var cfg Config
	FromEnv()(&cfg)

	if cfg.validationErr == nil {
		t.Fatal("期望配置校验失败记录错误，实际为 nil")
	}
	if cfg.validationErr.Code.Code != CacheError.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", cfg.validationErr.Code.Code, CacheError.Code)
	}

	// 聚合后的错误下挂各条配置错误，每条配置错误下挂底层 VALUE_* 原因。
	var first *apperr.Error
	if !errors.As(cfg.validationErr.Err, &first) {
		t.Fatalf("期望保留内层配置错误原因，实际: %T", cfg.validationErr.Err)
	}
	if first.Code.Code != CacheError.Code {
		t.Fatalf("内层配置错误码不匹配: got %d, want %d", first.Code.Code, CacheError.Code)
	}
	var inner *apperr.Error
	if !errors.As(first.Err, &inner) {
		t.Fatalf("期望保留底层 VALUE_* 原因，实际: %T", first.Err)
	}
	if inner.Code.Code != valueerr.InvalidFormat.Code {
		t.Fatalf("底层原因码不匹配: got %d, want %d", inner.Code.Code, valueerr.InvalidFormat.Code)
	}
	if inner.Err == nil {
		t.Fatal("期望保留原生解析根因，实际根因为空")
	}
}

// TestP5ApplyFromEnvDoesNotLeakSecret 断言配置失败的安全描述与 JSON 均不泄漏凭据。
func TestP5ApplyFromEnvDoesNotLeakSecret(t *testing.T) {
	const secret = "super-secret-redis-password"
	t.Setenv(string(env.NoSQLDriver), "redis")
	t.Setenv(string(env.NoSQLPass), secret)
	t.Setenv(string(env.NoSQLDatabase), "not-a-number")

	var cfg Config
	FromEnv()(&cfg)

	if cfg.validationErr == nil {
		t.Fatal("期望配置校验失败记录错误，实际为 nil")
	}
	if strings.Contains(cfg.validationErr.Error(), secret) {
		t.Fatalf("错误文本泄漏了 Redis 密码: %s", cfg.validationErr.Error())
	}
	payload, err := json.Marshal(cfg.validationErr)
	if err != nil {
		t.Fatalf("序列化错误失败: %v", err)
	}
	if strings.Contains(string(payload), secret) {
		t.Fatalf("错误 JSON 泄漏了 Redis 密码: %s", payload)
	}
	if !strings.Contains(string(cfg.validationErr.ErrorMessage), string(env.NoSQLDatabase)) {
		t.Fatalf("期望安全描述包含配置项名，实际: %s", cfg.validationErr.ErrorMessage)
	}
}

// TestP5PluginStartReportsConfigFailureAsCacheError 断言 Start 阶段的配置失败按所属域错误码返回。
func TestP5PluginStartReportsConfigFailureAsCacheError(t *testing.T) {
	t.Setenv(string(env.NoSQLDriver), "memory")
	t.Setenv(string(env.NoSQLMemoryMaxEntries), "not-a-number")

	_, factory := New(FromEnv())
	p := factory()

	startErr := p.Start(context.Background())
	if startErr == nil {
		t.Fatal("期望 Start 因配置非法而失败，实际为 nil")
	}
	if startErr.Code.Code != CacheError.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", startErr.Code.Code, CacheError.Code)
	}
	if startErr.Err == nil {
		t.Fatal("期望保留配置错误根因，实际根因为空")
	}
}

// TestP5PluginStartWithoutBackendUsesCacheError 断言未选择后端时按 CacheError 归一。
func TestP5PluginStartWithoutBackendUsesCacheError(t *testing.T) {
	_, factory := New()
	p := factory()

	startErr := p.Start(context.Background())
	if startErr == nil {
		t.Fatal("期望 Start 因未选择后端而失败，实际为 nil")
	}
	if startErr.Code.Code != CacheError.Code {
		t.Fatalf("错误码不匹配: got %d, want %d", startErr.Code.Code, CacheError.Code)
	}
	if startErr.Err == nil {
		t.Fatal("期望保留原始原因，实际根因为空")
	}
	if startErr.Err == nil {
		t.Fatal("期望保留原始原因，实际根因为空")
	}
}

// TestP5CacheErrorCodeUnchanged 断言插件错误码未被迁移改动。
func TestP5CacheErrorCodeUnchanged(t *testing.T) {
	if CacheError.Code != 50016 || CacheError.Output != "CACHE_ERROR" {
		t.Fatalf("插件错误码被改动: %+v", CacheError)
	}
}
