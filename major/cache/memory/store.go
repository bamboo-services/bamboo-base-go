package xCacheMemory

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

// Store 分片内存存储，作为所有内存缓存实现的底座。
//
// 通过 [NewStore] 构造，ShardCount/MaxEntries/DefaultTTL 零值时使用默认值。
// 使用方应在应用退出时调用 [Store.Close] 释放 janitor goroutine。
type Store struct {
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

// NewStore 构造一个内存存储实例。
//
// 参数：
//   - shardCount: 分片数，必须为 2 的幂；0 使用默认 16
//   - maxEntries: 单分片最大条目数，0 表示无上限；超限时按 LRU 淘汰该分片最久未访问项。
//     全局总容量约 = maxEntries × shardCount（分片间独立计数）
//   - defaultTTL: 默认 TTL，0 表示永不过期
//
// janitor 默认每 30 秒清理一次过期项，可通过 [Store.Close] 停止。
func NewStore(shardCount, maxEntries int, defaultTTL time.Duration) *Store {
	if shardCount <= 0 {
		shardCount = 16
	}
	power := uint64(1)
	for power < uint64(shardCount) {
		power <<= 1
	}
	shardCount = int(power)

	s := &Store{
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
func (s *Store) DefaultTTL() time.Duration { return s.defaultTTL }

// Close 停止后台 janitor，释放资源。
//
// 调用后 Store 仍可读写，但不再自动清理过期项。可安全多次调用。
func (s *Store) Close() {
	if s.janitor != nil && s.janitor.running.CompareAndSwap(true, false) {
		close(s.janitor.stopCh)
	}
}

// getShard 根据 key 的 FNV-1a hash 选取分片。
func (s *Store) getShard(key string) *memoryShard {
	// FNV-1a 64bit
	var h uint64 = 14695981039346656037
	for i := 0; i < len(key); i++ {
		h ^= uint64(key[i])
		h *= 1099511628211
	}
	return s.shards[h&s.shardMask]
}

// run janitor 主循环。
func (j *memoryJanitor) run(s *Store) {
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
func (s *Store) cleanup() {
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
func (s *Store) Get(key string) (any, bool) {
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
func (s *Store) Set(key string, value any, ttl time.Duration) {
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
func (s *Store) evict(sh *memoryShard) {
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
func (s *Store) Delete(key string) bool {
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
func (s *Store) Exists(key string) bool {
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
func (s *Store) Len() int {
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

// UpdateNoChange 是 [Store.Update] / [Store.UpdateKeepExpireAt] 闭包的哨兵返回值，
// 表示本次不产生任何变更（不更新 Value、不更新 ExpireAt、不移动 LRU）。
//
// 用于 Hash/Set/List 的条件追加场景：当闭包判定无需写入时返回本值，
// 区别于返回 nil（nil 表示删除整个条目）。
var UpdateNoChange = &struct{}{}

// Update 在单把分片锁内完成「读旧值 → 修改 → 写回」的原子操作。
//
// 解决 hash/set/list cache 的复合写竞争：原本的 Get → mutate → Set 模式在
// 解锁间隙会让多个 goroutine 并发写同一个 map/slice，触发
// `concurrent map writes` panic 或丢失更新。
//
// fn 接收旧值（不存在或已过期时为 nil），返回新值：
//   - 返回 [UpdateNoChange] 表示本次不产生任何变更（短路返回）
//   - 返回 nil 表示删除整个条目
//   - 返回非 nil 表示写入/覆盖（保留旧值类型时建议就地修改后返回原引用）
//
// ttl 为本次写入的过期时间，0 回退到 store.defaultTTL。
// fn 内部禁止执行阻塞 I/O 或长时间计算，否则会拖累整个分片的吞吐。
func (s *Store) Update(key string, ttl time.Duration, fn func(old any) any) {
	s.updateInternal(key, ttl, fn, false)
}

// UpdateKeepExpireAt 与 [Store.Update] 行为一致，唯一区别是不重设已存在条目的 ExpireAt。
//
// 用于 Hash/Set/List 的 NoSlide/KeepTTL 场景：追加数据但不延长 key 的整体 TTL。
// 若条目不存在则按 ttl/defaultTTL 计算新 ExpireAt（新 key 无原 TTL 可保留）。
//
// fn 的返回值约定与 [Store.Update] 相同（含 [UpdateNoChange] 哨兵）。
func (s *Store) UpdateKeepExpireAt(key string, ttl time.Duration, fn func(old any) any) {
	s.updateInternal(key, ttl, fn, true)
}

// updateInternal 是 [Store.Update] / [Store.UpdateKeepExpireAt] 的共享实现。
//
// keepExpireAt 为 true 时：entry 已存在则保留原 ExpireAt 不重设（仅更新 Value 与 LRU）；
// entry 不存在则按 ttl/defaultTTL 计算 ExpireAt（新 key 无原 TTL 可保留）。
// keepExpireAt 为 false 时：行为与原 Update 完全一致（无条件重设 ExpireAt）。
//
// 闭包返回值判断顺序：先 [UpdateNoChange] 短路返回，再 nil 删除，最后正常写入——
// 避免「无变化」语义与「删除」语义混淆。
func (s *Store) updateInternal(key string, ttl time.Duration, fn func(old any) any, keepExpireAt bool) {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()

	var old any
	existing, ok := sh.data[key]
	exists := ok && !existing.expired(time.Now())
	if exists {
		old = existing.Value
	}

	newVal := fn(old)
	// 先判断哨兵：无变化则直接返回，不做任何写入或 LRU 移动
	if newVal == UpdateNoChange {
		return
	}
	// 再判断 nil：删除整个条目
	if newVal == nil {
		if ok {
			sh.order.Remove(existing.elem)
			delete(sh.data, key)
		}
		return
	}

	// 计算新的 ExpireAt
	// - keepExpireAt=true 且 entry 存在且未过期：保留原 ExpireAt
	// - 其余情况（含 entry 已过期）：按 ttl/defaultTTL 计算（同 Set 逻辑）
	var expireAt time.Time
	if keepExpireAt && exists {
		expireAt = existing.ExpireAt
	} else if ttl > 0 {
		expireAt = time.Now().Add(ttl)
	} else if s.defaultTTL > 0 {
		expireAt = time.Now().Add(s.defaultTTL)
	}

	if ok {
		existing.Value = newVal
		existing.ExpireAt = expireAt
		sh.order.MoveToFront(existing.elem)
		return
	}
	e := &memoryEntry{Value: newVal, ExpireAt: expireAt}
	e.elem = sh.order.PushFront(&memoryEntryLRU{key: key, entry: e})
	sh.data[key] = e
	if s.maxEntries > 0 && sh.order.Len() > s.maxEntries {
		s.evict(sh)
	}
}

// SetCond 带条件地写入单 key 条目，用于 KeyCache 的 NX/XX/KEEPTTL 语义。
//
// 参数：
//   - nx: 为 true 时仅当 key 不存在（或已过期）才写入，否则返回 false
//   - xx: 为 true 时仅当 key 已存在且未过期才写入，否则返回 false
//   - keepTTL: 为 true 且 key 已存在时保留原 ExpireAt 不重设（等价于 Redis SET KEEPTTL）
//
// nx 与 xx 同传时 nx 优先（与 Redis SET NX XX 行为一致）。
// ttl 为 0 时使用 Store 的 defaultTTL；若 defaultTTL 也为 0 则永不过期。
// 写入成功返回 true，条件不满足或被淘汰不写入时返回 false。
// 达到 maxEntries 时淘汰最久未访问项。
func (s *Store) SetCond(key string, value any, ttl time.Duration, nx, xx, keepTTL bool) bool {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()

	now := time.Now()
	existing, ok := sh.data[key]
	exists := ok && !existing.expired(now)

	// nx 优先于 xx
	if nx && exists {
		return false
	}
	if xx && !exists {
		return false
	}

	// 计算 ExpireAt
	// - keepTTL=true 且 key 已存在：保留原 ExpireAt
	// - 其余情况：按 ttl/defaultTTL 计算（同 Set 逻辑）
	var expireAt time.Time
	if keepTTL && exists {
		expireAt = existing.ExpireAt
	} else if ttl > 0 {
		expireAt = now.Add(ttl)
	} else if s.defaultTTL > 0 {
		expireAt = now.Add(s.defaultTTL)
	}

	if exists {
		existing.Value = value
		existing.ExpireAt = expireAt
		sh.order.MoveToFront(existing.elem)
		return true
	}

	// 已过期或不存在：若 map 中残留过期条目先清理
	if ok {
		sh.order.Remove(existing.elem)
		delete(sh.data, key)
	}

	e := &memoryEntry{Value: value, ExpireAt: expireAt}
	e.elem = sh.order.PushFront(&memoryEntryLRU{key: key, entry: e})
	sh.data[key] = e

	if s.maxEntries > 0 && sh.order.Len() > s.maxEntries {
		s.evict(sh)
	}
	return true
}
