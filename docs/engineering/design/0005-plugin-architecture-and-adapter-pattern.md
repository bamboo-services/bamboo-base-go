# v2.0.0 插件体系架构与 Adapter 嵌套模块模式设计

> 状态：draft · 日期：2026-10-19 · Scope：plugin
> 承接 [0003 插件体系分类与第三方库选型调研](../research/0003-plugin-ecosystem-and-libraries.md)、[0001 生命周期插件运行时设计](./0001-lifecycle-plugin-runtime.md)、[0002 Context 体系与共享组件传递设计](./0002-context-propagation-and-shared-state.md)

## 目标与边界

本次设计让 v1.x 的 `major` 在新工作树中退役，将网络协议、任务调度、并发池与存储客户端收归 `plugins/*`。当前仓库仍是 v1 实现，`foundation`、`application` 及新的插件生命周期都是待落地方案，不是已有能力。首批整体 Release 与各模块均为 `v2.0.0`；module/import 带小写 `/v2`，目录不新增 `v2/` 层。后续按需发版与清单映射详见 [0003](./0003-foundation-module-consolidation.md#异常与兼容)。

本设计稿旨在锁定整个 `plugins/*` 体系的**通用组织范式与核心契约**，避免各插件各自为政：

1. **确立“主插件模块 + `adapter/<实现>/go.mod` 嵌套子模块”的多后端扩展范式**：
   - 凡是单一引擎的插件（如 `plugins/web-gin`、`plugins/grpc`、`plugins/cron`、`plugins/async`、`plugins/email`），顶层一个 `go.mod`，内部按职责划分子包；
   - 凡是支持多种重型外部驱动的插件（以 **`plugins/database`** 为代表），在主插件模块内聚生命周期、公共模型与契约子包，而将具体驱动统一挂在 **`adapter/<driver>/`** 目录下，且**每个 `adapter/<driver>` 内部嵌套独立的 `go.mod`**，实现内部子包与外部驱动依赖的彻底隔离。
2. **锁定七大官方插件的生命周期模式与依赖拓扑**：
   - **`Worker` 模式**：`plugins/web-gin`（预留 `web-*` 扩展空间）、`plugins/grpc`、`plugins/cron`（底层复用 `gocron/v2`）；
   - **`Component` 模式**：由 `plugins/cron` 经内核 `Activator` 动态激发的定时任务实例；
   - **`Tool` 模式**：`plugins/async`（底层复用 `ants/v2`，靠逆拓扑在关机时先于 DB 排空）、`plugins/database`、`plugins/cache`（单一模块兼管 Memory/Redis 并通过 `Exposer` 同时暴露 `*cache.Manager` 与 `*redis.Client`）、`plugins/email`（底层复用 `wneessen/go-mail`）。
3. **协议结果输出归口至 `httpresult` 与 `grpcresult`**：
   - 彻底移除原 `major/result`（`xResult`），由 `plugins/web-gin/httpresult` 与 `plugins/grpc/grpcresult` 分别消费底座的 `apperr.Error` 与 `envelope.Response` 标准定义，各自决策 HTTP JSON 与 gRPC Status 的报错显示内容。

## 工作方式

| 当前工作树                                                              | 本次设计                                          |
| ----------------------------------------------------------------------- | ------------------------------------------------- |
| HTTP 能力散在 major/defined/common，数据库驱动独立在 plugins/database/* | 插件主模块收编通用职责，驱动移入 adapter 独立模块 |
| Async 裸开协程，Cron 直接调用业务函数                                   | Async 管池与排空，Cron 只触发内核 Component       |

### 1. 插件目录全景与 `adapter/` 嵌套 `go.mod` 模式

为什么在 `plugins/database` 里要把具体驱动收进 `adapter/*` 并各自嵌套 `go.mod`？

在 Go 语言的 Module 规则中，同一个模块下的普通子目录共享顶层 `go.mod` 的全部依赖；但只要某个子目录（如 `adapter/postgres/`）内部拥有自己独立的 `go.mod`，Go 工具链就会将其视为一个**独立的嵌套模块（Nested Module）**——主模块 `plugins/database/go.mod` 不会包含任何嵌套模块的依赖。

这样设计能同时拿到两个极大好处：

- **目录心智模型一目了然**：直接放在 `plugins/database/` 根下的 `dialect/`、`models/`、`gormlog/` 是主插件自带的内部子包（共享 `plugins/database/go.mod`）；而放在 `adapter/` 下的 `mysql/`、`postgres/`、`sqlite/`、`oracle/`、`sqlserver/` 则是按需选装的独立驱动模块。开发者再也不会分不清哪个目录是公共子包、哪个目录是独立发版的驱动。
- **重型驱动零污染**：业务项目如果只用 PostgreSQL，只需引入 `plugins/database` 与 `plugins/database/adapter/postgres`，编译产物里绝不会混入半行 MySQL、SQLite 或需要 CGO 环境才能编译的 Oracle 驱动。

完整的 `plugins/` 目录布局如下：

```text
plugins/
├── web-gin/                  # package webgin (Worker)
│   ├── go.mod
│   ├── plugin.go             # Start(Listen) / Run(Serve) / Stop(Shutdown)
│   ├── bind/                 # 泛型请求参数绑定 (原 major/utility/bind.go)
│   ├── header/               # HTTP Header 常量与 Bearer 提取 (原 defined/http)
│   ├── httpresult/           # 依据 apperr 决策 HTTP 状态码与 envelope.Response
│   ├── middleware/           # BridgeRequest / CORS / Panic / 响应兜底
│   └── validator/            # validator/v10 实例级标签与中文翻译器
├── grpc/                     # package grpcplugin (Worker)
│   ├── go.mod
│   ├── plugin.go             # Start(Listen+Health) / Run / Stop
│   ├── grpcresult/           # 依据 apperr 决策 gRPC codes.Code 与 Protobuf 响应
│   ├── interceptor/          # BridgeRequest 与错误转换 Unary/Stream 拦截器
│   └── generate/             # buf 生成的 Protobuf 基础契约
├── cron/                     # package cron (Worker + 子 Component)
│   ├── go.mod                # 复用 github.com/go-co-op/gocron/v2
│   └── plugin.go             # 到点调 Activator.Activate 激发子 Component
├── async/                    # package async (Tool)
│   ├── go.mod                # 复用 github.com/panjf2000/ants/v2 (零 Gin 依赖)
│   └── plugin.go             # Detach 上下文 + 有界池 + Stop 阶段排空
├── database/                 # package database (Tool)
│   ├── go.mod                # 仅依赖 foundation, application, gorm.io/gorm
│   ├── plugin.go             # Start 建池+Ping / Stop 关池 / Expose *gorm.DB
│   ├── dialect/              # package dialect — Adapter 接口与驱动注册表
│   ├── models/               # package models  — BaseEntity 与 PageResponse[T]
│   ├── gormlog/              # package gormlog — GORM slog 日志适配器
│   └── adapter/              # 扩展包目录：每个驱动拥有嵌套的独立 go.mod！
│       ├── mysql/            #   module .../plugins/database/adapter/mysql/v2
│       │   ├── go.mod        #     require gorm.io/driver/mysql
│       │   └── mysql.go
│       ├── postgres/         #   module .../plugins/database/adapter/postgres/v2
│       │   ├── go.mod        #     require gorm.io/driver/postgres
│       │   └── postgres.go
│       ├── sqlite/           #   module .../plugins/database/adapter/sqlite/v2
│       │   ├── go.mod        #     require github.com/libtnb/sqlite
│       │   └── sqlite.go
│       ├── oracle/           #   module .../plugins/database/adapter/oracle/v2
│       │   ├── go.mod        #     require github.com/oesand/gorm-oracle
│       │   └── oracle.go
│       └── sqlserver/        #   module .../plugins/database/adapter/sqlserver/v2
│           ├── go.mod        #     require gorm.io/driver/sqlserver
│           └── sqlserver.go
├── cache/                    # package cache (Tool)
│   ├── go.mod                # 内聚 Memory 分片复合存储与 go-redis/v9
│   ├── plugin.go             # Expose *cache.Manager 与 *redis.Client
│   ├── driver/               # KeyCache / HashCache / SetCache / ListCache
│   ├── memory/               # 16 分片复合结构内存后端 + janitor 协程
│   └── redis/                # Redis 后端实现与日志 Hook
└── email/                    # package email (Tool)
    ├── go.mod                      # 复用 github.com/wneessen/go-mail
    └── plugin.go                   # EmailPlugin: SMTP 客户端与模板渲染
```

上图画的是源码目录，`v2` 不作为新目录出现。每个模块的 go.mod 与跨模块 import 均带各自主版本后缀；例如父模块 `.../plugins/database/v2` 的 dialect 是 `.../plugins/database/v2/dialect`，驱动则是另一个模块 `.../plugins/database/adapter/postgres/v2`。首批 tag 仍按目录写为 `plugins/database/adapter/postgres/v2.0.0`，具体映射见 [0003](./0003-foundation-module-consolidation.md#源码目录不搬模块身份升级到-v2)。

### 2. `plugins/database` 与 `adapter/*` 的装配机制

`plugins/database` 支持两种装配方式，既能显式传 Adapter（最推荐、零全局状态），也能兼容从环境变量自动装配：

1. **显式注入 Adapter（推荐）**：
   每个 `adapter/<driver>` 导出一个实现 `dialect.Adapter` 接口的构造函数（如 `postgres.New(dsn)` 或 `postgres.FromEnv()`）。业务在注册插件时直接传入：
   `database.New(database.WithAdapter(postgres.FromEnv()))`。
   这种方式在编译期就能确认驱动已引入，且完全不经过任何全局 map。
2. **环境变量动态分发（兼容 `DATABASE_DRIVER`）**：
   每个 `adapter/<driver>` 同时保留 `init()` 向 `dialect.Register(driverName, factory)` 注册工厂的能力。当业务使用 `database.New(database.FromEnv())` 并匿名导入对应 `adapter/<driver>` 时，`DatabasePlugin.Start(startCtx)` 按 `env.DatabaseDriver` 选择已注册的适配器。这只保留 `DATABASE_DRIVER` 配置习惯，不保留旧 Go API 或入口。
3. **生命周期与 `Exposer` 暴露**：
   - `Start(startCtx)`：调用 `adapter.Open()` 获取 `gorm.Dialector`，挂载 `gormlog.New()`，执行 `gorm.Open`、设置连接池参数，并调用 `sqlDB.PingContext(startCtx)` 验证连通性；内核随后自动调用 `Expose()` 将 `ctxkey.DatabaseKey -> *gorm.DB` 写入 `SharedContainer`；
   - `Stop(cleanupCtx)`：调用 `sqlDB.Close()` 关闭底层连接池。

数据库的注册定义明确写 `Key: PluginKey, Provides: []ctxkey.Key{ctxkey.DatabaseKey}`。消费者声明 `Requires: []ctxkey.Key{ctxkey.DatabaseKey}`，内核在启动前就能找到数据库提供者；`Expose()` 只负责实际交付句柄，不再决定依赖图。

同样，CachePlugin 用独立主键保存插件实例。Memory 模式只声明并暴露 `CacheManagerKey`，Redis 模式声明并暴露 `CacheManagerKey` 与 `RedisClientKey`。模式在构造注册定义时确定，不能先承诺 Redis 能力、启动后却悄悄缺席。

### 3. Async 负责真退，逆拓扑负责保留依赖

将基于 `ants/v2` 的 `plugins/async` 定位为 **`Tool` 插件**（而非 `Worker`），是利用 `0001` 的 Kahn 逆拓扑分层关机实现优雅排空的关键设计：

```text
启动拓扑顺序 (自下而上)                                关机逆拓扑顺序 (自上而下)
────────────────────────                             ────────────────────────
Layer 0: [database]  [cache]  [email]                1. 先停 Layer 2: [web-gin] [grpc] [cron]
                 ▲                                      (入口关闭，不再产生任何新的 async.Submit)
                 │ Requires                                        │
Layer 1:      [async] (ants/v2 池)                                 ▼
                 ▲                                   2. 再停 Layer 1: [async].Stop(stopCtx)
                 │ Requires                             (拒收新任务，等待池中在途任务全部跑完)
Layer 2: [web-gin]   [grpc]   [cron]                               │
                                                                   ▼
                                                     3. 最后停 Layer 0: [database] [cache] [email]
                                                        (所有异步任务已写完库，安全关闭连接池)
```

- **提交期**：先检查 App 已进入运行期且尚未关闭新任务准入，启动期返回 ErrNotRunning，关机拒新返回 ErrShuttingDown；不能在 Start 中提交会并发读取尚未冻结容器的任务。再从容器取 Pool，在同一池准入锁内检查 closed、登记任务与计数，再提交给 ants。关闭准入与开始等待也使用这把锁，不能仅“读一次原子标记，再 Add”。底层提交失败时撤回登记、退还计数并取消 Context；返回错误的任务不应留下悬空句柄。
- **Context 与依赖**：Detach 剥离原请求取消与临时数据，同时接入当前 App 的 Grace 到期通知。保存每个任务的 Cancel，在真实返回后释放通知挂钩。取消不依赖 Async.Stop 已经开始：Handler 即使正等待异步任务，任务也能独立收到全局取消。Async 的 Requires 必须覆盖其任务会使用的 DB/Cache/Email 等共享资源，消费者则声明依赖 PoolKey；不能从完整容器视图推断闭包到底会访问什么。
- **排空期**：Stop 关准入后，用 application.GraceContext 等自然完成。Grace 到期的全局通知独立取消残留任务，Stop 仍继续等待函数真退；Stop Context 到总截止也不允许提前返回冒充停稳。真正归零后才进入 CleanupContext 清理池，确认自管 worker 协程退出再返回。

默认采用有界并发、非阻塞拒绝；若配置排队，队列也必须有界。ReleaseTimeout 的返回不能替代业务任务真实完成的计数，更不能强杀运行中的 Go 函数。总上限到期时 App 在外层报告 Async 卡在 Stop，保留它声明的依赖。

首版不保证断电后恢复任务，也不承诺“零丢失”。需要持久化交付应使用消息队列或任务系统，而不是只依靠内存池。

### 4. `plugins/cron`（`gocron/v2`）与内核 `Activator` 的协同

`plugins/cron` 底层采用社区活跃的 `github.com/go-co-op/gocron/v2`，但它**不让 `gocron` 自己裸跑业务函数**，而是将 `gocron` 作为精准的时间触发器，把任务实例的创建、防重入与生命周期全权交给内核 `Activator`：

1. 业务先把具体的定时任务注册为 `Component`（`app.RegisterComponent(jobInfo, jobFactory)`，默认 `OverlapSkip`）；
2. 再在 `cron.New(cron.WithCronJob("0 0 * * *", jobKey))` 中绑定调度规则与目标 `Component` 的 `ctxkey.Key`；显式采用 Queue 时，接纳的 tick 按同一定义 FIFO 等待，父 Cron 关闭会取消尚未派发项，不补跑关机积压；
3. `CronPlugin.Start(startCtx)` 初始化 `gocron.Scheduler` 并校验所有表达式；`Run(lifetimeCtx)` 启动调度器，每次到点触发时调用：
   `activator.Activate(lifetimeCtx, jobKey, application.WithPayload(tickTime))`；
4. `CronPlugin.Stop(stopCtx)` 调用 `scheduler.Shutdown()` 停止发出新 tick，而正在跑的子 `Component` 由内核等待其 `Run` 与 `Stop` 完整收尾后再关闭底层 `database`/`cache`。

### 5. 网络入口：实例校验与真实排空

Gin 插件持有自己的 validator 与 translator。框架 bind 包的 Bind/Data/Query/URI/Header 先通过不调用 Gin 全局校验器的解码路径获取数据，再显式调用当前插件的验证器，最后统一转换错误。实例绑定器通过插件的请求中间件挂入私有请求 Context，不存在包级可变绑定器。

不能先调用 ShouldBind*，再声称补一次实例校验就完成隔离：Gin 默认路径已经调用 binding.Validator，可能提前拒绝输入或执行别的 App 的规则。因此框架绑定不替换包级 binding.Validator，也不走会隐式调用它的默认绑定路径。业务直接使用 Gin.ShouldBind* 是另一条路径，不保证享有本插件的实例级规则。

HTTP/gRPC 适配器的 forceCtx、调用计数、Stream Context 包装与 Stop 顺序以 [0002](./0002-context-propagation-and-shared-state.md#3-插件注册后怎么一路传给四大-pluginrunner) 为准。HTTP 默认保证普通 Handler；WebSocket/hijacked 需独立受管 Worker，不用普通 HTTP 测试替代其验收。

### 6. `httpresult` 与 `grpcresult`：消费同一份 `apperr` 决策报错展示

底座 `foundation/apperr` 与 `foundation/envelope` 只定义纯数据结构；当业务返回 `*apperr.Error` 时，两个协议插件的结果包按各自协议规范做转换：

| 维度             | `plugins/web-gin/httpresult`                                                                    | `plugins/grpc/grpcresult`（配合拦截器）                                                                           |
| ---------------- | ----------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| **输入契约**     | `*gin.Context` + `*apperr.Error`（或成功 `data`）                                               | `context.Context` + `*apperr.Error`（或 Protobuf 响应）                                                           |
| **状态码决策**   | 取 `int(err.Code.Code / 100)` 作为 HTTP 状态码（如 `40014 → 400`、`50000 → 500`）               | 按 `err.Code.Code / 100` 映射为 gRPC 标准 `codes.Code`（如 `400xx → InvalidArgument`、`401xx → Unauthenticated`） |
| **响应体封装**   | 构造 `envelope.Response` JSON：自动填入 `ctxkey.GetRequestKey(ctx)` 与 `CalcOverheadTime(ctx)`  | 成功时向 Protobuf `BaseResponse` 注入 `Context` 与 `Overhead`；失败时将 `apperr` 元信息写入 `status.Status`       |
| **报错内容脱敏** | `Message` 始终展示；`ErrorMessage` 只使用业务显式提供的安全内容；Debug 也不直接暴露底层原始 Err | 同样仅附业务明确给出的安全详情，原始 Err 只记日志                                                                 |

## 数据与接口

错误码段映射必须有确定兜底，不让两个调用方自己猜：

| HTTP 分组        | gRPC Code                                     |
| ---------------- | --------------------------------------------- |
| 400              | InvalidArgument                               |
| 401 / 403 / 404  | Unauthenticated / PermissionDenied / NotFound |
| 408 / 504        | DeadlineExceeded                              |
| 409 / 412        | Aborted / FailedPrecondition                  |
| 429              | ResourceExhausted                             |
| 499              | Canceled                                      |
| 500 / 501        | Internal / Unimplemented                      |
| 502 / 503        | Unavailable                                   |
| 未列出的错误分组 | Unknown                                       |

非法 HTTP 分组统一输出 HTTP 500；未覆盖的合法错误分组按表降为 gRPC Unknown，原业务码仍通过受控详情保留。内部 Err 仅记日志，不因开启 Debug 就直接拼成 SQL/底层原始错误返回。需要附带 ErrorMessage 时仍由业务显式提供安全内容。

### 1. `plugins/database` 与 `adapter` 核心接口

```go
package dialect

import "gorm.io/gorm"

// Adapter 是具体数据库驱动扩展包（plugins/database/adapter/*）必须实现的契约。
type Adapter interface {
    // Name 返回驱动名称（如 "mysql", "postgres", "sqlite"）。
    Name() string
    // Open 构造 GORM Dialector（在 DatabasePlugin.Start 阶段调用）。
    Open() (gorm.Dialector, error)
}
```

```go
package database

import (
    "context"

    "github.com/bamboo-services/bamboo-base-go/application/v2"
    "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
    "github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
    "github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
    "gorm.io/gorm"
)

// PluginKey 是数据库插件自身在容器中的主标识。
const PluginKey ctxkey.Key = "plugin_database"

// New 返回主键为 PluginKey、Provides 为 [ctxkey.DatabaseKey] 的定义及工厂。
func New(opts ...Option) (application.PluginInfo, func() application.Plugin)

// WithAdapter 显式注入一个具体数据库驱动适配器（如 postgres.FromEnv()）。
func WithAdapter(adapter dialect.Adapter) Option

// DatabasePlugin 实现 application.Plugin 与 ctxkey.Exposer。
type DatabasePlugin struct {
    // 内部持有 *gorm.DB 与配置
}

func (p *DatabasePlugin) Start(ctx context.Context) error
func (p *DatabasePlugin) Stop(ctx context.Context) error

// Expose 在 Start 成功后将 *gorm.DB 暴露到 ctxkey.DatabaseKey。
func (p *DatabasePlugin) Expose() map[ctxkey.Key]any {
    return map[ctxkey.Key]any{
        ctxkey.DatabaseKey: p.db,
    }
}

// Get 从 ctx 中提取已绑定当前请求上下文的 *gorm.DB（内部执行 db.WithContext(ctx)）。
func Get(ctx context.Context) (*gorm.DB, *apperr.Error)
func MustGet(ctx context.Context) *gorm.DB
```

### 2. `plugins/async`（基于 `ants/v2`）核心接口

```go
package async

import (
    "context"

    "github.com/bamboo-services/bamboo-base-go/application/v2"
    "github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
)

const PluginKey ctxkey.Key = "plugin_async"
const PoolKey ctxkey.Key = "context_async_pool"

// New 返回 Async Tool 定义与工厂，主键为 PluginKey，Provides 包含 PoolKey。
// 使用哪些数据库、缓存或邮件资源，需在其 Requires 中明确声明，不能仅靠完整容器视图猜测。
func New(opts ...Option) (application.PluginInfo, func() application.Plugin)

// Submit 从 parentCtx 剥离超时信号（保留 SharedContainer 与 RequestKey），将 fn 提交至 ants 协程池执行。
// 若应用已进入关机排空阶段，返回 ErrPoolClosed。
// WithRequires 声明本池内任务使用的资源，构造 PluginInfo.Requires。
func WithRequires(keys ...ctxkey.Key) Option

func Submit(
    parentCtx context.Context,
    fn func(ctx context.Context),
    opts ...TaskOption,
) (*Task, error)
```

### 3. `plugins/web-gin/httpresult` 与 `plugins/grpc/grpcresult` 核心接口

```go
package httpresult

import (
    "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
    "github.com/gin-gonic/gin"
)

func Success(ctx *gin.Context, message string)
func SuccessWithData(ctx *gin.Context, message string, data any)
func Error(ctx *gin.Context, err *apperr.Error)
func AbortError(ctx *gin.Context, err *apperr.Error)
```

```go
package grpcresult

import (
    "context"

    "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
    "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/generate"
)

func Success(ctx context.Context, message string) *generate.BaseResponse
func SuccessWith[T any](ctx context.Context, message string) T
func ToStatusError(ctx context.Context, err *apperr.Error) error
```

## 异常与兼容

v2 不提供旧 `major` 或旧访问器兼容壳；业务迁移到新插件构造器、`application` 及领域访问器。旧项目继续固定已发布 v1 版本，两条使用路径不在同一套新入口中混搭。

1. **未注入或未导入数据库 `adapter/*` 的报错指引**：
   当 `DatabasePlugin.Start(startCtx)` 发现既未通过 `WithAdapter` 传入适配器、也未在 `dialect` 注册表中找到对应驱动时，直接返回明确错误：`数据库驱动 [postgres] 未加载，请传入 WithAdapter(postgres.FromEnv()) 或匿名导入 plugins/database/adapter/postgres/v2`。
2. **关机期间调用 `async.Submit` 的安全拒绝**：
   当上层自定义协程在 `AsyncPlugin.Stop` 开始后仍尝试调用 `async.Submit` 时，直接返回 `ErrPoolClosed`，绝不向正在关闭的 `ants.Pool` 强塞任务引发 panic。
3. **非 `*apperr.Error` 未知错误的兜底转换**：
   `web-gin` 的统一响应兜底中间件与 `grpc` 的 `ResponseBuilder` 拦截器捕获到非 `*apperr.Error` 的普通 `error` 时，自动包装为 `apperr.ServerInternalError`（50000），在日志中记录完整原始错误，而对客户端仅展示安全的标准错误文案。

## 备选与取舍

每种数据库驱动都放主模块里，目录更简单，但主模块 tidy 会把这些驱动一并纳入依赖图。嵌套模块让安装边界与目录边界一致，代价是模块数量和发布映射更复杂；它不能替代 GOWORK=off 的独立消费验证。

Cron 仅做表达式触发时，robfig/cron 已经足够，且范围更小。继续选择 gocron/v2 是为了复用间隔、一次性等调度入口，不把它的防重入机制与内核 Overlap 再叠一套。Async 的 ants 接口较低层，conc/pond 对业务错误与 Context 更直接；选择 ants 不免除我们自己补任务计数、取消和排空的责任。库选型对比见 [research/0003](../research/0003-plugin-ecosystem-and-libraries.md)。

内存 Cache 保留 Hash/Set/List 是为了维持业务 API，不是说纯 KV 库不能服务普通缓存。若未来需要 Redis 服务端级别的原子性、脚本或分布式一致性，Memory 不冒充等价后端，应收窄接口承诺。若 Async 为持久任务补出重试日志或跨进程租约，也应退出内存池范围，重新比较成熟任务系统。

## 验证

插件重构尚未实现。落地后各模块先跑 `GOWORK=off go test -race ./...`，再覆盖这些跨模块场景：

1. 数据库主模块 `GOWORK=off go list -m all` 不包含具体驱动；独立示例只导入一个 Adapter，`GOWORK=off go list -deps ./...` 不包含未选驱动。不要用整个 Workspace 的模块图代替这项检查。
2. Memory 的 Provides 只有 CacheManagerKey，Redis 模式另含 RedisClientKey；缺少 Redis 能力时在工厂前失败。Stop 真正关闭 janitor/连接池后条目退役。
3. 启动期/关机拒新、Submit 与 Stop 并发、底层提交失败、池满拒绝、任务 panic 都不泄漏计数；Grace 到期任务收到取消，忽略取消的任务仍令 Stop 等待并保留 DB。释放测试屏障后才允许关闭 DB。另测 Handler 等 Task.Wait、任务等取消而 Async.Stop 尚未开始，确认 Grace 通知独立到达，不形成关闭顺序死锁。
4. 用 gocron 的间隔调度接口配置 50ms 周期，激活耗时 120ms 的 Component，验证 OverlapSkip 与独立追踪 ID；显式 Queue 验证 FIFO、队满、Cron 关闭取消未派发 tick。不要把五段 Cron 表达式误当成毫秒调度接口。嵌套子树也须完整收拢。
5. 两个 Gin 插件为同名标签配置相反规则，框架 Bind 分别按各自实例校验；不得修改 Gin 全局校验器，也不得在解码时隐式调用它。
6. 真实 HTTP/gRPC 取消与排空按 0002 验证，包括 Serve 已退但 Handler 未退；WebSocket/hijack 不计入默认保证。
7. 同一份 40014 BodyError 与 50000 ServerInternalError 经两个结果包得到确定映射；覆盖映射表兜底、非法 HTTP 码与内部 Err 不外泄。SuccessWith 的 T 必须是符合 BaseResponse 约束的 Protobuf 指针，误用时明确报编程错误，不能对 any 作无条件成功承诺。

## 待确认

插件归属、真实 Stop、实例级绑定和默认网络边界已经确认。版本与 tag 统一按 [0003](./0003-foundation-module-consolidation.md#异常与兼容)：目录不加 v2 层，module/import 带 /v2，首批全模块 v2.0.0，后续按需发版。无剩余阻塞性方案选择，编码与真实发布验收不作为本轮已完成能力。
