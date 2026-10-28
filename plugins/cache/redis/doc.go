// Package redis 提供基于 Redis 客户端的分布式缓存后端实现。
//
// 本包实现了 [github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver]
// 中定义的全部泛型缓存接口（[KeyCache]、[HashCache]、[SetCache]、[ListCache]）。
// 依托 Redis 键值存储的高性能与持久化能力，适用于分布式服务部署、跨进程状态共享以及大数据集缓存。
//
// # 客户端依赖与连接池
//
// 本包所有缓存实现均依赖成熟的 [github.com/redis/go-redis/v9.Client]：
//
//   - 客户端实例通过构造函数显式注入（如 [NewKeyCache]、[NewHashCache] 等）；
//   - Redis 客户端内部自带连接池机制且本身完全并发安全，本包的所有缓存实例可被多个 goroutine
//     安全并发调用；
//   - 本包不主动关闭注入的 Redis 客户端，其生命周期与网络连接由上层插件容器或创建者统一维护。
//
// # 键编码与字段约束
//
//   - 键与字段转换：所有键类型 K 与字段类型 F 均通过注入的 [driver.KeyEncoder]（未注入时
//     回退至 [driver.DefaultKeyEncoder]）转换为标准字符串；
//   - 哈希字段反射约束：[HashCache.GetAll] 在运行时要求泛型字段类型 F 的底层类型必须为 string，
//     否则将明确返回错误，防止因类型不匹配静默返回空结果；
//   - 值序列化：所有泛型值 V 均通过注入的 [driver.Codec]（默认 [driver.JSONCodec]）
//     序列化为字节切片后送入 Redis。
//
// # Redis 命令映射与数据结构
//
// 各泛型缓存接口与 Redis 原生命令的映射关系如下：
//
//   - [KeyCache]：映射为 Redis String 命令（GET、SET、DEL、EXISTS 等）；
//   - [HashCache]：映射为 Redis Hash 命令（HGET、HSET、HSETNX、HDEL、HGETALL、HEXISTS 等）；
//   - [SetCache]：映射为 Redis Set 命令（SADD、SMEMBERS、SISMEMBER、SCARD、SREM 等）；
//   - [ListCache]：映射为 Redis List 命令（LPUSH、RPUSH、LRANGE、LINDEX、LLEN、LPOP、RPOP、LREM 等）。
//     针对 [ListCache.Prepend]，为保证切片首元素在头部最左侧，内部在调用 LPUSH 前自动反转参数顺序。
//
// # 条件写入与 TTL 策略
//
// 本包完整支持 driver 中定义的函数式写选项 [driver.SetOption]：
//
//   - [driver.WithNX] 与 [driver.WithXX]：单键操作映射为 SET NX / SET XX 或预检，
//     条件不满足时安全跳过且不返回错误；
//   - [driver.WithKeepTTL]：更新键内容时保留现有生存时间，避免覆盖已有 TTL；
//   - [driver.WithNoSlide]：写入集合、哈希或列表元素时不触发 EXPIRE 续期，防止高频写入导致热点永不过期。
//
// # 错误处理与未命中语义
//
// 当 Redis 报告 [github.com/redis/go-redis/v9.Nil] 错误时，本包将其统一视作缓存未命中，
// 返回 (nil, false, nil)，不产生系统级错误。其他网络中断、权限鉴权失败或命令语法错误则透明向外传递。
package redis
