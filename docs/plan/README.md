# 实施计划看板（Plan）

这里记录“从当前代码怎样走到目标设计”，与 `docs/engineering/` 中的长期设计分开。活动计划平铺在 `docs/plan/`；终态移入 `docs/plan/archive/`。取号时同时扫描两处，规则见 [维护公约](../engineering/MAINTENANCE.md)。

## 进行中计划

| 编号 | 计划                                                                     | Scope      | 状态  |
| ---- | ------------------------------------------------------------------------ | ---------- | ----- |
| 0002 | [v2 强类型错误返回迁移计划](./0002-foundation-strong-error-migration.md) | foundation | draft |

0002已完成规范/计划起草、方向裁决、[P0技术台账](../engineering/research/0005-foundation-strong-error-p0-inventory.md)与[引用/实现者](../engineering/research/0006-foundation-strong-error-p0-callers.md)，现补[P1开工准备](./0002-foundation-strong-error-migration.md#p1开工准备包2026-11-09仅文档)：候选码、16种原生RPC兼容分类、实施边界与门禁。2026-11-09用户确认没有仓外V2消费者，已知范围仅仓内C01–C06；P0整体待用户审阅准出，P1尚未获Go实施授权，继续draft，不自动ready。

## 已归档计划

| 编号 | 计划                                                                      | Scope     | 终态（done / cancelled / superseded） |
| ---- | ------------------------------------------------------------------------- | --------- | ------------------------------------- |
| 0001 | [v2 全仓重构实施计划](./archive/0001-lifecycle-v2-full-reconstruction.md) | lifecycle | done                                  |

0001 实施计划已于 2026-10-28 完成总体验收并归档。全量 14 个 v2 模块已完全接管旧工作树，全仓质量门禁（test/race/vet/lint-go-all/lint-docs/generate-proto/check-v2-consumers/check-v2-examples）全绿通过，Oracle 与 SQL Server 驱动因缺少物理专属环境诚实保留显式阻塞。

本目录是计划与归档的唯一台账，不再保留顶层 plan/ 的重复入口；编号同时扫描当前目录与 archive/。
