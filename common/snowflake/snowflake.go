package xSnowflake

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// 雪花算法配置常量
const (
	// epoch 自定义纪元时间：2023-07-25 00:00:00 UTC
	// 从此时间开始计算时间戳，可用约 69 年
	epoch int64 = 1690214400000

	// 位数分配
	geneBits       uint8 = 6  // 基因位数（支持 64 种业务类型）
	datacenterBits uint8 = 3  // 数据中心位数（支持 8 个数据中心）
	nodeBits       uint8 = 3  // 节点位数（支持 8 个节点）
	sequenceBits   uint8 = 10 // 序列号位数（每毫秒 1024 个）

	// 最大值
	maxGene         int64 = -1 ^ (-1 << geneBits)       // 63
	maxDatacenterID int64 = -1 ^ (-1 << datacenterBits) // 7
	maxNodeID       int64 = -1 ^ (-1 << nodeBits)       // 7
	maxSequence     int64 = -1 ^ (-1 << sequenceBits)   // 1023

	// 位移量
	sequenceShift   = 0
	nodeShift       = sequenceBits                                        // 10
	datacenterShift = sequenceBits + nodeBits                             // 13
	geneShift       = sequenceBits + nodeBits + datacenterBits            // 16
	timestampShift  = sequenceBits + nodeBits + datacenterBits + geneBits // 22
)

// Node 雪花算法节点
//
// 每个节点实例负责生成唯一的雪花 ID。
// 节点通过数据中心 ID 和节点 ID 的组合来保证分布式环境下的唯一性。
// ID 中可嵌入业务类型基因，便于从 ID 中直接识别数据类型。
type Node struct {
	mu           sync.Mutex
	datacenterID int64
	nodeID       int64
	lastTime     int64
	sequence     int64
}

// NewNode 创建新的雪花算法节点
//
// 参数说明:
//   - datacenterID: 数据中心 ID (0-7)
//   - nodeID: 节点 ID (0-7)
//
// 返回值:
//   - *Node: 雪花算法节点实例
//   - error: 如果参数超出范围则返回错误
func NewNode(datacenterID, nodeID int64) (*Node, error) {
	if datacenterID < 0 || datacenterID > maxDatacenterID {
		return nil, fmt.Errorf("数据中心 ID 必须在 0-%d 之间，当前值: %d", maxDatacenterID, datacenterID)
	}
	if nodeID < 0 || nodeID > maxNodeID {
		return nil, fmt.Errorf("节点 ID 必须在 0-%d 之间，当前值: %d", maxNodeID, nodeID)
	}
	return &Node{
		datacenterID: datacenterID,
		nodeID:       nodeID,
	}, nil
}

// Generate 生成新的雪花 ID
//
// 该方法是线程安全的，可以在并发环境中使用。
//
// 参数说明:
//   - gene: 业务基因类型 (0-63)，可选，默认为 GeneDefault(0)
//
// 返回值:
//   - SnowflakeID: 生成的雪花 ID
//   - error: 如果基因类型超出范围则返回错误
func (n *Node) Generate(gene ...Gene) (SnowflakeID, error) {
	g := GeneDefault
	if len(gene) > 0 {
		g = gene[0]
	}

	if !g.IsValid() {
		return 0, fmt.Errorf("基因类型必须在 0-%d 之间，当前值: %d", maxGene, g)
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	now := time.Now().UnixMilli()

	if now == n.lastTime {
		n.sequence = (n.sequence + 1) & maxSequence
		if n.sequence == 0 {
			// 当前毫秒内序列号用尽，等待下一毫秒
			for now <= n.lastTime {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		n.sequence = 0
	}

	n.lastTime = now

	id := ((now - epoch) << timestampShift) |
		(int64(g) << geneShift) |
		(n.datacenterID << datacenterShift) |
		(n.nodeID << nodeShift) |
		n.sequence

	return SnowflakeID(id), nil
}

// MustGenerate 生成新的雪花 ID，如果发生错误则 panic
//
// 参数说明:
//   - gene: 业务基因类型 (0-63)，可选，默认为 GeneDefault(0)
//
// 返回值:
//   - SnowflakeID: 生成的雪花 ID
func (n *Node) MustGenerate(gene ...Gene) SnowflakeID {
	id, err := n.Generate(gene...)
	if err != nil {
		panic(err)
	}
	return id
}

// DatacenterID 返回节点的数据中心 ID
//
// 返回值:
//   - int64: 数据中心 ID
func (n *Node) DatacenterID() int64 {
	return n.datacenterID
}

// NodeID 返回节点的节点 ID
//
// 返回值:
//   - int64: 节点 ID
func (n *Node) NodeID() int64 {
	return n.nodeID
}

// SnowflakeID 雪花 ID 类型
//
// 64 位结构:
//   - 1 位符号位（不使用）
//   - 41 位时间戳（毫秒级，约 69 年有效期）
//   - 6 位基因（支持 64 种业务类型，Gene=0 时等同于普通 ID）
//   - 3 位数据中心 ID（支持 8 个数据中心）
//   - 3 位节点 ID（支持 8 个节点）
//   - 10 位序列号（每毫秒 1024 个 ID）
//
// 在标准雪花 ID 基础上嵌入业务类型基因，便于从 ID 识别数据类型。
// JSON 序列化为字符串格式，避免 JavaScript 中的精度丢失问题。
type SnowflakeID int64

// MarshalJSON 将雪花 ID 序列化为 JSON 字符串
//
// 返回值:
//   - []byte: JSON 格式的字符串（带引号）
//   - error: 序列化错误
func (s SnowflakeID) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatInt(int64(s), 10))
}

// UnmarshalJSON 从 JSON 反序列化雪花 ID
//
// 支持字符串格式和数字格式的反序列化。
//
// 参数说明:
//   - data: JSON 数据
//
// 返回值:
//   - error: 反序列化错误
func (s *SnowflakeID) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		// 兼容数字格式
		var num int64
		if err := json.Unmarshal(data, &num); err != nil {
			return fmt.Errorf("解析雪花 ID 失败: %w", err)
		}
		*s = SnowflakeID(num)
		return nil
	}
	num, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return fmt.Errorf("解析雪花 ID 字符串失败: %w", err)
	}
	*s = SnowflakeID(num)
	return nil
}

// MarshalBinary 实现 encoding.BinaryMarshaler 接口
//
// 将雪花 ID 序列化为字节数组，使用字符串格式存储。
// 适用于需要二进制序列化的场景（如缓存、消息队列等）。
//
// 注意: 使用值接收者以确保值类型也能被序列化（Redis 等场景需要）
//
// 返回值:
//   - []byte: 字节数组形式的 ID
//   - error: 序列化错误（当前实现始终返回 nil）
func (s SnowflakeID) MarshalBinary() ([]byte, error) {
	return []byte(s.String()), nil
}

// UnmarshalBinary 实现 encoding.BinaryUnmarshaler 接口
//
// 从字节数组反序列化雪花 ID。
// 字节数组应为十进制字符串格式。
//
// 参数说明:
//   - data: 字节数组形式的 ID 数据
//
// 返回值:
//   - error: 反序列化错误
func (s *SnowflakeID) UnmarshalBinary(data []byte) error {
	val, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return fmt.Errorf("解析雪花 ID 字节数据失败: %w", err)
	}
	*s = SnowflakeID(val)
	return nil
}

// Value 实现 driver.Valuer 接口，用于 GORM 写入数据库
//
// 返回值:
//   - driver.Value: 数据库驱动值（int64）
//   - error: 始终为 nil
func (s SnowflakeID) Value() (driver.Value, error) {
	return int64(s), nil
}

// Scan 实现 sql.Scanner 接口，用于 GORM 从数据库读取
//
// 参数说明:
//   - value: 数据库返回的值
//
// 返回值:
//   - error: 扫描错误
func (s *SnowflakeID) Scan(value interface{}) error {
	if value == nil {
		*s = 0
		return nil
	}
	switch v := value.(type) {
	case int64:
		*s = SnowflakeID(v)
	case []byte:
		num, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return fmt.Errorf("扫描雪花 ID 失败: %w", err)
		}
		*s = SnowflakeID(num)
	default:
		return fmt.Errorf("不支持的雪花 ID 类型: %T", value)
	}
	return nil
}

// Int64 返回雪花 ID 的 int64 值
//
// 返回值:
//   - int64: ID 的数值形式
func (s SnowflakeID) Int64() int64 {
	return int64(s)
}

// String 返回雪花 ID 的字符串表示
//
// 返回值:
//   - string: ID 的字符串形式
func (s SnowflakeID) String() string {
	return strconv.FormatInt(int64(s), 10)
}

// IsZero 判断是否为零值
//
// 返回值:
//   - bool: 如果为零值返回 true
func (s SnowflakeID) IsZero() bool {
	return s == 0
}

// Timestamp 提取时间戳
//
// 返回值:
//   - time.Time: ID 创建时间
func (s SnowflakeID) Timestamp() time.Time {
	ms := (int64(s) >> timestampShift) + epoch
	return time.UnixMilli(ms)
}

// Gene 提取业务基因类型
//
// 返回值:
//   - Gene: 业务基因类型
func (s SnowflakeID) Gene() Gene {
	return Gene((int64(s) >> geneShift) & maxGene)
}

// DatacenterID 提取数据中心 ID
//
// 返回值:
//   - int64: 数据中心 ID
func (s SnowflakeID) DatacenterID() int64 {
	return (int64(s) >> datacenterShift) & maxDatacenterID
}

// NodeID 提取节点 ID
//
// 返回值:
//   - int64: 节点 ID
func (s SnowflakeID) NodeID() int64 {
	return (int64(s) >> nodeShift) & maxNodeID
}

// Sequence 提取序列号
//
// 返回值:
//   - int64: 序列号
func (s SnowflakeID) Sequence() int64 {
	return int64(s) & maxSequence
}
