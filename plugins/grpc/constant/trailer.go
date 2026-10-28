package xGrpcConst

// Trailer 定义 gRPC Trailer 元数据键的强类型别名。
type Trailer string

// String 返回元数据的字符串表示形式。
//
// 该方法实现了 `fmt.Stringer` 接口，直接将 `Trailer` 类型转换为 `string` 类型返回。
func (md Trailer) String() string {
	return string(md)
}

// 常用 gRPC Trailer 键常量定义。
const (
	TrailerRequestUUID Trailer = "x-request-uuid" // 定义用于传递请求唯一标识符的元数据键，通常用于在中间件或拦截器中标识请求的唯一性。
)
