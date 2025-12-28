package xInit

import (
	xConsts "github.com/bamboo-services/bamboo-base-go/constants"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/snowflake"
	"go.uber.org/zap"
)

// SnowflakeInit 初始化雪花算法节点
//
// 该方法从环境变量读取节点配置:
//   - SNOWFLAKE_DATACENTER_ID: 数据中心 ID
//   - SNOWFLAKE_NODE_ID: 节点 ID
//
// 若未配置环境变量，则根据机器特征（MAC 地址或主机名）自动生成节点 ID。
// 初始化后的节点实例会在 SystemContextInit 中注入到请求上下文。
//
// 注意:
//   - 该方法应在 LoggerInit 之后调用，以确保日志记录器可用。
//   - 初始化失败会触发 Fatal 级别日志并终止程序。
func (r *Reg) SnowflakeInit() {
	zap.L().Named(xConsts.LogINIT).Info("初始化雪花算法节点")

	if err := xSnowflake.InitDefaultNode(); err != nil {
		zap.L().Named(xConsts.LogINIT).Fatal("雪花算法节点初始化失败", zap.Error(err))
	}

	// 获取节点信息并记录日志
	node := xSnowflake.GetDefaultNode()
	geneNode := xSnowflake.GetDefaultGeneNode()

	// 生成测试 ID 验证节点正常工作
	testID := node.Generate()
	testGeneID := geneNode.MustGenerate(xSnowflake.GeneSystem)

	zap.L().Named(xConsts.LogINIT).Info("雪花算法节点初始化成功",
		zap.Int64("datacenter_id", node.DatacenterID()),
		zap.Int64("node_id", node.NodeID()),
		zap.String("test_id", testID.String()),
		zap.Int64("gene_datacenter_id", geneNode.DatacenterID()),
		zap.Int64("gene_node_id", geneNode.NodeID()),
		zap.String("test_gene_id", testGeneID.String()),
	)
}
