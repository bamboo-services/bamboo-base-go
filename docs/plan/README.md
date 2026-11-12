# 实施计划看板（Plan）

这里记录“从当前代码怎样走到目标设计”，与 `docs/engineering/` 中的长期设计分开。活动计划平铺在 `docs/plan/`；终态移入 `docs/plan/archive/`。取号时同时扫描两处，规则见 [维护公约](../engineering/MAINTENANCE.md)。

## 进行中计划

| 编号 | 计划                                                                     | Scope      | 状态        |
| ---- | ------------------------------------------------------------------------ | ---------- | ----------- |
| 0002 | [v2 强类型错误返回迁移计划](./0002-foundation-strong-error-migration.md) | foundation | in-progress |

0002已完成规范/计划起草、方向裁决、[P0技术台账](../engineering/research/0005-foundation-strong-error-p0-inventory.md)与[引用/实现者](../engineering/research/0006-foundation-strong-error-p0-callers.md)。2026-11-09按用户授权完成[P1错误核心与准出记录](./0002-foundation-strong-error-migration.md#p1准出记录2026-11-09)：19项追加码、16种原生RPC纯分类、nil安全原语和持久编译/协议测试，已由Leader复核。已知消费者仅仓内C01–C06，无仓外V2消费者；计划仍为in-progress，P2–P7及发布不因P1通过而获得授权。

P1已提交为 `7c4f7fe`、P2已提交为 `fad7597`（均未推送）。2026-11-09按用户授权完成P2实现：Worker丢链修复、typed/ctxkey/env、snowflake/toolkit、logging强类型迁移与[准出记录](./0002-foundation-strong-error-migration.md#p2准出记录2026-11-09待用户审阅)（词表84→91项，全量门禁通过）；开工依据见[P2开工准备包](./0002-foundation-strong-error-migration.md#p2开工准备包2026-11-09)。2026-11-10用户批准P3开工并已完成：application内核契约/账本与全部插件实现者、examples及test消费者[一次原子切换](./0002-foundation-strong-error-migration.md#p3准出记录2026-11-10待用户审阅)为*apperr.Error（零新增码，全量门禁通过），已提交为 `c8abf577`（未推送）；cron/async专属码列为P5准备项。2026-11-12补齐[P4施工清单](./0002-foundation-strong-error-migration.md#p4施工清单2026-11-12仅准备尚未实施)：数据库与五适配器接口、低层错误归属候选、工作包、验收门禁及回滚边界均已列明；2026-11-12用户已确认dialect四项/models一项的归属、500分组与评审边界；[评审结论](./0002-foundation-strong-error-migration.md#p4错误归属与评审结论)已写回。随后用户于2026-11-12批准P4实际实施：50033–50037已集中登记（词表91→96项），[公共契约、五适配器与模型迁移及Leader准出](./0002-foundation-strong-error-migration.md#p4准出记录2026-11-12待用户审阅)已完成，待用户审阅；P5–P7、提交推送与发布仍未启动。

## 已归档计划

| 编号 | 计划                                                                      | Scope     | 终态（done / cancelled / superseded） |
| ---- | ------------------------------------------------------------------------- | --------- | ------------------------------------- |
| 0001 | [v2 全仓重构实施计划](./archive/0001-lifecycle-v2-full-reconstruction.md) | lifecycle | done                                  |

0001 实施计划已于 2026-10-28 完成总体验收并归档。全量 14 个 v2 模块已完全接管旧工作树，全仓质量门禁（test/race/vet/lint-go-all/lint-docs/generate-proto/check-v2-consumers/check-v2-examples）全绿通过，Oracle 与 SQL Server 驱动因缺少物理专属环境诚实保留显式阻塞。

本目录是计划与归档的唯一台账，不再保留顶层 plan/ 的重复入口；编号同时扫描当前目录与 archive/。
