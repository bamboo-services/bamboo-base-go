package xUtil

import (
	"context"
	"fmt"
	"strings"

	xError "github.com/bamboo-services/bamboo-base-go/error"
	xGrpcConst "github.com/bamboo-services/bamboo-base-go/grpc/constant"
	"google.golang.org/grpc/metadata"
)

// ExtractMetadata 从 gRPC 传入上下文中提取指定键的元数据值
//
// 该函数用于从 gRPC 请求的元数据中获取特定键对应的值。
// 它会遍历该键下的所有值，并返回第一个非空白字符串。
//
// 参数说明:
//   - ctx: `context.Context` 请求上下文，用于错误追踪。
//   - key: `xGrpcConst.Trailer` 需要提取的元数据键名。
//
// 返回值:
//   - string: 找到的第一个非空白元数据值。
//   - error: 如果上下文中不存在元数据或指定键无有效值，返回 `xError.NotExist` 错误。
func ExtractMetadata(ctx context.Context, key xGrpcConst.Metadata) (string, *xError.Error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", xError.NewError(ctx, xError.NotExist, "缺少 gRPC Trailer", false)
	}

	values := md.Get(key.String())
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed, nil
		}
	}
	return "", xError.NewError(ctx, xError.NotExist, xError.ErrMessage(fmt.Sprintf("元数据中不存在有效值: %s", key.String())), false)
}
