# v2.0.0 Context 体系与共享组件传递设计

> 状态：draft · 日期：2026-10-19 · Scope：context
> 承接 [v2.0.0 Context 体系与共享组件传递调研](../research/0002-context-propagation-and-shared-state.md) 与 [0001 生命周期插件运行时设计](./0001-lifecycle-plugin-runtime.md)

## 目标与边界

本设计稿承接前期调研（[research/0002](../research/0002-context-propagation-and-shared-state.md)），并与 [0001 生命周期插件运行时设计](./0001-lifecycle-plugin-runtime.md) 配套落地。

> **单一来源说明**：本文是 v2.0.0 所有 Context 语义（生命周期 Context 派生、共享组件容器 `SharedContainer`、启动期 `Requires` 白名单视图、以及向各路 `PluginRunner` 传递机制）的**唯一权威设计文档**；`0001` 仅调用本文定义的接口。

根据已拍板的核心决策，本设计锁定以下三条硬边界：

1. **键类型合一，但身份与产物分清**：插件主键与资源键都使用 `foundation/ctxkey.Key`，不另造 `PluginID`。主键标识插件实例；需要暴露底层句柄时，在 `PluginInfo.Provides` 静态声明其它键。例如数据库插件的主键是 `plugin_database`，提供的 `DatabaseKey` 则保存 `*gorm.DB`。
2. **启动期严格 `Requires` 白名单拦截**：消费者可以声明插件主键，也可以声明资源键。内核在调用任何工厂前按 `Provides` 解析提供者并建图；但 `Start(startCtx)` 的读取权限仍只限于实际写进 `Requires` 的键。没有声明 `DatabaseKey`，调用 `database.MustGet(startCtx)` 就会触发 `ErrUndeclaredDependency`。
3. **先声明、再交付**：Tool/Worker 的 `Start` 成功后，插件实例默认按主键入库；实现 `ctxkey.Exposer` 的插件再交付 `Provides` 声明的底层句柄。`Expose()` 的键集合必须与 `Provides` 一致，不能偷偷加键、漏键或覆盖主键。配置影响产物时，在注册定义中就确定这组键，而不是启动后临时发明能力。

涉及模块：`foundation/ctxkey`（键值与 `Exposer` 契约）、`application`（容器、提供者解析与 Context 桥接），以及数据库、缓存、邮件和各入口插件。当前工作树仍使用 v1 的裸切片与全局 Extractor；本文描述的是后续替代方案，不是已经落地的能力。

## 工作方式

### 1. `SharedContainer` 的内部结构与三态流转

每个 `application.App` 实例内部独自持有一个 `*sharedContainer`，彻底取代 v1.x 的 `ContextNodeList` 裸切片和全局 `globalContextExtractor`。

```text
启动阶段 (Building)                  运行阶段 (Frozen)                 关机阶段 (Stopping)
──────────────────────             ──────────────────────           ──────────────────────
每个 Tool/Worker Start 成功后：      全部 Worker.Run 启动前：          每当某插件 Stop 执行完毕：
1. 校验 Expose 交付的完整键集合     容器置为 frozen = true            将该插件名下所有 Key 的状态
2. 写入主键及已声明的资源键         运行期 Get[T] 走 O(1) 查表        标记为 EntryStopped；
3. 所有条目记录同一个 ownerKey      不再增加或删除 map 条目           超时未退的执行另记残留
4. 将本批条目标为 EntryReady
```

容器内部对每次读取按如下规则执行检查：

- **启动期受限视图（`startScopeView`）**：
  当内核为插件 `ownerKey` 准备 `startCtx` 时，不会把裸容器直接塞进去，而是包一层带白名单集合 `allowed map[ctxkey.Key]struct{}`（由 `info.Requires` 构建）的 `startScopeView`。在 `Start(startCtx)` 内调用 `Get[T](startCtx, targetKey)` 时：
  1. 先查 `targetKey` 是否在 `allowed` 白名单里——不在就直接返回 `ErrUndeclaredDependency`（哪怕 `targetKey` 碰巧已经在前面启动好了也绝不放行，彻底治住靠注册顺序蒙混过关的隐式耦合）；
  2. 在白名单里，再按 `providerByKey[targetKey]` 确认提供者已启动，读取对应 `EntryReady` 条目。拓扑用提供者主键排顺序，白名单用消费者声明的原键管访问，两者不混在一起。
- **运行期完整视图（无白名单限制，O(1) 直出）**：
  当所有 Tool/Worker 完成 `Start` 后，`lifetimeCtx` 挂载完整容器视图。读取不再做启动期白名单拦截，但这不是绕过生命周期依赖声明的许可：插件在 `Run/Stop` 或后台任务中使用的共享资源，同样应写入 `Requires`，否则内核无法据此保证停机顺序。哈希表查找本身是 O(1)；端到端是否零分配仍需实测，不在这里先作承诺。
- **关机期退役拦截（`EntryStopped`）**：
  每个容器条目内部用 `atomic.Uint32` 维护状态（`EntryReady = 1`、`EntryStopped = 2`）。当关机按逆拓扑推进、某个插件（比如 `DatabasePlugin`）执行完自己的 `Stop(cleanupCtx)` 后，内核立刻将它登记的主键及 `Expose()` 出来的全部附加键状态原子翻转为 `EntryStopped`。如果此后还有超时未退的野协程拿着老 `ctx` 调 `database.Get(ctx)`，容器立刻拦截并返回 `ErrComponentStopped`。

### 2. 换取消信号，不换共享底座

内核用私有的 `containerCtxKey`、身份凭据键与关机预算键组合 Context。容器只保存 App 级资源，实例身份只由内核签发，预算只控制等待与清理时限，三者不要揉成一份可写元数据。

| Context               | 从哪派生、何时结束                                 | 可读内容                                         |
| --------------------- | -------------------------------------------------- | ------------------------------------------------ |
| startCtx              | 未取消共享根 + 默认 15s，Start 返回即取消          | Requires 精确白名单，不给后台执行长期使用        |
| lifetimeCtx / compCtx | 未取消共享根 + 内核控制的 Cancel；第 20 秒协作取消 | 完整容器、私有 App/实例凭据、描述性 InstanceMeta |
| stopCtx               | 未取消共享根 + 本次关机总截止，默认第 60 秒        | 完整容器与本轮预算，不继承已取消运行信号         |
| graceCtx              | 从 stopCtx 取固定排空截止，默认第 20 秒            | 用来判断何时从自然排空转为协作取消               |
| cleanupCtx            | 纯资源收尾开始时创建，默认最长 10s，且不晚于总截止 | 同一容器与独立清理信号，不给整个应用续期         |

Start/Run/Stop 都由内核受管协程监督。App 的强制取消根由关机定时器在 Grace 到期独立触发，不依赖某个插件的 Stop 已经开始。Context 到期只是通知；方法忽略取消时，内核不伪造返回，也不并发再调 Stop。协调器到总上限返回残留快照，方法真实完成仍由监督器记录。

Component 的 Stop 在 Run 和子树结束后执行，直接使用 cleanupCtx。Worker/Async 的 Stop 还要排空，先拿 stopCtx，经 GraceContext 得到排空截止，等执行真退后才通过 CleanupContext 创建纯清理窗口。不能把 10s 清理倒计时从自然排空开始就算起。协调器到总上限返回之后，若已接纳实例才进入自己的唯一一次 Stop，则获得原配置中最长 10s 的独立尾部清理窗口；这只完成该实例，不重启底层资源的关机遍历。

### 3. 插件注册后，怎么一路传给四大 `PluginRunner`？

当 DatabasePlugin、CachePlugin 等基础设施完成 `Start`，并把 `*gorm.DB`、`*cache.Manager`、`*redis.Client` 装进 SharedContainer 后，共享 Context 按以下四条路径传给入口插件和业务代码：

#### （1）`GinRunner`（`plugins/web-gin`）：双保险桥接 + 彻底废除 `globalContextExtractor`

1. **引擎初始化（`GinRunner.Start`）**：
   创建 `*gin.Engine` 时，强制开启 `engine.ContextWithFallback = true`（让 `gin.Context.Value(key)` 在查不到 `c.Keys` 时自动穿透查询 `c.Request.Context().Value(key)`）。同时，把已挂载 `SharedContainer` 的 `startCtx` 传给所有路由注册函数 `RouteRegistrar(startCtx, engine)`。
2. **每次 HTTP 请求进入（内置 `ContextBridgeMiddleware`）**：
   中间件生成或提取本次请求的 `requestID`，调用一行 `application.BridgeRequest`：

   ```go
   reqCtx := application.BridgeRequest(
       c.Request.Context(),
       r.lifetimeCtx,
       requestID,
   )
   c.Request = c.Request.WithContext(reqCtx)
   ```

   `BridgeRequest` 内部做的是**自定义组合 Context（`bridgedContext`）**：
   - Deadline/Done/Err 委托给协议插件准备的请求 Context。客户端取消仍有效；HTTP ReadTimeout/WriteTimeout 是 I/O 时限，不等于给业务函数自动设执行 deadline。需要业务超时的路由应显式创建请求超时 Context。
   - `Value(key)` 查询容器或私有身份键时，从 `r.lifetimeCtx` 取当前 App 与 Worker 的内核凭据；查询 `RequestKey` 时取本次 requestID，其余请求级数据委托原生请求 Context。
     业务读取组件仍走标准 `ctx.Value`。Handler 调 `Activate(reqCtx, jobKey)` 时，内核也能确认是哪一个 Worker 接纳了请求；请求 ID 可以变化，但父级实例身份不会被业务提交的数据替换。

HTTP 插件在 Start 从 application.ForceContext(startCtx) 派生自身 forceCtx，通过 http.Server.BaseContext 接入请求根。这份通知只在全局 Grace 到期触发，不受短命 startCtx 取消影响，也不是刚收到关机信号就取消。Handler 外层登记活跃执行，进入和关闭准入采用同一同步边界，确保 Stop 不会在新增 Handler 登记前误判归零。

Stop 调 Server.Shutdown(graceCtx) 停接入并排空；Grace 到期后取消 forceCtx、调用 Server.Close，再等 Handler 真退。Serve 返回和 Shutdown 超时都不等于 Handler 已结束，只有自管执行归零后才允许 Stop 返回。WebSocket/hijacked 连接不在默认跟踪承诺内，需要独立受管 Worker。

#### （2）gRPC：保留客户端取消，再接入服务端取消

gRPC 与 HTTP 一样，在 Start 从 application.ForceContext(startCtx) 派生本插件的 forceCtx，再在 Unary/Stream 拦截器中分别跟踪活跃调用。对原生 RPC ctx 派生取消分支，通过 `context.AfterFunc(r.forceCtx, cancelRPC)` 接入本插件强制取消根；请求结束时停止回调、取消派生分支并退还计数。Stream 必须包装 ServerStream.Context，不能只改局部变量。

随后调 `BridgeRequest(rpcCtx, r.lifetimeCtx, traceID)` 挂入容器与可信 Worker 身份。客户端 deadline/metadata 保留，Grace 结束后的服务端取消也能进入业务函数。

GrpcRunner.Stop 先开始 GracefulStop；Grace 到期则取消自管调用并调 Server.Stop，但仍等业务真实返回和 Serve 结束。GracefulStop 本身不接收 Context，需由适配器协调，不能当成天然带超时的 Stop。

#### （3）`CronRunner`（`plugins/cron`）：经 `Activator` 自动派发独立 `compCtx`

当 `CronRunner.Run(lifetimeCtx)` 到点触发某个定时任务时，它调用：

```go
handle, err := r.activator.Activate(
    r.lifetimeCtx,
    job.ComponentKey,
    application.WithPayload(tickTime),
)
```

内核 `Activator` 造出新的 `Component` 实例后，直接用 `newComponentContext` 生成本次执行专属的 `compCtx` 传给 `Component.Run(compCtx)`：

- 里面自带应用级 `SharedContainer`（调 `database.MustGet(compCtx)` 直接通）；
- 自动生成了本次 tick 独享的 `ctxkey.RequestKey`（每次定时任务执行在日志里都有独立的 TraceID，不再混成一团）；
- 可以通过 `application.PayloadFrom[time.Time](compCtx)` 取出当次触发参数 `tickTime`。

#### （4）`Async`（`plugins/async`）：调用 `Detach` 剥离超时与临时脏数据

当业务在 HTTP 或 gRPC 请求里调用新的 `async.Submit(reqCtx, fn)` 时，插件内部调用 `application.Detach(reqCtx)`：

- 从未取消的基底派生，值只继承 SharedContainer 与 RequestKey；取消链不继承原请求，而是接到当前 App 的强制取消根。返回的 Cancel 同时负责结束该任务并移除通知挂钩。原请求的超时、错误码和临时数据不搬过去。
- 也不复制原 Worker/Component 的激活凭据，避免入口已停后，脱离请求的任务继续冒用父级身份。Async 自己跟踪计数并负责排空；任务 Context 在全局 Grace 到期就收到取消，不必等 Async.Stop。普通 Detach 不代替任务登记，也不是脱离生命周期监督的许可。Detach 不接受启动期受限视图，BridgeRequest 的应用侧参数也必须来自内核签发的运行 Context；误用是编程错误，不允许借桥接把 startScopeView 升格为完整容器。
- 因而仅持有 Detach Context 的后台函数不能直接激活 Component，调用会明确报缺少合法父级。若将来确有这种需求，应新增明确的受管父实例能力，不暗中延长旧请求父级的寿命。

## 数据与接口

### 1. `foundation/ctxkey` 包扩展接口

在最底层的 `foundation/ctxkey`（详见 [0003 设计稿](./0003-foundation-module-consolidation.md)）中保持零外部框架依赖，定义 `Exposer` 接口：

```go
package ctxkey

// Exposer 供包装器插件交付 PluginInfo.Provides 中声明的底层句柄。
// 内核在 Start 成功后调用并检查完整键集合，通过后再一起登记进共享容器。
type Exposer interface {
    Expose() map[Key]any
}
```

### 2. `application` 包 Context 与组件提取接口

```go
package application

import (
    "context"

    "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
    "github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

// EntryState 表示共享容器内组件条目的生命周期状态。
type EntryState uint8

const (
    EntryReady   EntryState = iota + 1 // 已完成 Start，可正常读取
    EntryStopped                       // 已执行 Stop 退役，拒绝读取
)

// InstanceMeta 是身份的只读描述；复制或自行构造此结构不会获得激活权限。
type InstanceMeta struct {
    InstanceID       string
    PluginKey        ctxkey.Key
    ParentKey        ctxkey.Key // 顶层 Worker 为空，其余为父插件键
    ParentInstanceID string     // 实例树以此关联，顶层 Worker 为空
    TraceID          string     // 当前运行或请求的 RequestKey
}

// WithPayload 在调用 Activator.Activate 时向新建的 Component 实例传递当次触发参数。
func WithPayload(payload any) ActivateOption

// PayloadFrom 从 Component 的 ctx 中类型安全地提取 Activate 时传入的参数。
func PayloadFrom[T any](ctx context.Context) (T, bool)

// InstanceMetaFrom 从运行期 ctx 中提取当前实例的身份元信息。
func InstanceMetaFrom(ctx context.Context) (InstanceMeta, bool)

// BridgeRequest 挂载应用容器、可信父实例身份与 requestID。
// 请求取消仍来自 reqCtx；应用强制取消由协议插件合入其请求根 Context。
func BridgeRequest(
    reqCtx context.Context,
    appLifetimeCtx context.Context,
    requestID string,
) context.Context

// Detach 只保留容器与追踪 ID，不继承临时数据或父实例激活凭据。
func Detach(parentCtx context.Context) (context.Context, context.CancelFunc)

// GraceContext 使用本轮固定排空截止，不因晚进入某一层而续期。
func GraceContext(stopCtx context.Context) (context.Context, context.CancelFunc)

// CleanupContext 在协调期间不越过总截止；已接纳实例的晚到收尾使用独立上限。
// 两项预算函数仅接受内核发放的 Stop Context，误传请求 ctx 是编程错误。
func CleanupContext(stopCtx context.Context) (context.Context, context.CancelFunc)

// ForceContext 只提供当前 App 的 Grace 到期取消通知，不挂容器/身份，也不暴露 Cancel。
// 可从受限 startCtx 取这份内核控制通知，不能把 startCtx 自身作为长期取消根。
func ForceContext(ctx context.Context) context.Context

// Get 从 ctx 挂载的 SharedContainer 中按 key 读取类型化组件实例（O(1) 查找，未命中时回退 ctx.Value）。
func Get[T any](ctx context.Context, key ctxkey.Key) (T, *apperr.Error)

// MustGet 为 Get 的 panic 版本，供确认依赖已就绪的业务 Handler 简写调用。
func MustGet[T any](ctx context.Context, key ctxkey.Key) T
```

领域访问器放在各自插件中，例如 `database.Get(ctx)` / `database.MustGet(ctx)` 调用 `application.Get[*gorm.DB](ctx, ctxkey.DatabaseKey)`，再返回 `db.WithContext(ctx)`，把当前请求或任务的取消信号与追踪 ID 带入 GORM 会话。foundation 与 application 本身不依赖 GORM。

## 异常与兼容

### 1. 按真实原因报错

`Get[T](ctx, key)` 遇到异常时，按以下场景返回带精确原因的 `*apperr.Error`（`MustGet[T]` 则 panic 相同信息）：

1. **未挂载容器（`ErrContainerNotMounted`）**：传入的是裸 `context.Background()` 且未走 `BridgeRequest` / `Detach`，报错：`上下文未挂载应用 SharedContainer，请检查是否漏调 BridgeRequest/Detach`；
2. **启动期未声明依赖（`ErrUndeclaredDependency`）**：插件 `ownerKey` 在 `Start(startCtx)` 中读取未列入自身 `Requires` 的 `key`，报错：`插件 [ownerKey] 未在 Requires 中声明依赖 [key]，禁止在 Start 阶段越权读取`；
3. **关机后访问已退役组件（`ErrComponentStopped`）**：目标组件已完成 `Stop`，条目状态为 `EntryStopped`，报错：`组件 [key] 已停止卸载，禁止继续访问`；
4. **类型断言不匹配（`ErrComponentTypeMismatch`）**：键存在但实际类型与泛型参数 `T` 不符，报错：`组件 [key] 实际类型为 %T，无法断言为目标类型 %T`。

5. **没有该组件（`ErrComponentNotFound`）**：容器已挂载，但既无该键条目，原生 ctx.Value 也没有该请求级值，返回缺失错误；不伪装成类型不符。启动白名单校验始终先执行，禁止利用 fallback 绕过声明检查。

### 2. `Provides` 与 `Exposer` 的交付校验

内核先按 `0001` 的规则预留全部主键和资源键，冲突在调用工厂前返回 `ErrDuplicateExposedKey`。一个插件的主键与 `Provides` 不能相互覆盖；同一键也不能有两个提供者。

`Start` 成功后，先检查 `Expose()` 的完整结果，再发布这批条目。多出未声明键、漏掉声明键、返回 nil 或类型化 nil 句柄，或声明了 `Provides` 却未实现 `Exposer`，都按 `ErrExposedKeysMismatch` 中止启动：先清理当前已启动实例，再逆序回滚此前实例，不留下半批可读取的资源。无附加资源的插件可省略 `Exposer`，此时 `Provides` 必须为空。

以缓存为例：Memory 模式只声明 `CacheManagerKey`；Redis 模式声明 `CacheManagerKey` 与 `RedisClientKey`。后端在构造注册定义时已经确定，因此启动前就能知道 Redis 是否真的可供依赖。

### 3. v1 调用方主动迁移

v2 不保留 `major/utility/context` 的旧访问器兼容壳。业务需要迁移 import、键类型与访问入口；数据库改用 `database.Get/MustGet`，通用组件改用 `application.Get/MustGet`，不能把旧 `xError.Error` 与新 `apperr.Error` 当成同一类型。

固定已发布 v1 版本的项目仍使用其原有实现，不会因本次设计自动切换到新容器。这里保留的是“从 Context 读取共享资源”的使用思路，不是源码级零改动承诺。

## 备选与取舍

把每个共享值都 WithValue 到每次请求也能工作，但启动视图与关机状态容易散落到多条链。这里保留一份容器，再组合取消与身份；它不是替代显式依赖声明的 DI 黑盒。

请求直接继承应用运行 Context 容易在第一次退出信号到来时全被取消；完全只服从原请求又无法在 Grace 后催停。因此由协议插件合并独立 forceCtx，BridgeRequest 负责值与身份。行为依据见 [context](https://pkg.go.dev/context) 与 [HTTP Shutdown](https://pkg.go.dev/net/http#Server.Shutdown)。

若身份要跨进程传输，必须重评：私有 Context 凭据不是网络鉴权。若基准测试发现桥接带来明显额外分配，应先测量再减层，不继续宣称未经验证的零成本。

## 验证

代码尚未实现。落地后在 application 模块执行 `GOWORK=off go test -race ./...`，web-gin/grpc 模块分别补适配测试：

1. 静态键冲突在工厂前失败；Expose 多键、漏键、nil、缺 Exposer 时回滚，不发布半批条目。
2. Requires 只写 DatabaseKey，也能按提供者排序并读取；只写插件主键不获得额外资源键的 Start 权限。
3. 取消 startCtx 或运行 Context 不影响新建 stopCtx；协调期间纯清理不越过总截止。残留实例在总上限之后才开始 Stop 时，另获原配置中的尾部清理窗口，不重复清理旧实例。
4. 真实 HTTP/gRPC 保留客户端取消、deadline 与 metadata；Grace 内不中断正常请求，之后服务端取消可进入业务。仅取消 Worker lifetimeCtx 不得被误当成请求已取消。
5. Shutdown 与 Serve 已返回、测试 Handler 仍阻塞时，Stop 仍等待、DB 仍存活。gRPC Unary/Stream 做等价验证。
6. Gin ContextWithFallback 读同一容器；嵌套 Component 按 ParentInstanceID 归属。自报身份、跨 App、已结束父级与祖先递归均被拒绝。
7. Detach 只保留容器与追踪 ID，不保留激活凭据；原请求取消不影响任务，但全局 Grace 到期会取消它，即使 Async.Stop 尚未开始。完成信号仍等函数真实返回，Cancel 回收通知挂钩不泄漏。
8. 两个 App 的容器、身份与 forceCtx 互相隔离；EntryStopped 拦截新读取，但不能撤销业务先前已经持有的裸句柄。

## 待确认

容器、身份、排空与迁移方向已经确认，本篇无剩余阻塞性方案选择。协议适配与取消竞态仍需实现后验证，不把文档确认当成运行验证通过。
