package xGrpcUtil_test

import (
	"context"
	"fmt"

	xGrpcConst "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/constant"
	xGrpcUtil "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/utility"
	"google.golang.org/grpc/metadata"
)

// ExampleExtractMetadata 演示从传入上下文元数据中安全提取指定键的有效字符串值。
func ExampleExtractMetadata() {
	// 构造包含传入元数据的上下文
	md := metadata.Pairs(
		xGrpcConst.MetadataRequestUUID.String(), "uuid-trace-123456",
		xGrpcConst.MetadataAppAccessID.String(), "   ", // 空白无效值
	)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// 成功提取有效非空白元数据
	val, err := xGrpcUtil.ExtractMetadata(ctx, xGrpcConst.MetadataRequestUUID)
	fmt.Println(err == nil)
	fmt.Println(val)

	// 提取仅包含空白字符的键时返回错误
	_, errBlank := xGrpcUtil.ExtractMetadata(ctx, xGrpcConst.MetadataAppAccessID)
	fmt.Println(errBlank != nil)

	// 提取不存在的键时返回错误
	_, errNotExist := xGrpcUtil.ExtractMetadata(ctx, xGrpcConst.MetadataAppSecretKey)
	fmt.Println(errNotExist != nil)

	// Output:
	// true
	// uuid-trace-123456
	// true
	// true
}
