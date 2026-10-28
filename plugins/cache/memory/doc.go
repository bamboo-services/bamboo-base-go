// Package memory 提供基于进程内分片内存的高性能缓存后端实现。
//
// 本包实现了 [github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver]
// 中定义的全部泛型缓存接口（[KeyCache]、[HashCache]、[SetCache]、[ListCache]）。
// 适用于单实例部署、本地高频只读缓存或低延迟热点数据暂存等无需跨进程共享的业务场景。
//
// # 底层分片存储与并发模型
//
// [Store] 作为所有内存缓存结构共享的底层存储引擎：
//
//   - 采用 FNV-1a 64 位哈希算法将键均匀路由到多个独立分片（Shard）；
//   - 每个分片持有独立的读写互斥锁（[sync.RWMutex]），写操作仅锁住目标分片，
//     大幅削减多 goroutine 并发争用，显著提升多核并行读写吞吐；
//   - 分片数量在初始化时向上取整为 2 的幂（默认 16 分片）。
//
// # 内存容量控制与 LRU 淘汰
//
// 为防止内存无限制增长，[Store] 提供了实例级容量保护：
//
//   - 单分片容量上限：由 maxEntries 指定每个分片允许容纳的最大条目数（0 表示无上限）；
//   - LRU 淘汰策略：每个分片维护独立双向链表，记录条目访问热度。读取或写入未过期条目时
//     自动将其移至链表头部；当分片条目数超过 maxEntries 时，自动从尾部淘汰最久未访问的条目；
//   - 全局有效条目上限约为 shardCount × maxEntries。
//
// # 生命周期与 Janitor 清理
//
// [Store] 结合惰性淘汰与后台主动扫描双重过期机制：
//
//   - 惰性检查：在 [Store.Get]、[Store.Exists] 等操作被调用时，若发现目标条目已过期，
//     立即就地从 map 和 LRU 链表中移除并返回未命中；
//   - 后台清理器（Janitor）：后台定时协程每 30 秒执行一次全分片扫描，回收长期未被访问的过期条目；
//   - 优雅退出：调用方应在应用关机或实例废弃时调用 [Store.Close]，安全等待 Janitor 协程
//     及进行中的清理任务完全退出，避免后台协程泄漏。
//
// # 原子复合操作与并发安全
//
// [Store] 提供 [Store.Update]、[Store.UpdateKeepExpireAt] 与 [Store.SetCond] 方法：
//
//   - 在单把分片写锁内闭包执行「读取旧值 -> 业务变更 -> 写回新值」的原子流程，
//     避免了先读再改导致的切片越界或 map 并发写入 panic；
//   - 支持哨兵返回值 [UpdateNoChange]，在条件不满足时短路返回，不触发多余写操作与 LRU 移动。
//
// # 泛型缓存结构适配
//
// 基于 [Store] 构造了 4 种符合 driver 规范的泛型缓存门面：
//
//   - [NewKeyCache]：构造字符串键值缓存 [KeyCache]；
//   - [NewHashCache]：构造哈希表缓存 [HashCache]；
//   - [NewSetCache]：构造唯一集合缓存 [SetCache]；
//   - [NewListCache]：构造双端列表缓存 [ListCache]。
package memory
