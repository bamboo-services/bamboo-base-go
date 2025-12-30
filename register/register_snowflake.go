package xReg

import (
	xLog "github.com/bamboo-services/bamboo-base-go/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/snowflake"
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
	log := xLog.WithName(xLog.NamedINIT)
	log.Info(r.Context, "初始化雪花算法节点")

	if err := xSnowflake.InitDefaultNode(); err != nil {
		log.SugarError(r.Context, "雪花算法节点初始化失败", "error", err)
		panic("雪花算法节点初始化失败: " + err.Error())
	}

	// 获取节点信息并记录日志
	node := xSnowflake.GetDefaultNode()

	// 生成测试 ID 验证节点正常工作
	testID := node.MustGenerate()                          // 普通 ID（Gene=0）
	testGeneID := node.MustGenerate(xSnowflake.GeneSystem) // 基因 ID

	log.SugarInfo(r.Context, "雪花算法节点初始化成功",
		"datacenter_id", node.DatacenterID(),
		"node_id", node.NodeID(),
		"test_id", testID.String(),
		"test_gene_id", testGeneID.String(),
		"test_gene_id_gene", testGeneID.Gene().String(),
	)
}
