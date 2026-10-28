// Package snowflake 提供嵌入业务基因的 64 位分布式雪花发号器与位段解析工具。
//
// 本包属于 foundation 基座层，面向分布式微服务系统的高性能主键生成。
// 在标准 Twitter Snowflake 结构基础上拓展了 6 位业务基因位段，
// 兼顾全局唯一性、时间单调递增性与数据分类分片路由能力。
//
// # 职责与适用场景
//
// 本包主要解决多服务节点并发环境下的唯一 ID 生成问题，
// 避免中心化数据库自增主键带来的性能瓶颈与单点故障。
//
// 适用场景包括：
//   - 分布式业务实体全局唯一主键发号；
//   - 异步任务与调用链链路追踪标识生成；
//   - 结合业务基因实现无额外查询的分库分表路由与业务类型识别。
//
// # ID 结构与位段分配
//
// [SnowflakeID] 为 64 位带符号整数（int64），最高符号位固定为 0，
// 确保数值始终为正整数。具体位段分配自高位至低位如下：
//
//   - 1 位符号位：固定为 0；
//   - 41 位时间戳：毫秒级时间差，以 2023-07-25 00:00:00 UTC（毫秒值 1690214400000）
//     作为自定义纪元起点，可稳定运行约 69 年（至 2092 年）；
//   - 6 位业务基因：取值范围 0 至 63，支持 64 种独立业务实体分类；
//   - 3 位数据中心 ID：取值范围 0 至 7，最多容纳 8 个独立数据中心；
//   - 3 位工作节点 ID：取值范围 0 至 7，每个数据中心最多容纳 8 个发号节点；
//   - 10 位序列号：取值范围 0 至 1023，每毫秒单节点最多生成 1024 个递增 ID。
//
// # 核心类型与调用路径
//
// [Node] 是发号核心类型，持有工作节点配置与发号状态计数器。
// 调用方通过 [NewNode] 指定数据中心 ID 与节点 ID 实例化节点。
//
// [SnowflakeID] 是发号输出结果与承载类型，提供位段信息提取、格式化与协议编解码接口。
//
// [Gene] 代表业务基因类型，预置系统级与通用业务级枚举，亦支持调用方按需扩展。
//
// [GeneCalc] 是轻量级基因计算器，由 [CalcGene] 构造，
// 提供基于 FNV-1a 算法的哈希映射功能。
//
// # 典型用法
//
// 以下代码演示节点的初始化、发号及位段信息提取：
//
//	package main
//
//	import (
//		"fmt"
//		"log"
//
//		"github.com/bamboo-services/bamboo-base-go/foundation/v2/snowflake"
//	)
//
//	func main() {
//		// 初始化发号节点（数据中心 1，节点 1）
//		node, err := snowflake.NewNode(1, 1)
//		if err != nil {
//			log.Fatalf("创建节点失败: %v", err)
//		}
//
//		// 生成带有订单业务基因的雪花 ID
//		id, err := node.Generate(snowflake.GeneOrder)
//		if err != nil {
//			log.Fatalf("发号失败: %v", err)
//		}
//
//		fmt.Println("ID 字符串:", id.String())
//		fmt.Println("业务基因:", id.Gene())
//		fmt.Println("生成时间:", id.Timestamp())
//	}
//
// # 业务基因与分片路由
//
// 业务基因占据 ID 的第 16 至 21 位（自最低位从 0 计数）。
// 基因取值范围划分为两大区间：
//   - 系统保留基因（0 至 15）：供框架内核或基础组件使用（如 [GeneSystem]、[GeneUser] 等）；
//   - 业务扩展基因（16 至 63）：供业务系统自定义实体类别（如 [GeneOrder]、[GenePayment] 等）。
//
// 当需要将关联实体（如子订单、支付流水）路由至主实体（如用户）所在分片时，
// 可借助 [CalcGene] 提供的哈希工具，提取关联键特征嵌入新 ID 中：
//
//	userGene := snowflake.CalcGene().Hash(userID)
//	orderID, err := node.Generate(userGene)
//
// # 序列化与存储交互
//
// [SnowflakeID] 全面适配常用的序列化与持久化协议：
//   - JSON 编码：输出带引号的十进制数字字符串，防止前端 JavaScript
//     因超过 Number.MAX_SAFE_INTEGER（2^53 - 1）而发生精度截断；
//     JSON 解码则同时向前兼容字符串与纯数值两种输入格式；
//   - 数据库交互：实现 [database/sql/driver.Valuer] 与 [database/sql.Scanner]，
//     以 int64 原生数值存储；扫描为 NULL 时自动重置为零值；
//   - 二进制传输：实现 [encoding.BinaryMarshaler] 与 [encoding.BinaryUnmarshaler]，
//     采用十进制字节文本格式编码。
//
// # 并发与故障语义
//
// [Node] 实例的 [Node.Generate] 方法是并发安全的，内部使用互斥锁保证发号序列单调递增，
// 适合在应用生命周期内作为单例共享。
//
// 关键边界与错误处理策略如下：
//   - 参数越界：[NewNode] 对超出 [0, 7] 范围的入参返回校验错误；
//     [Node.Generate] 对超出 [0, 63] 范围的基因参数返回校验错误；
//   - 空指针调用：未实例化的 nil 节点调用 [Node.Generate] 将返回 [ErrNilNode]，
//     调用 [Node.MustGenerate] 则直接引发 panic；
//   - 反序列化空指针：nil 接收器调用 UnmarshalJSON、UnmarshalBinary
//     或 Scan 方法将返回 [ErrNilReceiver]，解析失败时不会污染原值；
//   - 时钟回拨：若检测到系统时钟回退，[Node.Generate] 将立即返回
//     包含 [ErrClockBackwards] 的错误；
//   - 序列号耗尽：单毫秒内如果 1024 个序列号完全用尽，节点将主动自旋等待至下一毫秒继续发号。
package snowflake
