package application

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

func TestContextValue(t *testing.T) {
	keyFoo := ctxkey.Key("test_foo")
	keyBar := ctxkey.Key("test_bar")

	tests := []struct {
		name        string
		state       EntryState
		expectValue any
		expectFound bool
	}{
		{
			name:        "Ready_ReturnsValue",
			state:       EntryReady,
			expectValue: "foo-service",
			expectFound: true,
		},
		{
			name:        "Stopped_ReturnsNil",
			state:       EntryStopped,
			expectValue: nil,
			expectFound: false,
		},
		{
			name:        "Unavailable_ReturnsNil",
			state:       EntryUnavailable,
			expectValue: nil,
			expectFound: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			container := newSharedContainer("test-app")
			container.setEntry(keyFoo, keyFoo, "foo-service", tc.state)
			container.setEntry(keyBar, keyBar, "bar-healthy", EntryReady)

			// 1. 测试 Detach 语义
			baseCtx := context.WithValue(context.Background(), containerCtxKey, container)
			detached, cancel := Detach(baseCtx)
			defer cancel()

			val := detached.Value(keyFoo)
			if tc.expectFound {
				if val != tc.expectValue {
					t.Fatalf("detached.Value expected %v, got %v", tc.expectValue, val)
				}
			} else {
				if val != nil {
					t.Fatalf("detached.Value expected nil, got %v", val)
				}
			}
			// 健康键应不受影响
			if barVal := detached.Value(keyBar); barVal != "bar-healthy" {
				t.Fatalf("detached.Value healthy key expected bar-healthy, got %v", barVal)
			}

			// 2. 测试 BridgedContext 语义与坏状态防透穿原 ctx
			reqCtx := context.WithValue(context.Background(), keyFoo, "req-fallback-polluted")
			bridged := BridgeRequest(reqCtx, baseCtx, "req-123")

			bridgedVal := bridged.Value(keyFoo)
			if tc.expectFound {
				if bridgedVal != tc.expectValue {
					t.Fatalf("bridged.Value expected %v, got %v", tc.expectValue, bridgedVal)
				}
			} else {
				if bridgedVal != nil {
					t.Fatalf("bridged.Value expected nil (no fallback to reqCtx), got %v", bridgedVal)
				}
			}
			// 健康键应不受影响
			if barVal := bridged.Value(keyBar); barVal != "bar-healthy" {
				t.Fatalf("bridged.Value healthy key expected bar-healthy, got %v", barVal)
			}
		})
	}
}

func TestContextValueDynamicStateChange(t *testing.T) {
	keyFoo := ctxkey.Key("dynamic_foo")
	container := newSharedContainer("test-app")
	container.setEntry(keyFoo, keyFoo, "active-val", EntryReady)

	baseCtx := context.WithValue(context.Background(), containerCtxKey, container)
	detached, cancel := Detach(baseCtx)
	defer cancel()

	if detached.Value(keyFoo) != "active-val" {
		t.Fatalf("expected active-val before stop")
	}

	// 动态变更状态至 Stopped
	entry, ok := container.getEntry(keyFoo)
	if !ok {
		t.Fatalf("entry not found")
	}
	atomic.StoreUint32(&entry.state, uint32(EntryStopped))

	if detached.Value(keyFoo) != nil {
		t.Fatalf("expected nil after stop")
	}
}
