package xCacheDriver

import "time"

// SetOption 是缓存写操作的函数式选项，用于在单次调用中覆盖默认行为（如过期时间）。
//
// 由 [WithTTL] 等构造函数产出，经 [ApplySet] 在写操作入口合成最终 [SetConfig]。
// 业务侧可在 Set / Add / Prepend 等写方法尾部追加任意数量的选项：
//
//	_ = kc.Set(ctx, "user:1", &u, xCache.WithTTL(5*time.Minute))
//
// 设计为函数式选项而非方法参数，便于后续无破坏性地扩展更多控制项
//（如仅当不存在时写入、条件续期、版本号等）。
type SetOption func(*SetConfig)

// SetConfig 持有单次写操作的运行时配置，由 [ApplySet] 从实例默认值与选项列表合成。
//
// 各后端实现读取其字段决定实际行为，无需感知选项的构造细节。
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
	// 对 KeyCache.Set 等价于 Redis SET KEEPTTL；对 Hash/Set/List 等价于 NoSlide。
	KeepTTL bool
	// NoSlide 写入但不续期（不滑动 TTL）。
	// 主要用于 Hash/Set/List 的追加操作：添加数据但不延长 key 的整体 TTL。
	// 对 KeyCache.Set 无意义（Set 本身设新 TTL），传入时被忽略。
	NoSlide bool
}

// ApplySet 将实例默认 TTL 与选项列表合成最终 [SetConfig]。
//
// base 通常取自 Manager / cache 实例构造时注入的默认 TTL（0 表示默认永久）。
// 选项按顺序应用，后者覆盖前者；nil 选项被跳过以保证调用方安全性。
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
//
// ttl > 0 时按此值过期；ttl <= 0 时表示永不过期（即便实例默认 TTL > 0，本次也设为永久）。
// 不调用本选项则沿用 [ApplySet] 的 base。
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
//
// 与 [WithXX] 互斥；若同时传入，NX 优先（XX 被忽略）。
//
//	_ = kc.Set(ctx, "lock:order:1", &token, xCache.WithNX(), xCache.WithTTL(30*time.Second))
func WithNX() SetOption {
	return func(c *SetConfig) { c.NX = true }
}

// WithXX 设置仅当键已存在时写入的条件。
//
// 与 [WithNX] 互斥；若同时传入，NX 优先（XX 被忽略）。
//
//	_ = kc.Set(ctx, "config:feature", &v, xCache.WithXX())
func WithXX() SetOption {
	return func(c *SetConfig) { c.XX = true }
}

// WithKeepTTL 保留原有 TTL，覆盖值但不重设过期时间。
//
// 对 KeyCache.Set 等价于 Redis SET KEEPTTL；对 Hash/Set/List 等价于 [WithNoSlide]。
// 与 [WithTTL] 互斥；若同时传入，KeepTTL 优先（TTL 被忽略）。
//
//	_ = kc.Set(ctx, "user:1", &u, xCache.WithKeepTTL())
func WithKeepTTL() SetOption {
	return func(c *SetConfig) { c.KeepTTL = true }
}

// WithNoSlide 写入但不续期（不滑动 TTL）。
//
// 主要用于 Hash/Set/List 的追加操作：添加数据但不延长 key 的整体 TTL。
// 对 KeyCache.Set 无意义（Set 本身设新 TTL），传入时被忽略。
//
//	_ = lc.Append(ctx, "queue:1m", []string{"a", "b"}, xCache.WithNoSlide())
func WithNoSlide() SetOption {
	return func(c *SetConfig) { c.NoSlide = true }
}
