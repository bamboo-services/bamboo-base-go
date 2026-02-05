package xCtxUtil

import (
	"context"

	xConsts "github.com/bamboo-services/bamboo-base-go/context"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/snowflake"
)

// GetSnowflakeNode 从上下文获取雪花算法节点
//
// 如果上下文中不存在节点，则返回默认节点。
// 这确保了即使在非 HTTP 请求上下文中也能正常生成 ID。
//
// 参数说明:
//   - ctx: context.Context 上下文
//
// 返回值:
//   - *xSnowflake.Node: 雪花算法节点
func GetSnowflakeNode(ctx context.Context) *xSnowflake.Node {
	value := ctx.Value(xConsts.SnowflakeNodeKey)
	if value != nil {
		if node, ok := value.(*xSnowflake.Node); ok {
			return node
		}
	}
	// 回退到默认节点
	return xSnowflake.GetDefaultNode()
}

// MustGenerateSnowflakeID 使用上下文中的节点生成雪花 ID（panic 版本）
//
// 生成普通雪花 ID（Gene=0）。
// 如果生成失败，会触发 panic。
//
// 参数说明:
//   - ctx: context.Context 上下文
//
// 返回值:
//   - xSnowflake.SnowflakeID: 生成的雪花 ID
func MustGenerateSnowflakeID(ctx context.Context) xSnowflake.SnowflakeID {
	return GetSnowflakeNode(ctx).MustGenerate()
}

// GenerateSnowflakeID 使用上下文中的节点生成雪花 ID（错误返回版本）
//
// 生成普通雪花 ID（Gene=0）。
//
// 参数说明:
//   - ctx: context.Context 上下文
//
// 返回值:
//   - xSnowflake.SnowflakeID: 生成的雪花 ID
//   - error: 生成失败时返回错误
func GenerateSnowflakeID(ctx context.Context) (xSnowflake.SnowflakeID, error) {
	return GetSnowflakeNode(ctx).Generate()
}

// MustGenerateGeneSnowflakeID 使用上下文中的节点生成带基因的雪花 ID（panic 版本）
//
// 如果生成失败，会触发 panic。
//
// 参数说明:
//   - ctx: context.Context 上下文
//   - gene: 业务基因类型
//
// 返回值:
//   - xSnowflake.SnowflakeID: 生成的雪花 ID
func MustGenerateGeneSnowflakeID(ctx context.Context, gene xSnowflake.Gene) xSnowflake.SnowflakeID {
	return GetSnowflakeNode(ctx).MustGenerate(gene)
}

// GenerateGeneSnowflakeID 使用上下文中的节点生成带基因的雪花 ID（错误返回版本）
//
// 参数说明:
//   - ctx: context.Context 上下文
//   - gene: 业务基因类型
//
// 返回值:
//   - xSnowflake.SnowflakeID: 生成的雪花 ID
//   - error: 生成失败时返回错误
func GenerateGeneSnowflakeID(ctx context.Context, gene xSnowflake.Gene) (xSnowflake.SnowflakeID, error) {
	return GetSnowflakeNode(ctx).Generate(gene)
}
