// Package models 提供 GORM 通用模型基类、雪花主键自动注入、业务基因识别与通用分页模型。
//
// 本包为应用业务实体定义标准字段约定与生命周期钩子，主要涵盖基础模型、软删除模型、
// 雪花 ID 上下文自动填充以及通用的分页请求与响应规范。
//
// # 模型基类与雪花主键注入
//
// 本包提供两套基础实体结构：
//
//   - [BaseEntity]：通用基础实体，包含主键 [BaseEntity.ID]（[snowflake.SnowflakeID]）
//     与毫秒级时间戳 [BaseEntity.CreatedAt]、[BaseEntity.UpdatedAt]。
//   - [BaseEntityWithSoftDelete]：带软删除特性的基础实体，在 [BaseEntity] 基础上增加
//     [gorm.DeletedAt] 软删除索引字段。
//
// 两类基类均实现了 GORM 的创建钩子 [BaseEntity.BeforeCreate] 与更新钩子 [BaseEntity.BeforeUpdate]：
//
//   - ID 自动生成：在 BeforeCreate 执行时，若实体当前 ID 为零值，钩子将从 GORM Statement
//     关联的 [context.Context] 中提取 [*snowflake.Node]（键为 [ctxkey.SnowflakeNodeKey]）；
//     若上下文缺失该节点或节点实例为 nil，BeforeCreate 将立即终止并返回明确错误，
//     防止向数据库插入非法零值主键；
//   - 显式 ID 保留：若实体在创建前已显式赋予了非零 ID，钩子将安全保留原 ID，不再尝试从
//     上下文读取雪花发号器节点；
//   - 业务基因识别：若实体类型实现了 [GeneProvider] 接口，发号器将采用其实体定义的
//     业务基因生成雪花 ID；若未实现，默认采用 [snowflake.GeneDefault]。
//
// # 分页请求与响应
//
// 本包提供通用的数据库分页规范化工具：
//
//   - [PageRequest]：统一分页请求入参，包含 Page、Size 与 [PageSort]（[SortAsc] / [SortDesc]）。
//     调用 [PageRequest.Normalize] 或 [PageRequest.NormalizeWithConfig] 可将非法参数
//     （如 page <= 0、size <= 0、超出上限或非法排序方向）规范化为安全值。
//   - [PageRequest.Offset] 与 [PageRequest.Limit]：在规范化基础上直接计算 SQL 偏移量与查询行数。
//   - [PageResponse]：泛型分页响应信封，携带当前页码、总页数、总条目数、每页容量及泛型数据项切片。
//     提供 [NewPage]、[NewPageFromRequest]、[NewPageNoData] 等构造函数。
//   - [PageProvider]：可供自定义业务请求实现的接口，通过 [GetPageRequest] 可统一抽取并规范化分页参数。
//
// # 错误语义
//
// 在雪花主键生成与模型钩子执行过程中可能返回以下哨兵错误：
//
//   - [ErrNilStatement]：GORM Statement 句柄为 nil。
//   - [ErrNilContext]：GORM Statement 中的上下文为 nil。
//   - [ErrSnowflakeNodeNotFound]：上下文中未找到有效的雪花节点。
//   - [ErrNilSnowflakeNode]：上下文中获取到的雪花节点指针为 nil 或 typed nil。
//
// # 并发与使用边界
//
// 实体实例由当前数据库事务或操作协程独占使用，不保证并发读写安全性。
// 全局分页默认配置 [DefaultPageConfig] 为包级变量，如需全局定制应在服务启动初始化阶段完成配置。
package models
