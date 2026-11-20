package application

import (
	"sync"
	"sync/atomic"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

type containerEntry struct {
	key      ctxkey.Key
	ownerKey ctxkey.Key
	value    any
	state    uint32 // EntryReady, EntryStopped, EntryUnavailable
}

type sharedContainer struct {
	appID       string
	frozen      atomic.Bool
	stateReader func() AppState
	mu          sync.RWMutex
	entries     map[ctxkey.Key]*containerEntry
}

type publishItem struct {
	key      ctxkey.Key
	ownerKey ctxkey.Key
	val      any
	state    EntryState
}

func newSharedContainer(appID string) *sharedContainer {
	return &sharedContainer{
		appID:   appID,
		entries: make(map[ctxkey.Key]*containerEntry),
	}
}

func (c *sharedContainer) freeze() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.frozen.Store(true)
}

func (c *sharedContainer) publishBatch(items []publishItem) *apperr.Error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.frozen.Load() {
		return newAppErr(ErrRegistrationClosed, "共享容器已冻结，拒绝写入")
	}

	// 校验 batch 内与当前已入库条目是否存在冲突（先到先得，不覆盖原 owner）
	for _, item := range items {
		if _, exists := c.entries[item.key]; exists {
			return newAppErr(ErrDuplicateExposedKey, "键 "+string(item.key)+" 已被注册")
		}
	}

	// 检查 batch 内部自重复
	seen := make(map[ctxkey.Key]struct{}, len(items))
	for _, item := range items {
		if _, ok := seen[item.key]; ok {
			return newAppErr(ErrDuplicateExposedKey, "发布批次内键 "+string(item.key)+" 重复")
		}
		seen[item.key] = struct{}{}
	}

	// 原子提交：无任何冲突后一并入库
	for _, item := range items {
		entry := &containerEntry{
			key:      item.key,
			ownerKey: item.ownerKey,
			value:    item.val,
		}
		entry.state = uint32(item.state)
		c.entries[item.key] = entry
	}
	return nil
}

func (c *sharedContainer) setEntry(key, ownerKey ctxkey.Key, val any, state EntryState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry := &containerEntry{
		key:      key,
		ownerKey: ownerKey,
		value:    val,
	}
	entry.state = uint32(state)
	c.entries[key] = entry
}

func (c *sharedContainer) getEntry(key ctxkey.Key) (*containerEntry, bool) {
	if c.frozen.Load() {
		entry, ok := c.entries[key]
		return entry, ok
	}
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	return entry, ok
}

func (c *sharedContainer) setOwnerState(ownerKey ctxkey.Key, state EntryState) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, entry := range c.entries {
		if entry.ownerKey == ownerKey {
			atomic.StoreUint32(&entry.state, uint32(state))
		}
	}
}

type startScopeView struct {
	container   *sharedContainer
	ownerKey    ctxkey.Key
	allowedKeys map[ctxkey.Key]struct{}
}

func (v *startScopeView) getEntry(key ctxkey.Key) (*containerEntry, *apperr.Error) {
	if _, ok := v.allowedKeys[key]; !ok {
		return nil, newAppErr(ErrUndeclaredDependency, "插件 ["+string(v.ownerKey)+"] 未在 Requires 中声明依赖 ["+string(key)+"]，禁止在 Start 阶段越权读取")
	}
	entry, ok := v.container.getEntry(key)
	if !ok {
		return nil, newAppErr(ErrComponentNotFound, "组件 ["+string(key)+"] 未找到")
	}
	return entry, nil
}
