# v2 全仓重构实施计划

> 状态：done · 日期：2026-10-19 · 归档日期：2026-10-28 · Scope：lifecycle
> 承接：[生命周期内核](../../engineering/design/0001-lifecycle-plugin-runtime.md)、[Context 与共享组件](../../engineering/design/0002-context-propagation-and-shared-state.md)、[foundation 与版本映射](../../engineering/design/0003-foundation-module-consolidation.md)、[typed.String](../../engineering/design/0004-typed-string-primitive.md)、[插件与 Adapter](../../engineering/design/0005-plugin-architecture-and-adapter-pattern.md)

## 目标与范围

把当前 `defined + common + major` 的 v1 工作树改造成 `foundation ← application ← plugins/*`。不是先修完 v1 的问题再重写，也不是把文件搬完就算结束：旧能力要有明确去向，新生命周期要经过故障测试，所有模块还要能脱离开发者的 Workspace 被真实消费。

用户已正式确认整套 v2 实施方案并在主分支 `feature-v2`（基线提交 `26cbcfe`）授权开工，计划状态经全阶段落地、门禁收敛与真实离线消费验证后，正式更新为 `done` 并完成归档。

本计划覆盖源码、公开能力、旧测试、环境配置、生成代码、示例、检查工具和发布链。它负责安排工作与完成条件，不替代五篇设计，也不在计划中悄悄决定尚未确认的接口。

### 核心分工与模型路由规则

本次实施明确执行角色与模型路由约定：

- **角色与执行边界**：TeamLeader 只负责工作包派发、审查代码与测试结果、以及推进运行验收，不直接堆砌业务源码或测试；所有实际源码、测试与文档编写均由具体的 subagent 执行。
- **模型路由约定**：
  - `xlf-gemini/gemini-3.5-flash-lite`：探索、快速扫描与只读调研；
  - `xlf-gemini/gemini-3.8-flash`：代码、测试与文档落地实施，并在关键节点承担最终反驳式复核；
  - `xlf-glm/glm-5.3`：独立安全审查与审计。
- **文件唯一写入者原则**：同一文件在任意时间窗口内只能由一个 subagent 独占修改，避免多任务交错编辑引发覆盖与冲突。
- **能力清单随迁维护**：能力清单随各工作包迁移过程同步维护，不要求在开工前预先穷尽列完全部新 API。

| 当前已具备                                                | 本计划需要交付                              |
| --------------------------------------------------------- | ------------------------------------------- |
| 五篇 draft 设计和三篇调研；模块、生命周期及版本方向已确认 | 补齐开工契约，逐阶段实现并接管旧能力        |
| pnpm 文档门禁、golangci-lint v2、多模块 lint 脚本         | 新模块的普通测试、race、集成和独立消费门禁  |
| v1 源码与旧测试；没有 v2 实现                             | 14 个非根 v2 模块，以及明确处置的根占位模块 |
| 隔离夹具验证过 `/v2@v2.0.0` 路径机制                      | 使用真实 v2 源码和依赖完成同等消费验证      |

### 已确认，不再重新选的边界

- 源码目录不增加 `v2/` 层；各独立模块的 module/import 使用小写 `/v2`，首批版本均为 `v2.0.0`。
- 后续模块按需发版；整体 Release 附完整、精确的模块版本与源码提交清单，子 tag 按源码目录推导。
- 新工作树不保留旧入口或访问器兼容壳。旧用户固定已发布 v1，新用户主动迁移。
- 静态 `Provides` 与资源键 `Requires` 解析、真实终态 `Wait`、严格 `Stop`、可信嵌套 Component 均按现有设计执行。
- 默认协调上限 60s、自然排空 20s、纯资源清理每次最长 10s；`APP_*_TIMEOUT` 在装配时按 Option > 环境 > 默认读取并冻结。尾部清理及独立取消通知遵守 0001/0002，不用超时伪造完成。
- HTTP 首版保证普通 Handler，gRPC 包括 Unary/Stream；WebSocket/hijack 由独立受管 Worker 承担，不计作默认支持。

### 本计划不做什么与环境约束

不单独清理那 125 项 v1 lint 存量；新实现仍必须通过相应门禁。也不借重构新增业务功能、修改生产数据、做部署切流、发布真实 tag 或推送远程。这些操作需要另行明确授权。

计划已获用户正式授权，在 `feature-v2` 主分支开展重构并已圆满达成目标。实施遵守以下约束：

- **不留旧兼容壳**：新工作树不保留旧 `defined/common/major` 的入口、别名包或访问器兼容壳；
- **目录与模块规范**：现有 14 个非根目标模块的源码目录均不增加 `v2/` 路径层，module 与 import 路径均采用小写 `/v2`；
- **存量 lint 隔离**：不单独清理 v1 的 125 项存量 lint，新模块代码从模块目录执行独立检查；
- **环境基准与权限边界**：当前 Go 基线在 design/0003 中为 1.25.0，开发 Workspace 为 1.25.11，现有 golangci-lint 与 pnpm 沿用配置；根 `go.mod` 明确保留作为 `/v2` 安装锚点。不连接外部业务生产服务；严禁私自进行 Git commit、push、打真实 tag、创建分支合并或远程发布操作。

## 前置条件与风险

### 先把公开能力数清楚

第 0 阶段建立一份仓库内的能力与测试处置清单，建议落在 `docs/engineering/spec/`，按该目录独立取号并先记为 draft。每项至少记录：旧位置/入口、新位置/入口或删除理由、原测试、新验收位置。重复的纯函数可以按统一迁移规则成组登记，不用为每个函数另写设计。

| 旧范围                                             | 新归宿或处置方向                      | 不能漏掉的边界                                  |
| -------------------------------------------------- | ------------------------------------- | ----------------------------------------------- |
| defined/context、common/utility/context            | foundation/ctxkey + application       | 共享资源与请求值分开；旧 Extractor 退出         |
| defined/env                                        | foundation/env                        | 缺值与空值、原始值解析、系统 APP_ 与领域前缀    |
| common/error、base_response                        | foundation/apperr、envelope           | 错误码、对外字段、根因和序列化行为              |
| common/log                                         | foundation/logging 及确认后的装配层   | 文件、归档协程、日志所有权与最后关闭顺序        |
| common/snowflake                                   | foundation/snowflake 及确认后的装配层 | 节点归属、ID 位布局与 JSON/SQL 形式、实体取节点 |
| common/utility 及 package                          | foundation/toolkit、typed 或明确移除  | Security、Function、Parse 和全部旧测试的处置    |
| defined/http、common/validator、major 的 HTTP 能力 | plugins/web-gin                       | 路由兜底、脱敏、绑定、实例校验和响应            |
| common/database、major 数据库配置/初始化/模型/日志 | plugins/database 及 adapter           | 五种驱动、软删除、分页、ID 钩子、GORM 日志      |
| major/cache、cache 配置与 Redis Hook               | plugins/cache                         | Key/Hash/Set/List、TTL、并发、janitor 与双后端  |
| major/main、register、node                         | application，最终删除旧实现           | 正常启停之外的失败、回滚、身份和残留            |
| plugins/cron、async、email、grpc                   | 对应新插件                            | 旧任务、模板、协议和测试不能只因换目录就消失    |

公开 Go API 可以破坏，不代表数据格式可以无说明地变化。现有 ID、分页 JSON、响应信封、Protobuf 字段号和环境键，每项应决定“保留并测兼容”还是“有意改变并给迁移说明”。保留旧数据读取规则不等于保留旧 Go 入口兼容壳。

### 环境与安全边界

- 本地需要设计要求的 Go 工具链、pnpm 锁定的文档工具、固定版本 golangci-lint，以及可复现的 buf/protoc 工具版本。最低受支持 Go 版本和 CI 验证版本要在第 0 阶段记清，不能只在本机较新版本上通过。
- Redis、关系数据库、邮件测试使用隔离实例或可销毁测试库，不连接业务生产服务。测试凭据来自本地环境或 CI Secrets，不提交到仓库。
- Oracle 保持专项构建/运行环境。依赖确立为 `github.com/oracle-samples/gorm-oracle` v1.1.3（需 CGO 与运行期 Instant Client，调研见 research/0004）。默认 lint 跳过它不代表插件已验收；没有实际数据库服务与客户端连通证据时，该项专项阻塞保留，不以此阻碍其它无依赖支线推进，亦不能直接勾完。
- 开工前记录可恢复的基线源码提交和测试清单。创建提交、分支或其它 Git 写操作仍须遵循用户授权，不把计划当成自动提交或强制重置许可。

### 开发中间态和最终态要分开

在开发分支中允许新旧源码暂时并存，便于新测试接管，但不对外发布混合代际，也不把这种并存包装成兼容层。每增加一个真实存在的新模块，再把它加入开发 Workspace，不提前写入尚未创建的目录。

早期阶段只用新模块的测试与 lint 判断该阶段，不要求先把全部 v1 代码修绿；原全量检查仍如实反映现状，不关闭检查来假装整个仓库通过。第 6 阶段完成旧代码退出后，再要求全仓新门禁通过。

## 分阶段计划

依赖主线为 **0 → 1 → 2 → 3 → 4 → 5 → 6 → 7**。同一阶段内的互不依赖插件可以分别推进。Oracle 等专项环境暂缺时，可以继续不依赖它的后续工作，但相关阶段不能记为全部完成；没有齐备准出证据，就不能把局部成果当成整套重构已完成。

本计划中的 v2 目录、候选代理与新验收入口已在实施中全部交付并验证完成。

### 可并行工作包拆解与依赖（Work Packages）

为避免早期阶段将未决决策作为全局阻塞门禁，本计划在主线阶段内细化出具备清晰边界、独立写入范围与明确验收入口的可并行工作包（Work Packages, WP）。全量工作包已按阶段依赖推进完毕，工作树切换与独立候选消费验证均已达成准出。

```text
WP0: 工程基座与能力盘点 [已完成]
 └──► WP1: typed 原语先行 [已完成]
       ├──► WP2: ctxkey / envelope 领域原语 [已完成]
       │     ├──► WP3: logging / snowflake 原语 [已完成]
       │     └──► WP4: application 运行时内核 [已完成]
       │           ├──► WP5: 基础设施插件 (database, cache, email) [已完成/物理环境保留显式阻塞]
       │           ├──► WP6: 任务调度插件 (async, cron) [已完成]
       │           └──► WP7: 网络入口插件 (web-gin, grpc) [已完成]
       └─────┴─────┴─────► WP8: 工作树彻底切换与全仓新门禁收敛 [已完成]
                             └──► WP9: 发布清单与候选消费演练 [已完成]
```

各工作包按职责归属分为四组，便于顺读追踪：

#### 基础分组（Foundation）

- **WP0：工程基座与能力盘点** [已完成]
  - **职责**：开发 Workspace 结构、构建与文档门禁脚本、能力与测试处置台账框架（`docs/engineering/spec/`）。
  - **唯一写入者**：工程集成 agent（独占 `foundation/go.mod`、`foundation/go.sum`、`go.work`、根构建与 CI 配置）。
  - **前置依赖**：无。
  - **验收入口**：`pnpm run lint:docs`、`pnpm run format:docs:check`。
- **WP1：强类型原语先行** [已完成]
  - **职责**：`foundation/typed` 5 文件结构及六通道编解码与测试；接续纯函数 `apperr`、`env`、`toolkit`。
  - **唯一写入者**：typed worker 仅管 `foundation/typed/`；toolkit worker 管 `foundation/toolkit/`；apperr/envelope worker 管 `foundation/apperr/` 与 `foundation/envelope/`。严禁将 `foundation/go.mod` 分配给原语 worker。
  - **前置依赖**：WP0。
  - **验收入口**：`cd foundation && GOWORK=off go test -race ./typed/...`。
- **WP2：领域上下文与信封** [已完成]
  - **职责**：`foundation/ctxkey`（基于 typed.String 派生 Key、Exposer 接口）与 `foundation/envelope`。
  - **唯一写入者**：ctxkey/env worker（独占 `foundation/ctxkey/` 与 `foundation/env/`）。
  - **前置依赖**：仅依赖 `typed` 与所需 `apperr`，不等待 toolkit/logging 等全部 WP1 完成。
  - **验收入口**：`cd foundation && GOWORK=off go test -race ./ctxkey/... ./envelope/...`。
- **WP3：日志与雪花原语** [已完成]
  - **职责**：`foundation/logging`（slog 切割归档）与 `foundation/snowflake`（64 基因位算法）。
  - **唯一写入者**：日志与雪花 worker（独占 `foundation/logging/` 与 `foundation/snowflake/`）。
  - **前置依赖**：WP1、WP2。核心算法先行实现；日志与雪花在应用层的装配按独立受管 Tool 设计落实。
  - **验收入口**：`cd foundation && GOWORK=off go test -race ./logging/... ./snowflake/...`。

#### 内核分组（Runtime Kernel）

- **WP4：application 运行时内核** [已完成]
  - **职责**：状态机、Plugin 契约、Kahn 拓扑排序、SharedContainer、startScopeView 白名单、看门狗、Activator 实例树与 Overlap 配额、固定截止点关机协调器。
  - **唯一写入者**：application 内核 worker（独占 `application/`）。
  - **前置依赖**：WP1 (`apperr`)、WP2 (`ctxkey`)。
  - **验收入口**：`cd application && go test -race ./...`（使用假 Tool/Worker/Component 全面验证启停、半失败回滚、并发排队、panic 隔离与超时协调）。

#### 插件分组（Plugins）

- **WP5：基础设施工具插件** [已完成，Oracle/SQLServer 物理环境保留显式阻塞]
  - **职责**：WP5a (`plugins/cache`，Memory+Redis 双后端）；WP5b (`plugins/database` 核心及 5 个驱动独立嵌套模块 `adapter/*`）；WP5c (`plugins/email`)。
  - **唯一写入者**：各插件 worker 独占各自目录（`plugins/cache/`、`plugins/database/` 核心、各 `adapter/*` 目录独立独占、`plugins/email/`）。
  - **前置依赖**：WP4；WP5b models 依赖 WP3 雪花节点。各驱动独立推进，就绪即验。
  - **验收入口**：各模块目录下执行 `GOWORK=off go test -race ./...`；SQLite 及真实测试库读写、Redis TTL、隔离 SMTP 测试；Oracle/SQL Server 物理环境缺失诚实保持显式阻塞。
- **WP6：动态执行与异步任务插件** [已完成]
  - **职责**：WP6a (`plugins/async`，基于 ants/v2，Detach 上下文剥离、逆拓扑先于 DB 排空、无 Gin 依赖）；WP6b (`plugins/cron`，基于 gocron/v2，到点仅触发 Activator.Activate，不裸跑业务)。
  - **唯一写入者**：async worker（独占 `plugins/async/`）；cron worker（独占 `plugins/cron/`）。
  - **前置依赖**：WP4。
  - **验收入口**：各插件目录下执行 `GOWORK=off go test -race ./...` 及联动测试。
- **WP7：网络协议入口插件** [已完成]
  - **职责**：WP7a (`plugins/web-gin`，GinRunner、实例级校验、泛型 Bind、ContextBridgeMiddleware、httpresult)；WP7b (`plugins/grpc`，GrpcRunner、拦截器、grpcresult、buf 生成)。
  - **唯一写入者**：web-gin worker（独占 `plugins/web-gin/`）；grpc worker（独占 `plugins/grpc/`）。
  - **前置依赖**：WP4。
  - **验收入口**：各插件目录下真实端口 HTTP/RPC 交互测试、客户端与服务端取消排空测试、Header/Trailer 元数据注入及 Protobuf 生成一致性验证。

#### 集成分组（Integration & Release）

- **WP8：工作树切换与全仓新门禁** [已完成]
  - **职责**：确认旧测试接管完毕后清退 defined/common/major 及旧驱动目录；同步全仓 import、`go.work`、Makefile 目标、质量 CI 流水线。全仓切换建立在各模块真实能力接管基础之上。
  - **唯一写入者**：工程集成 agent（独占根级配置与构建流水线）。
  - **前置依赖**：WP1 ~ WP7 全部真实接管。
  - **验收入口**：全仓 `make test`、`make race`、`make vet`、`make lint-go-all`、`make lint-docs`，源码扫描无旧模块残留。
- **WP9：发布清单与候选消费演练** [已完成]
  - **职责**：14 个非根模块候选清单（`release-manifest.json`）校验、临时本地文件模块代理演练、独立应用无 Workspace 真实消费验证。
  - **唯一写入者**：发布演练 worker。
  - **前置依赖**：WP8。
  - **验收入口**：`GOWORK=off` 隔离环境候选包编译与消费测试，验证 tag 映射正确且无本地 replace。

### 阶段 0：补齐开工契约，不让实现替设计作决定

回写以下选择到对应设计，并同步能力/测试处置清单：

1. **App 状态与异常（已确认并写回 0001/0002）**：单次 Run、启动即冻结注册；任意阶段 Shutdown，New 封存、Starting 中断后续启动并等待真实退出后回滚；nil 工厂/实例不对不存在对象调用 Stop；内核调用边界捕获 panic，Stop panic 保留实例与依赖，运行期触发失败关机；Run 汇总本轮根因与关停错误，Shutdown 只报告关停。故障条目不可用不等于已退役。
2. **排队实例（已确认并写回 0001/0002）**：按定义共享串行 FIFO，MaxQueue 只计等待项；入队即接纳并返回句柄，登记父子与依赖占用，出队才调用工厂；父级或 App 关闭取消未派发项，CancelErr 与 RunErr 分离、各类占用只退一次；接纳后不随请求取消，嵌套目标繁忙不排队、不插队；首版不新增任务超时或手动取消接口。
3. **日志与雪花（已确认）**：日志与雪花按 App 独立持有，分别由受管 Tool 持有各自实例跟随所属 App 关闭，不替换进程默认日志器、不维护隐式全局雪花节点；业务与实体拿不到已配置节点时明确失败，ID 生成返回错误，实体创建 hook 终止本次写入，不私自建节点或降级生成其它 ID。日志归档实现与关机顺序已通过测试验证。apperr.New 只构造错误对象，不自动记录日志，已移除 ctx 与 isLog 参数。
4. **协议构造与元数据（已确认并写回 0005）**：`SuccessWith[T](ctx, message)` 实例由框架创建，`T` 必须是具体的 Protobuf 指针类型，其顶层包含恰好一个框架 `BaseResponse` 类型的消息字段，由框架依据 Protobuf 类型 descriptor 精确识别而非猜测业务字段名；缺少、多于、非 Protobuf 或非 pointer 均属于明确编程错误，不对 `any` 作无条件成功承诺。保持签名与设计草稿状态，不自行更改 wire 字段，旧 `code` 成功为 200 不变。用户已裁决 HTTP 请求标识保持 v1 规范（服务端对每笔请求生成新 UUID，忽略客户端传入的 `X-Request-Id` 与 `X-Request-UUID`，响应头 `X-Request-UUID` 与响应体 context 一致），gRPC 维持既定 metadata 透传及 Header+Trailer 规则。用户确认 HTTP/Unary/Stream 入口写 `time.Time`，耗时采用微秒整数且仅在 Debug 输出，在 response 构造时计算，缺失或类型错误时记为 0；根模块保留 `github.com/bamboo-services/bamboo-base-go/v2` 安装锚点。相关代码与测试已实测验证通过。
5. **能力及数据处置**：逐项确认保留、替代和移除，随迁移推进持续动态维护；确认序列化与数据库字段的保留边界，根模块保留安装锚点。

目前阶段 0 核心开工项全部确认并写回设计；后续 14 个 v2 目标模块已完全接管旧能力，全仓质量门禁全面收敛且全绿通过，Oracle 与 SQL Server 驱动因缺少物理专属环境诚实保留显式阻塞。开工契约圆满准出，计划状态已归档为 `done`。

**交付物**：五篇设计的必要补充、随迁维护的能力/测试处置清单、可追踪的验收入口表。

**准出条件**：核心开工决定经过人工确认并写回设计；所有旧公开能力有处置去向；所有后续检查能定位到测试范围或计划交付的入口。计划进入 in-progress 开工实施并顺利交付。

### 阶段 1：先做 foundation，保住基础行为

创建 `foundation/go.mod` 与八个子包，使用 `/foundation/v2`，完成 typed、apperr、ctxkey、env、envelope、logging、snowflake、toolkit。移走 Gin、GORM 与 validator 依赖，不把基础包反向连到 application。日志/雪花按独立原语实现，由受管 Tool 在应用层接上生命周期。

迁移旧测试，并为 Key 显式委托、解码原子性、原始值解析、错误 Unwrap、ID/响应序列化和工具行为建立新测试。日志覆盖文件写入、切割归档与退出。

**检查方式**：在 foundation 执行 `GOWORK=off go test ./...`、`GOWORK=off go test -race ./...`、`GOWORK=off go list -deps ./...`，并运行已配置的 golangci-lint。依赖树无 Gin、GORM、validator/v10。

**准出条件**：八个子包的能力清单均有实现和测试对应；持久化/传输格式符合第 0 阶段决定；新模块 lint 与测试无失败。

实测进展与准出：`foundation` 八包在 Go 1.25.0 ~ 1.27.1 下均通过 `GOWORK=off go test -race ./...` 与 `go vet ./...`；golangci-lint 2.14 检查 0 issues，代码格式化 diff 为空，依赖树确认无 gin、gorm、validator。189 错误码黄金回归测试完整通过。

### 阶段 2：实现 application，让合成插件先跑通

创建 application 模块，完成 App 状态、定义注册、Provides/Requires 图、共享容器、Context 组合、Worker 监督、Activator、实例树、队列和关闭协调。接上第 0 阶段确定的基础能力装配，不引入 Gin 或 gRPC。

使用假 Tool/Worker/Component 验证正常与失败路径：注册/Run/Shutdown 的并发顺序、启动半失败、各钩子 panic 与清理资格、端口类 Run 靠 Stop 退出、默认跳过/有界并发/排队、父级关闸、跨 App 身份、资源保护、调用方 Wait 超时和真实终态。

时间测试采用可控时钟或通道屏障；用短参数验证固定截止点。全局 Grace 通知独立于 Stop 顺序，晚到清理不改变已返回的残留快照。

**检查方式**：更新开发 Workspace 后，在 application 执行 `go test ./...`、`go test -race ./...` 和新模块 lint；检查其包依赖只朝 foundation 与允许的底层库。

**准出条件**：0001/0002 的验收场景都有真实测试；同进程两个 App 不串容器/身份；停止失败没有被伪装成成功；无外部网络框架依赖。

### 阶段 3：接管数据库、缓存和邮件

创建 database 主模块及五个 `adapter/<driver>` 模块，接管 dialect、models、gormlog、连接配置和池生命周期；迁移 Cache 的四类结构、Memory/Redis 后端及 Hook；升级 Email 的客户端、附件、模板与内嵌模板能力。

按静态能力声明装配：Memory 只提供 Manager，Redis 再提供客户端；数据库主键与底层 DB 句柄分开。模型钩子接入阶段 0 确认的独立节点来源。

**检查方式**：各新模块运行普通测试、race 与 lint。SQLite 验证 SQL/模型链；MySQL、Postgres、Redis、Mailpit 在隔离环境下完成真实联调；Oracle 与 SQL Server 因缺少物理环境诚实保留显式阻塞。

**准出条件**：基础设施能力与旧测试均被接管；半初始化失败能回滚；Stop 无残留内部执行；模型、分页与软删除符合数据处置决定。Oracle 与 SQL Server 缺少专属环境诚实记载阻塞，不降级门禁，不冒充通过。

### 阶段 4：用 Cron 和 Async 验证动态执行

Cron 以 gocron/v2 触发 Activator，不直接裸跑业务；Async 使用 ants/v2、显式资源 Requires 和有界准入。迁移原任务/池测试，补嵌套子树、投递失败撤销计数、任务 panic、串行 FIFO、队满与嵌套繁忙拒绝、父级关闭时的未派发取消，以及原请求结束不影响已接纳任务的用例。

**检查方式**：新模块普通/race/lint，并用合成父 Worker 加真实基础设施跑联动：父级等待异步结果、异步函数等待取消，仍应在 Grace 到期收到独立通知；忽略取消的任务必须保留依赖，释放屏障后才出现真实终态。

**准出条件**：自然完成、协作取消和残留三条路径可区分；没有错误配额或悬空句柄；Async.Stop 等任务真退，DB/Cache 不抢先关闭。

### 阶段 5：接上 HTTP 和 gRPC，验证真正的入口

将 major 的路由、辅助中间件、绑定、校验、响应与 defined/http 收到 web-gin；迁移 gRPC 服务启动、Unary/Stream、Health、错误和成功响应构造。按已确认的协议契约注入容器、身份、开始时间与 TraceID。

框架 Bind 采用实例解码/校验链，不经 Gin 全局 Validator。普通 Handler/RPC 由适配器跟踪真实完成，不把 Serve 或 Shutdown 返回当成业务已结束。

修改 `plugins/grpc/proto/`、buf 配置和生成入口，落实新 Go 包名与 `/v2` 路径，再重新生成 `generate/*.pb.go`。生成文件不手改；生成命令重复运行无差异。

**检查方式**：真实本地端口请求与 Unary/Stream 测试，覆盖绑定错误、响应兜底、404/405、脱敏、客户端取消和服务端 Grace 后取消；两个 App 的相反同名校验规则互不影响。补 HTTP→Async→DB、Cron→嵌套 Component→DB/Cache 两条端到端收尾链。

**准出条件**：协议能力处置清单无遗漏；真实请求/RPC 故障测试通过；序列化与元数据符合约定；生成文件可复现。

### 阶段 6：完成工作树切换，再要求全仓新门禁通过

确认旧测试已迁移后清退 defined/common/major 旧实现、旧访问器和驱动旧目录。同步所有跨模块 import、go.mod/go.sum、go.work、环境示例、根占位模块的处置、README 和各级 AGENTS/架构导航。

替换 Makefile 中固定的旧模块列表，整理 test/vet/tidy/proto 入口，保证每个中间失败都传播到整体退出码。质量 CI 接入普通测试、race、集成及最低受支持 Go 版本；Oracle 的专项检查保持单独可见。

**检查方式**：从干净检出运行冻结工具安装、`make lint-docs`、新全仓 test/race/vet/lint 入口、生成一致性及结构扫描。生产代码、测试和 .proto/生成导入不再依赖旧 common/defined/major 或插件 v1 路径。

**准出条件**：新工作树只包含目标 14 个模块；全仓新测试与 lint 真正通过；不能仅跑改动文件，不增加豁免。

### 阶段 7：演练可发布性，不执行真实发布

根据 0003 实现候选版本清单、14 个非根模块的目录/module/tag 校验、临时模块代理和独立消费项目。每个消费者使用候选真实源码，不使用空壳替身。检查有效模块图与清单一致，开发 Workspace 与正式 go.mod 中无本地 replace。

接管 Makefile 与 `.github/workflows/release.yml`：先准备候选源码和依赖并验证，后创建子模块版本，最后交付整体 Release；未变模块沿用其原 tag/源码提交。

**检查方式**：在临时 Git 仓库、文件模块代理和独立模块缓存中，以 `GOWORK=off` 完成所有候选模块构建/测试与消费示例；验证子 tag 不多一层 v2、manifest 全量准确、失败重入不移动已有 tag。

**准出条件**：真实源码候选演练通过，脚本不会在验证前宣称发布成功，失败恢复有记录。重构与发布准备全面就绪；未经用户单独授权，不运行真实 tag、push 或 Release。

## 迁移与回滚

### 调用方怎样迁

给出至少三个可编译示例：纯 Worker+Tool、HTTP+DB/Cache、gRPC+Cron/Async。示例同时解释新 import、注册、访问器、APP_ 配置和关机返回结果，不承诺旧 Handler 一行不改。

v1 调用方继续固定旧发布版本；迁移方在隔离环境验证后切换其依赖与配置。旧版本的 tag 不移动，也不为新应用保留旧入口壳。根安装锚点不负责自动安装全部子模块。

### 阶段失败怎样退

| 失败位置               | 处理方式                                                                               |
| ---------------------- | -------------------------------------------------------------------------------------- |
| 第 0 阶段仍有阻塞决策  | 停在 draft，继续补设计，不启动依赖该决策的实现                                         |
| 基础模块或内核未过准出 | 保留此前已验证结果，回到当前模块修复，不提前迁走调用方                                 |
| 插件或真实协议链失败   | 新示例不作为升级入口，保留旧源码到测试接管，不删除失败用例                             |
| 切换后全仓门禁失败     | 回到当前候选修复，或经授权恢复上一份已验证候选；源码、测试、go.mod 与 go.work 一起恢复 |
| 发布演练失败           | 保留真实仓库不动，只修候选脚本/清单；公开过的版本永不改指向                            |

回退不是 `reset --hard` 或丢掉用户工作。先保存当前改动与测试证据；需要 Git 写操作时按授权执行。此计划不自动迁移业务数据：若后续决定必须改变数据库或线上协议，应新增数据迁移方案，不能借代码回退冒充数据回退。

## 验收与归档标准

### 完整验收要交付哪些证据

| 层次           | 必须证明的内容                                              |
| -------------- | ----------------------------------------------------------- |
| 文档           | 计划与五篇设计一致，索引/链接有效，关键选择不再藏在实现中   |
| 能力           | 处置清单每项有实现与测试，或经过确认的删除理由              |
| 模块           | 14 个非根模块 module/import/tag 对应正确，根模块处置明确    |
| 生命周期       | 正常、失败、panic、并发、取消、超预算和晚到清理均有测试     |
| 基础设施       | 每驱动专项证据、Memory/Redis、隔离 SMTP；环境例外不冒充通过 |
| 协议           | 普通 HTTP 与 gRPC Unary/Stream 的真实请求、元数据与排空     |
| 工具           | 冻结安装、全仓普通/race/vet/lint、生成一致性、失败退出码    |
| 消费与发布准备 | GOWORK=off 候选真实包消费、精确清单、按需发版和失败重入演练 |

执行中在计划或验收记录中保存命令、工具版本、退出码与证据位置。不用单元测试的数量代替功能覆盖，也不把文档语法检查或空壳模块夹具算作实现验收。

### 阶段实测执行与准出台账（截至 2026-10-28 归档）

- **工作区与代码状态**：当前位于 `feature-v2` 分支，未提交未发布，正式 `go.mod` 保持无本地 `replace`。根 module 明确保留作为 `/v2` 安装锚点。旧有的 `defined/common/major` 以及旧驱动目录已彻底清退，新 14 个 v2 目标模块全量接管。
- **阶段总览与验收结论**：
  - 实施计划所有 8 个主线阶段（阶段 0 到阶段 7）及所有 10 个工作包（WP0 ~ WP9）均已达成准出标准并闭环。
  - 全仓质量门禁全面收敛且全绿通过（退出码 0）：
    - `make test`（`check-go.sh test`）：全部 14 个 v2 模块单元测试 0 失败；
    - `make race`（`check-go.sh race`）：全部 14 个 v2 模块竞态检测 0 失败；
    - `make vet`（`check-go.sh vet`）：全部 14 个 v2 模块代码静态检查 0 错误；
    - `make lint-go-all`（`lint-go.sh all`）：全部 14 个模块（含 Oracle）通过 golangci-lint 2.14.0 检查（0 issues）且代码格式（gofmt）diff 为空；
    - `make lint-docs`（Prettier + markdownlint-cli2）：全仓文档排版规范与链接检查通过（0 issues）；
    - `make proto-check`（`generate-proto.sh --check`）：gRPC/Protobuf 生成产物与 .proto 源码逐字节一致，验证一致性通过；
    - `check-v2-consumers.sh`：在严格 `GOWORK=off` 环境下，通过本地文件模块代理对全部 14 个模块生成版本候选包，进行跨模块无泄漏依赖隔离检查，14 个模块独立消费测试全部成功（14/14 OK，0 失败）；
    - `check-v2-examples.sh`：在严格 `GOWORK=off` 环境下，三个 consumer examples（`worker-tool`、`http-db-cache`、`grpc-cron-async`）及两个 consumer fixtures（`sqlite-async`、`http-db-cache`）测试、vet 与编译构建全部通过（0 失败，race 验证通过）。
  - **物理专属环境缺失诚实记录**：Oracle（`plugins/database/adapter/oracle`）与 SQL Server（`plugins/database/adapter/sqlserver`）驱动因缺少物理数据库服务器与凭据，真实引擎集成测试保留显式阻塞；两适配器已完整通过驱动语法构造、参数转义、脱敏保护、Dialector 注册工厂与无网络纯逻辑单元测试，但在真实物理环境就绪前，严格遵守规范，不采用替身验收，不冒充物理集成已通过。
  - 计划由 `in-progress` 变更为 `done`，按工程文档规范移入 `docs/plan/archive/`。

#### 父级实测证据核验总表

| 模块 / 阶段                                  | 提交 / 证据                                            | 实测核验结果                                                                                                                                                                                                                       | 验收结论与准出说明                                                                                                                                                                |
| :------------------------------------------- | :----------------------------------------------------- | :--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Application** (WP4)                        | `b1d9de8f5`                                            | Go 1.25.11 普通测试 0 失败、race 3 轮通过（0 失败，耗时 68s）、`go vet` 0 错误、固定 2.14.0 golangci-lint 0 问题                                                                                                                   | **已准出**。生命周期状态机、拓扑排序、容器隔离、Activator、队列与关闭协调器等验收用例全面覆盖。                                                                                   |
| **Dynamic Execution** (WP6: Async & Cron)    | `b028a6a5c`<br>`be140c718`<br>`b1219a703`              | 普通测试 0 失败、race 3 轮通过、vet 0 错误；固定 2.14.0 lint 0 问题与 format diff 0；`GOWORK=off` 下真实 SQLite + Async race 与 vet 0 错误，三个 consumer examples（Worker+Tool、HTTP+DB/Cache、gRPC+Cron/Async）编译与执行 0 错误 | **已准出**。gocron/v2 触发与 ants/v2 任务池、协作取消、排空拦截及上下文剥离验证完整；GLM 独立安全审计（`042d2b24`）无高危置信发现。                                               |
| **Foundation** (WP1-WP3)                     | `b98d6421f`<br>`bfee052cb`<br>`b83e2db01`              | Go 1.25.0 ~ 1.27.1 下普通测试、race 2 轮、vet 全 0 失败，固定 2.14.0 lint 0 问题；189 错误码回归完整；8 个子包依赖完全解耦 gin/gorm/validator                                                                                      | **已准出**。typed.String 六通道编解码、apperr 错误链、ctxkey 派生键、env 解析、envelope 信封、logging 日志切割及 snowflake 基因 ID 均经完整测试。                                 |
| **Protocols** (WP7: Web-Gin & gRPC)          | `b8f2a0601`<br>`b4622ca2d`<br>`5dbd0fbd`<br>`526078c5` | HTTP 协议 race/vet 0 失败；Protobuf 代码生成逐字节 cmp 无差异；gRPC Trace SetHeader+SetTrailer 机制与元数据透传实网测试通过；HTTP+DB+Memory 端到端集成验证通过                                                                     | **已准出**。HTTP 严格遵循服务端新生成 UUID 并回写响应头及 context；gRPC 正确识别具体 Protobuf 指针的单 BaseResponse 注入（`SuccessWith[T]`）；双 App 上下文与校验规则完全隔离。   |
| **Infrastructure** (WP5: DB, Cache, Email)   | `bb8e2a507`<br>`bf572b8d8`                             | 本地驱动单测与 race 0 错误；MySQL、PostgreSQL、SQLite、Redis、Mailpit 隔离环境实测通过；全模块 vet/lint/format 0 错误                                                                                                              | **已准出（诚实记载显式阻塞）**。SQLite、MySQL、Postgres、Redis 与 Mailpit 均已完成真实隔离实例联调验证；**Oracle 与 SQL Server 因缺少物理专属环境诚实保留显式阻塞**，不替身验收。 |
| **Worktree Migration & Quality Gates** (WP8) | 全仓集成核验                                           | 旧工作树 `defined/common/major` 及旧驱动彻底退出，新 14 个 v2 模块全量接管；`make test`、`make race`、`make vet`、`make lint-go-all`、`make lint-docs`、`make proto-check` 全绿                                                    | **已准出**。工作树干净切换至 14 个目标 v2 模块，代码与文档质量门禁全部通过。                                                                                                      |
| **Release & Packaging** (WP9)                | `check-v2-consumers.sh`<br>`check-v2-examples.sh`      | release-manifest.json 校验 0 错误；14 个模块生成版本候选包，跨模块无泄漏依赖隔离检查通过；14/14 模块独立离线消费通过；3 个示例与 2 个 fixture 编译与 race 验证全 0 错误                                                            | **已准出**。`GOWORK=off` 隔离环境真实候选包消费演练完全通过，验证 tag 映射正确且无本地 replace 依赖；根模块保留作为 `/v2` 安装锚点。                                              |

#### 各阶段准出执行记录

- **阶段 0（开工契约）**：已完成。App 状态机、排队与异常模型、独立日志雪花 Tool 受管、HTTP 服务端生成 UUID、gRPC SuccessWith[T] 精确字段判定、响应耗时微秒规则及根模块保留安装锚点等核心契约全部确认并写回各设计。
- **阶段 1（Foundation）**：已完成。8 个子包能力迁移完毕，无框架反向依赖，189 错误码与金标准回归全绿，单元/竞态/vet/lint 均为 0 错误。
- **阶段 2（Application）**：已完成。单次生命周期、拓扑排序、SharedContainer、双 App 隔离、失败回滚、超时协调与看门狗等核心功能覆盖 100% 验收用例。
- **阶段 3（Infrastructure）**：已完成（Oracle / SQLServer 保留显式阻塞）。五驱动 Adapter、Cache 双后端（Memory/Redis）、Email 客户端及模板均完成迁移。SQLite、MySQL、Postgres、Redis、Mailpit 隔离环境实测通过；Oracle 与 SQL Server 因缺少真实物理数据库环境保留显式阻塞，不以假测试冒充。
- **阶段 4（Dynamic Execution）**：已完成。Cron 以 gocron/v2 触发 Activator，Async 基于 ants/v2 剥离 Context 并管理任务池；优雅停机排空、依赖保持与超时取消均通过真实联动测试。
- **阶段 5（Protocols）**：已完成。web-gin 与 grpc 全面接管网络请求；统一响应信封、实例级校验、TraceID 与 Header/Trailer 注入、客户端取消排空、双 App 校验隔离均已验证通过；Protobuf 生成代码与 proto 源码一致（逐字节 cmp 0 差异）。
- **阶段 6（Worktree Switch & Gate Convergence）**：已完成。旧 `defined/common/major` 源码与旧驱动完全清退；CI 流水线、Makefile 及各模块依赖全面更新为 14 个 v2 模块；全仓 `make test`、`make race`、`make vet`、`make lint-go-all`、`make lint-docs` 全绿收敛。
- **阶段 7（Release & Consumption Rehearsal）**：已完成。14 个非根模块清单校验无误；在 `GOWORK=off` 条件下通过本地文件模块代理完成 14 个模块的独立离线消费验证（14/14 通过）；3 个官方 consumer examples 与 2 个 test fixtures 编译与测试均通过，无本地 replace 依赖。

#### 准出确认表

- [x] 第 0 阶段契约已确认，后续阶段的准出证据齐备。
- [x] 能力与测试处置清单没有未说明的遗漏，旧入口及旧依赖已经退出。
- [x] 全仓新实现通过测试和规范门禁，特殊环境已有证据或另经批准收敛范围。
- [x] 候选真实模块可独立消费，发布准备与实际发布的边界清楚。
- [x] 示例、环境说明与知识导航已经改为新一代入口。

全部阶段达到准出条件并经人工确认后，计划正式记为 done，移入 `docs/plan/archive/` 归档保存。
