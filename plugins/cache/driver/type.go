package driver

// CacheType 标识缓存后端的实现类型。
//
// 用于在运行期判断当前缓存的底层实现，进而决定底层特有操作。
// 零值空串等价于 [CacheTypeNone]，表示未启用任何内置缓存实现。
type CacheType string

// 常用缓存后端类型常量定义。
const (
	CacheTypeRedis  CacheType = "redis"  // 使用 Redis 作为缓存后端，适用于分布式与跨进程共享场景。
	CacheTypeMemory CacheType = "memory" // 使用程序内内存作为缓存后端，适用于单实例或可接受最终一致性的场景。
	CacheTypeNone   CacheType = "none"   // 不启用内置缓存实现，业务侧可自行通过注册机制装配缓存节点。
)

// String 返回缓存类型的字符串表示，便于日志输出与错误信息拼接。
func (c CacheType) String() string { return string(c) }

// IsRedis 返回当前类型是否为 Redis 后端。
func (c CacheType) IsRedis() bool { return c == CacheTypeRedis }

// IsMemory 返回当前类型是否为程序内内存后端。
func (c CacheType) IsMemory() bool { return c == CacheTypeMemory }

// Enabled 返回是否启用了内置缓存实现。
//
// 零值与显式声明 [CacheTypeNone] 均视为未启用。
func (c CacheType) Enabled() bool { return c != "" && c != CacheTypeNone }
