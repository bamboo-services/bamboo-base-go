package cache

import "time"

// WithMemory 配置程序内内存作为缓存后端，返回 [CacheOption]。
//
// 隐式将缓存类型置为 CacheTypeMemory。可通过 [MemoryOption] 可变参数调整默认 TTL、容量等。
//
// 使用示例：
//
//	xOption.WithCache(xOptionCache.WithMemory(xOptionCache.WithMemoryDefaultTTL(30*time.Minute)))
func WithMemory(opts ...MemoryOption) CacheOption {
	return func(c *CacheConfig) {
		c.typeVal = CacheTypeMemory
		c.memory = MemoryOptions{DefaultTTL: 30 * time.Minute}
		for _, o := range opts {
			if o != nil {
				o(&c.memory)
			}
		}
	}
}

// MemoryOption 是 [MemoryOptions] 的二级选项。
type MemoryOption func(*MemoryOptions)

// WithMemoryDefaultTTL 设置默认过期时间。
func WithMemoryDefaultTTL(d time.Duration) MemoryOption {
	return func(m *MemoryOptions) { m.DefaultTTL = d }
}

// WithMemoryMaxEntries 设置最大条目数。
func WithMemoryMaxEntries(n int) MemoryOption { return func(m *MemoryOptions) { m.MaxEntries = n } }

// WithMemoryShardCount 设置分片数。
func WithMemoryShardCount(n int) MemoryOption { return func(m *MemoryOptions) { m.ShardCount = n } }
