# v2.0.0 生命周期插件运行时设计

> 状态：draft · 日期：2026-10-19 · Scope：lifecycle
> 承接 [v2.0.0 生命周期内核与三模式插件机制调研](../research/0001-lifecycle-plugin-runtime.md)

## 目标与边界

前期调研（[research/0001](../research/0001-lifecycle-plugin-runtime.md)）已经把 v1.x 绑死 Gin、节点缺关闭钩子、以及 `goroutine.go` 假退出的病根扒清楚了。本稿把这些问题转成生命周期契约。单次 App、任意阶段关机、钩子异常、错误归属与 Component 排队规则已经确认；其它尚未完成的开工决定在文末标明，不把方向确认当成所有细节都已定稿。

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
└── option.go       # Option 与 Config（Start/Grace/Cleanup/ShutdownTimeout）
```

### 2. App 只运行一轮，关闭也不能把它复活

同一 App 的状态只能向前走。Running 的状态发布与运行期准入开启采用同一同步边界；关闭请求若先取得状态变更权，就不得再从 Starting 越过它启动新的运行体。运行结果和关机结果分开缓存；“资源是否清理干净”也与“关机协调器是否已经返回”分开记录。

| 状态       | 含义与可调用入口                                                   |
| ---------- | ------------------------------------------------------------------ |
| New        | 可以 Register；第一次 Run 进入 Starting；Shutdown 可直接封存空 App |
| Starting   | 已冻结注册；依次校验与启动；Shutdown 可中断启动                    |
| Running    | 运行和激活准入已打开；Shutdown 进入 Stopping                       |
| Stopping   | 不再启动插件或激活任务；并发 Shutdown 等同一次协调结果             |
| Stopped    | 协调结束且所有已创建实例都已真实收尾；资源释放错误仍可能非 nil     |
| Incomplete | 协调已返回，但有在途或无法确认已清理的实例及被保留依赖             |

New → Starting → Running 是正常启动；启动失败也可以从 Starting 进入 Stopping 回滚，未触碰资源的配置/建图失败可直接结束为 Stopped。New 上的 Shutdown 不调用任何工厂或钩子，返回 nil 并进入 Stopped。Incomplete 不因晚到任务结束自动改成 Stopped，因为可能仍保留着尚未进入 Stop 的提供者；原关机快照与结果不变。

第一次 Run 无论成功还是失败，都会消耗这次运行机会。重复或并发 Run 立即返回 ErrAlreadyRun，不加入等待、不创建第二组协程，也不改变第一次结果。若尚未 Run 就已被 Shutdown 封存，随后 Run 返回 ErrAppClosed。需要再跑一轮就重新 New；首版不支持重启同一个 App。

Register 只允许 New。第一次 Run 领取运行权与冻结定义、Shutdown 封存空 App、并发注册使用同一状态锁，谁先完成状态变更谁生效。晚到的 Register 返回 ErrRegistrationClosed，不能出现“建图后还漏进一个定义”。注册时复制 Requires/Provides 等元数据切片；不在持锁状态调用用户工厂或钩子。

Shutdown 在 Starting 发生时，立即停止安排下一个工厂/钩子并取消当前 startCtx，不先等待应用完整启动。当前 factory/Start/Expose 若仍在执行，监督器等它真实结束后才决定清理资格；到总上限仍未结束就报告残留，不并发 Stop。已成功创建并进入过 Start 的实例按规则回滚，只有构造但尚未进入 Start 的实例无需 Stop。即使当前 Start 响应关闭后返回 nil，也不能越过关闭标记进入 Running。

启动中关闭的 Run 返回 ErrStartupInterrupted，并保留同期启动错误与关机错误；Shutdown 只报告清理是否成功。正在运行时的正常主动关闭不因 Context 取消自动变成运行失败。应用信号监听在取得运行权后、第一处可能分配资源的调用前就装好，启动中也能收到退出请求。

### 3. 从注册到启动：五步点火流水线

调用 `app := application.New(opts...)` 创建应用实例后，整个启动过程严格按以下五步推进：

```text
1. Register*(info, factory)   ──► 仅往 app.defs 字典存定义与工厂函数（零外部资源分配）
2. app.Run() 触发 buildGraph  ──► 校验 Key/Provides/Requires，解析提供者，检查缺依赖/环/Component 作用域，生成 layers
3. 按层初始化 Tool 与 Worker   ──► 调 factory() 建实例 → 注入受限 startCtx → 调 Start(startCtx)
                                  → 端口绑好/连接 Ping 通即算 Ready → 写入 SharedContainer
4. 冻结容器并启动 Worker.Run  ──► 冻结 SharedContainer 为只读 → 开闸放行(gate.Open)
                                  → 为每个 Worker 起受管协程跑 Run(lifetimeCtx)
5. 阻塞等本轮运行结束       ──► 信号监听从启动前已生效；关闭请求或 Worker 异常 → 同一次 Shutdown()
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
  所有 Tool/Worker 启动成功后，由内核监督 Worker.Run。Worker 完成事件与进入 Stopping 使用同一状态同步边界判定：完成事件若先发生，哪怕返回 nil，也记为 ErrWorkerUnexpectedExit 并触发失败关机；停止状态若先发布，按约定的正常退出/取消结果不再误报。不能只靠一条稍后处理的 errChan 消息猜测退出时的状态。
  进入 Stopping 时统一将全部已创建 Worker 标记为 expectedStop，不等逆拓扑轮到各自 Stop。已经记录的运行根因不会被这个标记清除；panic 和其它真实非关闭错误仍被记录，不把“预期停止”解释成吞掉所有错误。协议适配器应把其约定的正常关闭返回值规范化为 nil 或明确的取消结果。

### 4. 运行期：先接纳任务，再决定何时创建实例

定时器或业务代码通过 Activator.Activate 触发 Component，不能自己裸开 goroutine 绕过实例监督。每次接纳都对应一个唯一任务身份；这不等于立刻构造插件对象。

#### 准入成功时就交付句柄

内核在同一准入同步边界完成 App/父实例状态、身份和配额检查，以及任务登记。可信父级必须属于当前 App 的活跃 Worker/Component；裸 Context、跨 App 或已结束的父级返回 ErrInvalidActivationParent，关闭准入返回 ErrShuttingDown。Activate 调用 Context 在接纳前已取消时，返回 ctx.Err()，不交付句柄。

接纳成功后，立即登记 InstanceID、ParentKey/ParentInstanceID 和已解析的 Requires 依赖占用，并返回 RunHandle。队列项也属于已接纳的子任务，父级不能因为它没有运行就漏掉它；依赖占用是内核引用登记，不提前分配数据库连接或插件资源。只有真正派发时才在受管 goroutine 中调用工厂。

拒绝接纳一律返回 nil 句柄：跳过为 ErrComponentSkipped，并发超限为 ErrConcurrencyLimitExceeded，队满为 ErrQueueFull，嵌套排队受限为 ErrNestedQueueWouldBlock。拒绝不留下子任务、容量或依赖占用。

#### 重入策略只定义一套配额口径

| 策略                | 准入行为                                               |
| ------------------- | ------------------------------------------------------ |
| OverlapSkip（默认） | 已有未真实收尾的执行实例就跳过，不排队                 |
| OverlapAllow(n)     | 最多 n 个执行槽，超限拒绝，不排队                      |
| OverlapQueue(q)     | 同一定义共享一个串行执行槽和最多 q 个等待项，FIFO 取出 |

执行槽从派发、调用工厂前占用，直到该实例及子树真正收尾才释放，不能只等 Run 返回。Stop panic 或仍有在途清理时保留槽位。MaxQueue 只数等待项，不包括正在构造、启动、运行、等子树或清理的实例；Queue 的并发固定为 1，不再读取 MaxConcurrency。

例如 Queue(3) 可以同时有一个正在执行的 A，以及 B→C→D 三个等待项；E 到达时拒绝。A 真正完成后，在锁内释放槽并认领下一个仍合法的队首，再在锁外调用工厂；新请求不能抢在前序等待项前面。这个队列按 App 中的 Component 定义共享，不为每个父级另开一份。

#### 未派发取消，与执行中取消分开记

| 状态变化              | 谁占用什么、如何结束                                                                      |
| --------------------- | ----------------------------------------------------------------------------------------- |
| 接纳 → Queued         | 占等待容量、父子登记与依赖引用；不调用工厂或任何钩子                                      |
| Queued → Dispatched   | 锁内核对父级/App 仍允许派发，取出队首、转换为执行槽；原身份和依赖引用继续沿用，不再加一遍 |
| Queued → Cancelled    | 原子移除等待项，公布未派发取消结果，各释放一次等待容量、父子登记和依赖引用                |
| Dispatched → 最终结果 | 按实际进入过的生命周期及子树收尾，再释放执行槽和全部登记                                  |

父实例关闭子任务准入或 App 进入 Stopping 时，取消其尚未派发的项，不留到 Grace 中再开始。取消和派发与关闸共享同步边界，只有一个分支能赢。若派发先赢，该项按在途实例监督，关闭通知不能并发 Stop；尚未进入的工厂/Start 入口仍应复查停止标记，不为了凑齐流程强行调用后续钩子。Stop 资格沿用前面的 App 规则。

未派发取消的句柄已经履行全部真实收尾义务，因为根本没有对象和钩子需要清理。结果为 OutcomeCancelled、Dispatched=false、CancelRequested=true、CancelErr=取消原因，FactoryErr/StartErr/RunErr/StopErr 均为 nil；Wait 返回这个稳定结果和 CancelErr，不往 RunErr 编造一段业务返回值。

父级关闭的原因为 ErrParentClosing，App 关闭为 ErrShuttingDown，两者都应能通过 errors.Is 匹配 context.Canceled。并发发生时保留先在线性化边界确定的原因，不事后改写结果。即使某等待项因错误路径被多次观察，完成状态、done 关闭和计数退还也只能提交一次。

#### 接纳后不跟随原请求，也不新增任务 deadline

接纳成功的任务从 App 的未取消基底派生自己的 Context，保留独立身份/追踪 ID，并受父级生命周期与 App 关闭协调监管。原 HTTP/gRPC 请求结束不移除队列项，也不取消已经派发的任务；任务依然是受管的，不是野协程。只有接纳前的 ctx.Err() 参与本次拒绝判定，接纳后的请求取消不再回头撤销它。

首版不提供 WithTaskTimeout 或排队过期计时器：等待项会等到可派发、父级关闭或 App 关闭；Start 保留其独立 StartTimeout，Stop 和关机保留原预算。Wait(waitCtx) 到期永远只结束这次等待。首版也不额外增加手动取消任务的公开接口。

WithPayload 保存本次接纳参数，不自动深拷贝任意指针、map 或 slice；调用方应传不可变值或自备快照，接纳后不再修改任务会读取的共享可变数据。队列有界指等待项数量有界，不承诺任意大 Payload 的字节总量也自动有界。

#### 嵌套不允许堵在另一个繁忙队列后面

激活目标已出现在祖先定义链上时仍返回 ErrActivationCycle。除此之外，父级是 Component 且目标采用 OverlapQueue 时，仅在目标槽空闲、没有前序等待项时立即接纳；否则返回 ErrNestedQueueWouldBlock，不产生子句柄、不插队。Worker 及其可信请求 Context 可以正常进入等待队列。

这样保留嵌套子树，但首版不维护全局等待环图。它阻止的是调度器强加的嵌套排队等待；业务代码自行循环等其它句柄或外部锁，仍需自己保证不会死锁。

派发后按有效对象资格执行 Start→Run，Run 返回时关闭自己的子任务准入并处理取消队列，等待已接纳子树收拢后再 Stop。正常返回的阶段错误原样保留，真实收尾后才公布结果、释放槽位并关闭句柄；工厂失败和 Stop panic 仍按后文异常表处理。

### 5. 关机期：先排空，再清理，但不假装能强杀协程

收到信号、Worker 异常退出、运行期清理 panic 或宿主主动调用 Shutdown 后，内核只开启一次协调流程。进入 Stopping 与关闭全局/实例准入、统一标记全部 Worker 的 expectedStop 在同一同步边界完成；随后才触发可能使 Run 退出的取消或 Stop。自然排空和协调截止均从这次关机开始计时，默认第 20 秒与第 60 秒，不随拓扑层数续期。

这 60 秒内，内核按提供者依赖图逆序安排停止。活动 Component 子树也占用它声明的资源；资源只有在所有已接纳消费者真实收尾后才能停止。不能只看父 Worker 自己的 Run 是否返回。

- **Worker 先停接入，再收拢自己的执行**：expectedStop 与激活关闸已在进入 Stopping 时全局完成，不等轮到该层。按逆拓扑执行 Quiesce/Stop 与等待；Quiesce 不释放仍被消费者使用的资源。若子任务声明使用父 Worker 的资源，先等相关子树结束，再执行会释放资源的 Stop。预期停止标记只改变退出判定，不允许提前关闭底层资源。
- **Component 先结束运行和子树，再清理自身**：Run 返回后关闭本实例子任务准入，等待已接纳子树的真实终态，最后调用自身 Stop。Start 仍在运行时也不能并发 Stop。
- **Tool 同样必须真实收拢内部工作**：例如 Async.Stop 要先拒新任务，等任务真正退出、回收池后才返回。不能只因等待 Context 到期，就返回一个超时错误，留下还在写库的任务。

到第 20 秒，独立的关机定时器取消 App 的强制取消根：实例运行 Context、网络请求根与 Async 派生任务都接入这份通知。它不等待逆拓扑轮到哪个 Stop，因此不会出现“HTTP 等 Async、Async 等 Stop 才收到取消”的闭环。只有资源释放按依赖顺序推进，取消通知不按层排队。随后继续等真退；业务忽略 Done 就仍未结束。

**Stop 的正常返回条件不因 Context 超时变松。** 无论正常返回 nil 还是错误，插件都不得再有未上报、仍占用生命周期依赖的执行。Stop panic 不属于正常返回，内核无法据此证明执行已经收拢，必须保留依赖并报告清理未完成。资源释放失败可以返回错误，但任务还活着时 Stop 必须继续等待。内核在外层监督 Stop，不把协调器自己卡进一次同步调用里。

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

// AppState 描述本轮 App，不代表某个任务的业务结果。
type AppState uint8

const (
    StateNew AppState = iota + 1
    StateStarting
    StateRunning
    StateStopping
    StateStopped
    StateIncomplete
)

// HookPanicError 保留内核调用边界上的 panic，堆栈仅供内部诊断。
type HookPanicError struct {
    PluginKey  ctxkey.Key
    InstanceID string // 尚未分配实例时允许为空
    Stage      string // factory/start/expose/run/quiesce/stop
    Value      any
    Stack      []byte
}

func (e *HookPanicError) Error() string
func (e *HookPanicError) Unwrap() error // Value 为 error 时返回该错误，否则 nil

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
    MaxConcurrency int // 仅 OverlapAllow 使用，必须 >= 1
    MaxQueue       int // 仅 OverlapQueue 使用，必须 >= 1；不含单个执行槽
}

// 零值策略为 Skip。未知 Kind、对应容量小于 1、或不相关字段非零，在建图前拒绝。
// Queue 固定串行，不用 Allow 的 MaxConcurrency 为它偷偷增加并发。

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
    OutcomeSuccess    Outcome = iota + 1 // 执行成功且需要执行的清理均真实完成
    OutcomeFailed                        // Start/Run/Stop 失败；实际错误保留在结果中
    OutcomeCancelled                     // 未派发已取消，或执行明确响应取消且真实收尾
)

// RunResult 汇总一次实例运行的完整账单。
type RunResult struct {
    InstanceID      string
    PluginKey       ctxkey.Key
    Outcome         Outcome
    Dispatched      bool  // 已取得执行槽、进入构造监督；不等于 Start/Run 一定执行
    CancelRequested bool  // 是否收到内核取消请求，包含尚未派发就被取消
    CancelErr       error // 未执行 Run 时的取消原因，不冒充 RunErr
    FactoryErr      error // 工厂 panic 或返回无效实例；没有对象时不调用 Stop
    StartErr        error // Start 返回错误或 panic
    ExposeErr       error // Expose panic 或交付校验失败
    QuiesceErr      error // 静默钩子返回错误或 panic
    RunErr          error // Run 返回的原始错误
    StopErr         error // Stop 返回的清理错误
}

// RunHandle 是内核返回的运行监督句柄（非插件接口）。
type RunHandle interface {
    InstanceID() string
    // Wait 只在本实例需执行的生命周期与子树真实收尾后返回稳定结果。
    // 未派发取消/工厂无效且无清理资格时可结束；Stop panic 不证明已清理，不关闭句柄。
    // 调用方 ctx 仅控制本次等待；超时返回零值 RunResult 与 ctx.Err()，不取消任务。
    // Shutdown 超预算只产生独立残留快照，不封存此结果，也不关闭 handle.done。
    Wait(ctx context.Context) (RunResult, error)
}

// Activator 供 Worker 或业务按需激发 Component 实例。
type Activator interface {
    // ctx 用于可信身份及接纳前取消检查；接纳后不继承其 deadline/取消。
    // 成功立即返回已登记句柄；失败返回 nil 句柄，不留占用。
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
func (a *App) State() AppState
func (a *App) Run() error      // 本轮已观察的启动/运行根因 + 关机协调错误
func (a *App) Shutdown() error // 仅关机协调结果，多次调用共享
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

### 1. 无效实例和钩子 panic 分别处理

nil 工厂在 Register 时以 ErrNilFactory 拒绝，不登记定义。工厂在真正被调用时返回 nil 或类型化 nil，记为 ErrNilPlugin：没有有效对象就不调用 Start/Expose/Run/Stop。静态插件构造失败触发启动回滚；运行期 Component 构造失败仅结束这次任务。工厂只允许构造对象，禁止分配外部资源；如果违反这个契约又未交出对象，内核没有可供兜底清理的句柄。

内核在实际调用工厂和钩子的同一受管协程中 recover，转换为 HookPanicError，保留 PluginKey、可用的 InstanceID、Stage、原值与堆栈。不会恢复发生 panic 的函数继续执行，更不保证能回滚已发生的数据库写入。

| 失败边界                 | 内核下一步                                                                |
| ------------------------ | ------------------------------------------------------------------------- |
| factory panic / 无效实例 | 没有有效对象，不调用 Stop；应用启动失败回滚，Component 则仅本次失败       |
| Start panic              | 栈已展开后调用当前有效实例的唯一一次 Stop，再按所属作用域处理失败         |
| Expose panic / 校验失败  | 不发布本批条目；当前实例 Stop，再逆序回滚已启动插件                       |
| Worker.Run panic         | 记运行失败并触发 App 失败关机；关机已开始则只并入已有运行错误账本         |
| Component.Run panic      | 本任务失败；关闭子任务准入，收拢子树后再 Stop，不因单次业务失败关闭 App   |
| Quiesce panic            | 记关停错误，钩子栈退出后仍尝试 Stop，不因一次静默失败跳过清理             |
| Stop panic               | 记清理无法确认；不重试、不退役、不释放依赖；运行期发生则触发 App 失败关机 |

Start/Quiesce 等钩子若只是超时仍未返回，不能把它当作 panic 已展开来并发调用 Stop。Stop panic 后虽调用栈已经退出，仍可能留有未知自管执行；该实例不释放并发占位、不关闭 handle.done，Wait 只能在调用者自己的超时下退出等待，不伪造 OutcomeFailed 的“已完整收尾”终态。

Stop panic 的实例以阶段 stop-panic 和原 HookPanicError 进入 StuckInstances，并保留已解析依赖。协调器继续处理不依赖它的其它作用域；可推进的工作都处理完而仅剩已知无法确认的清理残留时，可以提前返回 StopIncompleteError，不必为已展开的 panic 空等满 60s。若仍有真实在途钩子，则按总协调上限等待。仅有未完成报告不构成再次调用 Stop 的许可。

recover 只覆盖这些调用边界。插件自行开的协程、进程级致命错误、os.Exit 或其它绕过正常返回的进程控制，不在这份恢复保证内。HookPanicError 与堆栈只进入内部账本/日志，不直接作为 HTTP/gRPC 对外详情。

### 2. Run 与 Shutdown 分别回答不同的问题

Run 只接受第一位调用者，返回整次运行在协调结果确定前观察到的启动/运行错误，再与 Shutdown 的关停错误用 errors.Join 汇总。Shutdown 缓存同一次关机的结果：Quiesce/Stop 错误和 StopIncompleteError，不把 Worker 的原始业务失败重复当成清理失败。

| 场景                         | Run 结果                                      | Shutdown 结果            |
| ---------------------------- | --------------------------------------------- | ------------------------ |
| 主动正常关闭，真实收尾成功   | nil                                           | nil                      |
| 配置/建图失败，未创建资源    | 原始校验错误                                  | nil                      |
| 启动中收到关闭，清理成功     | ErrStartupInterrupted，保留同期启动错误       | nil                      |
| Worker 失败，清理成功        | 运行根因，保留 Worker 非预期退出语义          | nil                      |
| Worker 失败且 Stop 出错      | 运行根因与关停错误的 Join                     | 关停错误                 |
| 存在卡住或 Stop panic 的实例 | 已知运行根因与 StopIncompleteError 等关停错误 | 同一次关停错误与残留快照 |

正常的 Component 失败通过其句柄返回，不自动污染 App 的运行结果；但 Component.Stop panic 是生命周期安全失败，关停结果中必须能追溯该原因。重复 Run/晚到 Register 的调用错误只返回给该调用者，不改变第一次运行账本。并发和重复 Shutdown 等同一次协调结果，不重复执行钩子。

Run/Shutdown 的返回结果都不随着晚到事件反复变化。总上限之后观察到的任务结果或异常继续记录在实例账本与日志中，不回写已经返回的应用结果；相关句柄只有在真实收尾条件满足时才公布终态。StateIncomplete 也不是任务 Outcome，不能混用。

### 3. 取消请求与业务结果的判定规则（防误判表）

内核把 CancelRequested 与真实返回值分开记录。未派发取消按 CancelErr 结束，不要求调用不存在的钩子；派发后尚未进入 Run 就被关闭中断时，同样以 CancelErr 记录原因，只清理已经进入 Start 的有效对象。任何真实准备期/清理错误优先保留为 OutcomeFailed，不被取消掩盖。工厂未产出有效实例时，可直接以 FactoryErr 形成失败结果；Start/Expose 失败则等有资格执行的 Stop 真正收尾后，以 OutcomeFailed 合并准备与清理错误。如果 QuiesceErr 非 nil，但实例随后真实收尾，则 OutcomeFailed，并与 Run/Stop 错误合并保留。下表只适用于已进入 Run、无准备期/Quiesce 错误、且子树及 Stop 均已真实完成的实例；Stop panic 不套用此表提前封存终态。

| `CancelRequested` | `Run` 返回值                            | `Stop` 返回值    | 最终 `Outcome`     | `Wait` 返回的 `error`          | 说明                                                                        |
| ----------------- | --------------------------------------- | ---------------- | ------------------ | ------------------------------ | --------------------------------------------------------------------------- |
| `false` / `true`  | `nil`                                   | `nil`            | `OutcomeSuccess`   | `nil`                          | 哪怕取消信号在最后一刻到达，只要业务实打实返回 `nil` 且清理成功，就认作成功 |
| `false` / `true`  | `nil`                                   | `stopErr != nil` | `OutcomeFailed`    | `stopErr`                      | 业务跑完了但临时资源清理失败，不能算完全成功                                |
| `true`            | `context.Canceled` / `DeadlineExceeded` | `nil`            | `OutcomeCancelled` | `runErr`                       | 响应取消信号正常中止且清理干净                                              |
| `false` / `true`  | `bizErr != nil`                         | `nil`            | `OutcomeFailed`    | `bizErr`                       | 业务执行失败                                                                |
| `false` / `true`  | `runErr != nil`                         | `stopErr != nil` | `OutcomeFailed`    | `errors.Join(runErr, stopErr)` | **双错并存**：同时保留业务根因与清理错误，绝不互相覆盖                      |

### 4. 超出总协调上限或清理无法确认后的残留快照

当默认 60s 总协调上限耗尽仍有在途实例，或可推进工作处理完后仍存在 Stop panic 等无法确认清理的残留时：

1. **不把快照冒充终态**：内核保留该实例的活跃记录，`handle.done` 继续保持打开。`Outcome` 中不设 `Incomplete` 终态；“仍卡在 Run”只是本次关机观察到的事实，不代表它后来一定成功、失败或取消。
2. **保护在用底座**：如果一个 `Component` 卡在 `Run` 里没退，内核不调用它的 `Stop`，并沿着已解析的提供者依赖链保留仍在使用的资源。没有被残留实例占用的独立插件继续完成 `Stop`。Worker 与 Tool 也遵守同一保护规则；严格 Stop 契约不允许它们提前返回并隐藏仍在执行的工作。
3. **如实返回结构化错误**：Shutdown 的结果包含 `*StopIncompleteError`，Run 再将它与已知运行根因合并返回，调用方应通过 errors.As 检查而不是要求顶层错误恰好就是该类型，其中 `StuckInstances []StuckInstance` 是返回时的独立快照，记录 `PluginKey`、`InstanceID`、阶段与连带保留的依赖。快照不随后台结果变化；若实例后来完成了 `Run + Stop`，再公布它的真实终态、关闭句柄。宿主可以另用带超时的 `Wait` 等待，不由内核自行 `os.Exit`。

并发或重复调用 `Shutdown()` 取得同一次协调流程的结果，不会刷新原先的残留快照，也不会重复调用 `Stop`。这与实例句柄是两回事：快照说的是“当时还剩谁”，`Wait` 说的是“这个实例最终怎样了”。

### 5. 与 v1.x 的兼容与迁移边界

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
4. OverlapSkip/Allow/Queue 的执行槽和等待容量分别计数；Queue 按定义串行 FIFO，跨父级提交仍共用顺序；Run 已返回而 Stop 未完时不启动下一项。队满/嵌套忙拒绝不产生句柄或任何占用。
5. 不同父实例的嵌套子树不串归属；父 Run 失败也要等已接纳子树收尾后才 Stop；祖先递归激活被拒绝。
6. Wait 调用方超时只退出等待；真实成功不被 CancelRequested 篡改，Start/Run/Stop 错误可分别通过 errors.Is 检查。
7. 参数缩短后验证关机固定截止点，不随拓扑层数续期。清理在排空之后开始，故意运行越过旧第 30 秒但不越过新总上限；纯清理仍被限制在其自身预算内。
8. 用可由测试释放的阻塞任务验证残留：Shutdown 到总上限返回、依赖未停、句柄未完成；在总上限之后释放任务，确认它获得仍有效的独立尾部清理窗口，唯一一次 Stop 真正结束后才出现真实终态；旧快照不变，被保留提供者未被自动强拆。
9. Async 的任务虽然不是 RunnablePlugin，也必须使 Tool.Stop 挂起到真退；HTTP/gRPC 的“Serve 返回、Handler 未退”不能被判为停稳。再构造 Handler 等 Task.Wait、Task 等 ctx.Done 的链路，确认取消在 Grace 到期独立到达，即使 Async.Stop 尚未开始也不会死等。
10. 四项环境值覆盖默认值，显式 Option 覆盖环境；非法值、空值、无单位、负数与 Grace 不小于总上限均在工厂前报错。New 后修改环境不影响该 App，也不改变进行中的关机截止点。
11. 并发 Register/Run/Shutdown 只有一个合法状态顺序；重复 Run 不执行第二轮，Run 前关闭不调用工厂，启动中关闭不继续进入后续插件或 Running。已有结果不被非法调用或晚到事件改写。
12. nil 工厂、nil/类型化 nil 实例不调用不存在对象的 Stop；依次注入 factory/Start/Expose/Run/Quiesce/Stop panic，验证 HookPanicError 的阶段、原值与堆栈、清理资格、错误保留和钩子至多调用一次。
13. Component.Run panic 只结束本任务；Component.Stop panic 则触发失败关机，实例与依赖不被退役，Wait 不交付伪终态。Quiesce panic 后仍尝试 Stop。分别验证 Run/Shutdown 错误归属，以及 errors.Is/As 能找到原始根因。
14. A 依赖 B，逆拓扑尚在等待 A.Stop 时，B.Run 响应全局 Grace 取消返回；验证不误报 ErrWorkerUnexpectedExit。反向让 B 在停止状态发布前先退出，必须保留运行根因；关机中发生的 panic/真实错误也不被 expectedStop 吞掉。
15. 入队立即有句柄、父子登记和依赖占用，工厂调用次数仍为零；原请求随后取消不改变等待项，Wait 超时也不移除它。接纳前已取消则拒绝。
16. 并发关闭父级/App、派发队首与重复取消同一等待项，只有一个状态分支完成；未派发取消后所有钩子均未调用，CancelErr 可匹配 context.Canceled，RunErr 仍为 nil，每种占用只退一次。
17. 构造 A 与 B 的繁忙串行队列及嵌套请求，验证立即 ErrNestedQueueWouldBlock，不产生等待环或插队；空闲且无等待项时可立即派发。用可控屏障证明首版没有隐藏的请求/任务 deadline 或排队过期计时器。

## 待确认

已确认单次 App 与异常处理，以及共享串行 FIFO、入队即接纳、未派发取消、接纳后与原请求取消独立、嵌套繁忙不等待。首版不增加单任务超时或手动取消接口。原有 App 预算、身份与版本决定不变。

阶段 0 的 App 与队列项已经确认；日志与雪花装配、协议元数据以及能力/数据处置仍需继续收敛，不能据此把整个阶段勾为完成。

文档仍为 draft，未实现 application。RFC 评审应重点检验固定截止点、逆拓扑与活动子树占用的组合；运行正确性必须由后续测试证明。
