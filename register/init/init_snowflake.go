package xInit

import (
	"context"

	xLog "github.com/bamboo-services/bamboo-base-go/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/snowflake"
)

// SnowflakeInit 初始化并验证雪花算法默认节点。
//
// 该函数会初始化默认的雪花算法节点，生成测试 ID 以验证节点工作正常，
// 并记录节点 ID 和数据中心 ID 等关键信息。
//
// 参数 ctx 用于日志记录的上下文追踪。
// 返回初始化后的节点实例（any 类型，实际为 *Node），若初始化失败则返回 error。
func SnowflakeInit(ctx context.Context) (any, error) {
	log := xLog.WithName(xLog.NamedINIT)
	log.Info(ctx, "初始化雪花算法节点")

	if err := xSnowflake.InitDefaultNode(); err != nil {
		return nil, err
	}

	// 获取节点信息并记录日志
	node := xSnowflake.GetDefaultNode()

	// 生成测试 ID 验证节点正常工作
	testID := node.MustGenerate()                          // 普通 ID（Gene=0）
	testGeneID := node.MustGenerate(xSnowflake.GeneSystem) // 基因 ID

	log.SugarDebug(ctx, "雪花算法节点初始化成功",
		"datacenter_id", node.DatacenterID(),
		"node_id", node.NodeID(),
		"test_id", testID.String(),
		"test_gene_id", testGeneID.String(),
		"test_gene_id_gene", testGeneID.Gene().String(),
	)

	return xSnowflake.GetDefaultNode(), nil
}
