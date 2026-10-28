package xGrpcUtil

import (
	"context"
	"fmt"
	"strings"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	xGrpcConst "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/constant"
	"google.golang.org/grpc/metadata"
)

// ExtractMetadata 从 gRPC 传入上下文（Incoming Metadata）中提取指定键的元数据值。
//
// 该函数用于从传入的 gRPC 请求元数据（Incoming Metadata，非 Trailer）中获取特定键对应的值。
// 它会遍历该键下的所有值，并返回第一个经过 strings.TrimSpace 后非空的字符串。
//
// 参数说明:
//   - ctx: context.Context 请求上下文（即使为 nil 也不会 panic）。
//   - key: xGrpcConst.Metadata 需要提取的元数据键名。
//
// 返回值:
//   - string: 找到的第一个非空白元数据值（已去除首尾空白字符）。
//   - *apperr.Error: 如果上下文为 nil、未包含传入元数据或指定键无有效值，返回 apperr.NotExist 错误（错误码 40000），且不泄露敏感元数据值。
func ExtractMetadata(ctx context.Context, key xGrpcConst.Metadata) (string, *apperr.Error) {
	if ctx == nil {
		return "", apperr.New(apperr.NotExist, "缺少 gRPC 传入元数据", nil)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", apperr.New(apperr.NotExist, "缺少 gRPC 传入元数据", nil)
	}

	values := md.Get(key.String())
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed, nil
		}
	}
	return "", apperr.New(apperr.NotExist, apperr.ErrMessage(fmt.Sprintf("元数据中不存在有效值: %s", key.String())), nil)
}
