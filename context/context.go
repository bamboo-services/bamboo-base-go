package xCtx

type ContextKey string

const (
	Nil              ContextKey = ""                        // 空值
	Exec             ContextKey = "special_execution"       // 特殊执行
	RequestKey       ContextKey = "context_request_key"     // 上下文请求键
	ErrorCodeKey     ContextKey = "context_error_code"      // 上下文请求错误码
	ErrorMessageKey  ContextKey = "context_error_message"   // 上下文请求错误描述
	UserStartTimeKey ContextKey = "context_user_start_time" // 上下文用户请求开始时间
	DatabaseKey      ContextKey = "context_database"        // 上下文数据库客户端
	RedisClientKey   ContextKey = "context_redis_client"    // 上下文 Redis 客户端
	SnowflakeNodeKey ContextKey = "context_snowflake_node"  // 上下文雪花算法节点
)

// String 返回 ContextKey 的字符串表示形式。
func (s ContextKey) String() string {
	return string(s)
}

// IsNil 检查 ContextKey 是否为空值
func (s ContextKey) IsNil() bool {
	return s == Nil
}

// IsExec 检查 ContextKey 是否为特殊执行
func (s ContextKey) IsExec() bool {
	return s == Exec
}
