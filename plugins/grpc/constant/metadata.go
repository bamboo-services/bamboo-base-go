package xGrpcConst

// Metadata 定义 gRPC 元数据（Metadata）键的强类型别名。
type Metadata string

// String 返回元数据键的基础字符串表示形式。
//
// 该方法实现了 [fmt.Stringer] 接口，直接将 [Metadata] 类型转换为 string 返回。
func (md Metadata) String() string {
	return string(md)
}

// 常用 gRPC 请求元数据键常量定义。
const (
	MetadataAppAccessID  Metadata = "app-access-id"  // 用于传递应用访问凭证标识符（Access ID）的元数据键。
	MetadataAppSecretKey Metadata = "app-secret-key" // 用于传递应用访问密钥（Secret Key）的元数据键。
	MetadataRequestUUID  Metadata = "x-request-uuid" // 用于在请求元数据中传递分布式链路请求唯一标识符的键。
)
