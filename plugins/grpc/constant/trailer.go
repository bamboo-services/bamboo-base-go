package xGrpcConst

// Trailer 定义 gRPC 响应尾部（Trailer）元数据键的强类型别名。
type Trailer string

// String 返回 Trailer 元数据键的基础字符串表示形式。
//
// 该方法实现了 [fmt.Stringer] 接口，直接将 [Trailer] 类型转换为 string 返回。
func (md Trailer) String() string {
	return string(md)
}

// 常用 gRPC 响应尾部 Trailer 键常量定义。
const (
	TrailerRequestUUID Trailer = "x-request-uuid" // 在响应 Trailer 阶段回写请求唯一标识符（UUID）的元数据键。
)
