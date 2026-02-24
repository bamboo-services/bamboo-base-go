package xCache

import "context"

// KeyCache 定义了基于字符串（String）数据结构的缓存操作接口，用于管理单一键值对数据。
//
// 该接口提供了获取、设置和删除缓存项的基本能力，适用于需要快速存取单个对象或配置项的场景。
//
// 泛型参数：
//   - K: 缓存键的类型，用于标识特定的缓存项。
//   - V: 缓存值的类型。
//
// Get 方法根据键检索值，返回指向值的指针、是否存在以及可能的错误。
// Set 方法将键值对存入缓存，需处理 value 为 nil 的场景。
// Exists 方法检查指定键是否存在。
// Delete 方法从缓存中移除指定的键。
type KeyCache[K any, V any] interface {
	Get(ctx context.Context, key K) (*V, bool, error)
	Set(ctx context.Context, key K, value *V) error
	Exists(ctx context.Context, key K) (bool, error)
	Delete(ctx context.Context, key K) error
}

// HashCache 定义了基于哈希（Hash）数据结构的缓存操作接口，用于管理二维键值对数据。
//
// 该接口提供了对哈希表字段的增删改查能力，适用于需要存储对象属性、用户配置或分组数据的场景。
// 哈希结构为 key → field → value，支持对单个字段或多个字段进行操作。
//
// 泛型参数：
//   - K: 哈希键的类型，用于标识特定的哈希表。
//   - F: 字段键的类型，用于标识哈希表中的特定字段。
//   - V: 字段值的类型。
//   - S: 结构体的类型，用于指定已有的结构体字段。
//
// Get 方法获取指定字段的值，返回指向值的指针、是否存在以及可能的错误。
// Set 方法设置单个字段的值。
// GetAll 方法获取哈希表中的所有字段和值。
// GetAllStruct 方法获取哈希表中的所有字段和值，返回指定的结构体。
// SetAll 方法批量设置多个字段的值。
// SetAllStruct 方法批量设置多个字段的值，使用指定的结构体。
// Exists 方法检查指定字段是否存在。
// Remove 方法从哈希表中移除指定的字段。
// Delete 方法删除整个哈希表。
type HashCache[K any, F comparable, V any, S any] interface {
	Get(ctx context.Context, key K, field F) (*V, bool, error)
	Set(ctx context.Context, key K, field F, value *V) error
	GetAll(ctx context.Context, key K) (map[F]V, error)
	GetAllStruct(ctx context.Context, key K) (S, error)
	SetAll(ctx context.Context, key K, fields map[F]*V) error
	SetAllStruct(ctx context.Context, key K, value S) error
	Exists(ctx context.Context, key K, field F) (bool, error)
	Remove(ctx context.Context, key K, fields ...F) error
	Delete(ctx context.Context, key K) error
}

// SetCache 定义了基于集合（Set）数据结构的缓存操作接口，用于管理无序且元素唯一的集合数据。
//
// 该接口提供了添加成员、获取所有成员、移除指定成员以及删除整个集合的方法，适用于需要存储标签、权限或去重数据的场景。
//
// 泛型参数：
//   - K: 集合键的类型，用于标识特定的集合。
//   - V: 集合成员的值类型。
//
// Add 方法将一个或多个成员添加到集合中，已存在的成员会被忽略。
// Members 方法获取集合中的所有成员。
// IsMember 方法检查指定成员是否存在于集合中。
// Count 方法获取集合中的成员数量。
// Remove 方法从集合中移除指定的成员。
// Delete 方法删除整个集合。
type SetCache[K any, V any] interface {
	Add(ctx context.Context, key K, members ...V) error
	Members(ctx context.Context, key K) ([]V, error)
	IsMember(ctx context.Context, key K, member V) (bool, error)
	Count(ctx context.Context, key K) (int64, error)
	Remove(ctx context.Context, key K, members ...V) error
	Delete(ctx context.Context, key K) error
}

// ListCache 定义了基于列表（List）数据结构的缓存操作接口，用于管理有序且允许重复元素的列表数据。
//
// 该接口提供了头尾添加、范围查询、索引访问、弹出元素等方法，适用于需要维护顺序或实现队列/栈的场景。
//
// 泛型参数：
//   - K: 列表键的类型，用于标识特定的列表。
//   - V: 列表元素的值类型。
//
// Prepend 方法将一个或多个值插入到列表头部（左侧）。
// Append 方法将一个或多个值追加到列表尾部（右侧）。
// Range 方法按索引范围获取列表元素，支持负数索引（-1 表示最后一个元素）。
// Index 方法获取指定索引位置的元素，支持负数索引。
// Len 方法获取列表的长度。
// Pop 方法从列表头部弹出一个元素并返回。
// PopLast 方法从列表尾部弹出一个元素并返回。
// Remove 方法从列表中移除指定数量的匹配元素，count > 0 从头部开始，count < 0 从尾部开始，count = 0 移除所有匹配项。
// Delete 方法删除整个列表。
type ListCache[K any, V any] interface {
	Prepend(ctx context.Context, key K, values ...V) error
	Append(ctx context.Context, key K, values ...V) error
	Range(ctx context.Context, key K, start int64, end int64) ([]V, error)
	Index(ctx context.Context, key K, index int64) (*V, error)
	Len(ctx context.Context, key K) (int64, error)
	Pop(ctx context.Context, key K) (*V, error)
	PopLast(ctx context.Context, key K) (*V, error)
	Remove(ctx context.Context, key K, count int64, value V) error
	Delete(ctx context.Context, key K) error
}
