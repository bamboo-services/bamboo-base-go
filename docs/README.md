# 文档中心

本项目文档按模式分区管理，不同模式的文档独立建表、互不混排。工程文档的命名、取号与状态流转规则由 [`engineering/MAINTENANCE.md`](./engineering/MAINTENANCE.md) 统一约束。

## 模式导航

| 顶层模式                                | 子模式 / 入口                                                                             | 定位与职责                                                | 状态流转                             |
| --------------------------------------- | ----------------------------------------------------------------------------------------- | --------------------------------------------------------- | ------------------------------------ |
| **工程研发模式**（`docs/engineering/`） | [调研模式 `research/`](./engineering/research/)                                           | 客观现状、外部机制与实测证据                              | 无状态（仅记日期）                   |
| **工程研发模式**（`docs/engineering/`） | [设计模式 `design/`](./engineering/design/)                                               | 模块职责、数据接口、异常处理与验证方式                    | `draft → rfc → adr`                  |
| **工程研发模式**（`docs/engineering/`） | [规范模式 `spec/`](./engineering/spec/)                                                   | 跨模块标准、协议格式与长期架构红线                        | `draft → rfc → adr`                  |
| **工程研发模式**（`docs/engineering/`） | [手册模式 `runbook/`](./engineering/runbook/)                                             | 部署发布、告警排查与故障止血恢复 SOP                      | `draft → active → deprecated`        |
| **实施计划模式**（`docs/plan/`）        | [计划看板 `./plan/README.md`](./plan/README.md)                                           | 分阶段改造路线、兼容迁移与切流回滚（完结归入 `archive/`） | `draft → ready → in-progress → done` |
| **项目基建模式**（`docs/` 根级）        | [`scope-manage.md`](./scope-manage.md) · [`MAINTENANCE.md`](./engineering/MAINTENANCE.md) | 业务域词表总账与工程文档维护器公约                        | 持续维护                             |

## 工程文档台账（按模式分表）

### 调研模式（`research/`）

| 编号 | 文档                                                                                                         | Scope     | 调研日期   |
| ---- | ------------------------------------------------------------------------------------------------------------ | --------- | ---------- |
| 0001 | [v2.0.0 生命周期内核与三模式插件机制调研](./engineering/research/0001-lifecycle-plugin-runtime.md)           | lifecycle | 2026-10-17 |
| 0002 | [v2.0.0 Context 体系与共享组件传递调研](./engineering/research/0002-context-propagation-and-shared-state.md) | context   | 2026-10-17 |
| 0003 | [v2.0.0 插件体系分类与第三方库选型调研](./engineering/research/0003-plugin-ecosystem-and-libraries.md)       | plugin    | 2026-10-18 |

### 设计模式（`design/`）

| 编号 | 文档                                                                                                                   | Scope      | 阶段状态 |
| ---- | ---------------------------------------------------------------------------------------------------------------------- | ---------- | -------- |
| 0001 | [v2.0.0 生命周期插件运行时设计](./engineering/design/0001-lifecycle-plugin-runtime.md)                                 | lifecycle  | draft    |
| 0002 | [v2.0.0 Context 体系与共享组件传递设计](./engineering/design/0002-context-propagation-and-shared-state.md)             | context    | draft    |
| 0003 | [v2.0.0 foundation 地基模块合并与包重构设计](./engineering/design/0003-foundation-module-consolidation.md)             | foundation | draft    |
| 0004 | [v2.0.0 foundation/typed 强类型字符串原语与受控词汇设计](./engineering/design/0004-typed-string-primitive.md)          | typed      | draft    |
| 0005 | [v2.0.0 插件体系架构与 Adapter 嵌套模块模式设计](./engineering/design/0005-plugin-architecture-and-adapter-pattern.md) | plugin     | draft    |

五篇设计已按联动问答修订，阶段仍为 `draft`，不代表代码实现或运行验收通过。已确认静态资源键、嵌套任务归属、真实 Stop/Wait、派生 Key 显式编解码，以及环境预算启动读取后冻结；默认总协调 60s、自然排空 20s、纯清理每次最长 10s。版本映射已由 design/0003 收敛：源码目录不增 v2/ 层，module/import 带小写 /v2，首批所有模块均从 v2.0.0 起步；后续模块按需发版，每个整体 Release 附全量精确版本清单。

v2 全仓实施安排见 [0001 总实施计划](./plan/0001-lifecycle-v2-full-reconstruction.md)，当前为 draft。它承接五篇设计，第 0 阶段中 App 状态与异常、Component 队列规则已确认，其余开工契约继续收敛；计划登记不表示已经授权代码重构。

### 规范模式（`spec/`）

| 编号 | 文档 | Scope | 阶段状态 |
| ---- | ---- | ----- | -------- |

### 手册模式（`runbook/`）

| 编号 | 文档 | Scope | 状态 |
| ---- | ---- | ----- | ---- |
