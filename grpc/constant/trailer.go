package xGrpcConst

type Trailer string

// String 返回元数据的字符串表示形式。
//
// 该方法实现了 `fmt.Stringer` 接口，直接将 `Trailer` 类型转换为 `string` 类型返回。
func (md Trailer) String() string {
	return string(md)
}

const (
	TrailerRequestUUID Trailer = "x_request_uuid" // 定义用于传递请求唯一标识符的元数据键，通常用于在中间件或拦截器中标识请求的唯一性。
)
