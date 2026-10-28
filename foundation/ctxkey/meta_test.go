package ctxkey_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
)

func setEnvDebug(val string) func() {
	old, ok := os.LookupEnv(string(env.Debug))
	if val == "" {
		_ = os.Unsetenv(string(env.Debug))
	} else {
		_ = os.Setenv(string(env.Debug), val)
	}
	return func() {
		if ok {
			_ = os.Setenv(string(env.Debug), old)
		} else {
			_ = os.Unsetenv(string(env.Debug))
		}
	}
}

func TestIsDebugMode(t *testing.T) {
	tests := []struct {
		name     string
		envVal   string
		expected bool
	}{
		{"unset", "", false},
		{"true", "true", true},
		{"1", "1", true},
		{"TRUE", "TRUE", true},
		{"false", "false", false},
		{"0", "0", false},
		{"invalid", "not-a-bool", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := setEnvDebug(tc.envVal)
			defer cleanup()

			if got := ctxkey.IsDebugMode(); got != tc.expected {
				t.Errorf("IsDebugMode() with APP_DEBUG=%q = %v; want %v", tc.envVal, got, tc.expected)
			}
		})
	}
}

func TestCalcOverheadTime(t *testing.T) {
	// 1. 非调试模式：无论上下文如何始终返回 0
	t.Run("non_debug_mode_always_zero", func(t *testing.T) {
		cleanup := setEnvDebug("false")
		defer cleanup()

		tests := []struct {
			name string
			ctx  context.Context
		}{
			{"with_start_time", context.WithValue(context.Background(), ctxkey.UserStartTimeKey, time.Now().Add(-100*time.Millisecond))},
			{"nil_context", nil},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				if got := ctxkey.CalcOverheadTime(tc.ctx); got != 0 {
					t.Errorf("expected 0 in non-debug mode, got %d", got)
				}
			})
		}
	})

	// 2. 调试模式下的各项边界
	t.Run("debug_mode_cases", func(t *testing.T) {
		cleanup := setEnvDebug("true")
		defer cleanup()

		tests := []struct {
			name string
			ctx  context.Context
		}{
			{"nil_context", nil},
			{"missing_key", context.Background()},
			{"wrong_type", context.WithValue(context.Background(), ctxkey.UserStartTimeKey, "not-a-time")},
			{"zero_time", context.WithValue(context.Background(), ctxkey.UserStartTimeKey, time.Time{})},
			{"future_time", context.WithValue(context.Background(), ctxkey.UserStartTimeKey, time.Now().Add(1*time.Hour))},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				if got := ctxkey.CalcOverheadTime(tc.ctx); got != 0 {
					t.Errorf("expected 0, got %d", got)
				}
			})
		}

		// normal time (sleep slightly to get positive microseconds)
		start := time.Now().Add(-5 * time.Millisecond)
		ctxNormal := context.WithValue(context.Background(), ctxkey.UserStartTimeKey, start)
		overhead := ctxkey.CalcOverheadTime(ctxNormal)
		if overhead <= 0 {
			t.Errorf("expected positive overhead microseconds, got %d", overhead)
		}
		if overhead < 4000 || overhead > 1000000 {
			t.Errorf("expected overhead around 5000us, got %d", overhead)
		}
	})
}

func TestGetRequestKey(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{"nil_context", nil, ""},
		{"missing_key", context.Background(), ""},
		{"wrong_type", context.WithValue(context.Background(), ctxkey.RequestKey, 12345), ""},
		{"normal_value", context.WithValue(context.Background(), ctxkey.RequestKey, "req-123-abc"), "req-123-abc"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ctxkey.GetRequestKey(tc.ctx); got != tc.expected {
				t.Errorf("GetRequestKey() = %q; want %q", got, tc.expected)
			}
		})
	}
}

func TestGetErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{"nil_context", nil, ""},
		{"missing_key", context.Background(), ""},
		{"wrong_type", context.WithValue(context.Background(), ctxkey.ErrorMessageKey, struct{}{}), ""},
		{"normal_value", context.WithValue(context.Background(), ctxkey.ErrorMessageKey, "database timeout"), "database timeout"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ctxkey.GetErrorMessage(tc.ctx); got != tc.expected {
				t.Errorf("GetErrorMessage() = %q; want %q", got, tc.expected)
			}
		})
	}
}
