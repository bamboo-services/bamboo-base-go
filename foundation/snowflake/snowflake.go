package snowflake

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/valueerr"
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

// 节点发号与反序列化过程中的导出哨兵错误，调用方可通过 errors.Is 识别。
//
// 迁移到统一错误后，这三个哨兵不再直接作为返回值抛出，
// 而是作为对应统一错误的根因保留：错误码分别为
// [valueerr.InvalidReceiver]（50019，无效接收器）与 [ClockBackwards]（50027，时钟回退），
// 因此 errors.Is(err, ErrNilNode) 等判定依旧可达。
var (
	ErrNilNode        = errors.New("snowflake: nil node")              // 在未初始化的 nil [Node] 指针上调用了发号方法。
	ErrNilReceiver    = errors.New("snowflake: nil receiver")          // 在 nil 接收器指针上调用了反序列化或数据扫描方法。
	ErrClockBackwards = errors.New("snowflake: clock moved backwards") // 系统时钟发生回退，无法保证发号的时间单调递增性。
)

// Node 是 64 位分布式基因雪花算法的发号节点。
//
// 每个节点实例通过数据中心 ID 与节点 ID 的唯一组合保证分布式发号的全局互斥性，
// 支持在发号时嵌入 6 位业务基因，并内置并发锁确保多 goroutine 调用下的发号安全性。
//
// 零值 Node 不具备有效时钟状态，直接调用发号方法将受到未校验限制，建议始终通过 [NewNode] 构造。
type Node struct {
	mu           sync.Mutex
	datacenterID int64
	nodeID       int64
	lastTime     int64
	sequence     int64
	clockFunc    func() int64 // 私有可替换时钟，仅用于测试；生产默认为 nil (调用 time.Now().UnixMilli())
}

// NewNode 创建并初始化一个雪花算法发号节点实例。
//
// datacenterID 与 nodeID 均受位宽限制，合法取值范围均为 [0, 7]。
// 任一参数超出合法范围时返回非 nil 的 [*apperr.Error]，错误码为
// [NodeInvalid]（50026），并把具体越界信息保留为根因；
// 成功构造的 [Node] 实例完全并发安全，可供应用内全局共享。
func NewNode(datacenterID, nodeID int64) (*Node, *apperr.Error) {
	if datacenterID < 0 || datacenterID > maxDatacenterID {
		return nil, apperr.New(
			NodeInvalid,
			apperr.ErrMessage(fmt.Sprintf("数据中心 ID 必须在 0-%d 之间", maxDatacenterID)),
			fmt.Errorf("数据中心 ID 必须在 0-%d 之间，当前值: %d", maxDatacenterID, datacenterID),
		)
	}
	if nodeID < 0 || nodeID > maxNodeID {
		return nil, apperr.New(
			NodeInvalid,
			apperr.ErrMessage(fmt.Sprintf("节点 ID 必须在 0-%d 之间", maxNodeID)),
			fmt.Errorf("节点 ID 必须在 0-%d 之间，当前值: %d", maxNodeID, nodeID),
		)
	}
	return &Node{
		datacenterID: datacenterID,
		nodeID:       nodeID,
	}, nil
}

func (n *Node) currentTime() int64 {
	if n.clockFunc != nil {
		return n.clockFunc()
	}
	return time.Now().UnixMilli()
}

// Generate 并发安全地生成一个新的 64 位雪花算法 ID。
//
// gene 为可选参数，表示需要嵌入 ID 的业务基因：未传或为空时默认使用 [GeneDefault]；
// 若传入的值超出合法范围 [0, 63]，将返回值为 [valueerr.OutOfRange]（40014）
// 的统一错误，原越界信息保留为根因。
//
// 在 nil 节点上调用将返回错误码为 [valueerr.InvalidReceiver]（50019）的统一错误，
// 并保留 [ErrNilNode] 为根因（errors.Is 仍可达）。
// 若当前时钟小于上一次发号时间戳，将返回错误码为 [ClockBackwards]（50027）
// 的统一错误，并保留 [ErrClockBackwards] 为根因；
// 若当前毫秒内序列号用尽，方法将主动等待下一毫秒并重置序列号后完成发号。
//
// 成功路径返回的 nil 是真正的 nil 错误（类型为 [*apperr.Error] 的零值指针），
// 调用方可用 == nil 直接判定成功。
func (n *Node) Generate(gene ...Gene) (SnowflakeID, *apperr.Error) {
	if n == nil {
		return 0, valueerr.From(valueerr.KindInvalidReceiver, "雪花发号节点未初始化", ErrNilNode)
	}

	g := GeneDefault
	if len(gene) > 0 {
		g = gene[0]
	}

	if !g.IsValid() {
		return 0, valueerr.From(
			valueerr.KindOutOfRange,
			apperr.ErrMessage(fmt.Sprintf("基因类型必须在 0-%d 之间", maxGene)),
			fmt.Errorf("基因类型必须在 0-%d 之间，当前值: %d", maxGene, g),
		)
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	now := n.currentTime()

	if now < n.lastTime {
		return 0, apperr.New(
			ClockBackwards,
			"系统时钟回退",
			fmt.Errorf("%w: current time %d < last time %d", ErrClockBackwards, now, n.lastTime),
		)
	}

	var nextSeq int64
	if now == n.lastTime {
		candidateSeq := (n.sequence + 1) & maxSequence
		if candidateSeq == 0 {
			// 当前毫秒内序列号用尽，等待下一毫秒
			for now <= n.lastTime {
				now = n.currentTime()
				if now < n.lastTime {
					return 0, apperr.New(
						ClockBackwards,
						"系统时钟回退",
						fmt.Errorf("%w: clock moved backwards while waiting next millisecond", ErrClockBackwards),
					)
				}
			}
			nextSeq = 0
		} else {
			nextSeq = candidateSeq
		}
	} else {
		nextSeq = 0
	}

	n.sequence = nextSeq
	n.lastTime = now

	id := ((now - epoch) << timestampShift) |
		(int64(g) << geneShift) |
		(n.datacenterID << datacenterShift) |
		(n.nodeID << nodeShift) |
		n.sequence

	return SnowflakeID(id), nil
}

// MustGenerate 生成新的雪花算法 ID，若发号过程中发生任何错误则直接触发 panic。
//
// 参数规则与 [Node.Generate] 一致；适用于应用启动阶段或确信参数合规且无需局部处理错误的场景。
func (n *Node) MustGenerate(gene ...Gene) SnowflakeID {
	id, err := n.Generate(gene...)
	if err != nil {
		panic(err)
	}
	return id
}

// DatacenterID 返回发号节点配置的数据中心 ID。
// 若接收器指针为 nil 则返回 0。
func (n *Node) DatacenterID() int64 {
	if n == nil {
		return 0
	}
	return n.datacenterID
}

// NodeID 返回发号节点配置的工作节点 ID。
// 若接收器指针为 nil 则返回 0。
func (n *Node) NodeID() int64 {
	if n == nil {
		return 0
	}
	return n.nodeID
}

// SnowflakeID 是嵌入业务基因的 64 位分布式唯一标识符（底层为 int64）。
//
// 二进制位段自高位至低位组织为：
//   - 1 位符号位（固定为 0）；
//   - 41 位毫秒级时间戳（基于 2023-07-25 00:00:00 UTC 自定义纪元，有效约 69 年）；
//   - 6 位业务基因（取值 0~63，标识业务类型并支持分片路由）；
//   - 3 位数据中心 ID（取值 0~7）；
//   - 3 位节点 ID（取值 0~7）；
//   - 10 位序列号（取值 0~1023，每毫秒最大 1024 个并发序号）。
//
// 零值 SnowflakeID 为 0，[SnowflakeID.IsZero] 返回 true。
// 本类型内置 JSON、数据库驱动及二进制编解码支持，JSON 格式输出为字符串避免精度丢失。
//
//nolint:revive // 保持与历史 SnowflakeID 类型名称一致
type SnowflakeID int64

// MarshalJSON 实现 [json.Marshaler] 接口，将雪花 ID 编码为带引号的十进制 JSON 字符串。
//
// 此举可防止前端 JavaScript 等弱类型环境因数值超过 2^53 - 1 产生精度丢失。
// 该方法的外部签名固定为 ([]byte, error)：内部使用强类型核心，
// 成功时返回真正的 nil 错误，失败时返回携带统一错误码的错误。
func (s SnowflakeID) MarshalJSON() ([]byte, error) {
	data, err := s.marshalJSONCore()
	if err != nil {
		return nil, err
	}
	return data, nil
}

// marshalJSONCore 是 [SnowflakeID.MarshalJSON] 的强类型核心。
//
// 内部以十进制文本编码再交给 JSON 编码器，保证输出为带引号的字符串。
func (s SnowflakeID) marshalJSONCore() ([]byte, *apperr.Error) {
	data, err := json.Marshal(strconv.FormatInt(int64(s), 10))
	if err != nil {
		return nil, valueerr.From(valueerr.KindUnsupportedTarget, "编码雪花 ID 失败", err)
	}
	return data, nil
}

// UnmarshalJSON 实现 [json.Unmarshaler] 接口，从 JSON 数据反序列化雪花 ID。
//
// 支持字符串格式（如 "123456"）以及向前兼容纯数字格式（如 123456）。
// 若接收器指针为 nil 则返回错误码为 [valueerr.InvalidReceiver]（50019）的统一错误，
// 并保留 [ErrNilReceiver] 为根因；遇到 JSON null 或非字符串/数字类型返回 [valueerr.InvalidType]（40015），
// 数值超出 int64 范围（含字符串与纯数字）返回 [valueerr.OutOfRange]（40014），其余解析失败返回
// [valueerr.InvalidFormat]（40013）；任何失败均保持接收器原值不变。
//
// 该方法的外部签名固定为 error：内部使用强类型核心，
// 成功时返回真正的 nil 错误。
func (s *SnowflakeID) UnmarshalJSON(data []byte) error {
	if err := s.unmarshalJSONCore(data); err != nil {
		return err
	}
	return nil
}

// parseDecimalID 解析十进制雪花 ID 文本，将溢出归一为 [valueerr.OutOfRange]（40014），
// 其余格式错误归一为 [valueerr.InvalidFormat]（40013）。
func parseDecimalID(raw string, invalidFormatMsg apperr.ErrMessage) (SnowflakeID, *apperr.Error) {
	num, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		if errors.Is(err, strconv.ErrRange) {
			return 0, valueerr.From(valueerr.KindOutOfRange, "雪花 ID 超出有效范围", err)
		}
		return 0, valueerr.From(valueerr.KindInvalidFormat, invalidFormatMsg, err)
	}
	return SnowflakeID(num), nil
}

// unmarshalJSONCore 是 [SnowflakeID.UnmarshalJSON] 的强类型核心。
//
// 按 JSON token 类型精确分类：null 与非字符串/数字类型映射为类型错误，
// 字符串或纯数字超出 int64 范围映射为越界错误，语法或格式不合法映射为格式错误；
// 解析失败不会污染接收器原值。
func (s *SnowflakeID) unmarshalJSONCore(data []byte) *apperr.Error {
	if s == nil {
		return valueerr.From(valueerr.KindInvalidReceiver, "雪花 ID 接收器为 nil", ErrNilReceiver)
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		var str string
		err := json.Unmarshal(data, &str)
		return valueerr.From(valueerr.KindInvalidFormat, "解析雪花 ID 失败", err)
	}
	if bytes.Equal(trimmed, []byte("null")) {
		return apperr.New(valueerr.InvalidType, "雪花 ID 不接受 JSON null")
	}

	switch trimmed[0] {
	case '"':
		var str string
		if err := json.Unmarshal(trimmed, &str); err != nil {
			return valueerr.From(valueerr.KindInvalidFormat, "解析雪花 ID 字符串失败", err)
		}
		id, appErr := parseDecimalID(str, "解析雪花 ID 字符串失败")
		if appErr != nil {
			return appErr
		}
		*s = id
		return nil

	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		var num int64
		if err := json.Unmarshal(trimmed, &num); err != nil {
			var typeErr *json.UnmarshalTypeError
			if errors.As(err, &typeErr) {
				if _, parseErr := strconv.ParseInt(string(trimmed), 10, 64); errors.Is(parseErr, strconv.ErrRange) {
					return apperr.New(valueerr.OutOfRange, "雪花 ID 超出有效范围", err, parseErr)
				}
			}
			return valueerr.From(valueerr.KindInvalidFormat, "解析雪花 ID 失败", err)
		}
		*s = SnowflakeID(num)
		return nil

	default:
		var str string
		err := json.Unmarshal(trimmed, &str)
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) {
			return valueerr.From(valueerr.KindInvalidFormat, "解析雪花 ID 失败", err)
		}
		return valueerr.From(valueerr.KindInvalidType, "雪花 ID 仅接受字符串或整数", err)
	}
}

// MarshalBinary 实现 [encoding.BinaryMarshaler] 接口，将雪花 ID 转换为十进制字符串格式的字节切片。
//
// 返回的 error 始终为 nil。
// 该方法的外部签名固定为 ([]byte, error)：内部使用强类型核心，
// 成功时返回真正的 nil 错误。
func (s SnowflakeID) MarshalBinary() ([]byte, error) {
	data, err := s.marshalBinaryCore()
	if err != nil {
		return nil, err
	}
	return data, nil
}

// marshalBinaryCore 是 [SnowflakeID.MarshalBinary] 的强类型核心。
func (s SnowflakeID) marshalBinaryCore() ([]byte, *apperr.Error) {
	return []byte(s.String()), nil
}

// UnmarshalBinary 实现 [encoding.BinaryUnmarshaler] 接口，从十进制字节切片反序列化雪花 ID。
//
// 若接收器指针为 nil 则返回错误码为 [valueerr.InvalidReceiver]（50019）的统一错误，
// 并保留 [ErrNilReceiver] 为根因；数值超出 int64 范围返回 [valueerr.OutOfRange]（40014），
// 其余解析错误返回 [valueerr.InvalidFormat]（40013），任何失败均保持原有值不变。
//
// 该方法的外部签名固定为 error：内部使用强类型核心，
// 成功时返回真正的 nil 错误。
func (s *SnowflakeID) UnmarshalBinary(data []byte) error {
	if err := s.unmarshalBinaryCore(data); err != nil {
		return err
	}
	return nil
}

// unmarshalBinaryCore 是 [SnowflakeID.UnmarshalBinary] 的强类型核心。
func (s *SnowflakeID) unmarshalBinaryCore(data []byte) *apperr.Error {
	if s == nil {
		return valueerr.From(valueerr.KindInvalidReceiver, "雪花 ID 接收器为 nil", ErrNilReceiver)
	}
	id, appErr := parseDecimalID(string(data), "解析雪花 ID 字节数据失败")
	if appErr != nil {
		return appErr
	}
	*s = id
	return nil
}

// Value 实现 [driver.Valuer] 接口，转换为 int64 原生数值写入数据库。
//
// 返回的 error 始终为 nil。
// 该方法的外部签名固定为 (driver.Value, error)：内部使用强类型核心，
// 成功时返回真正的 nil 错误。
func (s SnowflakeID) Value() (driver.Value, error) {
	value, err := s.valueCore()
	if err != nil {
		return nil, err
	}
	return value, nil
}

// valueCore 是 [SnowflakeID.Value] 的强类型核心。
func (s SnowflakeID) valueCore() (driver.Value, *apperr.Error) {
	return int64(s), nil
}

// Scan 实现 [database/sql.Scanner] 接口，从数据库驱动值扫描还原雪花 ID。
//
// 支持 int64、int、int32 数值类型，以及 []byte、string 十进制文本类型。
// 当传入 value 为 nil 时，将接收器重置为 0 并返回 nil；
// 若接收器指针为 nil 则返回错误码为 [valueerr.InvalidReceiver]（50019）的统一错误，
// 并保留 [ErrNilReceiver] 为根因；
// 遇到不支持的类型（错误码 [valueerr.InvalidType]，40015）、数值越界
// （错误码 [valueerr.OutOfRange]，40014）或文本解析失败
// （错误码 [valueerr.InvalidFormat]，40013）时保留原值并返回错误。
//
// 该方法的外部签名固定为 error：内部使用强类型核心，
// 成功时返回真正的 nil 错误。
func (s *SnowflakeID) Scan(value interface{}) error {
	if err := s.scanCore(value); err != nil {
		return err
	}
	return nil
}

// scanCore 是 [SnowflakeID.Scan] 的强类型核心。
func (s *SnowflakeID) scanCore(value interface{}) *apperr.Error {
	if s == nil {
		return valueerr.From(valueerr.KindInvalidReceiver, "雪花 ID 接收器为 nil", ErrNilReceiver)
	}
	if value == nil {
		*s = 0
		return nil
	}
	switch v := value.(type) {
	case int64:
		*s = SnowflakeID(v)
	case int:
		*s = SnowflakeID(v)
	case int32:
		*s = SnowflakeID(v)
	case []byte:
		id, appErr := parseDecimalID(string(v), "扫描雪花 ID 失败")
		if appErr != nil {
			return appErr
		}
		*s = id
	case string:
		id, appErr := parseDecimalID(v, "扫描雪花 ID 失败")
		if appErr != nil {
			return appErr
		}
		*s = id
	default:
		return apperr.New(
			valueerr.InvalidType,
			"不支持的雪花 ID 类型",
			fmt.Errorf("不支持的雪花 ID 类型: %T", value),
		)
	}
	return nil
}

// Int64 将雪花 ID 显式转换为原生的 int64 数值。
func (s SnowflakeID) Int64() int64 {
	return int64(s)
}

// String 返回雪花 ID 的十进制字符串表示。
func (s SnowflakeID) String() string {
	return strconv.FormatInt(int64(s), 10)
}

// IsZero 报告雪花 ID 是否为零值（数值为 0）。
func (s SnowflakeID) IsZero() bool {
	return s == 0
}

// Timestamp 解析并提取雪花 ID 中的时间戳位段，还原为毫秒精度的 [time.Time]。
func (s SnowflakeID) Timestamp() time.Time {
	ms := (int64(s) >> timestampShift) + epoch
	return time.UnixMilli(ms)
}

// Gene 解析并提取雪花 ID 中的 6 位业务基因，返回对应的 [Gene] 类型。
func (s SnowflakeID) Gene() Gene {
	return Gene((int64(s) >> geneShift) & maxGene)
}

// DatacenterID 解析并提取雪花 ID 中的数据中心 ID 位段（范围 0~7）。
func (s SnowflakeID) DatacenterID() int64 {
	return (int64(s) >> datacenterShift) & maxDatacenterID
}

// NodeID 解析并提取雪花 ID 中的发号工作节点 ID 位段（范围 0~7）。
func (s SnowflakeID) NodeID() int64 {
	return (int64(s) >> nodeShift) & maxNodeID
}

// Sequence 解析并提取雪花 ID 中的毫秒内自增序列号位段（范围 0~1023）。
func (s SnowflakeID) Sequence() int64 {
	return int64(s) & maxSequence
}
