package xCtx

// ContextNode 表示构成上下文树结构的基本节点单元，用于存储键值对形式的数据。
//
// Key 字段标识数据的类型或属性名称，Value 字段存储关联的具体数据内容。
type ContextNode struct {
	Key   ContextKey // Key 用于标识上下文节点的类型或属性名称，区分不同的数据项。
	Value any        // Value 存储与上下文节点关联的实际数据，可以是任何类型。
}

// ContextNodeList 表示上下文节点 Key-Value 对的有序集合，用于构建树形结构的链式路径或存储列表数据。
type ContextNodeList []ContextNode

// NewCtxNodeList 创建并初始化一个空的上下文节点列表，用于存储有序的键值对数据。
func NewCtxNodeList() ContextNodeList {
	return make(ContextNodeList, 0)
}

// GetList 返回底层的节点切片，提供对有序集合的直接访问。
func (c ContextNodeList) GetList() []ContextNode {
	return c
}

// Get 获取 ContextNode 中存储的数据值并返回。
func (c ContextNodeList) Get(key ContextKey) any {
	for _, node := range c {
		if node.Key == key {
			return node.Value
		}
	}
	return nil
}

// Append 将指定的键值对节点追加到列表末尾。
//
// 参数 key 表示节点的键，用于标识数据；参数 value 表示节点的值。
func (c *ContextNodeList) Append(key ContextKey, value any) {
	*c = append(*c, ContextNode{Key: key, Value: value})
}
