# v2.0.0 生命周期插件运行时设计

> 状态：draft · 日期：2026-10-19 · Scope：lifecycle
> 承接 [v2.0.0 生命周期内核与三模式插件机制调研](../research/0001-lifecycle-plugin-runtime.md)

## 目标与边界

前期调研（[research/0001](../research/0001-lifecycle-plugin-runtime.md)）已经把 v1.x 绑死 Gin、节点缺关闭钩子、以及 `goroutine.go` 假退出的病根扒清楚了。本稿把这些问题转成生命周期契约：已经确认的决定直接写进正文，尚未接上的排空与父子归属问题列在文末，不把方向确认当成所有细节都已定稿。

本次设计在仓库根下新建独立模块 **`application`**（模块路径 `github.com/bamboo-services/bamboo-base-go/application/v2`，纳入 `go.work`），专门充当新一代生命周期内核。首批整体 Release 与各独立模块均从 **v2.0.0** 起步；module/import 使用小写 `/v2`，源码目录不新增 `v2/` 层。后续按需发版及完整版本清单见 [0003](./0003-foundation-module-consolidation.md#异常与兼容)。模块间的单向依赖关系为：

```text
foundation  ←  application  ←  plugins/* (gin, grpc, cron, database, ...)
```

`application` 模块必须满足以下已锁定的实现边界：

1. **纯净无框架依赖**：`application` 只依赖底层的 `foundation` 模块（由原 `defined` + `common` 合并而成，详见 [0003 设计稿](./0003-foundation-module-consolidation.md)），不引入 Gin 或 gRPC；原 `major` 中的 Gin HTTP 服务能力剥离为平等的 `plugins/web-gin` 插件。当前工作树仍是 v1 实现，这些新模块与生命周期能力都是待实现设计。
2. **三模式统一生命周期**：
   - **`Worker`（常驻型）**：应用级单实例，实现 `RunnablePlugin`（`Start` + `Run` + `Stop`）。**就绪口径已锁定**：网络服务类 `Worker` 在 `Start(ctx)` 内完成 `net.Listen` 端口绑定与路由注册即算就绪，随后内核起协程跑 `Run(ctx)` 直接执行阻塞 `Serve`。
   - **`Component`（激发态）**：按需动态创建，同样实现 `RunnablePlugin`。每次通过内核 `Activator` 触发时都调用工厂新建独立实例，依次执行 `Start → Run → Stop`。**重入默认策略已锁定**：默认采用 `OverlapSkip`（上一轮未跑完则直接跳过本次触发）。
   - **`Tool`（工具/注入态）**：应用级单实例，实现 `Plugin`（`Start` + `Stop`）。有状态的数据库连接池在 `Start/Stop` 建池与关池；无状态的官方纯工具包同样必须登记插件并将 `Start/Stop` 写为空实现（`return nil`）。
3. **运行句柄与关机预算口径**：
   - `Wait` 只交付真实终态，调用者超时只结束本次等待；
   - 默认自然排空 `20s`，之后协作取消；纯资源收尾每次最多 `10s`；
   - 整个关机协调过程另设 `60s` 总上限。排空与清理分开，不再沿用旧的“20 + 10 = 30s 总预算”。

| 当前工作树                              | 本次设计后的目标                                   |
| --------------------------------------- | -------------------------------------------------- |
| v1 注册节点与 Runner，仍由 Gin 主导启动 | application 统一管理三种插件，具体协议下沉到插件   |
| 节点没有统一 Stop，缺少真实退出账本     | 按实例追踪 Start/Run/Stop 与子树，超时返回残留快照 |

首版不做自动重启、运行期新增插件定义或跨进程调度。它们需要另一套持久化与恢复契约，不能挤进这个进程内生命周期接口。

> **关于 Context 的单一来源约束**：所有插件方法接收的 `context.Context` 如何派生、如何挂载共享组件容器、如何做启动期 `Requires` 白名单拦截、以及如何传递给 `PluginRunner`，统一在 **[0002 Context 体系与共享组件传递设计](./0002-context-propagation-and-shared-state.md)** 中定义。本文只调用 `0002` 提供的内部构造函数，绝不重复定义。

## 工作方式

### 1. `application` 模块目录与职责划分

```text
application/
├── go.mod          # module github.com/bamboo-services/bamboo-base-go/application/v2
├── app.go          # App 主结构、New()、RegisterWorker/Component/Tool、Run()、Shutdown()
├── plugin.go       # Plugin、RunnablePlugin、Quiescer、PluginInfo、Mode 定义
├── graph.go        # Kahn 分层拓扑排序、环检测、Requires 作用域校验
├── supervisor.go   # Worker 协程监督、runHandle 实现、非预期早退捕获
├── activator.go    # Activator 实现、Component 并发配额与重入控制
└── option.go       # Option 与 Config（StartTimeout/CleanupTimeout/ShutdownBudget）
```

### 2. 从注册到启动：五步点火流水线

调用 `app := application.New(opts...)` 创建应用实例后，整个启动过程严格按以下五步推进：

```text
1. Register*(info, factory)   ──► 仅往 app.defs 字典存定义与工厂函数（零外部资源分配）
2. app.Run() 触发 buildGraph  ──► 校验 Key/Provides/Requires，解析提供者，检查缺依赖/环/Component 作用域，生成 layers
3. 按层初始化 Tool 与 Worker   ──► 调 factory() 建实例 → 注入受限 startCtx → 调 Start(startCtx)
                                  → 端口绑好/连接 Ping 通即算 Ready → 写入 SharedContainer
4. 冻结容器并启动 Worker.Run  ──► 冻结 SharedContainer 为只读 → 开闸放行(gate.Open)
                                  → 为每个 Worker 起受管协程跑 Run(lifetimeCtx)
5. 阻塞监听信号与看门狗通道   ──► 收到 SIGINT/SIGTERM 或任一 Worker 非预期退出 → 转入 Shutdown()
```

其中三个实现细节必须写死在代码里：

- **Kahn 分层拓扑（`graph.go`）**：
  `buildGraph()` 先建一张 `providerByKey` 表：每个插件主 `Key` 指向自身，`Provides` 中的资源键也指向这个插件。比如 `plugin_database` 声明提供 `DatabaseKey`，消费者写 `Requires: [DatabaseKey]`，内核就知道该先启动数据库插件。资源键只是同一个提供者的不同入口，不是额外的生命周期节点。
  然后将每条 `Requires` 解析成提供者插件，去重后进行 Kahn 排序，得到 `layers [][]ctxkey.Key`。启动从底层提供者往上走，关机按反方向收。启动白名单仍保留调用方实际声明的键：声明 `DatabaseKey`，不等于顺手获得数据库插件的其它暴露键。
  主 Key 与资源键共用一个查重空间；空键、非法键、重复声明、多个提供者争用同一键、未找到提供者和自依赖都在调用工厂前报错。插件主 Key 不得再列进自身 `Provides`，请求级的 `RequestKey` 等键也不得被注册为共享资源。
  `Component` 可以声明自己需要的 Tool/Worker，但不能声明 `Provides`，也不能成为任何静态依赖的提供者。解析依赖时遇到它就返回 `ErrInvalidDependencyScope`，避免把“按需才存在的任务”当成启动时可用的共享组件。
- **启动失败的原地回滚**：
  在顺着 `layers` 调 `inst.Start(startCtx)` 的过程中，只要第 $k$ 个插件返回了非 `nil` 错误，内核立刻停下脚步：先对已经返回错误的失败实例调用一次 `Stop`，再按逆序回滚此前已启动实例；回滚同样受总协调上限约束，启动错误与清理错误用 `errors.Join` 保留。若 Start 忽略自己的超时仍不返回，不能并发调用它的 Stop；内核转入失败关机，保留其依赖并如实报告卡在 Starting 的实例。只有它真正离开 Start 后，监督流程才能安排清理。
- **Worker 看门狗监督（`supervisor.go`）**：
  所有 `Tool` 和 `Worker` 的 `Start` 跑通后，内核为每个 `Worker` 分配一个 `*runHandle`，并起 goroutine 执行 `worker.Run(lifetimeCtx)`。如果当前应用还没拉响关机警报（`workerEntry.expectedStop.Load() == false`），某个 `Worker.Run` 却自己退出来了——哪怕它返回的是 `nil`——看门狗协程都会立刻构造 `ErrWorkerUnexpectedExit` 推入 `app.errChan`，触发全局失败关机。

### 3. 运行期：`Activator` 怎么激发与收敛 `Component`？

当 `CronRunner` 到点触发定时任务，或者某个业务服务需要按需拉起一个 `Component` 时，严禁自己写裸 `go job()`，必须调用内核注入的 `Activator.Activate(ctx, compKey, opts...)`。

`Activate` 在 `act.mu` 内完成身份核验、准入与父子登记。父级身份从内核写入 Context 的私有凭据中读取，不接受业务在 Payload 中自报 ParentKey。

1. **查身份与闸门**：凭据必须属于当前 App 的活跃 Worker 或 Component；裸 Context、跨 App 凭据或已经结束的父实例返回 `ErrInvalidActivationParent`。应用或父级已停止接纳新任务时返回 `ErrShuttingDown`。登记子实例与关闭父级闸门必须用同一把锁，不能让任务钻进检查与关闸之间。
2. **查重入策略（默认 `OverlapSkip`）**：
   - 若该 `Component` 配置为默认的 `OverlapSkip` 且当前活跃实例数 `activeCount > 0`，直接返回 `(nil, ErrComponentSkipped)`，不造新实例；
   - 若配置为 `OverlapAllow(maxConcurrency)`，检查 `activeCount < maxConcurrency`，超限则返回 `ErrConcurrencyLimitExceeded`；
   - 若配置为 `OverlapQueue(maxQueue)`，将激活请求塞入长度为 `maxQueue` 的有界等待队列，队满同样报错拒绝；
3. **造新实例并跑完整生命周期**：
   准入后生成唯一 `instanceID`，同时记录父插件 `ParentKey` 和父实例 `ParentInstanceID`。后者才是实例树的关联键：两个相同定义的父 Component 不能被混成同一个父级。新实例在受管 goroutine 中执行：
   - 调 `Start(startCtx)`，成功才进入 `Run(compCtx)`；启动失败也要清理已分配资源，但不能丢掉启动错误。
   - `Run` 返回后先关闭自己的子任务准入，再等已经接纳的子实例完成整棵子树。父任务不能一边退出一边继续生孩子。
   - 子树全部收拢后才调自身 `Stop`；`Start` 若仍未返回，也不得并发调用它的 `Stop`。
   - 真实收尾后合并 `Start/Run/Stop` 的错误并公布稳定结果，退还并发配额，最后关闭 `handle.done`。

嵌套不等于无限递归：激活目标的插件键若已经出现在当前祖先链上，返回 `ErrActivationCycle`，避免父实例占住同一定义的配额，又等待永远排不到的子实例。Overlap 配额按整个 App 中的 Component 定义统计，实例树另按 InstanceID 追踪。

### 4. 关机期：先排空，再清理，但不假装能强杀协程

收到信号、Worker 异常退出或宿主主动调用 `Shutdown()` 后，内核只开启一次协调流程。先关闭全局与实例激活闸门，再以当时的时刻确定两个固定截止点：自然排空在第 20 秒结束，整个协调过程在第 60 秒结束。不是每到一个插件就重新送它 20 秒。

这 60 秒内，内核按提供者依赖图逆序安排停止。活动 Component 子树也占用它声明的资源；资源只有在所有已接纳消费者真实收尾后才能停止。不能只看父 Worker 自己的 Run 是否返回。

- **Worker 先停接入，再收拢自己的执行**：先标记 expectedStop、关闭新激活；可选 Quiesce 只负责停接入，不能释放仍被消费者使用的资源。随后调用 Stop 触发 Run 退出，同时等待它自管的请求、内部协程和已接纳 Component 子树。若子任务声明使用父 Worker 本身的资源，先关准入并等这些子任务结束，再执行会释放该资源的 Stop。
- **Component 先结束运行和子树，再清理自身**：Run 返回后关闭本实例子任务准入，等待已接纳子树的真实终态，最后调用自身 Stop。Start 仍在运行时也不能并发 Stop。
- **Tool 同样必须真实收拢内部工作**：例如 Async.Stop 要先拒新任务，等任务真正退出、回收池后才返回。不能只因等待 Context 到期，就返回一个超时错误，留下还在写库的任务。

到第 20 秒，独立的关机定时器取消 App 的强制取消根：实例运行 Context、网络请求根与 Async 派生任务都接入这份通知。它不等待逆拓扑轮到哪个 Stop，因此不会出现“HTTP 等 Async、Async 等 Stop 才收到取消”的闭环。只有资源释放按依赖顺序推进，取消通知不按层排队。随后继续等真退；业务忽略 Done 就仍未结束。

**Stop 的返回条件不因 Context 超时变松。** 无论返回 nil 还是错误，插件都不得再有未上报、仍占用生命周期依赖的执行。资源释放失败可以返回错误，但任务还活着时 Stop 必须继续等待。内核在外层监督 Stop，不把协调器自己卡进一次同步调用里。

需要排空的 Worker/Tool.Stop 获得带总截止时间的 stopCtx。真正开始纯资源收尾后，才通过 0002 的 CleanupContext 派生一把最长 10 秒、且不越过总截止的清理 Context；Component 的 Stop 本身就是此类清理。若清理仍忽略超时不返回，照样保留活跃记录，不并发重试 Stop。

到第 60 秒，协调器返回残留快照，并停止继续推进尚未开始停止的底层提供者。**这不禁止已接纳实例完成自己的生命周期。** 比如 Component.Run 第 70 秒才退出，监督器仍等子树结束，再为它唯一的一次 Stop 派生独立、最长 10s 的尾部清理 Context；不把已经过期的全局 Context 直接交给它，也不重试已经进入过的 Stop。若它仍忽略取消，Wait 就仍等真实完成。

晚到收尾不改变已经返回的 Shutdown 错误与快照，也不是给整个应用续期。被保留但尚未进入 Stop 的底层提供者不会随之自动开启第二轮关机；宿主据残留报告决定退出或额外运维处理。区分的是“协调器最多等多久”与“残留实例真实活了多久”，内核不自行退出进程。

普通 HTTP Handler 和 gRPC Unary/Stream 在这套保证内；WebSocket/hijacked 连接不在默认 HTTP 插件的自动管理范围，需要独立受管 Worker。关闭连接也不等于业务回调已经返回。

## 数据与接口

以下为 `application` 模块的核心导出类型与签名：

```go
package application

import (
    "context"
    "time"

    "github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

// Mode 标识插件的三种运行形态。
type Mode uint8

const (
    ModeTool      Mode = iota + 1 // 工具/注入态：仅 Start/Stop，应用级单实例
    ModeWorker                    // 常驻型运行时：Start/Run/Stop，应用级单实例
    ModeComponent                 // 激发态运行时：Start/Run/Stop，每次激活新建实例
)

// OverlapKind 定义 Component 在上一轮尚未结束时的重入策略。
type OverlapKind uint8

const (
    OverlapSkip  OverlapKind = iota + 1 // 默认：跳过本次激活
    OverlapAllow                        // 允许有界并发（受 MaxConcurrency 限制）
    OverlapQueue                        // 有界排队等待（受 MaxQueue 限制）
)

// OverlapPolicy 描述 Component 的并发与重入配额。
type OverlapPolicy struct {
    Kind           OverlapKind
    MaxConcurrency int // OverlapAllow 时生效，必须 >= 1
    MaxQueue       int // OverlapQueue 时生效，必须 >= 1
}

// PluginInfo 是注册定义的唯一元信息来源；插件主键与资源键都使用 ctxkey.Key。
type PluginInfo struct {
    Key      ctxkey.Key    // 全局唯一插件标识；该键默认保存插件实例
    Name     string        // 人类可读名称，用于日志与排障报告
    Mode     Mode          // 由 RegisterWorker/Component/Tool 自动绑定并校验
    Provides []ctxkey.Key  // 静态声明额外暴露的资源键；Component 必须为空
    Requires []ctxkey.Key  // 所需插件主键或资源键；启动前解析到 Tool/Worker 提供者
    Overlap  OverlapPolicy // 仅 ModeComponent 生效，零值默认采用 OverlapSkip
}

// Plugin 是所有插件（含纯工具插件）必须实现的生命周期底线契约。
type Plugin interface {
    Start(ctx context.Context) error // 资源准备与端口监听；接收受限 startCtx
    Stop(ctx context.Context) error  // 真实排空与资源清理；返回后不得遗留自管执行
}

// RunnablePlugin 由 Worker 与 Component 共用。
type RunnablePlugin interface {
    Plugin
    Run(ctx context.Context) error // 阻塞执行体；接收实例 lifetimeCtx
}

// Quiescer 是 Worker 按需可选实现的静默扩展接口。
type Quiescer interface {
    Quiesce(ctx context.Context) error
}

// Outcome 表示运行实例的最终归宿分类。
type Outcome uint8

const (
    OutcomeSuccess    Outcome = iota + 1 // Run 返回 nil 且 Stop 返回 nil
    OutcomeFailed                        // Start/Run/Stop 失败；实际错误保留在结果中
    OutcomeCancelled                     // Run 明确返回 context 取消错误且 Stop 成功
)

// RunResult 汇总一次实例运行的完整账单。
type RunResult struct {
    InstanceID      string
    PluginKey       ctxkey.Key
    Outcome         Outcome
    CancelRequested bool  // 运行期间是否收到过内核取消请求
    StartErr        error // Start 失败时记录；未执行 Run 不得记作成功
    RunErr          error // Run 返回的原始错误
    StopErr         error // Stop 返回的清理错误
}

// RunHandle 是内核返回的运行监督句柄（非插件接口）。
type RunHandle interface {
    InstanceID() string
    // Wait 只在 Start/Run 与 Stop 的实际收尾完成后返回稳定的实例终态。
    // 调用方 ctx 仅控制本次等待；超时返回零值 RunResult 与 ctx.Err()，不取消任务。
    // Shutdown 超预算只产生独立残留快照，不封存此结果，也不关闭 handle.done。
    Wait(ctx context.Context) (RunResult, error)
}

// Activator 供 Worker 或业务按需激发 Component 实例。
type Activator interface {
    Activate(
        ctx context.Context,
        compKey ctxkey.Key,
        opts ...ActivateOption,
    ) (RunHandle, error)
}

// Config 定义内核全局超时预算，可通过 Option 覆盖默认值。
type Config struct {
    StartTimeout    time.Duration // 单个 Start 的协作超时，默认 15s
    GraceTimeout    time.Duration // 自然排空窗口，默认 20s，自关机开始计时
    CleanupTimeout  time.Duration // 每次纯资源清理上限，默认 10s
    ShutdownTimeout time.Duration // 整体协调上限，默认 60s
}

// 所有超时必须大于 0，GraceTimeout 必须小于 ShutdownTimeout。
// CleanupTimeout 的实际截止时间取本次开始 + 上限与全局截止时间的较早者。

// App 表示一个独立的应用程序生命周期内核实例。
type App struct { /* 内部持有拓扑图、SharedContainer、活跃实例表与信号通道 */ }

func WithStartTimeout(v time.Duration) Option
func WithGraceTimeout(v time.Duration) Option
func WithCleanupTimeout(v time.Duration) Option
func WithShutdownTimeout(v time.Duration) Option
func New(opts ...Option) *App
func (a *App) RegisterTool(info PluginInfo, factory func() Plugin) error
func (a *App) RegisterWorker(
    info PluginInfo,
    factory func() RunnablePlugin,
) error
func (a *App) RegisterComponent(
    info PluginInfo,
    factory func() RunnablePlugin,
) error
func (a *App) Activator() Activator
func (a *App) Run() error
func (a *App) Shutdown() error
```

### 环境变量只在启动装配时读取

上面的时间都是默认值，不是硬编码。系统／应用级配置统一使用 APP_ 前缀，为 application 增加四个环境变量键，定义放在 foundation/env，实际读取和预算校验放在 application：

| 环境变量             | Config 字段     | 默认值 |
| -------------------- | --------------- | ------ |
| APP_START_TIMEOUT    | StartTimeout    | 15s    |
| APP_GRACE_TIMEOUT    | GraceTimeout    | 20s    |
| APP_CLEANUP_TIMEOUT  | CleanupTimeout  | 10s    |
| APP_SHUTDOWN_TIMEOUT | ShutdownTimeout | 60s    |

优先级为 **显式 Option > 环境变量 > 默认值**。只有未由 Option 指定的字段才读取对应环境变量；未设置才用默认值，已设置为空也应报错。解析使用 env.Lookup 返回的 typed.String.Duration，支持 500ms、20s、1m 等 Go Duration 写法，不接受没有单位的裸数字。

最终值全部必须大于 0，且 GraceTimeout < ShutdownTimeout。协调器仍在运行时，CleanupTimeout 的有效窗口与剩余总时间取较小者；已经返回残留快照后的实例尾部清理使用原配置中的独立上限，不重新读取环境。显式传入 0 也是非法配置，不把它当作“用户没填”。

New 在装配时读取当前环境和 Option，存成 App 私有快照；解析或校验错误由 Run 在建图与任何工厂之前返回，不 panic、不悄悄回退。环境文件若需 godotenv 加载，必须由宿主先加载再 New，内核不偷偷监听或重读 .env。

部署改环境变量后重启生效。不提供热重载接口，不在每次请求或关机中重新 os.Getenv；修改父进程环境也不会更新已运行进程的快照。Shutdown 一旦开始，就从快照计算固定截止点，中途不允许延长或缩短。

## 异常与兼容

### 1. 取消请求与业务结果的判定规则（防误判表）

内核把 `CancelRequested` 与真实返回值分开记录。若 `Start` 失败，不调用 `Run`，但要等待实际清理完成，再以 `OutcomeFailed` 和 `errors.Join(startErr, stopErr)` 公布结果；其余实例等 `Run`、子树与 `Stop` 完整结束后，按下表封存结果：

| `CancelRequested` | `Run` 返回值                            | `Stop` 返回值    | 最终 `Outcome`     | `Wait` 返回的 `error`          | 说明                                                                        |
| ----------------- | --------------------------------------- | ---------------- | ------------------ | ------------------------------ | --------------------------------------------------------------------------- |
| `false` / `true`  | `nil`                                   | `nil`            | `OutcomeSuccess`   | `nil`                          | 哪怕取消信号在最后一刻到达，只要业务实打实返回 `nil` 且清理成功，就认作成功 |
| `false` / `true`  | `nil`                                   | `stopErr != nil` | `OutcomeFailed`    | `stopErr`                      | 业务跑完了但临时资源清理失败，不能算完全成功                                |
| `true`            | `context.Canceled` / `DeadlineExceeded` | `nil`            | `OutcomeCancelled` | `runErr`                       | 响应取消信号正常中止且清理干净                                              |
| `false` / `true`  | `bizErr != nil`                         | `nil`            | `OutcomeFailed`    | `bizErr`                       | 业务执行失败                                                                |
| `false` / `true`  | `runErr != nil`                         | `stopErr != nil` | `OutcomeFailed`    | `errors.Join(runErr, stopErr)` | **双错并存**：同时保留业务根因与清理错误，绝不互相覆盖                      |

### 2. 超出总协调上限后的残留快照

当默认 60s 总协调上限耗尽，仍有实例卡在 Start、Run、子树等待或 Stop 时：

1. **不把快照冒充终态**：内核保留该实例的活跃记录，`handle.done` 继续保持打开。`Outcome` 中不设 `Incomplete` 终态；“仍卡在 Run”只是本次关机观察到的事实，不代表它后来一定成功、失败或取消。
2. **保护在用底座**：如果一个 `Component` 卡在 `Run` 里没退，内核不调用它的 `Stop`，并沿着已解析的提供者依赖链保留仍在使用的资源。没有被残留实例占用的独立插件继续完成 `Stop`。Worker 与 Tool 也遵守同一保护规则；严格 Stop 契约不允许它们提前返回并隐藏仍在执行的工作。
3. **如实返回结构化错误**：`app.Shutdown()` 与 `app.Run()` 返回 `*StopIncompleteError`，其中 `StuckInstances []StuckInstance` 是返回时的独立快照，记录 `PluginKey`、`InstanceID`、阶段与连带保留的依赖。快照不随后台结果变化；若实例后来完成了 `Run + Stop`，再公布它的真实终态、关闭句柄。宿主可以另用带超时的 `Wait` 等待，不由内核自行 `os.Exit`。

并发或重复调用 `Shutdown()` 取得同一次协调流程的结果，不会刷新原先的残留快照，也不会重复调用 `Stop`。这与实例句柄是两回事：快照说的是“当时还剩谁”，`Wait` 说的是“这个实例最终怎样了”。

### 3. 与 v1.x 的兼容与迁移边界

- v2 工作树不保留 `major/register`、`major/main` 或旧访问器兼容壳，也不承诺旧 import、键类型与错误类型原样兼容。新调用方主动迁移到 `application` 与对应插件。
- 已发布的 v1 模块不因这次工作树重组而被改写。旧项目可以继续固定其 v1 依赖版本；这不表示 v2 工作树还会保留一套可编译的旧入口。
- 原 `plugins/cron`、`plugins/grpc`、`plugins/email` 及新增的 `plugins/web-gin` 将提供新的插件实现。首批各模块均为 v2.0.0，后续模块独立升级，由整体 Release 清单记录精确组合，不把每次整体版本号机械填进所有 require。

## 备选与取舍

Fx 的 OnStart/OnStop 适合装配已有组件，oklog/run 适合组织阻塞执行单元；它们都比自建内核成熟。这里保留三模式与 Activator，是因为还需要管理每次激活的新实例、父子树及资源键依赖，不只是把几个 Serve 函数串起来。外部方案的证据和比较见 [research/0001](../research/0001-lifecycle-plugin-runtime.md#参考)。

我们不另建一个“Stop 已返回，但其实还在跑”的上报协议，而是要求插件自己等真退，再由外层预算控制协调器何时交回宿主。它少一套状态机，代价是插件作者必须追踪内部执行。

如果接入两个独立 Worker 适配器后，真实排空仍反复依赖插件外的补丁；或需求变成跨进程恢复、持久化任务、自动重试，应重新评估这个薄内核的范围。那时要比较成熟调度平台，而不是继续往 Plugin 接口里堆职责。

## 验证

本轮只修改设计文档，没有实现 application。实现后在模块目录执行 `GOWORK=off go test -race ./...`，至少覆盖：

1. 零 Gin 的 Worker + Tool 按提供者依赖顺序启停；非法键、资源键冲突、缺依赖、环与 Component 作用域错误均在任何工厂执行前失败。
2. Start 报错时当前实例也被清理，错误与回滚错误同时保留；Start 忽略超时仍在执行时不并发 Stop。
3. Worker.Run 非预期返回 nil 或 error 都触发失败关机；正常 Stop 先触发 Run 退出，再等自管执行收拢。
4. OverlapSkip/Allow/Queue 的配额、队满拒绝与退还；父实例关闭准入与子实例登记并发时，没有遗漏子任务。队列请求在父级关闭后不得再启动，返回已结束的失败句柄而不是永远等待。
5. 不同父实例的嵌套子树不串归属；父 Run 失败也要等已接纳子树收尾后才 Stop；祖先递归激活被拒绝。
6. Wait 调用方超时只退出等待；真实成功不被 CancelRequested 篡改，Start/Run/Stop 错误可分别通过 errors.Is 检查。
7. 参数缩短后验证关机固定截止点，不随拓扑层数续期。清理在排空之后开始，故意运行越过旧第 30 秒但不越过新总上限；纯清理仍被限制在其自身预算内。
8. 用可由测试释放的阻塞任务验证残留：Shutdown 到总上限返回、依赖未停、句柄未完成；在总上限之后释放任务，确认它获得仍有效的独立尾部清理窗口，唯一一次 Stop 真正结束后才出现真实终态；旧快照不变，被保留提供者未被自动强拆。
9. Async 的任务虽然不是 RunnablePlugin，也必须使 Tool.Stop 挂起到真退；HTTP/gRPC 的“Serve 返回、Handler 未退”不能被判为停稳。再构造 Handler 等 Task.Wait、Task 等 ctx.Done 的链路，确认取消在 Grace 到期独立到达，即使 Async.Stop 尚未开始也不会死等。
10. 四项环境值覆盖默认值，显式 Option 覆盖环境；非法值、空值、无单位、负数与 Grace 不小于总上限均在工厂前报错。New 后修改环境不影响该 App，也不改变进行中的关机截止点。

## 待确认

生命周期、真实 Stop、嵌套归属与启动期环境配置已经确认。版本方案也已按 [0003](./0003-foundation-module-consolidation.md#异常与兼容) 收敛：目录不增加 v2 层，module/import 带 /v2，首批全模块 v2.0.0，后续按需发布与全量清单配套。无剩余阻塞性方案选择。

文档仍为 draft，未实现 application。RFC 评审应重点检验固定截止点、逆拓扑与活动子树占用的组合；运行正确性必须由后续测试证明。
