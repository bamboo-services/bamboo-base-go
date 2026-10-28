package application

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

type dummyTypedNilPlugin struct {
	startCount int32
	stopCount  int32
}

func (d *dummyTypedNilPlugin) Start(ctx context.Context) error {
	atomic.AddInt32(&d.startCount, 1)
	return nil
}

func (d *dummyTypedNilPlugin) Stop(ctx context.Context) error {
	atomic.AddInt32(&d.stopCount, 1)
	return nil
}

type typedNilExposePlugin struct {
	exposeVal any
}

func (t *typedNilExposePlugin) Start(ctx context.Context) error { return nil }
func (t *typedNilExposePlugin) Stop(ctx context.Context) error  { return nil }
func (t *typedNilExposePlugin) Expose() map[ctxkey.Key]any {
	return map[ctxkey.Key]any{
		"resKey": t.exposeVal,
	}
}

func TestTypedNil_IsTypedNil(t *testing.T) {
	var nilPtr *dummyTypedNilPlugin
	var nilMap map[string]int
	var nilSlice []string
	var nilFunc func()
	var nilChan chan int
	var nilInterface any
	var typedNilPlugin Plugin = nilPtr

	tests := []struct {
		name string
		val  any
		want bool
	}{
		{name: "untyped nil", val: nil, want: true},
		{name: "typed nil pointer", val: nilPtr, want: true},
		{name: "typed nil map", val: nilMap, want: true},
		{name: "typed nil slice", val: nilSlice, want: true},
		{name: "typed nil func", val: nilFunc, want: true},
		{name: "typed nil chan", val: nilChan, want: true},
		{name: "untyped nil interface", val: nilInterface, want: true},
		{name: "typed nil plugin interface", val: typedNilPlugin, want: true},
		{name: "non-nil pointer", val: &dummyTypedNilPlugin{}, want: false},
		{name: "non-nil map", val: map[string]int{"a": 1}, want: false},
		{name: "non-nil slice", val: []string{"a"}, want: false},
		{name: "non-nil func", val: func() {}, want: false},
		{name: "non-nil chan", val: make(chan int), want: false},
		{name: "scalar int", val: 0, want: false},
		{name: "scalar string", val: "", want: false},
		{name: "scalar bool", val: false, want: false},
		{name: "scalar struct", val: struct{ A int }{A: 1}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTypedNil(tt.val)
			if got != tt.want {
				t.Fatalf("isTypedNil(%v [%T]) = %v, want %v", tt.val, tt.val, got, tt.want)
			}
		})
	}
}

func TestTypedNil_FactoryReturnsTypedNil(t *testing.T) {
	var dummy *dummyTypedNilPlugin

	app := New()
	err := app.RegisterTool(
		PluginInfo{Key: "typedNilTool"},
		func() Plugin {
			return dummy
		},
	)
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	runErr := app.Run()
	if runErr == nil {
		t.Fatal("expected Run error for typed nil plugin, got nil")
	}
	if !errors.Is(runErr, ErrNilPlugin) {
		t.Fatalf("expected ErrNilPlugin, got: %v", runErr)
	}

	if dummy != nil {
		if c := atomic.LoadInt32(&dummy.startCount); c != 0 {
			t.Fatalf("expected Start count 0, got %d", c)
		}
		if c := atomic.LoadInt32(&dummy.stopCount); c != 0 {
			t.Fatalf("expected Stop count 0, got %d", c)
		}
	}
}

func TestTypedNil_ExposeReturnsTypedNil(t *testing.T) {
	type myResource struct{}
	var nilRes *myResource

	app := New()
	err := app.RegisterTool(
		PluginInfo{
			Key:      "exposePlugin",
			Provides: []ctxkey.Key{"resKey"},
		},
		func() Plugin {
			return &typedNilExposePlugin{
				exposeVal: nilRes,
			}
		},
	)
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	runErr := app.Run()
	if runErr == nil {
		t.Fatal("expected Run error for typed nil exposed resource, got nil")
	}
	if !errors.Is(runErr, ErrExposedKeysMismatch) {
		t.Fatalf("expected ErrExposedKeysMismatch, got: %v", runErr)
	}

	// 验证未发布到容器
	if _, found := app.container.getEntry("resKey"); found {
		t.Fatal("typed nil resource should not be published to container")
	}
}
