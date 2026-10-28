package xGrpcConst_test

import (
	"fmt"

	xGrpcConst "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/constant"
)

// ExampleMetadata_String 演示 Metadata 元数据键的字符串转换与常量定义。
func ExampleMetadata_String() {
	key := xGrpcConst.MetadataRequestUUID
	fmt.Println(key.String())
	// Output:
	// x-request-uuid
}

// ExampleTrailer_String 演示 Trailer 元数据键的字符串转换。
func ExampleTrailer_String() {
	key := xGrpcConst.TrailerRequestUUID
	fmt.Println(key.String())
	// Output:
	// x-request-uuid
}
