package xGrpcUtil

import (
	"context"
	"fmt"
	"strings"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	xGrpcConst "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/constant"
	"google.golang.org/grpc/metadata"
)

// ExtractMetadata 从 gRPC 请求传入上下文（Incoming Metadata）中提取指定键的首个非空白字符串值。
//
// 该函数专门用于从传入的 RPC 请求元数据（Incoming Metadata，非响应阶段的 Trailer）中安全提取
// 键为 key 的值列表。它会按序遍历对应键的所有切片条目，并返回首个经过 [strings.TrimSpace] 去除两端空白后非空的字符串。
//
// # 错误与失败语义
//
// 若遇到以下任一情况，函数将返回包含 [apperr.NotExist]（业务错误码 40000）的结构化业务错误，且错误信息中不会打印敏感元数据内容：
//   - ctx 为 nil；
//   - ctx 未通过 [metadata.FromIncomingContext] 携带任何传入元数据；
//   - 元数据中不存在指定的 key，或该键下的所有值均为空白字符串。
//
// 该函数具备并发安全性，可在中间件或业务 RPC 实现中安全调用。
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
