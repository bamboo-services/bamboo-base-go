package xCtxUtil

import (
	xConsts "github.com/bamboo-services/bamboo-base-go/constants"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/snowflake"
	"github.com/gin-gonic/gin"
)

// GetSnowflakeNode 从上下文获取雪花算法节点
//
// 如果上下文中不存在节点，则返回默认节点。
// 这确保了即使在非 HTTP 请求上下文中也能正常生成 ID。
//
// 参数说明:
//   - c: gin.Context 上下文
//
// 返回值:
//   - *xSnowflake.Node: 雪花算法节点
func GetSnowflakeNode(c *gin.Context) *xSnowflake.Node {
	value, exists := c.Get(xConsts.ContextSnowflakeNode.String())
	if exists {
		if node, ok := value.(*xSnowflake.Node); ok {
			return node
		}
	}
	// 回退到默认节点
	return xSnowflake.GetDefaultNode()
}

// GetGeneSnowflakeNode 从上下文获取基因雪花算法节点
//
// 如果上下文中不存在节点，则返回默认节点。
//
// 参数说明:
//   - c: gin.Context 上下文
//
// 返回值:
//   - *xSnowflake.GeneNode: 基因雪花算法节点
func GetGeneSnowflakeNode(c *gin.Context) *xSnowflake.GeneNode {
	value, exists := c.Get(xConsts.ContextGeneSnowflakeNode.String())
	if exists {
		if node, ok := value.(*xSnowflake.GeneNode); ok {
			return node
		}
	}
	// 回退到默认节点
	return xSnowflake.GetDefaultGeneNode()
}

// GenerateSnowflakeID 使用上下文中的节点生成雪花 ID
//
// 参数说明:
//   - c: gin.Context 上下文
//
// 返回值:
//   - xSnowflake.SnowflakeID: 生成的雪花 ID
func GenerateSnowflakeID(c *gin.Context) xSnowflake.SnowflakeID {
	return GetSnowflakeNode(c).Generate()
}

// GenerateGeneSnowflakeID 使用上下文中的节点生成基因雪花 ID
//
// 参数说明:
//   - c: gin.Context 上下文
//   - gene: 业务基因类型
//
// 返回值:
//   - xSnowflake.GeneSnowflakeID: 生成的基因雪花 ID
//   - error: 生成错误（如果基因类型无效）
func GenerateGeneSnowflakeID(c *gin.Context, gene xSnowflake.Gene) (xSnowflake.GeneSnowflakeID, error) {
	return GetGeneSnowflakeNode(c).Generate(gene)
}

// MustGenerateGeneSnowflakeID 使用上下文中的节点生成基因雪花 ID
//
// 如果发生错误则 panic，适用于确定基因类型有效的场景。
//
// 参数说明:
//   - c: gin.Context 上下文
//   - gene: 业务基因类型
//
// 返回值:
//   - xSnowflake.GeneSnowflakeID: 生成的基因雪花 ID
func MustGenerateGeneSnowflakeID(c *gin.Context, gene xSnowflake.Gene) xSnowflake.GeneSnowflakeID {
	return GetGeneSnowflakeNode(c).MustGenerate(gene)
}
