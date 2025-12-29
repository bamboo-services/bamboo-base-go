package xSnowflake

import (
	"fmt"
	"hash/fnv"
	"net"
	"os"
	"strconv"
	"sync"

	xConstEnv "github.com/bamboo-services/bamboo-base-go/constants/env"
)

var (
	defaultNode     *Node
	defaultGeneNode *GeneNode
	nodeOnce        sync.Once
	initErr         error
)

// InitDefaultNode 初始化默认的雪花算法节点
//
// 节点 ID 优先从环境变量获取:
//   - SNOWFLAKE_DATACENTER_ID: 数据中心 ID
//   - SNOWFLAKE_NODE_ID: 节点 ID
//
// 若未配置则自动根据 MAC 地址或主机名生成。
//
// 返回值:
//   - error: 初始化错误
func InitDefaultNode() error {
	nodeOnce.Do(func() {
		datacenterID, nodeID := getIDsFromEnv()

		// 创建标准雪花节点
		defaultNode, initErr = NewNode(datacenterID, nodeID)
		if initErr != nil {
			return
		}

		// 创建基因雪花节点（ID 需要在基因节点范围内）
		geneDatacenterID := datacenterID % (maxGeneDatacenterID + 1)
		geneNodeID := nodeID % (maxGeneNodeID + 1)
		defaultGeneNode, initErr = NewGeneNode(geneDatacenterID, geneNodeID)
	})
	return initErr
}

// getIDsFromEnv 从环境变量获取数据中心 ID 和节点 ID
//
// 返回值:
//   - datacenterID: 数据中心 ID
//   - nodeID: 节点 ID
func getIDsFromEnv() (datacenterID, nodeID int64) {
	datacenterID = getEnvInt64(xConstEnv.SnowflakeDatacenterID.String(), -1)
	nodeID = getEnvInt64(xConstEnv.SnowflakeNodeID.String(), -1)

	// 如果环境变量未配置，自动生成
	if datacenterID < 0 || datacenterID > maxDatacenterID ||
		nodeID < 0 || nodeID > maxNodeID {
		datacenterID, nodeID = autoGenerateIDs()
	}

	return
}

// getEnvInt64 获取整数类型的环境变量值
//
// 参数说明:
//   - key: 环境变量名
//   - defaultValue: 默认值
//
// 返回值:
//   - int64: 环境变量值或默认值
func getEnvInt64(key string, defaultValue int64) int64 {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	num, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return defaultValue
	}
	return num
}

// autoGenerateIDs 自动生成数据中心 ID 和节点 ID
//
// 优先使用 MAC 地址生成，失败则使用主机名。
// 使用 FNV-1a 哈希算法生成稳定的 ID。
//
// 返回值:
//   - datacenterID: 数据中心 ID (0-31)
//   - nodeID: 节点 ID (0-31)
func autoGenerateIDs() (datacenterID, nodeID int64) {
	var hashInput []byte

	// 尝试从 MAC 地址获取
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			// 跳过回环接口和无 MAC 地址的接口
			if iface.Flags&net.FlagLoopback != 0 || len(iface.HardwareAddr) < 6 {
				continue
			}
			hashInput = iface.HardwareAddr
			break
		}
	}

	// 如果没有获取到 MAC 地址，使用主机名
	if len(hashInput) == 0 {
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "unknown-host"
		}
		hashInput = []byte(hostname)
	}

	// 使用 FNV-1a 哈希算法
	h := fnv.New64a()
	_, _ = h.Write(hashInput)
	hash := h.Sum64()

	// 从哈希值中提取数据中心 ID 和节点 ID
	datacenterID = int64(hash % uint64(maxDatacenterID+1))
	nodeID = int64((hash >> 5) % uint64(maxNodeID+1))

	return
}

// GetDefaultNode 获取默认的雪花算法节点
//
// 如果未初始化，会自动调用 InitDefaultNode() 进行初始化。
//
// 返回值:
//   - *Node: 默认节点实例
func GetDefaultNode() *Node {
	if defaultNode == nil {
		_ = InitDefaultNode()
	}
	return defaultNode
}

// GetDefaultGeneNode 获取默认的基因雪花算法节点
//
// 如果未初始化，会自动调用 InitDefaultNode() 进行初始化。
//
// 返回值:
//   - *GeneNode: 默认基因节点实例
func GetDefaultGeneNode() *GeneNode {
	if defaultGeneNode == nil {
		_ = InitDefaultNode()
	}
	return defaultGeneNode
}

// GenerateID 使用默认节点生成雪花 ID
//
// 返回值:
//   - SnowflakeID: 生成的雪花 ID
func GenerateID() SnowflakeID {
	return GetDefaultNode().Generate()
}

// GenerateGeneID 使用默认节点生成基因雪花 ID
//
// 参数说明:
//   - gene: 业务基因类型
//
// 返回值:
//   - GeneSnowflakeID: 生成的基因雪花 ID
//   - error: 生成错误
func GenerateGeneID(gene Gene) (GeneSnowflakeID, error) {
	return GetDefaultGeneNode().Generate(gene)
}

// MustGenerateGeneID 使用默认节点生成基因雪花 ID，如果发生错误则 panic
//
// 参数说明:
//   - gene: 业务基因类型
//
// 返回值:
//   - GeneSnowflakeID: 生成的基因雪花 ID
func MustGenerateGeneID(gene Gene) GeneSnowflakeID {
	return GetDefaultGeneNode().MustGenerate(gene)
}

// ParseSnowflakeID 从字符串解析雪花 ID
//
// 参数说明:
//   - s: ID 字符串
//
// 返回值:
//   - SnowflakeID: 解析后的雪花 ID
//   - error: 解析错误
func ParseSnowflakeID(s string) (SnowflakeID, error) {
	num, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("解析雪花 ID 失败: %w", err)
	}
	return SnowflakeID(num), nil
}

// MustParseSnowflakeID 从字符串解析雪花 ID，如果发生错误则 panic
//
// 参数说明:
//   - s: ID 字符串
//
// 返回值:
//   - SnowflakeID: 解析后的雪花 ID
func MustParseSnowflakeID(s string) SnowflakeID {
	id, err := ParseSnowflakeID(s)
	if err != nil {
		panic(err)
	}
	return id
}

// ParseGeneSnowflakeID 从字符串解析基因雪花 ID
//
// 参数说明:
//   - s: ID 字符串
//
// 返回值:
//   - GeneSnowflakeID: 解析后的基因雪花 ID
//   - error: 解析错误
func ParseGeneSnowflakeID(s string) (GeneSnowflakeID, error) {
	num, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("解析基因雪花 ID 失败: %w", err)
	}
	return GeneSnowflakeID(num), nil
}

// MustParseGeneSnowflakeID 从字符串解析基因雪花 ID，如果发生错误则 panic
//
// 参数说明:
//   - s: ID 字符串
//
// 返回值:
//   - GeneSnowflakeID: 解析后的基因雪花 ID
func MustParseGeneSnowflakeID(s string) GeneSnowflakeID {
	id, err := ParseGeneSnowflakeID(s)
	if err != nil {
		panic(err)
	}
	return id
}
