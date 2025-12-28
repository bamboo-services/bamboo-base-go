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
	// epoch 自定义纪元时间：2024-01-01 00:00:00 UTC
	// 从此时间开始计算时间戳，可用约 69 年
	epoch int64 = 1704067200000

	// 标准雪花算法位数分配
	datacenterBits uint8 = 5  // 数据中心位数
	nodeBits       uint8 = 5  // 节点位数
	sequenceBits   uint8 = 12 // 序列号位数

	// 最大值
	maxDatacenterID int64 = -1 ^ (-1 << datacenterBits) // 31
	maxNodeID       int64 = -1 ^ (-1 << nodeBits)       // 31
	maxSequence     int64 = -1 ^ (-1 << sequenceBits)   // 4095

	// 位移量
	nodeShift       = sequenceBits                             // 12
	datacenterShift = sequenceBits + nodeBits                  // 17
	timestampShift  = sequenceBits + nodeBits + datacenterBits // 22
)

// Node 雪花算法节点
//
// 每个节点实例负责生成唯一的雪花 ID。
// 节点通过数据中心 ID 和节点 ID 的组合来保证分布式环境下的唯一性。
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
//   - datacenterID: 数据中心 ID (0-31)
//   - nodeID: 节点 ID (0-31)
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
// 如果同一毫秒内序列号用尽，会等待到下一毫秒再生成。
//
// 返回值:
//   - SnowflakeID: 生成的雪花 ID
func (n *Node) Generate() SnowflakeID {
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
		(n.datacenterID << datacenterShift) |
		(n.nodeID << nodeShift) |
		n.sequence

	return SnowflakeID(id)
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
//   - 5 位数据中心 ID（支持 32 个数据中心）
//   - 5 位节点 ID（支持 32 个节点）
//   - 12 位序列号（每毫秒 4096 个 ID）
//
// JSON 序列化为字符串格式，避免 JavaScript 中的精度丢失问题。
// 实现了 driver.Valuer 和 sql.Scanner 接口，支持 GORM 数据库操作。
type SnowflakeID int64

// MarshalJSON 将雪花 ID 序列化为 JSON 字符串
//
// 序列化为字符串格式以避免 JavaScript 的 52 位精度限制。
//
// 返回值:
//   - []byte: JSON 格式的字符串（带引号）
//   - error: 序列化错误
func (s *SnowflakeID) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatInt(int64(*s), 10))
}

// UnmarshalJSON 从 JSON 反序列化雪花 ID
//
// 支持字符串格式和数字格式的反序列化，提供向后兼容性。
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

// Value 实现 driver.Valuer 接口，用于 GORM 写入数据库
//
// 返回值:
//   - driver.Value: 数据库驱动值（int64）
//   - error: 始终为 nil
func (s *SnowflakeID) Value() (driver.Value, error) {
	return int64(*s), nil
}

// Scan 实现 sql.Scanner 接口，用于 GORM 从数据库读取
//
// 支持 int64 和 []byte 两种数据库返回类型。
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
func (s *SnowflakeID) Int64() int64 {
	return int64(*s)
}

// String 返回雪花 ID 的字符串表示
//
// 返回值:
//   - string: ID 的字符串形式
func (s *SnowflakeID) String() string {
	return strconv.FormatInt(int64(*s), 10)
}

// IsZero 判断雪花 ID 是否为零值
//
// 返回值:
//   - bool: 如果为零值返回 true
func (s *SnowflakeID) IsZero() bool {
	return *s == 0
}

// Timestamp 提取雪花 ID 中的时间戳
//
// 返回值:
//   - time.Time: ID 创建时间
func (s *SnowflakeID) Timestamp() time.Time {
	ms := (int64(*s) >> timestampShift) + epoch
	return time.UnixMilli(ms)
}

// DatacenterID 提取雪花 ID 中的数据中心 ID
//
// 返回值:
//   - int64: 数据中心 ID
func (s *SnowflakeID) DatacenterID() int64 {
	return (int64(*s) >> datacenterShift) & maxDatacenterID
}

// NodeID 提取雪花 ID 中的节点 ID
//
// 返回值:
//   - int64: 节点 ID
func (s *SnowflakeID) NodeID() int64 {
	return (int64(*s) >> nodeShift) & maxNodeID
}

// Sequence 提取雪花 ID 中的序列号
//
// 返回值:
//   - int64: 序列号
func (s *SnowflakeID) Sequence() int64 {
	return int64(*s) & maxSequence
}
