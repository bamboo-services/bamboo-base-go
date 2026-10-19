# v2.0.0 生命周期内核与三模式插件机制调研

> 调研日期：2026-10-17
> 后续设计说明：本篇保留当时的候选接口与开放问题；静态 `Provides`、资源键依赖解析、真实终态 `Wait` 与 v2 主动迁移边界已转入 [design/0001](../design/0001-lifecycle-plugin-runtime.md)。下文的 `PluginID`、旧句柄签名与开放问题保留调研时点，不是当前契约。后续已确认 Stop 真退、嵌套子树与默认 60s 总协调上限（20s 自然排空、纯清理每次最长 10s），并支持启动读取环境预算后冻结；旧 30s 设想不再采用。后续还确认源码目录不增 v2 层，module/import 使用 /v2，首批模块均从 v2.0.0 起步，后续按需发版；映射见 [design/0003](../design/0003-foundation-module-consolidation.md#异常与兼容)。

## 背景

现在的 v1.x 名义上是基础组件库，骨子里其实是个绑死了 Gin 的 Web 启动器。翻开 `major/register/register.go:14-17`，核心结构 `Reg` 直接塞着 `Serve *gin.Engine`；到了 `major/main/runner.go:76-79`，`Runner` 一看 `reg.Serve == nil` 当场就 Panic；甚至连管初始化节点的 `major/register/node/node.go:222-230` 都直接 `import "github.com/gin-gonic/gin"`。如果你想用这套基础库写一个纯 gRPC 服务、一个只跑定时任务的后台进程，或者一个命令行工具，也得硬着头皮把整套 Gin 引擎拉起来陪跑。

生命周期本身也拧巴。`major/register/register.go:60-98` 在 `Register` 阶段就把读配置、连数据库、建缓存一股脑全干完了，而节点函数 `Node func(ctx context.Context) (any, error)`（`major/register/node/node.go:23`）只管生、不管死，压根没有关闭入口。到了退出的时候，`major/main/web.go:48-63` 由 HTTP 关闭协程一把调 `ctxCancel()` 统管全局；更要命的是 `major/main/goroutine.go:37-43`，监听协程一收到 `shutdownNotify` 广播就调 `Done()` 放行 `WaitGroup`，根本不等后台协程真正跑完。像 `plugins/cron/runner/runner.go:187-209` 超时就直接 `return` 扔下在途任务不管，`plugins/grpc/runner/runner.go:219-239` 报错只能自己调 `log.Panic`，都是因为底下缺一个真正管生命周期的内核。

**v2.0.0 要调研解决的核心问题很明确：把 Gin 从底座里请出去，让基础库回归成一个纯粹的生命周期内核，该用一套怎样的插件模型去同时托住常驻服务、按需任务与工具资源？**

围绕这个目标，我们已经确定了四条大方向作为调研前提：

1. **内核与具体框架彻底解耦**：内核只认识插件和生命周期，不认识 HTTP 路由或中间件。Gin、gRPC、Cron 统统作为平等的插件接入，不再有谁是主、谁是附庸的区别。
2. **所有插件按运行形态分成三种模式**：
   - **`Worker`（常驻型）**：跟着应用一起启动、一直跑到底的组件。不管是接外部流量的 HTTP / gRPC 服务，还是在内部循环干活的 Cron 调度器、消息队列消费者，都归这一类。
   - **`Component`（激发态）**：按需唤醒、跑完就收工的运行单元（也可以跑很久）。拿定时任务来说，Cron 调度器本身是个常驻的父级 `Worker`，每到整点触发跑的那一次具体任务，就是一个动态拉起来的 `Component` 实例。
   - **`Tool`（工具/注入态）**：提供数据库连接池、Redis 客户端这类被别人调用的资源能力，或者纯计算工具包。
3. **官方能力一律走插件登记**：用户自己引什么第三方 Go 包是用户的自由；但只要是我们基础库提供的能力，哪怕没有任何初始化逻辑，也必须按统一的 `plugins` 模式登记元信息，留下一条清晰的插件账本。
4. **所有插件强制实现 `Start` 和 `Stop`**：允许方法体写空实现（直接 `return nil`），但方法签名一个都不能少——工具态也一样。在此之上，`Worker` 和 `Component` 共用多一个 `Run(ctx context.Context) error` 的 `RunnablePlugin` 接口，由内核统一调度和监督；元信息只放在注册定义里，不在实例上重复挂 `Info()`；分阶段停流量的 `Quiesce` 降级为按需可选扩展；超时、重试、并发上限这些策略全部收归内核配置层，不往基础接口里塞。

为了别一口吃成胖子，首版也有几件明确**不碰**的事：不搞靠反射或注解猜依赖的自动 DI 容器；不支持应用跑起来之后动态加新的插件定义；不做崩溃自动重启或失败自动重试；也不碰跨进程的分布式调度。另外，首版的 `Requires` 依赖声明只允许指向应用级的 `Worker` 或 `Tool`，不能去依赖一个按需才存在的 `Component` 定义——毕竟启动时 `Component` 还没影呢，等它等于死锁。

## 发现

```text
启动：登记定义(零副作用) → 校验依赖图 → 按拓扑序逐个「调工厂建实例 → Start(准备资源)」
      → 内核起协程跑全部 Worker.Run 并挂上 RunHandle → 就绪检查通过 → 开闸放行(Running)

Cron：DB(Tool) Start 建池并 Ping → Cron(Worker) Start 装好调度表 → 内核跑 Cron.Run
      → 整点 tick 触发 → 经内核 Activator 新建 Job(Component) 实例
      → Job.Start → 内核跑 Job.Run → Job.Run 真正返回 → Job.Stop → 记录终态

关机：撤就绪 + 关入口闸门(拒新流量与新激活) → 按「上层消费者先停、底层依赖后停」逐层推进：
      ├─ Worker：打预期停止标记 → 调 Stop(与 Run 并发安全，触发 Run 退出) → 等 Stop 与 Run 都完成
      └─ Component：先等 Run 自然跑完（超时再 cancel 它并继续等真退）→ 调 Stop
      → 等所有子任务实例彻底跑完 Stop → 最后调 DB(Tool).Stop 关连接池
```

### 1. 外部方案对比：为什么定成 `Start / Run / Stop`？

在敲定这套接口前，我们对照了 Go 生态里四个代表性项目的做法（基于 2026-10-17 核对的官方源码与文档）：

- **Uber Fx v1.24.0**（`docs/src/lifecycle.md`）只提供 `OnStart` 和 `OnStop` 两个带硬超时的短钩子，要求你在 `OnStart` 里自己 `go serve()`。这有两个老毛病：一是后台协程跑起来之后如果半路崩了，框架默认两眼一抹黑；二是新手特别容易顺手拿带 15 秒超时的 `OnStart` context 去跑后台循环，结果服务刚启动 15 秒后台工人全自动退出了。
- **Kratos v2.8.4**（`transport/http/server.go`）的 `Server` 接口只有 `Start` 和 `Stop`，但它的 `Start` 内部直接阻塞在 `Serve` 上。这样框架确实能用 `errgroup` 抓到运行期错误，代价是“端口刚绑好”和“服务一直在跑”混成了一件事。
- **Temporal Go SDK v1.37.0**（`worker/worker.go`）明确拆开了只报启动期错误的非阻塞 `Start()`，以及阻塞跑到底并上报运行期致命错误的 `Run()`。
- **`oklog/run` v1.1.0**（`group.go`）则守住了一条底线：只要有一个执行单元退出，就逐个调中断函数，并且**必须等所有执行函数实打实返回了**，`Group.Run()` 才算结束。

老实讲，这几家没有哪一家能直接照抄，但放在一起看，取舍就非常清楚了：我们把“准备资源”留在带超时保护的 `Start`，把“持续干活的主循环”拆到由内核起协程托管的 `Run`，把“收摊清理”交给 `Stop`。这样插件作者不用在每个插件里手搓一套 `WaitGroup` 和错误缓存通道，内核也能牢牢盯住每个 `Run` 的死活。

### 2. 从登记到启动：注册不碰资源，按依赖顺序点火

插件登记（`RegisterWorker`、`RegisterComponent`、`RegisterTool`）都是挂在具体 `App` 实例上的方法，不搞进程级全局单例——这样同一个进程里跑多个 `App`（比如单测并行或主副实例）才不会互相串台。

登记阶段只做一件事：把插件元信息（`PluginInfo`）和工厂函数存起来。**工厂函数只准 `new` 结构体、填配置，严禁在工厂里建数据库连接、绑端口或开文件。** 理由很实际：要是你在工厂里把连接建了一半报错返回 `nil`，内核手里连个插件实例都没有，想调 `Stop` 帮你擦屁股都找不到对象。把所有资源分配都摁死在 `Start` 里，“登记期零副作用”才不是一句空话。

在真正动手建实例之前，内核先拿着登记的 `PluginInfo` 把依赖图验一遍：ID 有没有重复、`Requires` 写的依赖存不存在、有没有循环依赖、模式和注册入口对不对得上、有没有违法去依赖 `Component`。这一步纯算内存数据，不调任何工厂。

校验过了，再按 `Requires` 的拓扑顺序，从最底层的被依赖方开始逐个往上启动 `Tool` 和 `Worker`（`Component` 这时候只是一纸定义，按兵不动）：

1. 先调工厂造出实例（返回 `nil` 直接报错中止）；
2. 再调 `Start(startCtx)` 准备资源（比如数据库建池并 `Ping`，HTTP 服务绑好 `net.Listener`）；
3. 要是某个插件的 `Start` 报错了，插件自己得先把刚拉起来的那一半内部协程停掉，内核也会兜底调一次它的 `Stop`，然后把前面已经 `Start` 成功的插件按反方向挨个 `Stop` 回滚掉；
4. 等所有 `Tool` 和 `Worker` 的 `Start` 都顺顺当当跑完，内核再统一为每个 `Worker` 起 goroutine 执行 `Run(lifetimeCtx)`，并用内核自己的 `RunHandle` 盯着。

这里有个很容易踩的坑：**端口绑上了、甚至 `Run` 的协程刚创建出来了，都不等于整个应用已经准备好接流量了。** 全部 `Worker` 的 `Run` 调度起来并确认没有立刻暴毙后，还得过一道就绪检查，最后才拉开入口闸门（admission gate）对外放量。

### 3. 三种模式怎么配合？拿 Cron + DB 走一遍

光讲概念容易飘，直接拿最常见的 **数据库（`Tool`） + Cron 调度器（`Worker`） + 定时报表任务（`Component`）** 看它们怎么咬合：

- **先看 `Tool`（数据库连接器与纯工具）**：
  数据库插件只实现 `Plugin`。启动时 `Start` 建好连接池、`Ping` 通数据库；运行期别的插件从容拿连接用；最后关机时等所有用数据库的组件都退干净了，`Stop` 再把连接池关掉。虽然 `sql.DB` 或 GORM 底下自己有维护心跳的后台协程，但我们不看它内部有没有协程，只看它对外扮演的角色——它是被动提供资源的底座，所以归为 `Tool`，不用硬凑一个空转的 `Run`。
  至于像加密、字符串处理这种纯函数工具包，同样要调 `RegisterTool` 登记身份，`Start` 和 `Stop` 直接写 `return nil` 就行，业务代码平时该怎么调函数还怎么调。（说句实在话，Go 语言层面没法阻止别人在没登记前直接调一个包级纯函数，我们靠工程规范守住边界，不在运行时搞奇技淫巧去拦截。）但凡是有状态的资源，只要还没 `Start` 或者已经 `Stop` 了，访问器一律返回明确错误，绝不悄悄给个 `nil` 埋雷。

- **再看 `Worker`（Cron 调度器）**：
  Cron 实现 `RunnablePlugin`。它的 `Start` 只负责把调度器和任务表装配好，绝不在 `Start` 里偷偷把调度循环跑起来；真正的阻塞调度循环写在 `Run` 里，由内核起协程跑。
  因为 `Worker` 的本分就是常驻，所以**只要应用还没发停止通知，`Worker.Run` 敢自己退出来——哪怕返回的是 `nil`——内核一律当成异常崩溃处理**，默认直接触发整个应用的失败退出流程。另外，插件内部如果为了干活自己开了辅助协程，当 `Run` 返回的那一刻，这些自己开的协程必须已经全部收拢完毕，不能人在曹营心在汉。

- **最后看 `Component`（每次整点激发的报表任务）**：
  整点时间一到，Cron 调度器要触发报表任务了。这时候，**严禁 Cron 自己写一句裸的 `go job()` 把任务扔出去**——一旦脱离内核，关机时谁也不知道还有几个任务在飞，跟 v1.x 的野协程就没区别了。
  正确路径是调用内核发放的激活器 `Activator`。内核在同一个锁里干完三件事：看一眼应用和父级 Cron 的闸门关没关、查一下并发配额够不够、在活动登记表里占好坑位并打上不可伪造的 `ParentID`。接着，**每次激活都调工厂新建一个独立的 `Component` 实例**——哪怕同一个定时任务每秒跑一次，每次也是全新的实例、全新的 ID 和独立状态，绝不拿同一个结构体实例反复 `Start/Stop`，从根上掐死重入时状态互踩的 bug。
  实例造好后，内核依次跑它的 `Start → Run → Stop`，最后公布结果、退还配额。如果上一次整点任务跑得慢，下一次 tick 又到了怎么办？内核在策略层提供三种明确的重入模式：
  - `Skip`：上一轮还没完，这一轮直接跳过；
  - `Allow`：允许并行跑，但卡死并发上限，超了就拒；
  - `BoundedQueue`：塞进有界队列排队，队列满了同样拒绝。
    不管选哪种，**绝对不允许出现无界队列或无限开协程**。

### 4. 关机怎么收：为什么 Worker 和 Component 的停法正好反过来？

> **关于 Context 的单一来源约束**：`Start`、`Run`、`Stop` 以及 `RunHandle.Wait` 所接收的 `context.Context` 如何划分超时职责、如何与外部信号隔离、以及插件注册后如何通过共享 Context 传递给 `PluginRunner`，全部统一收口在 **[0002 Context 体系与共享组件传递调研](./0002-context-propagation-and-shared-state.md)**。本文只描述生命周期调度动作，不再重复展开 Context 细则，避免后续修改产生差异。

收到退出信号后，所有阶段共享同一个关机总预算（不会每停一个插件就重新给它刷满 30 秒），按从上到下的顺序收摊：

第一步，先撤掉应用就绪状态、关死入口闸门，外部新流量和内部新的 `Activator` 激活统统拒之门外。

第二步，按依赖图的反方向——**上层消费者先停，底层提供者后停**——一层一层往下收。比如业务任务依赖内部 gRPC 服务，内部 gRPC 服务又依赖数据库，那就必须先把业务任务停干净，再停 gRPC，最后才能动数据库。如果某个端口既接外部用户流量、又被内部还在跑的任务当依赖调用，而插件适配层分不开这两股流量，那就老老实实按依赖顺序推迟停它——**宁可晚几秒停外部入口，也绝不能把在途任务脚下的地板提前抽掉。**

第三步，具体到单个实例怎么停，**`Worker` 和 `Component` 的顺序正好相反，这是整个机制里最不能搞混的地方**：

- **停 `Worker`：必须先调 `Stop`，同时等 `Run` 返回。**
  千万别写成“先等 `Worker.Run` 退出了再去调 `Worker.Stop`”——想想 `http.Server.Serve` 和 `grpc.Server.Serve` 就明白了，人家本来就是在 `Run` 里死等连接的，你不调 `Stop`（里面调 `Shutdown` / `GracefulStop`），`Run` 就会天荒地老一直阻塞下去，两边直接死锁。
  所以对 `Worker`，内核先给它打上“预期停止”标记（这样待会儿 `Run` 返回 `nil` 就不会被误报成崩溃），接着直接调 `Stop`（如果该 `Worker` 实现了可选的 `Quiescer` 且策略开启，可以在 `Stop` 前先调 `Quiesce` 停新请求并排空内部在途工作）。`Stop` 要求做到与正在跑的 `Run` 并发安全。等 `Stop` 返回了、并且 `RunHandle` 确认 `Run` 也实打实退出了，这个 `Worker` 本身才算落停。

- **停 `Component`：必须先等 `Run` 真正返回，再调 `Stop`。**
  `Component` 是一次具体的任务执行。业务代码还在 `Run` 里写文件、查数据库呢，要是你敢并发调它的 `Stop` 把临时目录删了、把句柄关了，当场就是数据损坏或 panic。
  所以对 `Component`，关机时先在宽限期（grace）里等它的 `Run` 自然跑完；要是宽限期到了还没跑完，内核再 cancel 它的 `lifetimeCtx` 发协作取消通知，但**依然要死等 `Run` 真正返回**，看到它确实退出来了，最后才调 `Stop` 做清理。

| 维度           | Drain（宽限期正常排空）               | Cancel（超时或强制取消）                              |
| -------------- | ------------------------------------- | ----------------------------------------------------- |
| 什么时候发生   | 正常关机、父级自然收工                | 宽限期预算用光了，或者宿主明确要求快速退出            |
| 新活儿接不接   | 闸门已关，一律不接                    | 闸门已关，一律不接                                    |
| 手头在跑的活儿 | 不动 `lifetimeCtx`，等它自己跑完      | cancel 掉 `lifetimeCtx` 催它退，但继续等 `Run` 真返回 |
| 怎么调 `Stop`  | `Run` 返回后用 `cleanupCtx` 调 `Stop` | 同样等 `Run` 真返回后，再用 `cleanupCtx` 调 `Stop`    |

第四步，**父子归属关系和资源依赖关系各算各的账，不许互相死等**。
还拿 Cron 举例：关机时，Cron 这个父级 `Worker` 调了 `Stop` 把定时器关掉、不再产生新 tick，它自己的 `Cron.Run` 很快就能退出来；只要 `Cron.Stop` 不去释放子任务正在用的东西，它就不必卡在 `Stop` 里傻等子任务。但是，内核的活动登记表还在盯着它生出来的那几个报表 `Component` 孩子！只要还有一个孩子没跑完自己的 `Stop`，Cron 所在的这个父子作用域就不算关干净，孩子们依赖的底层数据库（`Tool`）也绝对不会调 `Stop`。（反过来，如果是 `Component` 激发了子 `Component`，父实例则必须等孩子们全部进入终态后，才能调自己的 `Stop`。）等到所有上层消费者和它们的子孙后代全都彻底停稳了，最后一步才轮到关闭共享数据库。

### 5. 核心接口骨架与两个异常深水区

调研收敛出的核心接口骨架如下（超时、重试、并发配额等参数走内核配置结构，不往基础接口里塞）：

```go
// PluginInfo 是插件登记时的元信息，也是元信息的唯一来源；插件实例上不再挂 Info() 方法。
type PluginInfo struct {
    ID       PluginID
    Mode     Mode // Worker / Component / Tool，由注册入口绑定并校验
    Requires []PluginID
}

// Plugin 是所有插件（含工具态）都必须实现的基础生命周期契约，无实际动作时允许写空实现。
type Plugin interface {
    Start(ctx context.Context) error // 准备资源（ctx 语义详见 0002 调研文档）
    Stop(ctx context.Context) error  // 停止并释放自有资源（ctx 语义详见 0002 调研文档）
}

// RunnablePlugin 由 Worker 与 Component 共用；Tool 只实现 Plugin。
// Worker 的 Run 持续阻塞运行；Component 的 Run 执行单次任务，跑完即可正常返回。
type RunnablePlugin interface {
    Plugin
    Run(ctx context.Context) error // 执行运行体；返回前必须收拢插件自建的内部协程（ctx 语义详见 0002）
}

// Quiescer 是按需可选扩展：只有需要显式「先停新流量、排空内部在途，稍后再 Stop」的 Worker 才实现。
type Quiescer interface {
    Quiesce(ctx context.Context) error
}

// RunHandle 是内核侧的运行句柄（不是插件接口），用来监督一次 Run 并缓存其唯一结果。
type RunHandle interface {
    // Wait 阻塞等待结果；ctx 超时只退出调用方等待，不取消任务（详见 0002 调研文档）。
    Wait(ctx context.Context) error
}

// 注册入口挂在具体的 App/Registry 实例上（此处省略 receiver 示意），在编译期锁死工厂返回类型。
// 工厂函数只构造对象，不分配外部连接资源。
func RegisterWorker(info PluginInfo, factory func() RunnablePlugin) error
func RegisterComponent(info PluginInfo, factory func() RunnablePlugin) error
func RegisterTool(info PluginInfo, factory func() Plugin) error
```

在内核眼里，“定义”和“实例”的状态也是分开记的：

- **定义视角（Registry）**：只有 `Registered`。特别是 `Component`，刚注册完时只有定义，一个实例都没有。
- **实例视角（Runtime）**：`Created → Starting → Running → Stopping → Stopped`。对 `Component` 实例来说，结束后的业务终态细分为 `success`、`failed`、`cancelled`。

此外，有两个异常边界必须在后续设计中卡死：

1. **取消信号和业务成功撞车了，算谁的？**
   写并发任务最怕一种冤案：`Run` 里的业务代码其实已经把账入完库了，正走到最后一行 `return nil`；就在同一毫秒，关机超时触发了 `lifetimeCtx` 取消。如果内核偷懒，只看一眼 `ctx.Err() != nil` 就把这个任务盖章成 `cancelled`，上游就会误以为这笔业务没跑成。
   所以内核必须把 **“收到过取消请求（`CancelRequested`）”** 和 **“业务实际跑出来的返回值”** 分成两本账记，用互斥锁或 CAS 保证终态只提交一次：只要 `Run` 实打实返回了 `nil`，哪怕 `CancelRequested` 已经是 `true`，业务执行结果依然记为 **`success`**（当然，随后执行的 `Stop` 也必须成功，整个实例才算彻底干净）；只有 `Run` 明确返回了取消错误，终态才记为 **`cancelled`**；返回其他错误一律记为 **`failed`**；如果 `Run` 报错了、紧接着 `Stop` 清理也报错了，**两个错误一起保留上报**，绝不拿 `Stop` 的错把 `Run` 的根因盖掉。一句话：取消请求只是催促，不能篡改已经发生的业务事实，内核也绝不假装取消信号能自动回滚业务已经写进数据库的数据。
2. **遇上死活不退的“老赖”协程怎么办？**
   这是生命周期管理最需要诚实面对的边界：**Go 语言的 `context` 取消只是协作通知，内核没有任何合法手段去强杀一个死锁或忽略 `ctx.Done()` 的 goroutine。**
   如果某个插件在 `Start` 里直接卡死永不返回，或者关机总预算已经耗尽了，某个 `Worker` / `Component` 的 `Run` 或 `Stop` 还是死活不退，内核必须守住三条铁律：
   - **绝不假装退出了**：绝不干 v1.x（`major/main/goroutine.go:37-43`）那种“一收到关闭广播就调 `WaitGroup.Done()` 提前放行”的事；
   - **绝不强拆还在用的底座**：只要某个 `Component` 的 `Run` 还没真正返回，或者 `Start` 还卡在半道上，内核绝不并发调它的 `Stop`，也绝不关闭它声明依赖的数据库连接池——因为那段卡住的代码随时可能再往连接池里读写，强关就是制造二次崩溃；
   - **如实报告 `StopIncomplete`**：内核会把其它已经没有消费者占用的独立资源正常关掉，然后向宿主返回 `StopIncomplete`（暂定名，未返回的执行记为“运行未完成/结果待定”，不伪造 `cancelled`）以及一份残留未退出的实例清单。至于拿到这份报告后要不要直接调 `os.Exit(1)` 拔电源，交给最外层的 `main` 宿主拍板，内核自己绝不调 `os.Exit`。

## 结论

- **事实结论**：
  1. v1.x 在 `Reg` 结构、启动检查、关机主导权与节点包导入四处与 Gin 硬绑定，且 `goroutine.go` 与 `cron/runner` 存在未等待协程真实退出的设计缺陷；
  2. 用户已确认 v2.0.0 的核心规则：解耦 Gin、按 `Worker` / `Component` / `Tool` 三模式划分、官方能力一律登记插件、所有插件强制实现 `Start/Stop`（允许空实现），并由 `Worker` 与 `Component` 共用 `RunnablePlugin.Run`。
- **倾向（供后续编写 Design 时采用）**：
  1. 倾向采用“注册期只存元信息与无副作用工厂 → 启动期按 `Requires` 拓扑序调 `Start` → 内核起协程托管 `Worker.Run` → 反拓扑分层关机”的运行模型；
  2. 倾向对 `Worker` 采用“先 `Stop` 触发退出、再汇合确认 `Run` 返回”，对 `Component` 采用“先等 `Run` 返回、再调 `Stop` 清理”的反向停机序；
  3. 倾向将取消请求（`CancelRequested`）与 `Run` 实际返回结果分账记录，并在超出关机预算时如实返回 `StopIncomplete` 报告而非假退出。
- **开放问题（写详细 Design 前需拍板）**：
  1. **`RunHandle` 与 `Activator` 具体签名**：调用 `handle.Wait(ctx)` 到底等 `Run` 返回就算完，还是必须等 `Stop` 清理也跑完才返回？`Activator` 除了传 `Component` ID，怎么把当次触发的参数传给新建的实例？
  2. **异步预热的就绪协议**：像 HTTP/gRPC 这种进 `Run` 才开始 `Serve` 的插件，怎么通知内核“我已经真正能接活了”？是给个可选 readiness 回调，还是在 `Start` 里先把 `net.Listener` 建好就算就绪？
  3. **关机总预算与阶段分配**：默认关机总超时设多少（如 30s）？宽限排空期（grace）和强制取消等待期怎么分这块蛋糕？
  4. **Cron 重入与并发默认值**：不传配置时，Cron 任务默认用 `Skip` 还是 `Allow`？默认最大并发数和队列上限设多少？
  5. **非阻塞细节**：可选 `Quiescer` 的调用时机、注册入口错误返回风格、`StopIncomplete` 命名、以及 `/v2` 模块路径规划。

## 参考

- Uber Fx v1.24.0 Lifecycle 文档 — <https://github.com/uber-go/fx/blob/v1.24.0/docs/src/lifecycle.md>
- Kratos v2.8.4 Transport Server 接口与 HTTP 实现 — <https://github.com/go-kratos/kratos/blob/v2.8.4/transport/transport.go> · <https://github.com/go-kratos/kratos/blob/v2.8.4/transport/http/server.go>
- Temporal Go SDK v1.37.0 Worker 接口定义 — <https://github.com/temporalio/sdk-go/blob/v1.37.0/worker/worker.go>
- oklog/run v1.1.0 Group 实现 — <https://github.com/oklog/run/blob/v1.1.0/group.go>
- Go 标准库 `net/http.Server.Shutdown` 文档 — <https://pkg.go.dev/net/http#Server.Shutdown>
- gRPC 官方优雅关闭指南 — <https://grpc.io/docs/guides/server-graceful-stop/>
