package application

import (
	"context"
	"errors"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

func TestGetFrameworkErrorCodesAndCauses(t *testing.T) {
	key := ctxkey.Key("dependency")
	mount := func(value any, state EntryState) context.Context {
		container := newSharedContainer("error-code-regression")
		container.setEntry(key, key, value, state)
		return context.WithValue(context.Background(), containerCtxKey, container)
	}
	empty := newSharedContainer("empty-error-code-regression")
	emptyContext := context.WithValue(context.Background(), containerCtxKey, empty)
	undeclared := &startScopeView{container: empty, ownerKey: "consumer", allowedKeys: map[ctxkey.Key]struct{}{}}
	declared := &startScopeView{container: empty, ownerKey: "consumer", allowedKeys: map[ctxkey.Key]struct{}{key: {}}}

	tests := []struct {
		name  string
		ctx   context.Context
		code  *apperr.ErrorCode
		cause error
	}{
		{"nil context", nil, apperr.FrameworkContainerNotMounted, ErrContainerNotMounted},
		{"unmounted container", context.Background(), apperr.FrameworkContainerNotMounted, ErrContainerNotMounted},
		{"wrong container value", context.WithValue(context.Background(), containerCtxKey, "invalid"), apperr.FrameworkContainerNotMounted, ErrContainerNotMounted},
		{"nil container pointer", context.WithValue(context.Background(), containerCtxKey, (*sharedContainer)(nil)), apperr.FrameworkContainerNotMounted, ErrContainerNotMounted},
		{"undeclared dependency", context.WithValue(context.Background(), scopeViewCtxKey, undeclared), apperr.FrameworkDependencyUndeclared, ErrUndeclaredDependency},
		{"declared but missing dependency", context.WithValue(context.Background(), scopeViewCtxKey, declared), apperr.FrameworkComponentNotFound, ErrComponentNotFound},
		{"missing component", emptyContext, apperr.FrameworkComponentNotFound, ErrComponentNotFound},
		{"wrong component type", mount(42, EntryReady), apperr.FrameworkComponentTypeMismatch, ErrComponentTypeMismatch},
		{"nil nonpointer component", mount(nil, EntryReady), apperr.FrameworkComponentTypeMismatch, ErrComponentTypeMismatch},
		{"wrong native fallback type", context.WithValue(emptyContext, key, 42), apperr.FrameworkComponentTypeMismatch, ErrComponentTypeMismatch},
		{"stopped component", mount("value", EntryStopped), apperr.FrameworkComponentStopped, ErrComponentStopped},
		{"unavailable component", mount("value", EntryUnavailable), apperr.FrameworkComponentUnavailable, ErrComponentUnavailable},
		{"invalid component state", mount("value", EntryState(99)), apperr.FrameworkComponentInvalidState, ErrComponentUnavailable},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			value, err := Get[string](tc.ctx, key)
			if value != "" || err == nil {
				t.Fatalf("expected zero value and structured failure, got value=%q err=%v", value, err)
			}
			if err.Code != *tc.code || err.Code.Code/100 != 500 || !errors.Is(err, tc.cause) {
				t.Fatalf("framework code or original cause changed: got %+v, want %+v, cause %v", err, tc.code, tc.cause)
			}
		})
	}
}
