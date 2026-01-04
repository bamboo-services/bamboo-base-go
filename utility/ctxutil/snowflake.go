package xCtxUtil

import (
	xConsts "github.com/bamboo-services/bamboo-base-go/context"
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
	value, exists := c.Get(xConsts.SnowflakeNodeKey.String())
	if exists {
		if node, ok := value.(*xSnowflake.Node); ok {
			return node
		}
	}
	// 回退到默认节点
	return xSnowflake.GetDefaultNode()
}

// GenerateSnowflakeID 使用上下文中的节点生成雪花 ID
//
// 生成普通雪花 ID（Gene=0）。
//
// 参数说明:
//   - c: gin.Context 上下文
//
// 返回值:
//   - xSnowflake.SnowflakeID: 生成的雪花 ID
func GenerateSnowflakeID(c *gin.Context) xSnowflake.SnowflakeID {
	return GetSnowflakeNode(c).MustGenerate()
}

// GenerateGeneSnowflakeID 使用上下文中的节点生成带基因的雪花 ID
//
// 参数说明:
//   - c: gin.Context 上下文
//   - gene: 业务基因类型
//
// 返回值:
//   - xSnowflake.SnowflakeID: 生成的雪花 ID
func GenerateGeneSnowflakeID(c *gin.Context, gene xSnowflake.Gene) xSnowflake.SnowflakeID {
	return GetSnowflakeNode(c).MustGenerate(gene)
}
