# v2.0.0 插件体系分类与第三方库选型调研

> 日期：2026-10-18 · 范围：plugins/* 分类版图、原 major 拆解归宿与核心依赖库对比
> 后续设计说明：本文记录选型与代码现状，新插件生命周期尚未实现。已确认的组织方式以 [design/0005](../design/0005-plugin-architecture-and-adapter-pattern.md) 为准；最新版本方案为：源码目录不增 v2/ 层，所有独立模块的 module/import 带小写 /v2，首批整体 Release 与模块版本均为 v2.0.0；后续模块按需发版，整体清单记录精确组合。后续已确认 Async.Stop 必须等真退，Gin 框架绑定走实例校验，普通网络请求受管而 WebSocket/hijack 不作默认保证；启动期环境配置与 60s 默认总协调上限由 design/0001 统一定义。

## 调研问题

随着地基模块 `foundation`（见 [design/0003](../design/0003-foundation-module-consolidation.md)）与生命周期内核 `application`（见 [design/0001](../design/0001-lifecycle-plugin-runtime.md)、[design/0002](../design/0002-context-propagation-and-shared-state.md)）边界锁定，v1.x 里那个大包大揽的 `major` 模块将被彻底拆解，所有框架适配、网络协议、任务调度与有状态存储统一收敛到 `plugins/*` 层。

在逐个编写插件详细设计前，本调研先摸清三个问题：

1. **插件版图怎么划分？** 原 `major` 留下的 HTTP 组件、缓存系统、GORM 模型与日志适配器，以及剛从 `foundation` 剥离出来的 `http`、`validator`、`database` 注册表，各自归入哪个插件、对应 `Worker / Component / Tool` 哪种模式？
2. **现有 `plugins/{grpc,cron,async,email,database}` 有哪些实现痛点？** 对接 `Start / Run / Stop` 与 `SharedContainer` 时需要补上哪些缺口？
3. **底层开源库怎么选？** 哪些现成库已经是最优选，哪些地方（如 Cron 的调度需求或裸开协程的 Async）值得评估替代方案？

## 现状与发现

### 1. `Worker` 类网络入口插件：HTTP（Gin）与 gRPC

这类插件的共同特征是：**在 `Start(startCtx)` 阶段完成端口绑定（`net.Listen`）与路由/服务表挂载，在 `Run(lifetimeCtx)` 阶段阻塞执行 `Serve(listener)`，在 `Stop(cleanupCtx)` 阶段执行优雅停机。**

#### （1）HTTP / Web 插件（整合散落的 9 处 Gin 相关代码）

目前仓库里没有独立的 HTTP 插件，相关代码散落在 `major`、`defined` 和 `common` 三个模块共 9 个目录中：

- **引擎与启停**：`major/register/register_gin.go`（初始化 `*gin.Engine`）、`major/main/web.go`（构造 `*http.Server` 并调 `Shutdown`）；
- **中间件与路由兜底**：`major/middleware/`（CORS 与未写入响应兜底）、`major/helper/`（请求 Context 注入、HTTP 日志密码/Token 脱敏、Panic 恢复）、`major/route/`（NoRoute 404 / NoMethod 405）；
- **请求绑定与参数校验**：`major/utility/bind.go`（泛型 `Bind` 链式绑定）、`common/validator/`（8 个自定义 `validator/v10` 标签与中文翻译）、`major/validator/`（`GinValidateProvider` 与 40014 错误组装）；
- **协议结果输出（`httpresult`）与 Header**：原 `major/result/`（`xResult`）移入 HTTP 插件并重命名为 **`httpresult`**（`package httpresult`），基于底座的 `apperr.Error` 与 `envelope.Response` 标准定义，自动按 `Code/100` 决策 HTTP 状态码与报错展示内容并输出 JSON；原 `defined/http/` 同步迁入负责 Header 常量与 Bearer Token 提取。

**代码细节发现**：

- `major/main/web.go:25-39` 把 `ListenAndServe()` 塞在后台 goroutine 里跑，如果端口被占用，启动阶段毫不知情，只能在后台报错再反向触发全局 `ctxCancel()`。改用 `Start` 内先调 `net.Listen("tcp", addr)` 后，端口冲突能在启动期当场拦截并触发逆序回滚。
- `common/validator/translator.go:20-34` 使用包级全局变量 `var trans ut.Translator` 和 `var globalValidateProvider ValidateProvider`，不仅破坏多实例隔离，还在未加锁的情况下惰性初始化。收编进 HTTP 插件后，可将 `*validator.Validate` 与 `ut.Translator` 直接作为插件实例的私有字段，在 `Start` 中一次性完成绑定。
- 底层库方面：`github.com/gin-gonic/gin`（`v1.12.0`）与 `github.com/go-playground/validator/v10`（`v10.30.3`）配合极其紧密（Gin 的 `binding.Validator` 原生就是 `validator/v10`），无需更换底层库。

#### （2）gRPC 插件（`plugins/grpc`）

- 现有实现位于 `plugins/grpc/runner/runner.go`，底层采用 `google.golang.org/grpc` `v1.82.1` 与 `protobuf` `v1.36.11`。
- **代码细节发现**：
  1. `runner.go:219-239` 在后台协程里调 `net.Listen`，遇到错误直接调 `log.Panic` 炸掉整个进程；
  2. `runner.go:244-270` 的 `stopServer` 已经写了“先 `GracefulStop()`、超时回退 `Stop()`”的逻辑，但受限于 v1 `major/main/goroutine.go:37-43` 提前放行 `WaitGroup`，这段优雅关闭在进程退出时经常来不及跑完就被强行截断；
  3. 目前只默认注册了 `reflection.Register(grpcServer)`（`runner.go:209`），尚未内置标准的 `google.golang.org/grpc/health` 健康检查服务，在 K8s 环境下做 gRPC 原生探针不够顺手。
  4. **协议结果输出（`grpcresult`）统一命名与决策**：原 `plugins/grpc/result`（`xGrpcResult`）包名规范化为 **`grpcresult`**（`package grpcresult`），与 HTTP 插件的 `httpresult` 形成工整对仗；配合 Unary/Stream 响应拦截器，统一消费底座 `apperr.Error` 的错误码与错误信息来决策 gRPC `codes.Code` 状态码及 Protobuf `BaseResponse` 报错显示内容。

### 2. `Worker + Component` 调度与并发插件：Cron 与 Async

#### （1）Cron 定时调度插件（`plugins/cron`）

- 现有实现位于 `plugins/cron/runner/runner.go`，底层依赖 `github.com/robfig/cron/v3` `v3.0.1`。
- **代码细节发现**：
  - `runner.go:164-166` 在 `c.AddFunc` 里直接裸跑 `jobFn(ctx)`，既没有并发防重入保护，也共用同一个顶层 `ctx`（所有定时任务日志没有独立 TraceID）。
- **第三方库生态摸底（2026 年现状）**：
  - **`github.com/robfig/cron/v3`（`v3.0.1`）**：Go 社区最经典的 Cron 库，范围较小、零外部依赖；当前仓库仍使用 v3.0.1。本次不把“长期稳定”直接等同于不可用，维护状态判断应以仓库记录为准。它只负责解析五段/六段 Cron 表达式并按时触发 `func()`。
  - **`github.com/go-co-op/gocron/v2`（最新 `v2.22.0`，持续活跃维护）**：现代 Go 调度器首选库，内部表达式解析仍复用 `robfig/cron/v3`，但在此之上封装了完整的任务生命周期、原生 `context.Context` 注入、单例防重入模式（`LimitModeReschedule` / `LimitModeWait`）、一次性任务、随机抖动间隔（Jitter）以及前置/后置事件监听器。
  - **关键权衡点**：在我们的 v2 内核（[design/0001](../design/0001-lifecycle-plugin-runtime.md)）中，Cron 调度器本身是个父级 `Worker`，每到时间点只负责调用 `Activator.Activate(ctx, jobKey, WithPayload(tickTime))` 激发子级 `Component`——也就是说，**重入控制（默认 `OverlapSkip`）、并发限流、独立 `compCtx` 与关机排空计划由 `application.Activator` 统一实现，当前工作树尚无这一内核**。如果只把 Cron 当成“按时间表达式调一下 `Activator` 的触发器”，零外部依赖的 `robfig/cron/v3` 反而不易与内核的调度策略叠床架屋；而若希望支持更丰富的调度语法（如带随机抖动的间隔任务、每日多时间点、一次性延迟触发），`gocron/v2` 则更顺手。

#### （2）Async 异步任务插件（`plugins/async`）

- 现有实现位于 `plugins/async/async.go` 与 `context.go`。
- **代码细节发现**：
  1. **意外引入 Gin 依赖**：仅仅为了在 `async.go:24` 写一句 `if _, ok := parentCtx.(*gin.Context); ok { panic(...) }`，`plugins/async/go.mod:8` 竟然直接依赖了 `github.com/gin-gonic/gin`！在 0002 开启 Gin `ContextWithFallback` 并统一走 `application.Detach(parentCtx)` 后，这个检查和 Gin 依赖可以彻底删掉。
  2. **无界裸开协程 + 关机无人收尾**：每次调 `Async(ctx, fn)` 就直接 `go func()`，突发流量下没有并发上限保护；更重要的是，派发出去的 `*Task` 不受应用生命周期监管，收到 SIGTERM 关机信号时，主进程根本不知道还有多少个 `Async` 协程正在写数据库，直接关掉 DB 连接池就会导致异步任务批量报错。
- **v2 形态与协程池库摸底**：
  若将 `Async` 改造为应用级受管插件（在运行期限制最大并发、在关机 Grace 阶段等待在途异步任务排空），社区有三个主流方案：
  1. **`github.com/sourcegraph/conc`（`v0.3.0`）**：Sourcegraph 出品的结构化并发库，体量极轻（零外部依赖），核心是 `pool.ContextPool` 与自动捕获并传递 panic 堆栈的 `conc.WaitGroup`。非常适合做轻量级有界并发控制。
  2. **`github.com/alitto/pond/v2`（`v2.x`，活跃维护）**：专为现代 Go 设计的极简高性能协程池，原生支持 `SubmitErr(func() error)`、Context 绑定、有界任务队列、`StopAndWait()` 优雅排空以及运行指标统计（正在运行数、排队数、完成数）。
  3. **`github.com/panjf2000/ants/v2`（`v2.11.x`）**：14.3k Stars 的老牌协程池，极致压榨 goroutine 复用性能，但接口偏底层（任务签名不带 `ctx`），在业务级异步任务中不如 `pond/v2` 或 `conc` 直观。

### 3. `Tool` 类有状态资源插件：Database、Cache/Redis 与 Email

这类插件只实现 `Plugin`（`Start` + `Stop`），在 `Start` 中建立连接池并验活，在 `Stop` 中安全释放连接。

#### （1）Database 插件体系（GORM 核心 + 5 大驱动插件）

- `foundation` 剥离 `common/database` 后，GORM 相关的全部能力可以 neatly 聚合到 **`plugins/database`** 主模块（下挂按需引入的 `mysql`、`postgres`、`sqlite`、`oracle`、`sqlserver` 驱动插件）：
  - **驱动注册表**：原 `common/database/dialector.go`（`RegisterDialector` / `GetDialectorFactory`）；
  - **连接池生命周期**：原 `major/register/init/init_database.go` 與 `major/option/database/`，升级为实现 `application.Plugin` + `ctxkey.Exposer` 的 `DatabasePlugin`（`Start` 建连并 `PingContext`，暴露 `ctxkey.DatabaseKey -> *gorm.DB`；`Stop` 调 `sqlDB.Close()`）；
  - **日志与实体基类**：原 `major/log/gorm.go`（GORM 到 `foundation/logging` 的适配器）和原 `major/models/`（`BaseEntity` 雪花 ID 钩子与泛型 `PageResponse[T]`）。
- 底层库：继续使用 `gorm.io/gorm` `v1.31.2` 及各官方/纯 Go（`libtnb/sqlite`）驱动，生态成熟稳定。

#### （2）Cache 与 Redis 插件（`major/cache` + `major/hook/redis.go`）

- 原 `major/cache` 提供了一套非常漂亮的四象限泛型缓存接口：`KeyCache[K,V]`、`HashCache[K,F,V,S]`、`SetCache[K,V]`、`ListCache[K,V]`，支持 Redis（`github.com/redis/go-redis/v9` `v9.21.0`）与本地内存（`major/cache/memory/store.go`）双后端无缝切换。
- **内存缓存后端对比发现**：
  我们对比了 Go 社区热门的高性能内存缓存库 **`maypok86/otter`**（W-TinyLFU 算法，2026 年最活跃的现代泛型缓存）、**`dgraph-io/ristretto`** 与 **`allegro/bigcache`**：
  - 这些第三方库全部是**纯扁平 KV 缓存**（只提供 `Get(key)` / `Set(key, val, ttl)`）；
  - 而我们的 `Cache` 插件要求本地内存后端同时实现 **`HashCache`（哈希表字段级读写）、`SetCache`（集合交并差与成员判断）、`ListCache`（双端队列推入弹出）**，以便业务在单测或单机部署切到 `WithMemory()` 时，原本调 Redis Hash/Set/List 的业务代码一行不改就能跑！
  - 因此，现有的自研 16 分片 `memory.Store`（`major/cache/memory/store.go`）不可被纯 KV 库替代；迁入 `plugins/cache` 后，只需让 `CachePlugin` 实现 `Start/Stop`（在 `Stop` 中调用 `manager.Close()` 关停 `memoryJanitor` 协程或 `redis.Client`），并通过 `ctxkey.Exposer` 同时暴露 `ctxkey.CacheManagerKey` 与（当启用 Redis 时的）`ctxkey.RedisClientKey`。

#### （3）Email 邮件客户端插件（`plugins/email`）

- 现有 `plugins/email` 底层使用的是 **`github.com/wneessen/go-mail` `v0.8.1`**。
- **第三方库对比发现**：老牌的 `go-gomail/gomail` 已停更超过 7 年，而 `wneessen/go-mail` 是目前 Go 社区公认的现代标准替代品（持续高频发版，原生支持 Go 1.25、`context.Context` 超时控制、STARTTLS 强制策略与 `html/template` 渲染）。因此本次保留底层库，将 `InitClient` 包装为标准的 `Tool` 插件（实现 `Start` 配置校验与可选 `Stop`）。

## 对比

### 1. v2.0.0 插件分类版图与三模式归宿总表

| 插件模块与建议包名                                         | 运行模式                    | `Start / Run / Stop` 职责划分                                                                                                        | 容器入库与 `Exposer` 暴露键值                                                                | 收编的原 v1.x 代码位置                                                                                                                        |
| ---------------------------------------------------------- | --------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| `plugins/web-gin`（`package webgin`，含子包 `httpresult`） | `Worker`                    | `Start`：初始化 Gin、注册验证器与路由、执行 `net.Listen`；`Run`：`httpServer.Serve(ln)`；`Stop`：`httpServer.Shutdown(cleanupCtx)`   | 主键：HTTP 插件自身                                                                          | `major/{middleware,helper,route,validator}`、`major/result`（改名 `httpresult`）、`major/utility/bind.go`、`defined/http`、`common/validator` |
| `plugins/grpc`（含子包 `grpcresult`）                      | `Worker`                    | `Start`：装配拦截器、注册服务与 Health、执行 `net.Listen`；`Run`：`grpcServer.Serve(ln)`；`Stop`：`GracefulStop()` 超时回退 `Stop()` | 主键：gRPC 插件自身                                                                          | `plugins/grpc`（原 `xGrpcResult` 改名 `grpcresult`）                                                                                          |
| `plugins/cron`（`package cronplugin`）                     | `Worker` (+ 子 `Component`) | `Start`：校验表达式并装载任务表；`Run`：启动定时器，到点调 `Activator.Activate` 激发 `Component`；`Stop`：停止定时器不再发新 tick    | 主键：Cron 调度器自身                                                                        | `plugins/cron`                                                                                                                                |
| `plugins/async`（`package async`）                         | `Tool`                      | `Start`：初始化有界协程池；`Stop`：拒绝新任务并在清理预算内等待在途异步任务跑完                                                      | 主键：Async 调度器（新接口使用 `Submit(ctx, fn)`）                                           | `plugins/async`（移除 Gin 依赖）                                                                                                              |
| `plugins/database` + `database/*`                          | `Tool`                      | `Start`：按驱动构造 `Dialector`、`gorm.Open` 并 `PingContext`；`Stop`：`sqlDB.Close()`                                               | 主键：插件自身；`Exposer`：`ctxkey.DatabaseKey` (`*gorm.DB`)                                 | `common/database`、`major/option/database`、`major/register/init/init_database.go`、`major/models`、`major/log/gorm.go`、`plugins/database/*` |
| `plugins/cache`（`package cache`）                         | `Tool`                      | `Start`：初始化 Redis（并 `Ping`）或 Memory 分片 Store；`Stop`：关闭 Redis 连接池或停掉 `memoryJanitor`                              | 主键：缓存插件实例；`Provides`/`Exposer`：`CacheManagerKey`，Redis 模式另含 `RedisClientKey` | `major/cache`、`major/option/cache`、`major/register/init/init_cache.go`、`major/hook/redis.go`                                               |
| `plugins/email`（`package email`）                         | `Tool`                      | `Start`：校验 SMTP 配置、构造 `*mail.Client` 与模板管理器；`Stop`：空实现或关闭长连接                                                | 主键：`ctxkey.EmailClientKey` (`*email.Client`)                                              | `plugins/email`                                                                                                                               |

### 2. 核心领域开源库横向对比

| 领域             | 当前使用库                                                 | 备选现代库                                                                  | 核心差异与维护状态（截至 2026）                                                                                                                            | 适配度观察                                                                                                        |
| ---------------- | ---------------------------------------------------------- | --------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| **Cron 调度**    | `robfig/cron/v3` (`v3.0.1`)                                | **`go-co-op/gocron/v2`** (`v2.22.0`)                                        | `robfig/cron` 范围更小；`gocron/v2`（7.1k+ Stars）是当前社区最活跃的现代调度库，兼容 `robfig/cron` 语法且原生支持 `context.Context`、Jitter 与生命周期钩子 | **优先采用最热门社区库复用策略**：升级为 `go-co-op/gocron/v2`，外层对接内核 `Activator` 激发 `Component`          |
| **Async 并发池** | 裸 `go func()`                                             | **`panjf2000/ants/v2`** (`v2.11.x`) / `alitto/pond/v2` / `sourcegraph/conc` | 裸协程无上限且关机失管；`ants/v2`（14.3k+ Stars）是 Go 社区最热门、经过大规模生产验证的高性能协程池标杆                                                    | **优先采用最热门社区库复用策略**：底层复用 `panjf2000/ants/v2`，外层封装 `application.Detach(ctx)` 与关机排空等待 |
| **本地内存缓存** | 自研分片 `memory.Store`                                    | `maypok86/otter` / `dgraph-io/ristretto`                                    | `otter` 与 `ristretto` 拥有顶尖的 W-TinyLFU 淘汰算法，但**仅支持扁平 KV**；自研 `memory.Store` 原生支持 Key / Hash / Set / List 四大结构                   | 为保持与 Redis 后端在 Hash/Set/List 上的 100% 透明切换，保留自研 `memory.Store` 最合适                            |
| **邮件发送**     | `wneessen/go-mail` (`v0.8.1`)                              | `go-gomail/gomail`                                                          | `gomail` 已停更 7 年；`wneessen/go-mail` 活跃维护、原生支持 Context 与现代 TLS 策略                                                                        | 现有选型支持所需功能，本次保留                                                                                    |
| **HTTP 与校验**  | `gin-gonic/gin` (`v1.12.0`) + `validator/v10` (`v10.30.3`) | `gofiber/fiber` / `labstack/echo`                                           | Gin 内置 `validator/v10` 且具有 `ContextWithFallback`，与项目既有生态完全吻合                                                                              | 保持 Gin + `validator/v10` 组合，剥离为独立 HTTP 插件即可                                                         |

## 已知未知

本调研涉及的 5 项核心方向均已拍板并转入 [0005 插件体系架构与 Adapter 嵌套模块模式设计](../design/0005-plugin-architecture-and-adapter-pattern.md)：

1. **Web 插件命名为 `plugins/web-gin`**（`package webgin`，含子包 `httpresult`），预留未来扩展其他 Web 框架的空间；
2. **协议结果输出包定名为 `httpresult` 与 `grpcresult`**：分别归属 `plugins/web-gin` 与 `plugins/grpc`，统一依据底座 `foundation/apperr` 决策报错显示内容；
3. **Cron 与 Async 采用最热门社区库复用策略**：`plugins/cron` 选用 `github.com/go-co-op/gocron/v2`；`plugins/async` 定位为 `Tool` 插件并选用 `github.com/panjf2000/ants/v2`，依靠逆拓扑关机排空；
4. **`plugins/database` 采用 `adapter/*` 嵌套独立 `go.mod` 模式**：主模块内聚 `dialect`、`models`、`gormlog`，具体驱动置于 `plugins/database/adapter/{mysql,postgres,sqlite,oracle,sqlserver}/` 且各自嵌套独立 `go.mod`；
5. **缓存与 Redis 内聚为单一 `plugins/cache` 模块**：主键保存插件实例，静态 `Provides` 声明资源键，`Exposer` 交付对应句柄；Memory 模式只有 `CacheManagerKey`，Redis 模式另含 `RedisClientKey`。
