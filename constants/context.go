package xConsts

type ContextKey string

const (
	ContextRequestKey    ContextKey = "context_request_key"     // 上下文请求键
	ContextErrorCode     ContextKey = "context_error_code"      // 上下文请求错误码
	ContextErrorMessage  ContextKey = "context_error_message"   // 上下文请求错误描述
	ContextUserStartTime ContextKey = "context_user_start_time" // 上下文用户请求开始时间
	ContextDatabase      ContextKey = "context_database"        // 上下文数据库客户端
	ContextRedisClient   ContextKey = "context_redis_client"    // 上下文 Redis 客户端
	ContextSnowflakeNode ContextKey = "context_snowflake_node"  // 上下文雪花算法节点
)

// String 返回 ContextKey 的字符串表示形式。
func (s ContextKey) String() string {
	return string(s)
}
