# 工程文档维护器公约（MAINTENANCE）

本文件约束 `docs/engineering/` 与 `docs/plan/` 下文档的创建、取号、阶段演进与卫生维护。日常写作与巡检由 `engineering-design` 插件（`research-er` / `design-er` / `spec-er` / `runbook-er` / `plan-er` / `doc-clean`）执行。

## 1. 模式与目录边界

| 目录                                | 模式             | 何时写入                                     | 状态机                                                                                                                       |
| ----------------------------------- | ---------------- | -------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `docs/engineering/research/`        | 调研（research） | 摸清外部标准、竞品或技术机制，只记现状与证据 | 无状态（记 `调研日期：YYYY-MM-DD`）                                                                                          |
| `docs/engineering/design/`          | 设计（design）   | 具体功能/模块怎样工作、数据接口、异常与验证  | `draft` → `rfc` → `adr` / `rejected` / `superseded`                                                                          |
| `docs/engineering/spec/`            | 规范（spec）     | 跨模块通用标准、命名/目录约定、架构红线      | `draft` → `rfc` → `adr` / `rejected` / `superseded`                                                                          |
| `docs/engineering/runbook/`         | 手册（runbook）  | 部署上线、告警排查、故障止血与恢复 SOP       | `draft` → `active` → `deprecated` / `superseded`                                                                             |
| `docs/plan/` + `docs/plan/archive/` | 计划（plan）     | 分阶段实施路线、存量迁移与回滚步骤           | 活动态（`docs/plan/`）：`draft` → `ready` → `in-progress`；终态（`docs/plan/archive/`）：`done` / `cancelled` / `superseded` |

## 2. 命名与独立取号铁律

所有受控文档统一命名为 `NNNN-<scope>-<title>.md`：

1. **`NNNN`（四位独立序列）**：
   - `research`、`design`、`spec`、`runbook` 在各自目录内独立取现有最大编号 +1（空目录从 `0001` 起）；
   - `plan` 取新号时**必须同时扫描 `docs/plan/` 与 `docs/plan/archive/`**，取合并最大编号 +1，防止与已归档计划撞号；
   - 不同模式的序列互不占用（`research/0001`、`design/0001`、`spec/0001` 可同时并存）；跨模式演进靠文首 `承接 [NNNN](相对路径)` 链接，严禁跨目录续号。
2. **`<scope>`（业务域）**：取自 [`../scope-manage.md`](../scope-manage.md) 词表的英文单词列，与目录模式正交。
3. **`<title>`**：kebab-case 英文短标题。

## 3. 原位演进与归档纪律

1. **单文件原位演进**：`design` 与 `spec` 从 `draft`（起草）到 `rfc`（提案描述/评审）再到 `adr`（定稿约束/冻结）始终在原文件内更新状态与收敛内容，不跨目录搬家；
2. **定稿锁定**：`design` 与 `spec` 状态升为 `adr` 须经人工确认；升为 `adr` 后核心条款锁定，后续推翻须新建文档并将旧文标为 `superseded`（双向互链）；
3. **计划完结归档**：`docs/plan/` 下平铺活动计划；当计划进入 `done`、`cancelled` 或 `superseded` 时，必须移入 `docs/plan/archive/` 并修正文内相对路径；
4. **分模式台账同步**：每次新增、改状态或归档后，必须同步更新 [`../README.md`](../README.md) 或 [`../plan/README.md`](../plan/README.md) 对应模式分表，并可用 `doc-clean` 校验。
