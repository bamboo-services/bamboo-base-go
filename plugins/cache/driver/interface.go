package driver

import "context"

// KeyCache 定义基于字符串（String）数据结构的通用泛型缓存接口。
//
// 适用于管理单一实体、配置项或轻量级 JSON 对象的存取与生命周期。
//
// 泛型参数：
//   - K: 缓存键类型，由注入的 [KeyEncoder] 转换为底层字符串键；
//   - V: 缓存值类型，由注入的 [Codec] 执行序列化与反序列化。
type KeyCache[K any, V any] interface {
	// Get 根据键检索缓存值。
	//
	// 当键不存在或已过期时，返回 nil, false, nil，不产生系统错误。
	// 当反序列化失败或底层通信异常时，返回 nil, false, error。
	Get(ctx context.Context, key K) (*V, bool, error)

	// Set 将键值对写入缓存。
	//
	// 当 value 为 nil 时，语义等价于 [KeyCache.Delete]。
	// 可通过 opts 传入 [WithTTL]、[WithNX]、[WithXX]、[WithKeepTTL] 等选项控制写入行为。
	Set(ctx context.Context, key K, value *V, opts ...SetOption) error

	// Exists 检查指定的键是否存在且未过期。
	Exists(ctx context.Context, key K) (bool, error)

	// Delete 从缓存中移除指定的键。如果键原本不存在，操作应幂等成功，不返回错误。
	Delete(ctx context.Context, key K) error
}

// HashCache 定义基于哈希（Hash）数据结构的二维键值对缓存接口。
//
// 组织模型为 key -> field -> value，支持细粒度的字段读写以及完整结构体的映射。
//
// 泛型参数：
//   - K: 哈希外层键类型；
//   - F: 哈希字段类型，必须满足 [comparable] 约束；
//   - V: 字段值类型；
//   - S: 结构体映射类型，用于 [HashCache.GetAllStruct] 与 [HashCache.SetAllStruct]。
type HashCache[K any, F comparable, V any, S any] interface {
	// Get 获取哈希表中指定字段的值。
	//
	// 当键或字段不存在时，返回 nil, false, nil。
	Get(ctx context.Context, key K, field F) (*V, bool, error)

	// Set 设置哈希表中单个字段的值。
	//
	// 当 value 为 nil 时，等价于移除该字段。
	// 可通过 opts 指定 [WithNX]、[WithXX]、[WithNoSlide] 等选项。
	Set(ctx context.Context, key K, field F, value *V, opts ...SetOption) error

	// GetAll 获取哈希表中的所有字段与对应值。
	//
	// 当键不存在时，返回空 map 与 nil error。
	GetAll(ctx context.Context, key K) (map[F]V, error)

	// GetAllStruct 获取哈希表中的所有字段并映射为指定的结构体类型 S。
	GetAllStruct(ctx context.Context, key K) (S, error)

	// SetAll 批量设置哈希表中的字段。
	//
	// 映射中值为 nil 的字段将被移除。若设置后哈希表为空，底层条目可能被直接清理。
	SetAll(ctx context.Context, key K, fields map[F]*V, opts ...SetOption) error

	// SetAllStruct 将结构体 S 的字段批量写入哈希表。
	SetAllStruct(ctx context.Context, key K, value S, opts ...SetOption) error

	// Exists 检查哈希表中是否存在指定的字段。
	Exists(ctx context.Context, key K, field F) (bool, error)

	// Remove 从哈希表中移除一个或多个指定字段。
	Remove(ctx context.Context, key K, fields ...F) error

	// Delete 删除整个哈希表及其全部关联字段。
	Delete(ctx context.Context, key K) error
}

// SetCache 定义基于无序唯一集合（Set）数据结构的缓存接口。
//
// 适用于标签过滤、去重列表、在线状态与权限校验等场景。
//
// 泛型参数：
//   - K: 集合键类型；
//   - V: 集合成员值类型。
type SetCache[K any, V any] interface {
	// Add 将一组成员添加到集合中，已存在的成员将被自动去重忽略。
	//
	// members 参数采用切片形式，尾部保留 opts 变参以支持 [WithTTL]、[WithNX] 等选项。
	Add(ctx context.Context, key K, members []V, opts ...SetOption) error

	// Members 获取集合中的所有成员。当键不存在时返回空切片或 nil。
	Members(ctx context.Context, key K) ([]V, error)

	// IsMember 检查指定成员是否存在于集合中。
	IsMember(ctx context.Context, key K, member V) (bool, error)

	// Count 获取集合中的成员总数。
	Count(ctx context.Context, key K) (int64, error)

	// Remove 从集合中移除一个或多个指定成员。
	Remove(ctx context.Context, key K, members ...V) error

	// Delete 删除整个集合。
	Delete(ctx context.Context, key K) error
}

// ListCache 定义基于有序双端列表（List）数据结构的缓存接口。
//
// 支持头尾双向压入弹出、索引范围切片及按值删除，适用于队列、时间线及最近记录展示。
//
// 泛型参数：
//   - K: 列表键类型；
//   - V: 列表元素值类型。
type ListCache[K any, V any] interface {
	// Prepend 将一组值插入到列表头部（左侧）。
	//
	// 插入后列表最左侧的元素顺序与切片中的顺序保持一致。
	Prepend(ctx context.Context, key K, values []V, opts ...SetOption) error

	// Append 将一组值追加到列表尾部（右侧）。
	Append(ctx context.Context, key K, values []V, opts ...SetOption) error

	// Range 按起始与结束索引获取列表元素切片。
	//
	// 支持负数索引，-1 表示最后一个元素，-2 表示倒数第二个元素。
	Range(ctx context.Context, key K, start int64, end int64) ([]V, error)

	// Index 获取指定索引位置处的单个元素，支持负数索引。索引越界时返回 nil, nil。
	Index(ctx context.Context, key K, index int64) (*V, error)

	// Len 获取列表当前包含的元素总数。
	Len(ctx context.Context, key K) (int64, error)

	// Pop 从列表头部（左侧）弹出一个元素并将其从列表中移除。列表为空时返回 nil, nil。
	Pop(ctx context.Context, key K) (*V, error)

	// PopLast 从列表尾部（右侧）弹出一个元素并将其从列表中移除。列表为空时返回 nil, nil。
	PopLast(ctx context.Context, key K) (*V, error)

	// Remove 从列表中移除匹配指定值的元素。
	//
	// count > 0 时从左向右移除最多 count 个匹配项；
	// count < 0 时从右向左移除最多 -count 个匹配项；
	// count = 0 时移除所有与 value 匹配的项。
	Remove(ctx context.Context, key K, count int64, value V) error

	// Delete 删除整个列表。
	Delete(ctx context.Context, key K) error
}
