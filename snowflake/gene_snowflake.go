package xSnowflake

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// 基因雪花算法位数分配常量
const (
	// 基因雪花算法位数分配
	geneBits           uint8 = 6  // 基因位数（支持 64 种业务类型）
	geneDatacenterBits uint8 = 3  // 数据中心位数（支持 8 个数据中心）
	geneNodeBits       uint8 = 3  // 节点位数（支持 8 个节点）
	geneSequenceBits   uint8 = 10 // 序列号位数（每毫秒 1024 个）

	// 最大值
	maxGene             int64 = -1 ^ (-1 << geneBits)           // 63
	maxGeneDatacenterID int64 = -1 ^ (-1 << geneDatacenterBits) // 7
	maxGeneNodeID       int64 = -1 ^ (-1 << geneNodeBits)       // 7
	maxGeneSequence     int64 = -1 ^ (-1 << geneSequenceBits)   // 1023

	// 位移量
	geneSequenceShift   = 0
	geneNodeShift       = geneSequenceBits                                                // 10
	geneDatacenterShift = geneSequenceBits + geneNodeBits                                 // 13
	geneShift           = geneSequenceBits + geneNodeBits + geneDatacenterBits            // 16
	geneTimestampShift  = geneSequenceBits + geneNodeBits + geneDatacenterBits + geneBits // 22
)

// GeneNode 基因雪花算法节点
//
// 每个节点实例负责生成唯一的基因雪花 ID。
// 相比标准雪花 ID，基因雪花 ID 在 ID 中嵌入了业务类型基因，
// 便于从 ID 中直接识别数据类型。
type GeneNode struct {
	mu           sync.Mutex
	datacenterID int64
	nodeID       int64
	lastTime     int64
	sequence     int64
}

// NewGeneNode 创建新的基因雪花算法节点
//
// 参数说明:
//   - datacenterID: 数据中心 ID (0-7)
//   - nodeID: 节点 ID (0-7)
//
// 返回值:
//   - *GeneNode: 基因雪花算法节点实例
//   - error: 如果参数超出范围则返回错误
func NewGeneNode(datacenterID, nodeID int64) (*GeneNode, error) {
	if datacenterID < 0 || datacenterID > maxGeneDatacenterID {
		return nil, fmt.Errorf("数据中心 ID 必须在 0-%d 之间，当前值: %d", maxGeneDatacenterID, datacenterID)
	}
	if nodeID < 0 || nodeID > maxGeneNodeID {
		return nil, fmt.Errorf("节点 ID 必须在 0-%d 之间，当前值: %d", maxGeneNodeID, nodeID)
	}
	return &GeneNode{
		datacenterID: datacenterID,
		nodeID:       nodeID,
	}, nil
}

// Generate 生成新的基因雪花 ID
//
// 该方法是线程安全的，可以在并发环境中使用。
//
// 参数说明:
//   - gene: 业务基因类型 (0-63)
//
// 返回值:
//   - GeneSnowflakeID: 生成的基因雪花 ID
//   - error: 如果基因类型超出范围则返回错误
func (n *GeneNode) Generate(gene Gene) (GeneSnowflakeID, error) {
	if !gene.IsValid() {
		return 0, fmt.Errorf("基因类型必须在 0-%d 之间，当前值: %d", maxGene, gene)
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	now := time.Now().UnixMilli()

	if now == n.lastTime {
		n.sequence = (n.sequence + 1) & maxGeneSequence
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

	id := ((now - epoch) << geneTimestampShift) |
		(int64(gene) << geneShift) |
		(n.datacenterID << geneDatacenterShift) |
		(n.nodeID << geneNodeShift) |
		n.sequence

	return GeneSnowflakeID(id), nil
}

// MustGenerate 生成新的基因雪花 ID，如果发生错误则 panic
//
// 参数说明:
//   - gene: 业务基因类型 (0-63)
//
// 返回值:
//   - GeneSnowflakeID: 生成的基因雪花 ID
func (n *GeneNode) MustGenerate(gene Gene) GeneSnowflakeID {
	id, err := n.Generate(gene)
	if err != nil {
		panic(err)
	}
	return id
}

// DatacenterID 返回节点的数据中心 ID
//
// 返回值:
//   - int64: 数据中心 ID
func (n *GeneNode) DatacenterID() int64 {
	return n.datacenterID
}

// NodeID 返回节点的节点 ID
//
// 返回值:
//   - int64: 节点 ID
func (n *GeneNode) NodeID() int64 {
	return n.nodeID
}

// GeneSnowflakeID 基因雪花 ID 类型
//
// 64 位结构:
//   - 1 位符号位（不使用）
//   - 41 位时间戳（毫秒级，约 69 年有效期）
//   - 6 位基因（支持 64 种业务类型）
//   - 3 位数据中心 ID（支持 8 个数据中心）
//   - 3 位节点 ID（支持 8 个节点）
//   - 10 位序列号（每毫秒 1024 个 ID）
//
// 在标准雪花 ID 基础上嵌入业务类型基因，便于从 ID 识别数据类型。
// JSON 序列化为字符串格式，避免 JavaScript 中的精度丢失问题。
type GeneSnowflakeID int64

// MarshalJSON 将基因雪花 ID 序列化为 JSON 字符串
//
// 返回值:
//   - []byte: JSON 格式的字符串（带引号）
//   - error: 序列化错误
func (s *GeneSnowflakeID) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatInt(int64(*s), 10))
}

// UnmarshalJSON 从 JSON 反序列化基因雪花 ID
//
// 支持字符串格式和数字格式的反序列化。
//
// 参数说明:
//   - data: JSON 数据
//
// 返回值:
//   - error: 反序列化错误
func (s *GeneSnowflakeID) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		// 兼容数字格式
		var num int64
		if err := json.Unmarshal(data, &num); err != nil {
			return fmt.Errorf("解析基因雪花 ID 失败: %w", err)
		}
		*s = GeneSnowflakeID(num)
		return nil
	}
	num, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return fmt.Errorf("解析基因雪花 ID 字符串失败: %w", err)
	}
	*s = GeneSnowflakeID(num)
	return nil
}

// Value 实现 driver.Valuer 接口，用于 GORM 写入数据库
//
// 返回值:
//   - driver.Value: 数据库驱动值（int64）
//   - error: 始终为 nil
func (s *GeneSnowflakeID) Value() (driver.Value, error) {
	return int64(*s), nil
}

// Scan 实现 sql.Scanner 接口，用于 GORM 从数据库读取
//
// 参数说明:
//   - value: 数据库返回的值
//
// 返回值:
//   - error: 扫描错误
func (s *GeneSnowflakeID) Scan(value interface{}) error {
	if value == nil {
		*s = 0
		return nil
	}
	switch v := value.(type) {
	case int64:
		*s = GeneSnowflakeID(v)
	case []byte:
		num, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return fmt.Errorf("扫描基因雪花 ID 失败: %w", err)
		}
		*s = GeneSnowflakeID(num)
	default:
		return fmt.Errorf("不支持的基因雪花 ID 类型: %T", value)
	}
	return nil
}

// Int64 返回基因雪花 ID 的 int64 值
//
// 返回值:
//   - int64: ID 的数值形式
func (s *GeneSnowflakeID) Int64() int64 {
	return int64(*s)
}

// String 返回基因雪花 ID 的字符串表示
//
// 返回值:
//   - string: ID 的字符串形式
func (s *GeneSnowflakeID) String() string {
	return strconv.FormatInt(int64(*s), 10)
}

// IsZero 判断是否为零值
//
// 返回值:
//   - bool: 如果为零值返回 true
func (s *GeneSnowflakeID) IsZero() bool {
	return *s == 0
}

// Timestamp 提取时间戳
//
// 返回值:
//   - time.Time: ID 创建时间
func (s *GeneSnowflakeID) Timestamp() time.Time {
	ms := (int64(*s) >> geneTimestampShift) + epoch
	return time.UnixMilli(ms)
}

// Gene 提取业务基因类型
//
// 返回值:
//   - Gene: 业务基因类型
func (s *GeneSnowflakeID) Gene() Gene {
	return Gene((int64(*s) >> geneShift) & maxGene)
}

// DatacenterID 提取数据中心 ID
//
// 返回值:
//   - int64: 数据中心 ID
func (s *GeneSnowflakeID) DatacenterID() int64 {
	return (int64(*s) >> geneDatacenterShift) & maxGeneDatacenterID
}

// NodeID 提取节点 ID
//
// 返回值:
//   - int64: 节点 ID
func (s *GeneSnowflakeID) NodeID() int64 {
	return (int64(*s) >> geneNodeShift) & maxGeneNodeID
}

// Sequence 提取序列号
//
// 返回值:
//   - int64: 序列号
func (s *GeneSnowflakeID) Sequence() int64 {
	return int64(*s) & maxGeneSequence
}

// ToSnowflakeID 转换为标准 SnowflakeID
//
// 注意：此转换仅保留 ID 的数值，基因信息在解析时会被忽略。
// 如果需要保留业务类型信息，请继续使用 GeneSnowflakeID 类型。
//
// 返回值:
//   - SnowflakeID: 转换后的标准雪花 ID
func (s *GeneSnowflakeID) ToSnowflakeID() SnowflakeID {
	return SnowflakeID(*s)
}
