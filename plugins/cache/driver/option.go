package driver

import "time"

// SetOption 是缓存写操作的函数式选项，用于在单次调用中覆盖默认行为（如过期时间）。
type SetOption func(*SetConfig)

// SetConfig 持有单次写操作的运行时配置，由 [ApplySet] 从实例默认值与选项列表合成。
type SetConfig struct {
	// TTL 本次写入的过期时间。
	//   - > 0：按此值过期
	//   - 0：永不过期（覆盖实例默认 TTL 为永久）
	//   - 未被任何选项覆盖时，保留 [ApplySet] 的 base 入参（即实例默认 TTL）
	TTL time.Duration

	// NX 仅当键不存在时写入（与 XX 互斥，同传时 NX 优先）。
	NX bool
	// XX 仅当键已存在时写入（与 NX 互斥，同传时 NX 优先）。
	XX bool
	// KeepTTL 保留原有 TTL 不重设（覆盖值但不改变过期时间）。
	KeepTTL bool
	// NoSlide 写入但不续期（不滑动 TTL）。
	NoSlide bool
}

// ApplySet 将实例默认 TTL 与选项列表合成最终 [SetConfig]。
// 选项按顺序应用，后者覆盖前者；nil 选项被跳过。
func ApplySet(base time.Duration, opts []SetOption) SetConfig {
	cfg := SetConfig{TTL: base}
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}
	// NX 与 XX 互斥：同传时 NX 优先（与 Redis SET NX XX 行为一致）
	if cfg.NX && cfg.XX {
		cfg.XX = false
	}
	return cfg
}

// WithTTL 设置本次写入的过期时间，覆盖实例默认 TTL。
// ttl > 0 时按此值过期；ttl <= 0 时表示永不过期。
func WithTTL(ttl time.Duration) SetOption {
	return func(c *SetConfig) {
		if ttl > 0 {
			c.TTL = ttl
		} else {
			c.TTL = 0
		}
	}
}

// WithNX 设置仅当键不存在时写入的条件。
// 与 [WithXX] 互斥；若同时传入，NX 优先。
func WithNX() SetOption {
	return func(c *SetConfig) { c.NX = true }
}

// WithXX 设置仅当键已存在时写入的条件。
// 与 [WithNX] 互斥；若同时传入，NX 优先。
func WithXX() SetOption {
	return func(c *SetConfig) { c.XX = true }
}

// WithKeepTTL 保留原有 TTL，覆盖值但不重设过期时间。
func WithKeepTTL() SetOption {
	return func(c *SetConfig) { c.KeepTTL = true }
}

// WithNoSlide 写入但不续期（不滑动 TTL）。
func WithNoSlide() SetOption {
	return func(c *SetConfig) { c.NoSlide = true }
}
