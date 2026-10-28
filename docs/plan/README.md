# 实施计划看板（Plan）

这里记录“从当前代码怎样走到目标设计”，与 `docs/engineering/` 中的长期设计分开。活动计划平铺在 `docs/plan/`；终态移入 `docs/plan/archive/`。取号时同时扫描两处，规则见 [维护公约](../engineering/MAINTENANCE.md)。

## 进行中计划

| 编号 | 计划           | Scope | 状态 |
| ---- | -------------- | ----- | ---- |
| -    | 暂无进行中计划 | -     | -    |

当前暂无进行中的活动实施计划。

## 已归档计划

| 编号 | 计划                                                                      | Scope     | 终态（done / cancelled / superseded） |
| ---- | ------------------------------------------------------------------------- | --------- | ------------------------------------- |
| 0001 | [v2 全仓重构实施计划](./archive/0001-lifecycle-v2-full-reconstruction.md) | lifecycle | done                                  |

0001 实施计划已于 2026-10-28 完成总体验收并归档。全量 14 个 v2 模块已完全接管旧工作树，全仓质量门禁（test/race/vet/lint-go-all/lint-docs/generate-proto/check-v2-consumers/check-v2-examples）全绿通过，Oracle 与 SQL Server 驱动因缺少物理专属环境诚实保留显式阻塞。

本目录是计划与归档的唯一台账，不再保留顶层 plan/ 的重复入口；编号同时扫描当前目录与 archive/。
