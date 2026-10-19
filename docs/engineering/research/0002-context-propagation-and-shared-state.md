# v2.0.0 Context 体系与共享组件传递调研

> 调研日期：2026-10-17（承接 [0001](./0001-lifecycle-plugin-runtime.md)）
> 后续设计说明：本篇保留调研时对兼容性与桥接方式的探索。用户随后选择 v2 清晰切断，不保留旧访问器兼容壳；静态 `Provides`、精确资源键白名单与多键交付检查以 [design/0002](../design/0002-context-propagation-and-shared-state.md) 为准，下文的“调用方零改动”不再是实施承诺。后续同时明确 BridgeRequest 携带私有父实例凭据，协议插件合入独立取消根；Stop 按默认 60s 总协调上限与独立清理窗口执行，旧候选图不直接作为实现依据。

## 背景

上一篇 [0001 调研](./0001-lifecycle-plugin-runtime.md) 确立了 `Start`、`Run`、`Stop` 的生命周期主骨架。为了防止两份文档重复描述 `context.Context` 导致后续演进时出现口径打架，**所有与 Context 相关的调研与设计（含生命周期三类 Context 的职责边界、共享组件容器的挂载、以及向各路 `PluginRunner` 的传递机制）全部收口以本篇 0002 为唯一权威来源**，0001 仅做引用不再重复展开。

只要顺着 0001 的主骨架往下推演两步，马上就会撞上一个非常实际的问题：

**插件 `Start` 跑完之后，它建好的数据库连接池、Redis 客户端、雪花节点放在哪？后面的 `PluginRunner`（比如 Gin、gRPC、Cron）和业务 Handler 怎么从 `ctx` 里把它们掏出来？**

在 v1.x 里，`context.Context` 其实一肩挑了两担活儿：一担是 Go 标准库本职的**“取消信号与超时控制”**，另一担则是咱们基础库特有的**“共享组件背包”**。翻开现有代码，这条链路是这么跑的：

1. **初始化往里塞**：`major/register/node/node.go:23` 的节点签名是 `Node func(ctx context.Context) (any, error)`。每次 `Exec()`（`node.go:129-156`）跑完一个节点，就把返回的 `any` 往切片 `rn.value`（类型是 `xCtx.ContextNodeList`，定义在 `defined/context/reg_node.go:12`）里追加一份，同时 `context.WithValue(rn.Ctx, node.Key, val)` 挂一份，最后再把整条 `rn.value` 挂到 `xCtx.RegNodeKey` 下。
2. **各路 Runner 往外传**：
   - **Gin**：`major/register/node/node.go:222-230` 的 `InjectContext()` 中间件把 `rn.value` 塞进每个 HTTP 请求的 `c.Request.Context()`；
   - **gRPC**：`plugins/grpc/interceptor/unary/init_context.go:18-40` 和 `stream/init_context.go:30-56` 先从启动时传入的 `mainCtx` 里把 `RegNodeKey` 抠出来存成闭包变量，等每次 RPC 进来再 `WithValue` 塞给 `rpcCtx`；
   - **Cron**：`plugins/cron/runner/runner.go:164-166` 直接在闭包里捕获外层 `ctx` 传给 `jobFn(ctx)`；
   - **Async**：`plugins/async/context.go:16-36` 的 `detachContext` 为了脱离 HTTP 请求超时，新建一个 `context.Background()`，再手动把 `RegNodeKey` 和 `RequestKey` 从父 `ctx` 搬过去。
3. **业务代码往外掏**：业务调 `xCtxUtil.MustGetDB(ctx)`（`major/utility/context/database.go:25-47`）或 `xCtxUtil.Get[T](ctx, key)`（`major/utility/context/context.go:45-78`）时，先靠进程级全局变量 `globalContextExtractor`（`major/utility/context/gin_extractor_impl.go:14`）把 `*gin.Context` 转成标准 `ctx`，再去 `RegNodeKey` 的切片里从头到尾 `for` 循环遍历找组件。

这套老办法很直观，但在 v1.x 里已经暴露出四个硬伤，到了 v2.0.0 更是直接卡脖子：

- **新接口没有返回值了，组件怎么入袋？** 0001 里所有插件统一成了 `Start(ctx context.Context) error`，去掉了旧节点的 `(any, error)` 返回值。如果数据库插件在 `Start` 里连好了 `*gorm.DB`，它靠什么姿势把连接池交出来放进共享 Context？
- **换一把 Context 信号，难不成连共享组件一起丢？** 在 Go 里，`context.WithCancel` 只能往下派生不能往上改。v1.x 在 `major/register/register.go:90-96` 留了个著名的坑：路由注册器拿到的是还没被 `Runner` 包裹 `WithCancel` 的 `reg.Init.Ctx`；反过来，到了 v2，当我们为了安全关机给 `Stop` 换上一把全新的 `cleanupCtx`（基于未取消基底），或者为每次 Cron 触发新建 `Component` 的 `lifetimeCtx` 时，如果还靠老式 `context.WithValue` 单链表手搓搬运，稍一疏忽就会在新 `ctx` 里丢掉整个组件背包。
- **`ContextNodeList` 裸切片既慢又悬**：`defined/context/reg_node.go:12-39` 本质是个无锁切片 `[]ContextNode`，每次取 DB 都得线性扫描；更要命的是 `Append` 接收者是指针 `*ContextNodeList`，塞进 `WithValue` 的却是值类型 slice header，逼得 `major/register/node/node.go:199` 在 `UseAfterExec` 时还得专门写注释提醒“同步刷新 ctx 中关联的 slice header”。
- **全局 `globalContextExtractor` 绑架了整个进程**：`major/utility/context/gin_extractor_impl.go:14` 用一个包级全局变量存提取器，不但不支持多 `App` 实例隔离，还让“怎么从框架上下文里拿标准 `ctx`”变成了一处隐式的全局副作用。

**因此，本篇调研聚焦摸清三件事：**

1. 插件注册并 `Start` 之后，产出的组件实例怎么装进**共享容器（Shared Container）**；
2. 共享容器怎么跟 `startCtx`、`lifetimeCtx`、`cleanupCtx` 以及每次请求的 `requestCtx` **正交组合**（换取消信号绝不丢共享组件，换请求绝不串临时状态）；
3. 这份共享 Context 怎么一路顺畅地传给 **`PluginRunner`（Gin / gRPC / Cron）**、传给按需激活的 **`Component`**，最终落到业务熟悉的 `xCtxUtil.MustGetDB(ctx)` 手里。

## 发现

先把整条传递链路画在一张图上。你会发现，核心秘诀其实就是一句话——**把“不变的共享组件底座”和“随时会变的取消信号/请求数据”彻底拆成两层**：

```text
【第一阶段：插件启动与共享底座装配（App 内核主导）】
  RegisterTool(DB) ──► DB.Start(startCtx) 成功
                         └─► 内核把 *gorm.DB 写入 App 专属的 SharedContainer（按拓扑序逐步充实）
  RegisterWorker(Gin / gRPC / Cron 等 PluginRunner)
                         └─► 它们的 Start(startCtx) 与 Run(lifetimeCtx) 拿到的 ctx，
                             底部都自动焊死了同一个 SharedContainer 引用

【第二阶段：从 PluginRunner 向四类运行现场分发（桥接函数 BridgeRequest 主导）】
  ┌─ 1. Gin PluginRunner    ──► HTTP 请求进来，中间件调 BridgeRequest(c.Request.Context(), lifetimeCtx, reqID)
  │                             保留客户端断连信号 + 挂入 SharedContainer + 生成本次 HTTP RequestKey
  ├─ 2. gRPC PluginRunner   ──► RPC 调用进来，拦截器调 BridgeRequest(rpcCtx, lifetimeCtx, traceID)
  │                             保留 gRPC deadline/metadata + 挂入 SharedContainer + 生成/透传 RequestKey
  ├─ 3. Cron PluginRunner   ──► 整点 tick 调 Activator 激活 Job(Component)
  │                             内核直接为该实例生成带 SharedContainer + 新 TraceID 的独立 lifetimeCtx
  └─ 4. Async 异步任务      ──► 业务调 Async(reqCtx, fn)，内部调 Detach(reqCtx)
                                剥离原请求超时与临时错误码，只带走 SharedContainer + 原链路 RequestKey

【第三阶段：业务与插件消费（统一读取入口）】
  Handler / Service / Component.Run ──► 调 xCtxUtil.MustGetDB(ctx) 或 xCtxUtil.Get[T](ctx, key)
                                        直接从 ctx 底部的 SharedContainer 按 key O(1) 取出实例
```

### 1. 插件没有返回值了，组件怎么塞进共享容器？

在 0001 里，所有插件统一实现 `Start(ctx context.Context) error` 和 `Stop(ctx context.Context) error`。为了不破坏这个干净的签名，同时又能把 `*gorm.DB`、`*redis.Client`、`*EmailClient` 装进共享 Context，我们梳理出 **“插件自身默认入库 + 可选 `Exposer` 暴露底层句柄”** 的双轨机制：

- **第一轨（默认行为：插件实例自己入库）**：
  每个插件在注册时都有唯一的 `PluginInfo.ID`（可以直接让 `PluginID` 与 `xCtx.ContextKey` 互通，或者在 `PluginInfo` 里指定关联的 `ContextKey`）。只要这个应用级插件（`Tool` 或 `Worker`）的 `Start(startCtx)` 返回了 `nil`，**内核立刻把该插件结构体实例本身，按它的 ID 存进当前 `App` 的 `SharedContainer`**。
  举个例子，邮件插件 `EmailPlugin` 本身就对外提供 `.Send()` 方法，那它的 `Start` 一跑完，别人直接按 `EmailClientKey` 从 `ctx` 里拿到的就是这个 `EmailPlugin` 实例，根本不用多写一行代码。

- **第二轨（包装器插件：实现可选的 `Exposer` 接口交出内部句柄）**：
  有些插件自己只是个管生命周期的壳，业务真正想从 `xCtxUtil.MustGetDB(ctx)` 里拿的是它肚子里的 `*gorm.DB`；再比如缓存插件（对照 v1.x 的 `major/register/register.go:75-80`），除了交出 `*xCache.Manager`（存 `CacheManagerKey`），如果底层配的是 Redis，还得顺手把 `*redis.Client` 也交出来（存 `RedisClientKey`）供老代码直接调 Redis。
  对于这种“壳与肉分离”或者“一鱼多吃”的插件，只需要顺手实现一个极小的可选接口 `Exposer`：
  当它的 `Start(startCtx)` 成功返回后，内核检查它如果实现了 `Exposer`，就调一下它的 `Expose()`，把返回的 `map[xCtx.ContextKey]any` 一并登记进 `SharedContainer`（若出现未声明的键冲突直接报错）。

这么一来，`Start(ctx) error` 的签名一点没变脏，而下游用了几百处的 `xCtxUtil.MustGetDB(ctx)`、`MustGetRDB(ctx)`、`MustGetCacheManager(ctx)` **连一行代码都不用改**，照样能拿到原汁原味的 `*gorm.DB` 和 `*redis.Client`。

### 2. 告别裸切片：共享容器（`SharedContainer`）里面长什么样？

v1.x 的 `ContextNodeList`（`defined/context/reg_node.go:12`）是个裸切片，既怕并发改，每次查还得从头扫到尾。在 v2 里，每个 `App` 实例内部持有一个并发安全的 **`SharedContainer`**，彻底取代 `ContextNodeList`。

它的工作规矩非常硬朗：

1. **只有内核能写，插件和业务只能读**：
   挂进 `context.Context` 里暴露给外面的，只是一个只读视图（或者容器指针的受控读取方法）。业务代码和插件再也不能像 v1.x 那样拿到切片自己调 `Append`，从根上消灭了 slice header 漂移和数据竞态。
2. **启动期按 `Requires` 拓扑逐步可见，运行期只读锁死**：
   - 在启动阶段，内核按 `Requires` 拓扑序一个接一个调插件的 `Start(startCtx)`。比如 `DB` 先 `Start` 完，容器里就有了 `DB`；接着轮到依赖 `DB` 的 `UserRepo` 或 `GinRunner` 跑 `Start(startCtx)` 时，它们从 `startCtx` 里就能稳稳当当地拿到已经活过来的 `DB`。**更严格一步：在某个插件的 `Start` 期间，内核可以通过包装视图，只允许它读取自己在 `PluginInfo.Requires` 里白纸黑字声明过的依赖**——如果你没在 `Requires` 里写 `DatabaseKey`，却在 `Start` 里偷偷调 `MustGetDB(startCtx)`，内核当场就报错拦住你，防止你靠碰巧的注册顺序写出隐式耦合！
   - 等所有 `Tool` 和 `Worker` 的 `Start` 全部跑完、准备起 `Worker.Run` 之前，整个 `SharedContainer` 的字典就已经齐活并进入**只读冻结态**。此后在整个应用运行期间（成千上万次 HTTP 请求、gRPC 调用、Cron 触发），对容器的读取全是无锁的 `map` O(1) 查找，快得飞起，也绝不会有任何并发写冲突。
3. **关机时跟随生命周期标记“已退役”**：
   当关机推进到底层、某个 `Tool`（比如数据库）的 `Stop(cleanupCtx)` 已经执行完毕后，内核会在容器里把该键的状态翻转成 `Stopped`。万一这时候还有哪个漏网的野协程拿着老 `ctx` 去调 `GetDB(ctx)`，容器不会把已经关掉的死连接池递给它，而是直接返回“组件 `context_database` 已停止”的明确错误。

### 3. 三把生命周期 Context 钥匙 + 共享底座：怎么合体才不串门？

为了杜绝“拿启动超时 ctx 跑后台导致 15 秒暴毙”和“拿已取消的 ctx 去关库导致清理失败”这两大惨案，内核先把生命周期的取消信号拆成互不串门的三把钥匙，再把同一个 `SharedContainer` 焊在它们共同的根底座上：

```text
context.Background()
  └── withShared(app.container)               <-- 共享底座（只存容器指针，永不取消）
        ├── WithTimeout(startTimeout)         --> 1. 传给 Plugin.Start(startCtx) / Quiesce
        ├── WithCancel() [由内核关机阶段控制]  --> 2. 传给 Worker.Run(lifetimeCtx) / Component.Run(lifetimeCtx)
        └── WithTimeout(cleanupTimeout)       --> 3. 传给 Plugin.Stop(cleanupCtx)
```

这三把 Context 加上运行句柄上的 `Wait(ctx)`，各自守着严格的边界：

1. **短期钩子 `startCtx`（传给 `Start` 及可选 `Quiesce`）**：它带的超时只约束这一个钩子函数能跑多久，钩子一返回就作废；任何后台协程都严禁绑在这把 `ctx` 上。当 `Start` 返回后，`startCtx` 被 cancel，但底下的 `app.container` 毫发无损。
2. **实例存续期 `lifetimeCtx`（直接传给 `Run`）**：这是实例活着期间的命根子，从同一个 `withShared(app.container)` 底座上拉出长命分支。**它绝不直接继承外层的操作系统信号 `ctx` 或应用调用方 `ctx`**——外部按 `Ctrl+C` 时只会唤醒内核关机协调器，先走优雅排空，等宽限期过了还没退，才由内核主动去 cancel 对应实例的 `lifetimeCtx`。要是直接继承信号 `ctx`，一按 `Ctrl+C` 所有在途 SQL 当场全断，优雅排空就成了笑话。
3. **独立清理 `cleanupCtx`（传给 `Stop`）**：哪怕前面的 `Run` 已经因为超时被 cancel 掉了，`Stop` 拿到的依然是从 `withShared(app.container)` 底座上重新派生出来的鲜活分支，并单独挂上清理超时。这样在 `Stop` 里关连接、刷缓冲、或者从 `app.container` 查辅助配置都能正常走完。
4. **调用方等待 `Wait(ctx)`（挂在内核 `RunHandle` 上）**：这里的 `ctx` 只是调用方自己的“耐心期限”——超时了只代表调用方退出等待，绝不会暗中把正在跑的任务给 cancel 掉。

### 4. 重头戏：插件注册后，怎么一路传给各个 `PluginRunner`？

像 Gin、gRPC、Cron 这些在 v2 里全部变成了插件（也就是各自模块里的 `PluginRunner` / `Worker`），它们是怎么拿到共享 Context，又是怎么喂给每一次请求和每一个任务的？我们逐个拆开看：

#### （1）传给 `PluginRunner` 自身：通过 `Start(startCtx)` 与 `Run(lifetimeCtx)` 直接送达

不管你是 `GinRunner`、`GrpcRunner` 还是 `CronRunner`，只要登记成了 `Worker`：

- 在 `Start(startCtx)` 阶段（比如 Gin 挂载路由、装配中间件，或者 gRPC 注册 Service 实现时），内核传进来的 `startCtx` 已经挂好了 `SharedContainer`。路由注册函数或 Service 构造函数如果需要当场拿某个已启动的组件（比如拿配置、拿中间件依赖的缓存管理器），直接从 `startCtx` 里取就行——而且因为此时依赖已经按拓扑序 `Start` 完毕，绝不会拿到 `nil`。
- 到了 `Run(lifetimeCtx)` 阶段，内核把长命的 `lifetimeCtx` 传给 `PluginRunner`。**注意：跟 v1.x（`major/register/register.go:90-96`）那个“路由注册器拿到的 ctx 没有被 Runner `WithCancel` 包裹”的坑彻底说再见了**——在 v2 里，`PluginRunner` 在 `Run(lifetimeCtx)` 里拿到的就是货真价实受内核关机调度的运行期上下文。

#### （2）从 `GinRunner` 传给每个 HTTP 请求：`BridgeRequest` 一行桥接

当一个 HTTP 请求打进 Gin 时，`c.Request.Context()` 是 `net/http` 为这次 TCP 连接生成的原生 `ctx`（它管着客户端主动断开连接时的取消信号）。这时候，`plugins/gin` 内置的上下文中间件只需要调一行内核提供的桥接函数：

```go
// 把 lifetimeCtx 里的 SharedContainer 嫁接到当前 HTTP 请求的原生 ctx 上，
// 既保留了 net/http 原生的客户端断连取消信号，又注入了全套共享组件与本次请求的 RequestKey。
reqCtx := xCtx.BridgeRequest(c.Request.Context(), lifetimeCtx, requestID)
c.Request = c.Request.WithContext(reqCtx)
```

这里顺手还解决了一个大麻烦：**彻底干掉 `major` 层那个尴尬的全局变量 `globalContextExtractor`（`major/utility/context/gin_extractor_impl.go:14`）！**
为什么 v1.x 非得搞一个 `GinContextExtractor`？因为很多业务代码习惯直接把 `*gin.Context` 当成 `ctx` 传给 `xCtxUtil.MustGetDB(c)`，而 v1.x 把 `RegNodeKey` 塞在了 `c.Request.Context()` 里、没塞在 `c.Keys` 里（在旧版 Gin 或未开启 `ContextWithFallback` 时，`c.Value(key)` 只查 `c.Keys` 字符串字典）。
在 v2 的 `plugins/gin` 中间件里，我们做两手双保险：

1. 一方面在创建 `gin.Engine` 时默认开启 `engine.ContextWithFallback = true`（Gin 原生支持：当 `c.Keys` 查不到时，自动 fallback 到 `c.Request.Context().Value(key)`）；
2. 另一方面，中间件同时把共享容器引用写进 `c.Request.Context()` 和 `c.Set(...)`。

这样一来，无论业务人员传进 `MustGetDB` 的是 `*gin.Context` 还是标准的 `c.Request.Context()`，标准的 `ctx.Value(...)` 都能直接命中共享容器！**内核和 `common` 层再也不需要知道 Gin 的存在，更不需要任何全局 `SetContextExtractor` 钩子。**

#### （3）从 `GrpcRunner` 传给每个 RPC 调用：Unary / Stream 拦截器同样一行桥接

对照 v1.x 的 `plugins/grpc/interceptor/unary/init_context.go` 和 `stream/init_context.go`，在 v2 的 `GrpcRunner` 里，拦截器不再手动去抠 `RegNodeKey`，而是直接调同一个 `xCtx.BridgeRequest(rpcCtx, lifetimeCtx, traceID)`：

- gRPC 客户端传过来的 deadline、取消信号、metadata 原封不动保留在 `rpcCtx` 链条上；
- `SharedContainer` 被稳稳挂在 `rpcCtx` 下，gRPC 的 Service 实现里调 `xCtxUtil.MustGetDB(ctx)` 和 HTTP Handler 的体验一模一样。

#### （4）从 `CronRunner` 传给每次激发的 `Component`：内核 `Activator` 自动装配

在 v1.x 的 `plugins/cron/runner/runner.go:164-166` 里，所有定时任务都在裸跑同一个外层 `ctx`——既没有独立的任务 `TraceID`（日志打出来分不清是哪一轮跑的），也没法单独取消某一次卡住的执行。

在 v2 里，当 `CronRunner` 整点触发、调用内核 `Activator` 激活某个报表 `Component` 时，**根本不需要 `CronRunner` 自己去拼凑 `ctx`**。内核在创建这次 `Component` 实例时，会自动从应用的 `SharedContainer` 底座上派生出一把专属于本次执行的 `componentCtx`：

- 底下焊着同一份 `SharedContainer`（所以报表任务里调 `MustGetDB(ctx)`、调缓存、发邮件全通）；
- 自动生成本次激活独享的 `RequestKey`（链路追踪 ID）与实例身份（`InstanceID`、`ParentID`）；
- 挂着受本次激活超时与关机宽限期控制的独立取消信号。

#### （5）从请求/任务再派生 `Async` 后台协程：只带共享底座与 TraceID，不带脏状态

对照 v1.x 的 `plugins/async/context.go:16-36`，当业务在某个 HTTP 请求里调 `xAsync.Async(reqCtx, fn)` 时，内核提供统一的 `xCtx.Detach(reqCtx)` 函数：

- 它基于干净的未取消基底（不再受当前 HTTP 请求返回即 cancel 的影响）；
- 自动把 `reqCtx` 里的 `SharedContainer` 和链路追踪 `RequestKey` 继承过来；
- **故意丢弃**当前 HTTP 请求绑定的临时状态（比如 `UserStartTimeKey`、`ErrorCodeKey`、`ErrorMessageKey`），防止异步协程去污染或误读主请求的 HTTP 响应码。

| 场景                      | 拿到的 Context 从哪来                          | 取消信号归谁管                           | 里面有哪些东西                                                 |
| ------------------------- | ---------------------------------------------- | ---------------------------------------- | -------------------------------------------------------------- |
| `Plugin.Start(startCtx)`  | 内核从 `SharedContainer` 底座派生              | 启动超时（`startTimeout`）               | 排在它前面已 `Start` 成功的依赖组件                            |
| `Worker.Run(lifetimeCtx)` | 内核从 `SharedContainer` 底座派生              | 内核关机协调器（进入强制取消阶段才触发） | 应用全部已就绪的共享组件                                       |
| `Plugin.Stop(cleanupCtx)` | 内核从 `SharedContainer` 底座重新派生          | 独立清理超时（`cleanupTimeout`）         | 仍存活的底层依赖组件（与已取消的 `Run` 解耦）                  |
| HTTP / gRPC 请求 `reqCtx` | `BridgeRequest(原生请求 ctx, lifetimeCtx)`     | 客户端断连或请求超时                     | 全部共享组件 + 本次请求的 `RequestKey` 与请求级临时键值        |
| `Component.Run(compCtx)`  | 内核 `Activator` 从 `SharedContainer` 底座派生 | 本次执行超时或内核关机取消阶段           | 全部共享组件 + 本次激活生成的独立 `RequestKey` 与实例身份      |
| `Async` 协程 `asyncCtx`   | `Detach(父 reqCtx / compCtx)`                  | 异步任务自身的 `Cancel` 或应用级关机     | 全部共享组件 + 继承自父级的 `RequestKey`（剥离请求级临时键值） |

### 5. 核心接口草案与精准报错机制

为了兼容现有业务代码里的 `xCtxUtil.MustGetDB(ctx)` 和 `xCtxUtil.Get[T](ctx, key)`，我们把“插件暴露共享值”、“内核桥接 Context”和“类型化提取”收敛成下面这组干净的接口：

```go
// Exposer 是插件可选实现的接口。
// 如果插件不仅要把自身实例按 PluginInfo.ID 放入共享容器，还需要暴露底层句柄
// （例如 DatabasePlugin 暴露 *gorm.DB 到 DatabaseKey，CachePlugin 额外暴露 *redis.Client 到 RedisClientKey），
// 只需实现此方法；内核会在该插件 Start(ctx) 成功返回后立即调用并写入 SharedContainer。
type Exposer interface {
    Expose() map[xCtx.ContextKey]any
}

// SharedContainer 是单个 App 实例内部持有的只读共享组件容器（取代 v1.x 的 ContextNodeList 裸切片）。
// 启动期由内核按拓扑序写入，进入 Running 前冻结为只读；关机时随插件 Stop 标记条目为已退役。
type SharedContainer interface {
    Get(key xCtx.ContextKey) (val any, state EntryState, ok bool)
}

// BridgeRequest 将 App 运行期的 SharedContainer 与链路标识嫁接到外部协议（HTTP / gRPC）的原生请求 ctx 上。
// 保留 reqCtx 原有的客户端断连与 deadline 信号，不复制其它请求的临时状态。
func BridgeRequest(
    reqCtx context.Context,
    appLifetimeCtx context.Context,
    requestID string,
) context.Context

// Detach 从当前请求或任务 ctx 派生出一个脱离原超时/取消链的独立上下文（供 Async 等场景使用）。
// 仅保留 SharedContainer 与链路追踪 RequestKey，自动剥离 ErrorCodeKey 等请求级临时数据。
func Detach(parentCtx context.Context) (context.Context, context.CancelFunc)

// Get 与 MustGet 是面向业务和插件的统一泛型读取入口（保留在上下文工具包中，无需 globalContextExtractor）。
func Get[T any](ctx context.Context, key xCtx.ContextKey) (T, *xError.Error)
func MustGet[T any](ctx context.Context, key xCtx.ContextKey) T
```

这里有四个关键细节：

1. **内部用私有 `struct{}` 键挂容器，对外继续用 `xCtx.ContextKey` 查组件**：
   把 `SharedContainer` 放进标准 `context.WithValue` 时，内核使用包内未导出的 `type containerCtxKey struct{}` 作为 key，绝不暴露给外部覆盖；而业务调 `Get[T](ctx, xCtx.DatabaseKey)` 时，函数内部先通过 `containerCtxKey{}` 一步拿到 `SharedContainer`，再在容器的哈希表里 O(1) 查 `xCtx.DatabaseKey`。如果容器里没查到（比如业务自己在某处临时 `context.WithValue(ctx, customKey, val)` 塞了个请求级小数据），再 fallback 到标准 `ctx.Value(key)`——既保住了 100% 的向下兼容，又把主路径提速到了 O(1)。
2. **`MustGetDB(ctx)` 自动绑定当前请求/任务的 `ctx`**：
   保留 v1.x（`major/utility/context/database.go:34`）一个非常贴心的细节：`GetDB(ctx)` 从共享容器里拿到全局的 `*gorm.DB` 后，返回的是 `db.WithContext(ctx)`。这样当前 HTTP 请求的超时信号和 `RequestKey`（用于 GORM 慢日志追踪）就能自动带进每一次 SQL 查询里，而不会污染共享容器里存着的那个根 `*gorm.DB`。
3. **请求级临时数据留在请求自己的 `ctx` 层，绝不写进 `SharedContainer`**：
   像 `RequestKey`、`UserStartTimeKey`、`ErrorCodeKey`、`ErrorMessageKey` 这些跟单次 HTTP/gRPC 请求绑定的数据，依然通过请求层的 `WithValue` 挂载在当次请求的 `reqCtx` 上；`SharedContainer` 只存应用级的插件与资源实例。两边井水不犯河水。
4. **告别笼统报错，按容器条目状态精准定位问题**：
   在 v1.x 里，不管你是忘了注册数据库、顺序反了、还是关机后野协程来拿，统统只报一句 `在上下文中找不到数据库，真的注入成功了吗？`（`major/utility/context/database.go:45`）。在 v2 的 `SharedContainer` 里，每个条目都带着明确的生命周期状态（`EntryState`），`Get[T](ctx, key)` 会根据现场给出精准报错：
   - 传入的 `ctx` 压根没挂 `SharedContainer`：提示“当前 Context 未挂载应用共享容器（请检查是否漏调 BridgeRequest/Detach 或误传了裸 context.Background()）”；
   - 在插件 A 的 `Start(startCtx)` 期间去拿未在 `Requires` 中声明的插件 B：提示“插件 A 未在 Requires 中声明依赖 B，禁止在启动期越权访问”；
   - 组件已被 `Stop` 卸载后仍有残留协程访问：命中 `Stopped` 状态，报错“组件 [context_database] 已停止，禁止在关机卸载后继续访问”；
   - 泛型类型 `T` 断言不匹配：报错打印“组件 [key] 实际类型为 X，无法断言为目标类型 Y”。

## 结论

- **事实结论**：
  1. v1.x 靠 `ContextNodeList` 裸切片与单条 `WithValue` 链传递共享组件，在切换取消信号（如 `Async` 或清理阶段）时必须手搓复制，且依赖包级全局变量 `globalContextExtractor` 解包 `*gin.Context`；
  2. 将“只读共享组件底座（`SharedContainer`）”与“`startCtx` / `lifetimeCtx` / `cleanupCtx` / `reqCtx` 信号层”正交分离，可以在业务层 `xCtxUtil.MustGetDB(ctx)` / `Get[T](ctx, key)` 零改动的前提下，彻底解决信号串门、O(N) 切片扫描与全局提取器耦合问题。
- **倾向（供后续编写 Design 时采用）**：
  1. 倾向采用“插件自身默认按 ID 入库 + 包装器插件可选实现 `Exposer`”双轨机制，把 `Start(ctx) error` 初始化好的底层句柄装入 `SharedContainer`；
  2. 倾向由内核提供统一的 `BridgeRequest`（供 Gin / gRPC 等 `PluginRunner` 嫁接外部请求 `ctx`）与 `Detach`（供 `Async` 剥离请求超时与临时响应状态），并在 `GinRunner` 中利用 `ContextWithFallback` + 双写彻底移除 `globalContextExtractor`；
  3. 倾向在启动阶段对 `startCtx` 启用 `Requires` 白名单检查，并在关机 `Stop` 后将容器条目翻转为 `Stopped` 状态拦截野协程访问。
- **开放问题（写详细 Design 前需拍板）**：
  1. **`PluginID` 与 `xCtx.ContextKey` 是否合一**：在 0001 里插件用 `PluginID` 标识，在 v1.x 里取组件用 `xCtx.ContextKey`（如 `xCtx.DatabaseKey`）。v2 是直接让 `type PluginID = xCtx.ContextKey` 合二为一，还是分开定义再做映射？（倾向于直接合一，少造一个概念。）
  2. **启动期是否强制检查 `Requires` 白名单**：插件 A 在 `Start(startCtx)` 里调 `Get(startCtx, keyB)` 时，如果 A 没在 `Requires` 里写 B，是直接报错拦截（强制显式声明依赖），还是只要 B 碰巧先启动了就放行？（倾向于直接报错拦截。）
  3. **`Exposer` 接口形态**：包装器插件交出内部句柄时，是用 `Expose() map[xCtx.ContextKey]any`（支持像 Cache 那样一次交出 Manager 和 RedisClient 两个键），还是单键 `Provide() (xCtx.ContextKey, any)`？

## 参考

- Go 标准库 `context` 包源码与设计规范 — <https://pkg.go.dev/context>
- Gin `Context.Value` 与 `ContextWithFallback` 实现 — <https://github.com/gin-gonic/gin/blob/v1.10.0/context.go>
- 本仓库 v1.x 上下文实现：`defined/context/context.go`、`defined/context/reg_node.go`、`major/register/node/node.go`、`major/utility/context/`、`plugins/grpc/interceptor/`、`plugins/async/context.go`
