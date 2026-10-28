package application

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

func TestPluginTypesAndInterfaces(t *testing.T) {
	_ = StateNew
	_ = ModeWorker
	_ = OverlapSkip
	_ = OutcomeSuccess

	policy := OverlapPolicy{
		Kind:           OverlapAllow,
		MaxConcurrency: 5,
	}
	if policy.Kind != OverlapAllow || policy.MaxConcurrency != 5 {
		t.Fatalf("unexpected policy: %+v", policy)
	}

	info := PluginInfo{
		Key:     ctxkey.Key("test_plugin"),
		Name:    "Test Plugin",
		Mode:    ModeComponent,
		Overlap: OverlapPolicy{Kind: OverlapQueue, MaxQueue: 10},
	}
	if info.Key != "test_plugin" || info.Mode != ModeComponent {
		t.Fatalf("unexpected info: %+v", info)
	}

	opt := WithPayload("sample_payload")
	cfg := &activateConfig{}
	opt(cfg)
	if cfg.payload != "sample_payload" {
		t.Fatalf("expected payload %q, got %v", "sample_payload", cfg.payload)
	}
}

func TestErrorsAndSentinels(t *testing.T) {
	if !errors.Is(ErrParentClosing, context.Canceled) {
		t.Fatalf("expected ErrParentClosing to match context.Canceled")
	}
	if !errors.Is(ErrShuttingDown, context.Canceled) {
		t.Fatalf("expected ErrShuttingDown to match context.Canceled")
	}
	if !errors.Is(ErrParentClosing, ErrParentClosing) {
		t.Fatalf("expected ErrParentClosing to match itself")
	}

	rootErr := errors.New("underlying root error")
	hookErr := &HookPanicError{
		PluginKey:  ctxkey.Key("test_worker"),
		InstanceID: "inst_123",
		Stage:      "run",
		Value:      rootErr,
		Stack:      []byte("fake stack"),
	}

	if !errors.Is(hookErr, rootErr) {
		t.Fatalf("expected hookErr to unwrap to rootErr")
	}
	if hookErr.Error() == "" {
		t.Fatalf("expected non-empty error string")
	}

	// Value 非 error 时的 Unwrap
	nonErrHook := &HookPanicError{
		PluginKey: ctxkey.Key("test_tool"),
		Stage:     "start",
		Value:     "string panic",
	}
	if nonErrHook.Unwrap() != nil {
		t.Fatalf("expected nil unwrap for non-error panic value")
	}
}

func TestResolveConfig(t *testing.T) {
	// 清理测试环境变量，保留存在性并恢复
	envKeys := []string{
		"APP_START_TIMEOUT",
		"APP_GRACE_TIMEOUT",
		"APP_CLEANUP_TIMEOUT",
		"APP_SHUTDOWN_TIMEOUT",
	}
	type envBackup struct {
		val     string
		existed bool
	}
	origEnvs := make(map[string]envBackup)
	for _, k := range envKeys {
		val, existed := os.LookupEnv(k)
		origEnvs[k] = envBackup{val: val, existed: existed}
	}
	restoreEnvs := func() {
		for _, k := range envKeys {
			b := origEnvs[k]
			if b.existed {
				_ = os.Setenv(k, b.val)
			} else {
				_ = os.Unsetenv(k)
			}
		}
	}
	t.Cleanup(restoreEnvs)

	clearEnvs := func() {
		for _, k := range envKeys {
			_ = os.Unsetenv(k)
		}
	}
	clearEnvs()

	// 1. 默认解析
	cfg, err := resolveConfig()
	if err != nil {
		t.Fatalf("default resolveConfig failed: %v", err)
	}
	if cfg.StartTimeout != 15*time.Second || cfg.GraceTimeout != 20*time.Second ||
		cfg.CleanupTimeout != 10*time.Second || cfg.ShutdownTimeout != 60*time.Second {
		t.Fatalf("unexpected default config: %+v", cfg)
	}

	// 2. 环境变量解析
	_ = os.Setenv("APP_START_TIMEOUT", "5s")
	_ = os.Setenv("APP_GRACE_TIMEOUT", "12s")
	_ = os.Setenv("APP_CLEANUP_TIMEOUT", "8s")
	_ = os.Setenv("APP_SHUTDOWN_TIMEOUT", "30s")

	cfgEnv, err := resolveConfig()
	if err != nil {
		t.Fatalf("env resolveConfig failed: %v", err)
	}
	if cfgEnv.StartTimeout != 5*time.Second || cfgEnv.GraceTimeout != 12*time.Second ||
		cfgEnv.CleanupTimeout != 8*time.Second || cfgEnv.ShutdownTimeout != 30*time.Second {
		t.Fatalf("unexpected env config: %+v", cfgEnv)
	}

	// 3. Option 覆盖环境变量
	cfgOpt, err := resolveConfig(
		WithStartTimeout(3*time.Second),
		WithGraceTimeout(10*time.Second),
	)
	if err != nil {
		t.Fatalf("option resolveConfig failed: %v", err)
	}
	if cfgOpt.StartTimeout != 3*time.Second || cfgOpt.GraceTimeout != 10*time.Second ||
		cfgOpt.CleanupTimeout != 8*time.Second || cfgOpt.ShutdownTimeout != 30*time.Second {
		t.Fatalf("unexpected option override config: %+v", cfgOpt)
	}

	// 4. Option 显式提供 0 值或负数被拒绝
	_, err = resolveConfig(WithStartTimeout(0))
	if err == nil {
		t.Fatalf("expected error for explicit zero timeout")
	}

	_, err = resolveConfig(WithGraceTimeout(-1 * time.Second))
	if err == nil {
		t.Fatalf("expected error for negative timeout")
	}

	// 5. 环境变量无效，但被明确 Option 屏蔽（未指定字段才读环境变量）
	_ = os.Setenv("APP_START_TIMEOUT", "invalid_duration")
	// 此时若不传 WithStartTimeout，应报错
	_, err = resolveConfig()
	if err == nil {
		t.Fatalf("expected error for invalid APP_START_TIMEOUT")
	}
	// 传 WithStartTimeout 显式覆盖，APP_START_TIMEOUT 不应被读取
	cfgShielded, err := resolveConfig(WithStartTimeout(2 * time.Second))
	if err != nil {
		t.Fatalf("expected Option to shield invalid env: %v", err)
	}
	if cfgShielded.StartTimeout != 2*time.Second {
		t.Fatalf("expected 2s, got %v", cfgShielded.StartTimeout)
	}

	// 6. 环境变量设置为空被拒绝
	_ = os.Setenv("APP_START_TIMEOUT", "")
	_, err = resolveConfig()
	if err == nil {
		t.Fatalf("expected error for empty env")
	}

	// 7. 裸数字被拒绝
	_ = os.Setenv("APP_START_TIMEOUT", "100")
	_, err = resolveConfig()
	if err == nil {
		t.Fatalf("expected error for bare number in env")
	}

	// 8. 非正数在环境变量中被拒绝
	_ = os.Setenv("APP_START_TIMEOUT", "-5s")
	_, err = resolveConfig()
	if err == nil {
		t.Fatalf("expected error for negative duration in env")
	}

	// 9. Grace >= Shutdown 预算冲突拒绝
	clearEnvs()
	_, err = resolveConfig(
		WithGraceTimeout(60*time.Second),
		WithShutdownTimeout(60*time.Second),
	)
	if err == nil {
		t.Fatalf("expected error when Grace >= Shutdown")
	}

	// 10. 重复独立解析（模拟多次独立 New）
	cfg1, err1 := resolveConfig(WithStartTimeout(1 * time.Second))
	cfg2, err2 := resolveConfig(WithStartTimeout(2 * time.Second))
	if err1 != nil || err2 != nil {
		t.Fatalf("independent resolveConfig failed: %v, %v", err1, err2)
	}
	if cfg1.StartTimeout != 1*time.Second || cfg2.StartTimeout != 2*time.Second {
		t.Fatalf("configs should be independent: cfg1=%v, cfg2=%v", cfg1.StartTimeout, cfg2.StartTimeout)
	}
}
