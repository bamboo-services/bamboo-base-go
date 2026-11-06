# v2.0.0 foundation 能力与测试迁移清单

> 状态：draft · 日期：2026-10-27 · Scope：foundation
> 承接：[0003 foundation 模块合并与包重构设计](../design/0003-foundation-module-consolidation.md)、[0004 typed.String 强类型原语设计](../design/0004-typed-string-primitive.md)

## 目标与边界

本清单盘点已落盘 `foundation/` 八包（`apperr`、`ctxkey`、`env`、`envelope`、`logging`、`snowflake`、`toolkit`、`typed`）对旧 `defined/context`、`defined/env`、`common/error`、`common/base_response`、`common/log`、`common/snowflake`、`common/utility` 能力及测试的接管关系。

文档保持 `draft` 状态。仅盘点真实落盘代码与正常迁移约定，本清单不新增标准。

## 正常迁移约定

1. **命名空间与别名退出**：旧 `x*` 导入别名与旧命名空间包装器退出，改用顶层纯函数直接导出。
2. **上下文解耦**：旧 `ValidateProvider` 与 HTTP 验证器向 `web-gin` 的迁移尚未完成，不等于废除功能；纯错误构造 `apperr.New` 移除 `ctx` 与 `isLog`，仅负责纯错误实例装配，不自动打日志。
3. **系统键迁移**：`defined/env` 中系统级环境变量统一规整至 `APP_*`，业务领域变量保持独立前缀。
4. **命名一致性与保留行为**：
   - 环境变量中 `NoSql` 更名为 `NoSQL`（发生于 `env`，非工具包）；
   - 错误码大写缩写更名（如 `IP`/`API`/`JSON`/`XML`）已落地且 golangci-lint 0 issues；
   - 基因雪花 ID 采用 6 bit 基因支持 64 类业务分类、保留 epoch 时间基准、JSON 序列化规范、密码哈希与加密算法。

## 模块能力与测试处置清单

### 1. 强类型原语与上下文键

- **旧路径**：`defined/context`、`common/utility/context`
- **新路径**：`foundation/typed/`、`foundation/ctxkey/`
- **测试路径**：对应旧测试，由 `foundation/typed/{convert,doc,index,operate,parse}_test.go` 与 `foundation/ctxkey/{key,meta}_test.go` 承接
- **处置理由**：旧 `ContextKey` 原本即为 string 类型；改用 typed 强类型原语派生键，提供编译期类型安全与只读元数据派生。

### 2. 环境读取与配置

- **旧路径**：`defined/env`
- **新路径**：`foundation/env/`
- **测试路径**：对应旧 `defined/env/env_test.go`，由 `foundation/env/key_test.go` 承接
- **处置理由**：提供强类型安全解析与缺值/空值回退，剥离全局初始化隐式依赖。

### 3. 错误模型与响应信封

- **旧路径**：`common/error`、`common/base_response.go`
- **新路径**：`foundation/apperr/`、`foundation/envelope/`
- **测试路径**：由 `foundation/apperr/{all_codes,error,new}_test.go` 与 `foundation/envelope/response_test.go` 承接
- **处置理由**：解耦 Gin 上下文；`envelope.Response.Data` 为 `any` 载荷（非泛型载荷），纯错误构造无副效应。
- **目录收口与重编（2026-11-06）**：迁移期曾由189项缩至30项，再误以“内核实际使用”保留9项通用码；该中间口径由用户审查纠正。当前 `apperr` 仅保留8项 `FRAMEWORK_*` 的500系列码，协议与领域定义下放所属包；全库65项按 [错误码分配与受控词表规范](./0002-foundation-error-code-vocabulary.md) 重编，Code/Output全局唯一。下游须同步迁移，旧编号不提供别名；根因链、脱敏与按状态分组映射的机制不变。
- **插件专属目录**：邮件服务码由 `plugins/email` 维护，缓存操作码由 `plugins/cache` 维护，事务、记录与数据库约束码由 `plugins/database` 维护。上层按需使用 `apperr.New` 包装；不自动改写插件既有哨兵错误与底层错误，不恢复验证码校验等业务规则预设。

### 4. 日志与雪花生成

- **旧路径**：`common/log`、`common/snowflake`
- **新路径**：`foundation/logging/`、`foundation/snowflake/`
- **测试路径**：对应旧 `common/snowflake/{gene_calc,snowflake}_test.go`，由 `foundation/logging/{handler,rotator}_test.go` 与 `foundation/snowflake/{gene_calc,snowflake}_test.go` 承接
- **处置理由**：`logging` 与 `snowflake` 目前仅提供可独立实例化的原语能力，App 受管 Tool 装配尚未完成；雪花 6 bit 基因（支持 64 类）位布局与 JSON 行为保持兼容。

### 5. 工具库能力接管

- **旧路径**：`common/utility/common.go`（`Ptr`/`Val`/`Contains`/`ToBool`）、`common/utility/package/`
- **新路径**：`foundation/toolkit/`
- **测试路径**：对应旧 `common/utility/package/{encryption,function,generate,parse,password,security,string,time,validate}_test.go` 群组，由 `foundation/toolkit/{common,crypto,function,generate,parse,password,security,string,time,valid}_test.go` 承接
- **处置理由**：`toolkit` 按 10 份源文件与 10 份测试文件扁平组织为同一个包（非拆分加密/安全等多个子包），移除历史无用别名与命名空间包装，方法按既定迁移不遗漏。

## 实测验证状态与遗漏

- **实测代码与测试阶段准出**：验收任务 `b98d6421f` 退出码 0，`foundation` 八包在 Go 1.27.1 与最低 Go 1.25.0 下通过 `GOWORK=off go test -race` 与 `go vet`；golangci-lint 2.14 检查 0 issues、代码格式化 diff 为空、依赖树无 gin/gorm/validator。189 错误码黄金回归测试已恢复 `FILE_DOWNLOAD_ERROR` 40051，无编码冲突。
- **后续未完成范围**：App Tool 装配、Redis 真实消费通道、候选完整模块消费和各插件验收仍归属后续阶段；设计持续保持 `draft`，计划保持 `in-progress`，不宣称整仓完成。
