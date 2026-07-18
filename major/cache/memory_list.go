package xCache

import (
	"context"
	"time"
)

// memoryListCache [ListCache] 的内存实现。
//
// 内存中以 [][]byte 存储有序元素切片，整体作为 [memoryEntry.Value] 存入 [memoryStore]。
type memoryListCache[K any, V any] struct {
	store *memoryStore
	codec Codec
	enc   KeyEncoder
	ttl   time.Duration
}

// NewMemoryListCache 构造一个基于内存的 [ListCache] 实现。
func NewMemoryListCache[K any, V any](store *memoryStore, codec Codec, enc KeyEncoder, ttl time.Duration) ListCache[K, V] {
	if codec == nil {
		codec = JSONCodec{}
	}
	return &memoryListCache[K, V]{store: store, codec: codec, enc: enc, ttl: ttl}
}

// loadOrCreate 仅用于 Get 路径（只读）。写路径必须走 [memoryStore.Update] 保证原子性。
func (c *memoryListCache[K, V]) loadOrCreate(key K) ([][]byte, bool) {
	k := EncodeKey(c.enc, key)
	if value, ok := c.store.Get(k); ok {
		if l, ok := value.([][]byte); ok {
			return l, false
		}
	}
	return nil, true
}

// normalizeIndex 把负数索引转换为正数索引（-1 表示最后一个元素）。
//
// 返回 (绝对索引, 是否越界)。
func normalizeIndex(idx int64, length int) (int, bool) {
	if idx < 0 {
		idx = int64(length) + idx
	}
	if idx < 0 || idx >= int64(length) {
		return 0, false
	}
	return int(idx), true
}

// Prepend 将一个或多个值插入到列表头部（左侧）。
//
// Prepend(k, a, b, c) 后列表头部为 [a, b, c, ...原元素]。
// Memory 后端直接操作切片，无需像 Redis LPUSH 那样反转参数。
// 通过 [memoryStore.Update] 在单把锁内完成读-改-写，避免 append 共享底层数组的并发污染。
func (c *memoryListCache[K, V]) Prepend(ctx context.Context, key K, values ...V) error {
	if len(values) == 0 {
		return nil
	}
	encoded := make([][]byte, 0, len(values))
	for _, v := range values {
		data, err := c.codec.Marshal(v)
		if err != nil {
			return err
		}
		encoded = append(encoded, data)
	}
	k := EncodeKey(c.enc, key)
	c.store.Update(k, c.ttl, func(old any) any {
		l, _ := old.([][]byte)
		// 必须新建切片，避免在共享底层数组上写
		newList := make([][]byte, 0, len(encoded)+len(l))
		newList = append(newList, encoded...)
		newList = append(newList, l...)
		return newList
	})
	return nil
}

// Append 将一个或多个值追加到列表尾部（右侧）。
//
// 通过 [memoryStore.Update] 保证原子性。新建切片避免共享底层数组污染。
func (c *memoryListCache[K, V]) Append(ctx context.Context, key K, values ...V) error {
	if len(values) == 0 {
		return nil
	}
	encoded := make([][]byte, 0, len(values))
	for _, v := range values {
		data, err := c.codec.Marshal(v)
		if err != nil {
			return err
		}
		encoded = append(encoded, data)
	}
	k := EncodeKey(c.enc, key)
	c.store.Update(k, c.ttl, func(old any) any {
		l, _ := old.([][]byte)
		newList := make([][]byte, 0, len(encoded)+len(l))
		newList = append(newList, l...)
		newList = append(newList, encoded...)
		return newList
	})
	return nil
}

// Range 按索引范围获取列表元素，支持负数索引（-1 表示最后一个元素）。
func (c *memoryListCache[K, V]) Range(ctx context.Context, key K, start int64, end int64) ([]V, error) {
	k := EncodeKey(c.enc, key)
	value, ok := c.store.Get(k)
	if !ok {
		return nil, nil
	}
	l, ok := value.([][]byte)
	if !ok {
		return nil, nil
	}
	length := len(l)
	si, _ := normalizeIndex(start, length)
	ei, _ := normalizeIndex(end, length)
	if si > ei {
		return nil, nil
	}
	result := make([]V, 0, ei-si+1)
	for i := si; i <= ei; i++ {
		var v V
		if err := c.codec.Unmarshal(l[i], &v); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, nil
}

// Index 获取指定索引位置的元素，支持负数索引。越界时返回 nil, nil。
func (c *memoryListCache[K, V]) Index(ctx context.Context, key K, index int64) (*V, error) {
	k := EncodeKey(c.enc, key)
	value, ok := c.store.Get(k)
	if !ok {
		return nil, nil
	}
	l, ok := value.([][]byte)
	if !ok {
		return nil, nil
	}
	idx, ok := normalizeIndex(index, len(l))
	if !ok {
		return nil, nil
	}
	var v V
	if err := c.codec.Unmarshal(l[idx], &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Len 获取列表的长度。
func (c *memoryListCache[K, V]) Len(ctx context.Context, key K) (int64, error) {
	k := EncodeKey(c.enc, key)
	value, ok := c.store.Get(k)
	if !ok {
		return 0, nil
	}
	l, ok := value.([][]byte)
	if !ok {
		return 0, nil
	}
	return int64(len(l)), nil
}

// Pop 从列表头部弹出一个元素并返回。列表为空时返回 nil, nil。
//
// 通过 [memoryStore.Update] 保证读-改-写原子性。
func (c *memoryListCache[K, V]) Pop(ctx context.Context, key K) (*V, error) {
	k := EncodeKey(c.enc, key)
	var result *V
	c.store.Update(k, c.ttl, func(old any) any {
		l, _ := old.([][]byte)
		if len(l) == 0 {
			return nil // 空列表删除 key（若存在）
		}
		data := l[0]
		// 反序列化在锁外做以避免 codec 慢导致拖累分片；先暂存 raw
		raw := make([]byte, len(data))
		copy(raw, data)
		// 闭包内只解码，不写外部状态（result 是闭包捕获的指针）
		var v V
		if err := c.codec.Unmarshal(raw, &v); err != nil {
			// 解码失败保留列表原样，不弹出
			return l
		}
		result = &v
		newList := l[1:]
		if len(newList) == 0 {
			return nil
		}
		return newList
	})
	return result, nil
}

// PopLast 从列表尾部弹出一个元素并返回。列表为空时返回 nil, nil。
//
// 通过 [memoryStore.Update] 保证读-改-写原子性。
func (c *memoryListCache[K, V]) PopLast(ctx context.Context, key K) (*V, error) {
	k := EncodeKey(c.enc, key)
	var result *V
	c.store.Update(k, c.ttl, func(old any) any {
		l, _ := old.([][]byte)
		if len(l) == 0 {
			return nil
		}
		data := l[len(l)-1]
		raw := make([]byte, len(data))
		copy(raw, data)
		var v V
		if err := c.codec.Unmarshal(raw, &v); err != nil {
			return l
		}
		result = &v
		newList := l[:len(l)-1]
		if len(newList) == 0 {
			return nil
		}
		return newList
	})
	return result, nil
}

// Remove 从列表中移除指定数量的匹配元素。
//
// count > 0 从头部开始移除最多 count 个；count < 0 从尾部开始；count = 0 移除全部。
// 通过 [memoryStore.Update] 保证原子性。
func (c *memoryListCache[K, V]) Remove(ctx context.Context, key K, count int64, value V) error {
	target, err := c.codec.Marshal(value)
	if err != nil {
		return err
	}
	k := EncodeKey(c.enc, key)
	c.store.Update(k, c.ttl, func(old any) any {
		l, _ := old.([][]byte)
		if len(l) == 0 {
			return nil
		}
		var removed int64
		newList := make([][]byte, 0, len(l))
		if count >= 0 {
			maxRemove := count
			if count == 0 {
				maxRemove = int64(len(l))
			}
			for _, data := range l {
				if removed < maxRemove && string(data) == string(target) {
					removed++
					continue
				}
				newList = append(newList, data)
			}
		} else {
			// 从尾部开始：反向遍历就地删除
			maxRemove := -count
			for i := len(l) - 1; i >= 0 && removed < maxRemove; i-- {
				if string(l[i]) == string(target) {
					l = append(l[:i], l[i+1:]...)
					removed++
				}
			}
			newList = l
		}
		if len(newList) == 0 {
			return nil
		}
		return newList
	})
	return nil
}

// Delete 删除整个列表。
func (c *memoryListCache[K, V]) Delete(ctx context.Context, key K) error {
	c.store.Delete(EncodeKey(c.enc, key))
	return nil
}
