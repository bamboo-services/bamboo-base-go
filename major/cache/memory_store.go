package xCache

import (
	"container/list"
	"sync"
	"sync/atomic"
	"time"
)

// memoryEntry 通用缓存条目，承载任意数据结构的值。
//
// Value 字段约定（由各 cache 实现维护）：
//   - KeyCache: []byte（已序列化的值）
//   - HashCache: map[F][]byte（field → value）
//   - SetCache: map[string]struct{}（成员序列化后取 string 作为 key）
//   - ListCache: [][]byte（有序元素切片）
//
// elem 指向所在分片 LRU 链表的节点，用于 O(1) 前移与淘汰。
type memoryEntry struct {
	Value    any
	ExpireAt time.Time // 零值表示永不过期
	elem     *list.Element
}

// expired 判断条目是否已过期。
func (e *memoryEntry) expired(now time.Time) bool {
	return !e.ExpireAt.IsZero() && now.After(e.ExpireAt)
}

// memoryStore 分片内存存储，作为所有内存缓存实现的底座。
//
// 通过 [NewMemoryStore] 构造，ShardCount/MaxEntries/DefaultTTL 零值时使用默认值。
// 使用方应在应用退出时调用 [memoryStore.Close] 释放 janitor goroutine。
type memoryStore struct {
	shards     []*memoryShard
	shardMask  uint64
	maxEntries int           // 0 表示无上限（每个分片独立计数）
	defaultTTL time.Duration // 0 表示永不过期
	janitor    *memoryJanitor
}

// memoryShard 单个分片，承载一段 hash 空间的数据。
type memoryShard struct {
	mu    sync.RWMutex
	data  map[string]*memoryEntry
	order *list.List
}

func newMemoryShard() *memoryShard {
	return &memoryShard{
		data:  make(map[string]*memoryEntry),
		order: list.New(),
	}
}

// memoryJanitor 后台清理器，周期扫描过期条目。
type memoryJanitor struct {
	interval time.Duration
	stopCh   chan struct{}
	running  atomic.Bool
}

// NewMemoryStore 构造一个内存存储实例。
//
// 参数：
//   - shardCount: 分片数，必须为 2 的幂；0 使用默认 16
//   - maxEntries: 单分片最大条目数，0 表示无上限；超限时按 LRU 淘汰该分片最久未访问项。
//     全局总容量约 = maxEntries × shardCount（分片间独立计数）
//   - defaultTTL: 默认 TTL，0 表示永不过期
//
// janitor 默认每 30 秒清理一次过期项，可通过 [memoryStore.Close] 停止。
func NewMemoryStore(shardCount, maxEntries int, defaultTTL time.Duration) *memoryStore {
	if shardCount <= 0 {
		shardCount = 16
	}
	power := uint64(1)
	for power < uint64(shardCount) {
		power <<= 1
	}
	shardCount = int(power)

	s := &memoryStore{
		shards:     make([]*memoryShard, shardCount),
		shardMask:  uint64(shardCount - 1),
		maxEntries: maxEntries,
		defaultTTL: defaultTTL,
	}
	for i := range s.shards {
		s.shards[i] = newMemoryShard()
	}

	s.janitor = &memoryJanitor{
		interval: 30 * time.Second,
		stopCh:   make(chan struct{}),
	}
	s.janitor.running.Store(true)
	go s.janitor.run(s)

	return s
}

// DefaultTTL 返回构造时配置的默认 TTL。
func (s *memoryStore) DefaultTTL() time.Duration { return s.defaultTTL }

// Close 停止后台 janitor，释放资源。
//
// 调用后 Store 仍可读写，但不再自动清理过期项。可安全多次调用。
func (s *memoryStore) Close() {
	if s.janitor != nil && s.janitor.running.CompareAndSwap(true, false) {
		close(s.janitor.stopCh)
	}
}

// getShard 根据 key 的 FNV-1a hash 选取分片。
func (s *memoryStore) getShard(key string) *memoryShard {
	// FNV-1a 64bit
	var h uint64 = 14695981039346656037
	for i := 0; i < len(key); i++ {
		h ^= uint64(key[i])
		h *= 1099511628211
	}
	return s.shards[h&s.shardMask]
}

// run janitor 主循环。
func (j *memoryJanitor) run(s *memoryStore) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()
	for {
		select {
		case <-j.stopCh:
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

// cleanup 遍历所有分片删除过期项。
func (s *memoryStore) cleanup() {
	now := time.Now()
	for _, sh := range s.shards {
		sh.mu.Lock()
		for k, e := range sh.data {
			if e.expired(now) {
				sh.order.Remove(e.elem)
				delete(sh.data, k)
			}
		}
		sh.mu.Unlock()
	}
}

// Get 读取条目，若不存在或已过期返回 nil, false。
//
// 返回的是 Value 字段在锁内读取的引用（any），调用方不再持有 *memoryEntry，
// 因此与并发的 [Set]/[Update]（替换整个 Value 字段）不会产生 data race。
// 若 Value 是切片/map 类型，调用方应只读不写；如需修改请走 [Update] 闭包。
func (s *memoryStore) Get(key string) (any, bool) {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	e, ok := sh.data[key]
	if !ok {
		return nil, false
	}
	if e.expired(time.Now()) {
		sh.order.Remove(e.elem)
		delete(sh.data, key)
		return nil, false
	}
	sh.order.MoveToFront(e.elem)
	return e.Value, true
}

// Set 写入条目。
//
// ttl 为 0 时使用 Store 的 defaultTTL；若 defaultTTL 也为 0 则永不过期。
// 达到 maxEntries 时淘汰最久未访问项。
func (s *memoryStore) Set(key string, value any, ttl time.Duration) {
	expireAt := time.Time{}
	if ttl > 0 {
		expireAt = time.Now().Add(ttl)
	} else if s.defaultTTL > 0 {
		expireAt = time.Now().Add(s.defaultTTL)
	}

	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()

	if e, ok := sh.data[key]; ok {
		e.Value = value
		e.ExpireAt = expireAt
		sh.order.MoveToFront(e.elem)
		return
	}

	e := &memoryEntry{Value: value, ExpireAt: expireAt}
	e.elem = sh.order.PushFront(&memoryEntryLRU{key: key, entry: e})
	sh.data[key] = e

	if s.maxEntries > 0 && sh.order.Len() > s.maxEntries {
		s.evict(sh)
	}
}

// evict 淘汰最久未访问项。调用方需持有 sh.mu 写锁。
func (s *memoryStore) evict(sh *memoryShard) {
	back := sh.order.Back()
	if back == nil {
		return
	}
	lru := back.Value.(*memoryEntryLRU)
	sh.order.Remove(back)
	delete(sh.data, lru.key)
}

// memoryEntryLRU 链表节点载荷，记录 key 用于淘汰时反查 map。
type memoryEntryLRU struct {
	key   string
	entry *memoryEntry
}

// Delete 删除条目，返回是否曾存在（且未过期）。
func (s *memoryStore) Delete(key string) bool {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	e, ok := sh.data[key]
	if !ok {
		return false
	}
	sh.order.Remove(e.elem)
	delete(sh.data, key)
	return true
}

// Exists 判断键是否存在且未过期。不更新 LRU 顺序。
func (s *memoryStore) Exists(key string) bool {
	sh := s.getShard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	e, ok := sh.data[key]
	if !ok {
		return false
	}
	return !e.expired(time.Now())
}

// Len 返回未过期条目总数。仅供监控/调试使用。
func (s *memoryStore) Len() int {
	now := time.Now()
	n := 0
	for _, sh := range s.shards {
		sh.mu.RLock()
		for _, e := range sh.data {
			if !e.expired(now) {
				n++
			}
		}
		sh.mu.RUnlock()
	}
	return n
}

// Update 在单把分片锁内完成「读旧值 → 修改 → 写回」的原子操作。
//
// 解决 hash/set/list cache 的复合写竞争：原本的 Get → mutate → Set 模式在
// 解锁间隙会让多个 goroutine 并发写同一个 map/slice，触发
// `concurrent map writes` panic 或丢失更新。
//
// fn 接收旧值（不存在或已过期时为 nil），返回新值：
//   - 返回 nil 表示删除整个条目
//   - 返回非 nil 表示写入/覆盖（保留旧值类型时建议就地修改后返回原引用）
//
// ttl 为本次写入的过期时间，0 回退到 store.defaultTTL。
// fn 内部禁止执行阻塞 I/O 或长时间计算，否则会拖累整个分片的吞吐。
func (s *memoryStore) Update(key string, ttl time.Duration, fn func(old any) any) {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()

	var old any
	if e, ok := sh.data[key]; ok && !e.expired(time.Now()) {
		old = e.Value
	}

	newVal := fn(old)
	if newVal == nil {
		if e, ok := sh.data[key]; ok {
			sh.order.Remove(e.elem)
			delete(sh.data, key)
		}
		return
	}

	expireAt := time.Time{}
	if ttl > 0 {
		expireAt = time.Now().Add(ttl)
	} else if s.defaultTTL > 0 {
		expireAt = time.Now().Add(s.defaultTTL)
	}

	if e, ok := sh.data[key]; ok {
		e.Value = newVal
		e.ExpireAt = expireAt
		sh.order.MoveToFront(e.elem)
		return
	}
	e := &memoryEntry{Value: newVal, ExpireAt: expireAt}
	e.elem = sh.order.PushFront(&memoryEntryLRU{key: key, entry: e})
	sh.data[key] = e
	if s.maxEntries > 0 && sh.order.Len() > s.maxEntries {
		s.evict(sh)
	}
}
