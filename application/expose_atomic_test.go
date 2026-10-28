package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

type dummyNilPtr struct{}

// 漏键插件
type missingKeysPlugin struct{}

func (p *missingKeysPlugin) Start(ctx context.Context) error { return nil }
func (p *missingKeysPlugin) Stop(ctx context.Context) error  { return nil }
func (p *missingKeysPlugin) Expose() map[ctxkey.Key]any {
	return map[ctxkey.Key]any{}
}

// 多键插件
type extraKeysPlugin struct{}

func (p *extraKeysPlugin) Start(ctx context.Context) error { return nil }
func (p *extraKeysPlugin) Stop(ctx context.Context) error  { return nil }
func (p *extraKeysPlugin) Expose() map[ctxkey.Key]any {
	return map[ctxkey.Key]any{
		"k1": "val1",
		"k2": "val2",
	}
}

// typednil 插件
type typedNilPlugin struct{}

func (p *typedNilPlugin) Start(ctx context.Context) error { return nil }
func (p *typedNilPlugin) Stop(ctx context.Context) error  { return nil }
func (p *typedNilPlugin) Expose() map[ctxkey.Key]any {
	var ptr *dummyNilPtr
	return map[ctxkey.Key]any{
		"k1": ptr,
	}
}

// panic 插件
type panicExposePlugin struct{}

func (p *panicExposePlugin) Start(ctx context.Context) error { return nil }
func (p *panicExposePlugin) Stop(ctx context.Context) error  { return nil }
func (p *panicExposePlugin) Expose() map[ctxkey.Key]any {
	panic("expose explosive")
}

// 阻塞并验证锁外调用的插件
type blockingExposePlugin struct {
	inExpose chan struct{}
	release  chan struct{}
}

func (p *blockingExposePlugin) Start(ctx context.Context) error { return nil }
func (p *blockingExposePlugin) Stop(ctx context.Context) error  { return nil }
func (p *blockingExposePlugin) Expose() map[ctxkey.Key]any {
	close(p.inExpose)
	<-p.release
	return map[ctxkey.Key]any{
		"res_block": "ok",
	}
}

// 正常成功插件
type successPlugin struct{}

func (p *successPlugin) Start(ctx context.Context) error { return nil }
func (p *successPlugin) Stop(ctx context.Context) error  { return nil }
func (p *successPlugin) Expose() map[ctxkey.Key]any {
	return map[ctxkey.Key]any{
		"res1": "val1",
		"res2": "val2",
	}
}

func TestExposeAtomic_ValidationFailuresNoPartialPublish(t *testing.T) {
	cases := []struct {
		name     string
		info     PluginInfo
		plugin   Plugin
		errCheck func(error) bool
	}{
		{
			name:     "漏键失败无半发布",
			info:     PluginInfo{Key: "p_missing", Provides: []ctxkey.Key{"k1"}},
			plugin:   &missingKeysPlugin{},
			errCheck: func(err error) bool { return errors.Is(err, ErrExposedKeysMismatch) },
		},
		{
			name:     "多键失败无半发布",
			info:     PluginInfo{Key: "p_extra", Provides: []ctxkey.Key{"k1"}},
			plugin:   &extraKeysPlugin{},
			errCheck: func(err error) bool { return errors.Is(err, ErrExposedKeysMismatch) },
		},
		{
			name:     "typednil失败无半发布",
			info:     PluginInfo{Key: "p_typed_nil", Provides: []ctxkey.Key{"k1"}},
			plugin:   &typedNilPlugin{},
			errCheck: func(err error) bool { return errors.Is(err, ErrExposedKeysMismatch) },
		},
		{
			name:   "panic捕获无半发布",
			info:   PluginInfo{Key: "p_panic", Provides: []ctxkey.Key{"k1"}},
			plugin: &panicExposePlugin{},
			errCheck: func(err error) bool {
				var hp *HookPanicError
				return errors.As(err, &hp)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := New()
			err := app.publishExposedKeys(tc.info, tc.plugin)
			if err == nil || !tc.errCheck(err) {
				t.Fatalf("expected error mismatch, got %v", err)
			}
			// 验证主键和资源键均不存在（无主键半发布）
			if _, ok := app.container.getEntry(tc.info.Key); ok {
				t.Fatalf("main key %s should not exist after failure", tc.info.Key)
			}
			for _, rk := range tc.info.Provides {
				if _, ok := app.container.getEntry(rk); ok {
					t.Fatalf("resource key %s should not exist after failure", rk)
				}
			}
		})
	}
}

func TestExposeAtomic_SuccessAllReadable(t *testing.T) {
	app := New()
	info := PluginInfo{Key: "p_ok", Provides: []ctxkey.Key{"res1", "res2"}}
	err := app.publishExposedKeys(info, &successPlugin{})
	if err != nil {
		t.Fatalf("publish unexpected error: %v", err)
	}

	// 主键与所有资源键一并可读
	mainE, ok := app.container.getEntry("p_ok")
	if !ok || mainE.value == nil {
		t.Fatalf("main entry should be readable")
	}
	r1, ok := app.container.getEntry("res1")
	if !ok || r1.value != "val1" {
		t.Fatalf("res1 should be val1")
	}
	r2, ok := app.container.getEntry("res2")
	if !ok || r2.value != "val2" {
		t.Fatalf("res2 should be val2")
	}
}

func TestExposeAtomic_DuplicateKeyRejection(t *testing.T) {
	app := New()
	info1 := PluginInfo{Key: "p_first", Provides: []ctxkey.Key{"res1", "res2"}}
	if err := app.publishExposedKeys(info1, &successPlugin{}); err != nil {
		t.Fatalf("first publish failed: %v", err)
	}

	// 尝试重复发布相同主键，必须拒绝并不覆盖原 entry
	info2 := PluginInfo{Key: "p_first", Provides: []ctxkey.Key{"k1", "k2"}}
	err := app.publishExposedKeys(info2, &extraKeysPlugin{})
	if !errors.Is(err, ErrDuplicateExposedKey) {
		t.Fatalf("expected ErrDuplicateExposedKey, got: %v", err)
	}

	ent, ok := app.container.getEntry("p_first")
	if !ok {
		t.Fatalf("entry not found")
	}
	if _, isSuccess := ent.value.(*successPlugin); !isSuccess {
		t.Fatalf("original owner overwritten!")
	}
}

func TestExposeAtomic_FreezeRejection(t *testing.T) {
	app := New()
	app.container.freeze()

	info := PluginInfo{Key: "p_late"}
	err := app.publishExposedKeys(info, &successPlugin{})
	if err == nil {
		t.Fatalf("expected error on frozen container, got nil")
	}
	if _, ok := app.container.getEntry("p_late"); ok {
		t.Fatalf("key should not be written to frozen container")
	}
}

func TestExposeAtomic_UserCodeDoesNotHoldLock(t *testing.T) {
	app := New()
	inExpose := make(chan struct{})
	release := make(chan struct{})

	p := &blockingExposePlugin{
		inExpose: inExpose,
		release:  release,
	}
	info := PluginInfo{Key: "p_block", Provides: []ctxkey.Key{"res_block"}}

	var pubErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		pubErr = app.publishExposedKeys(info, p)
	}()

	// 等待进入 Expose() 用户代码，此时完全无 sleep 纯通道同步
	<-inExpose

	// 此时若持有 container.mu，则 container.getEntry 或读写锁会死锁/阻塞。
	// 我们尝试获取读锁与写锁来无阻塞证明 container 未被加锁。
	acquiredRead := make(chan bool, 1)
	go func() {
		app.container.mu.RLock()
		count := len(app.container.entries)
		isFrozen := app.container.frozen.Load()
		app.container.mu.RUnlock()
		if count < 0 || isFrozen {
			t.Errorf("unexpected container state in read lock: count=%d, frozen=%v", count, isFrozen)
		}
		acquiredRead <- true
	}()

	select {
	case <-acquiredRead:
		// 成功证明 Expose 执行时不持有写锁
	case <-time.After(2 * time.Second):
		t.Fatalf("container locked while Expose() is executing!")
	}

	acquiredWrite := make(chan bool, 1)
	go func() {
		app.container.mu.Lock()
		app.container.entries[ctxkey.Key("probe_key")] = &containerEntry{state: uint32(EntryReady)}
		delete(app.container.entries, ctxkey.Key("probe_key"))
		app.container.mu.Unlock()
		acquiredWrite <- true
	}()

	select {
	case <-acquiredWrite:
		// 成功证明 Expose 执行时不持有读锁
	case <-time.After(2 * time.Second):
		t.Fatalf("container locked while Expose() is executing!")
	}

	// 释放 Expose
	close(release)
	wg.Wait()

	if pubErr != nil {
		t.Fatalf("publish returned error: %v", pubErr)
	}
}
