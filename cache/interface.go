package xCache

import "github.com/gin-gonic/gin"

// HashCache 定义了基于哈希（Hash）数据结构的缓存操作接口，用于管理任意类型的键值对数据。
//
// 该接口提供了获取、设置和删除缓存项的基本能力，适用于需要快速存取单个对象或配置项的场景。
//
// 泛型参数：
//   - K: 缓存键的类型，用于标识特定的缓存项。
//   - V: 缓存值的类型。
//
// Get 方法根据键检索值，返回指向值的指针、是否存在以及可能的错误。
// Set 方法将键值对存入缓存，需处理 value 为 nil 的场景。
// Delete 方法从缓存中移除指定的键。
type HashCache[K any, V any] interface {
	Get(ctx *gin.Context, key K) (*V, bool, error)
	Set(ctx *gin.Context, key K, value *V) error
	Delete(ctx *gin.Context, key K) error
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
// Remove 方法从集合中移除指定的成员。
// DeleteSet 方法删除整个集合。
type SetCache[K any, V any] interface {
	Add(ctx *gin.Context, key K, members ...V) error
	Members(ctx *gin.Context, key K) ([]V, error)
	Remove(ctx *gin.Context, key K, members ...V) error
	DeleteSet(ctx *gin.Context, key K) error
}

// ListCache 定义了基于列表（List）数据结构的缓存操作接口，用于管理有序且允许重复元素的列表数据。
//
// 该接口提供了添加元素、范围查询、移除指定元素以及删除整个列表的方法，适用于需要维护顺序或实现队列/栈的场景。
//
// 泛型参数：
//   - K: 列表键的类型，用于标识特定的列表。
//   - V: 列表元素的值类型。
//
// Add 方法将一个或多个值追加到列表末尾。
// Range 方法按索引范围获取列表元素，支持负数索引（-1 表示最后一个元素）。
// Remove 方法从列表中移除指定数量的匹配元素，count > 0 从头部开始，count < 0 从尾部开始，count = 0 移除所有匹配项。
// DeleteList 方法删除整个列表。
type ListCache[K any, V any] interface {
	Add(ctx *gin.Context, key K, values ...V) error
	Range(ctx *gin.Context, key K, start int64, end int64) ([]V, error)
	Remove(ctx *gin.Context, key K, count int64, value V) error
	DeleteList(ctx *gin.Context, key K) error
}
