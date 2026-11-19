# v2.0.0 强类型错误返回迁移计划

> 状态：in-progress · 日期：2026-11-09 · Scope：foundation
> 承接：[强类型错误返回与协议适配规范](../engineering/spec/0003-foundation-strong-error-contract.md) · [错误码受控词表](../engineering/spec/0002-foundation-error-code-vocabulary.md)

## 目标与范围

从基线 `aaf73c684d2d108c0916bba1733ef8c608024831` 迁移到：SDK自有公开及内部失败结果的静态类型均为 `*apperr.Error`；固定外部方法集保留薄适配；HTTP/gRPC转换前已有明确的Code/Output，不能把通用500当作统一完成。

**2026-11-09用户已明确授权P1开工，并要求Leader派发minimax-m3.1-flash/max子代理。P0技术台账据此准出；P1实现及Leader准出检查已完成，只覆盖错误核心、19项新增码和纯分类/编译契约测试。P2实现及Leader准出检查已完成并提交为 `fad7597c62496ea5c0e7f280227bb6a2de977782`（未推送）：Worker丢链修复、typed/ctxkey/env、snowflake/toolkit、logging强类型迁移及7项新码登记（词表84→91项）。P3实现及Leader准出检查已完成：application内核契约/账本与全部插件实现者（含examples与test消费者）一次原子切换为*apperr.Error，零新增码，全量门禁通过，已提交为 `c8abf577`（未推送）。2026-11-12用户批准P4实际实施：已按已确认语义追加50033–50037（词表91→96项），公共契约、五适配器与模型迁移及Leader准出已完成；P4已提交为 `e4930971`（未推送）。2026-11-15用户确认主推提案并批准P5实施：已集中登记42903与50038–50041（词表96→101项），cache/email/cron/async及仓内调用方迁移与Leader准出已提交为 `c5efbc3`（未推送）。随后用户批准P6实施：HTTP/gRPC协议出口收敛与R12原生信封补齐（零新增码）已提交为 `7e456a7`（未推送）。2026-11-18至2026-11-19按用户授权完成P7类型契约守卫（独立 `error-contract` 模块 + 3处内部助函数清零）、全量静态/竞态/消费者门禁、6项发布离线演练及MySQL/PostgreSQL/Redis/Mailpit四容器真引擎集成验证，已本地提交为 `3db459a`（未推送）；公开RC与正式发布仍待另行授权。**不自动创建工作卡、推送或发布，不以本地门禁代替公开RC/正式发布验收。

包含14模块、自有接口和函数值、结果账本、错误聚合、插件实例、测试替身、示例及独立消费者。排除第三方实现、生成的 `.pb.go` 手工修改、工具脚本的错误风格改造，以及未获单独授权的GORM/Redis完整新门面。

### 现状与待实施目标

| 项目       | 已核实的基线                                        | 尚未实施的交付                        |
| ---------- | --------------------------------------------------- | ------------------------------------- |
| 错误目录   | 65项Code/Output统一，词表门禁存在                   | 按新增语义评审追加，保持旧编号不重排  |
| 公开失败   | 22个确定性反例仍无统一错误结构                      | 强类型签名、统一实例与正确错误码      |
| 原结构丢失 | `application/app.go:948` 用 `%v` 丢掉Worker返回错误 | 保留原始错误和全部子原因              |
| 协议适配   | 11类本地探针验证既有保留/兜底路径                   | 取消/超时及信封策略确认后的回归       |
| 标准接口   | JSON/XML/SQL/日志/网络/gRPC存在固定方法集           | 精确白名单、强类型核心及nil安全薄适配 |
| 质量门禁   | 现有错误码检查只覆盖目录                            | 独立的错误签名及例外契约门禁          |

## 前置条件与风险

1. spec0003原列方向性待确认已全部裁决：独立control/valueerr及分组、原生gRPC补齐信封、首发前集中同步已知消费者、RC验证后14模块首版v2.0.0。用户于2026-11-09明确授权P1实施，计划进入in-progress；规范仍为draft，不因实现获准而自动冻结为ADR。
2. 保留 `apperr → 标准库` 的基础依赖；P1新增依赖为 `control/valueerr → apperr`。后续 `typed/ctxkey/env/toolkit → valueerr`（按实际需要）仍待P2；ctxkey/env可继续委托typed。P1检查无环，其他基础包不得借协议插件取码。
3. 500组已有18项，SDK剩余32个槽位；新增语义必须先核算各状态组容量。超出容量须重评编号规范，不用万能500或业务号段掩盖。
4. Go接口方法和函数类型没有返回类型协变。改 `Plugin.Start/Stop` 会同时影响application、所有插件、builtin、测试替身、示例及下游实现者；中间补丁可在本地推进，但禁止交付不编译的混合版本。
5. 强类型nil跨 `error` 会成为非nil接口。GORM钩子/事务、JSON/XML、日志、gRPC以及错误聚合均需成功路径测试，不能只验证错误路径。
6. 已确认控制取消/超时采用499/408/504而非500。现有gRPC先匹配 `apperr.Error`，仍须验证强类型归一后保持Canceled/DeadlineExceeded；原生根因与正常停服分支不能因包装被误改。
7. 已暴露的原生数据库/Redis对象不在签名改造后自动获得新语义；新增门面属于另行确认的扩展，不能暗中扩大任务。

## 已确认的控制与值错误映射

以下为2026-11-07用户逐项裁决，不是当前已交付实现；精确边界见 [spec0003 R11](../engineering/spec/0003-foundation-strong-error-contract.md)。

| 责任/场景                  | 规划归属                                           | Code分组 | HTTP / gRPC            |
| -------------------------- | -------------------------------------------------- | -------- | ---------------------- |
| 执行上下文取消             | `foundation/control` / `CONTROL_CANCELED`          | `499xx`  | 499 / Canceled         |
| 请求接收或读取超时         | `foundation/control` / `CONTROL_REQUEST_TIMEOUT`   | `408xx`  | 408 / DeadlineExceeded |
| 内部操作deadline到期       | `foundation/control` / `CONTROL_DEADLINE_EXCEEDED` | `504xx`  | 504 / DeadlineExceeded |
| 值格式/范围/枚举等输入非法 | `foundation/valueerr` / `VALUE_*`                  | `400xx`  | 400 / InvalidArgument  |
| 不支持的目标或非法API契约  | `foundation/valueerr` / `VALUE_*`                  | `500xx`  | 500 / Internal         |
| 服务自身配置入口失败       | 所属框架/插件配置码，保留值错误原因                | `500xx`  | 500 / Internal         |

499是非标准HTTP扩展，断连不保证能写回；内部deadline用504是项目约定，不宣称它必然是网关。值转换底层不猜调用来源，配置入口按责任重新包装，禁止每层随意改码；不采用422方案。两个新包属于现有foundation模块，不新增模块；P1按spec0002登记19项追加，既有65项不变；真实消费者迁移仍按后续阶段验收。

## 已确认的原生gRPC信封策略

2026-11-08用户选择“全部补齐，保留原生语义”，详见 [spec0003 R12](../engineering/spec/0003-foundation-strong-error-contract.md)。SDK统一错误继续按自身码转换；原生context取消/超时补R11控制信封；合法非OK原生status缺BaseResponse时只追加一个兼容信封，保留RPC状态、显式Message与所有原Details。已有BaseResponse不覆盖、不再追加，不因追踪ID不同重写已有载荷。

P1需完成原生标准状态到兼容信封码的分类表，P6按该表实施和测试；不得猜测不存在的业务领域码，也不得把兼容业务Code/100再映射后覆盖原生Aborted/DataLoss等状态。原始Any须保留，含重复/冲突/不可解码信封的输入不擅自修复；未知包装层文本不抄到对外文案。编码/传输故障有明确降级诊断，不能承诺在物理故障下信封总能送达。

策略已选不代表原生分支已经修改；P1仅建立分类原语及预置码，真实normalizeRPCError和拦截器接入仍属P6。

## 已确认的下游窗口与RC首发策略

2026-11-08用户选择 [spec0003 R13](../engineering/spec/0003-foundation-strong-error-contract.md)：全部已知V2试用消费者在首个稳定V2正式发布前集中迁移；先公开 `v2.0.0-rc.N` 验证，再发布首批14模块 `v2.0.0` 与完整精确Manifest，继续 `/v2` 路径。按验收条件而非固定天数放行。

本轮只读查询远端全部tag及GitHub Release，未观察到V2标签或V2 Release；可见Release仍为V1。这是本轮时点事实，不证明不存在仓外分支/伪版本消费者；发布前重新核查。P0登记受控消费者、负责人、当前依赖/目标组合及自定义实现清单，仓库内示例不能代替仓外业务验收。V1业务不强迫同日升级；旧V2候选使用者先锁定旧完整组合，再在集中窗口内迁移，不混搭新旧方法集。

P7先经隔离代理验证候选，再在每次另获授权后发布公开RC。RC内容或依赖组合变化时递增候选身份，已公开tag不移动/覆盖；最终RC需完成已知消费者验收。正式稳定候选若改写RC依赖为稳定依赖，必须重新验证实际发布内容及MVS有效模块图，不能把它当作与RC字节不变的重命名。正式版本公开可取后还需独立验证，代理延迟不虚报通过。

正式V2后遵守patch兼容修复/minor兼容新增/破坏性API变更新主版本路径。当前发布工具限定V2且Action会改依赖、打tag并推送，未来新主版本或后续按需发版须另行适配/验收；本次首发统一14模块不自动扩大为永远全量重打tag的决定。

现阶段未创建任何RC或正式tag，未改发布依赖与Release CI，未证明发布链路已可安全执行。P1只扩展错误目录守卫，不将其作为发布验收；全部发布与消费者迁移实施项仍待后续阶段。

## 全库迁移清单

下表是模块级必查范围，不以方法名猜测例外。P0已完成[517条逐符号类型分类、固定签名展开与消费者模板](../engineering/research/0005-foundation-strong-error-p0-inventory.md)，并附[2277处引用和133条接口实现关系](../engineering/research/0006-foundation-strong-error-p0-callers.md)。附录词法候选仅保留定位历史，不再作为完整台账；记录数包含哨兵、继承投影和函数值载体，不是待改函数数量。

| 模块                                 | 自有强类型迁移对象                                                                                                                | 固定签名/特别边界                                                                                                       |
| ------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| `foundation`                         | typed解析、snowflake构造/生成、logging构造与内部写入、toolkit密码/时间、标准编解码的内部核心                                      | apperr.New已强类型；Unwrap、编解码/SQL/flag、slog.Handler、io.Writer/Closer保留薄适配；envelope无错误返回，不强加返回值 |
| `application`                        | Plugin/RunnablePlugin/Quiescer、RunHandle/Activator、Register/Run/Shutdown、配置/拓扑/容器/监督辅助、RunResult与关机账本、builtin | bridgedContext.Err、标准Unwrap不改；诊断类型作为统一错误的可遍历原因                                                    |
| `plugins/database`                   | Start/Stop、dialect.Adapter.Open/Lookup、自有注册校验、模型ID生成                                                                 | GORM BeforeCreate/BeforeUpdate保留error；gormlog.Trace的error只是参数，不列为失败返回                                   |
| `plugins/cache`                      | driver的Codec与Key/Hash/Set/List接口、memory/redis实现、插件生命周期、序列化错误出口                                              | Codec是本库接口，不是外部强制；键不存在保持非失败，原生Redis客户端属于逃生口                                            |
| `plugins/email`                      | Start/Stop、Client.Send/SendHTML/SendTemplate、Message.AttachFile、TemplateManager及辅助函数                                      | watchedConn.Close与外部拨号回调保留error；nil适配及SMTP原因保留                                                         |
| `plugins/web-gin`                    | 生命周期、Binder.New、Header.GetAuthorization、Engine.New/Validate/RegisterValidation、绑定解析辅助、自有validator注册回调        | Gin handler不返回error；第三方translator回调、HTTP最终输出与Recovery属于外部边界                                        |
| `plugins/grpc`                       | 生命周期、gate等待、自有错误归一核心；utility已强类型也需复验                                                                     | gRPC handler/interceptor、ToStatusError输出保留原生签名；状态/Details需验证                                             |
| `plugins/cron`                       | Start/Run/Quiesce/Stop与私有调度配置/关闭辅助                                                                                     | 第三方调度器结果在边界归一；不是因实现Worker而保留error                                                                 |
| `plugins/async`                      | Pool.Submit、Plugin/包级Submit、Start/Stop、Task.Wait/WaitFor、私有停机及准入                                                     | context原始原因保留；用户任务当前无error结果，不为本计划额外增加任务失败API                                             |
| `plugins/database/adapter/mysql`     | Open、DSN自有校验及延迟错误字段                                                                                                   | Adapter为本库接口，不能以返回gorm.Dialector为由豁免错误位置                                                             |
| `plugins/database/adapter/postgres`  | Open、DSN自有校验及延迟错误字段                                                                                                   | 同步database消费者与集成夹具                                                                                            |
| `plugins/database/adapter/sqlite`    | Open与空DSN延迟错误                                                                                                               | 保持纯Go驱动，独立消费者验证                                                                                            |
| `plugins/database/adapter/sqlserver` | Open、DSN自有校验及延迟错误字段                                                                                                   | 真实服务验收与本地配置拒绝测试分开                                                                                      |
| `plugins/database/adapter/oracle`    | Open、DSN自有校验及延迟错误字段                                                                                                   | 不把配置拒绝测试当作真实Oracle连接验收                                                                                  |

同时检查 `examples/v2/`、`test/consumer/`、`test/integration/` 和14模块所有 `_test.go` 中实现自有接口的类型；生成文件不手改。非生产故障注入函数可维持原生输入，但不能让消费者样例继续实现旧接口。

## 分阶段计划

### P0：形成逐符号分类和固定签名台账

- 动作：以 `.github/scripts/manifest/v2-modules.txt` 为14模块权威清单，枚举生产函数/方法结果、接口方法、匿名回调、函数类型、错误字段和错误通道。分为自有强类型、固定外部签名、原生原因/输入、无失败返回四类。
- 交付：符号到所属域/阶段/例外理由/调用方的台账；按spec W01–W12逐符号展开。记录完整签名、外部接口与验证方式，不以目录或方法名整体豁免。过期白名单也必须失败。
- 准出：14模块逐一有记录；spec待确认项有用户裁决；候选文件没有未归类符号。现有模块图和 `go list -deps ./foundation/...` 无反向依赖。
- 状态：技术核对与模板已交付，2026-11-09用户明确授权P1实施，P0据此准出。基线类型加载覆盖39生产包、287源文件、517契约、2277精确引用和133接口关系。已知消费者为仓内C01–C06；候选模块图、动态nil与真实控制语义仍按对应阶段验收，不把开工授权当作这些回归已通过。
- 成果：[逐符号/例外/消费者模板](../engineering/research/0005-foundation-strong-error-p0-inventory.md)与[仓内调用方/实现者](../engineering/research/0006-foundation-strong-error-p0-callers.md)。P0记录是基线只读证据；本轮仅按后述P1范围实施，不回写历史类型快照为新代码现状。

### P1：先稳定错误核心、码映射和负向编译测试

- 动作：确认统一错误构造/转接的nil语义；控制和值错误直接按spec0003 R11已选方向执行，不重新立题。细化具体符号与全局序号，确定其他新增错误、配置重分类和聚合主码；保留原始cause及诊断类型，不把未知错误全归一个500。
- 交付：按spec0002追加的control/valueerr及其他已批准定义、词表与守卫归属；全部合法非OK标准gRPC状态到兼容信封码/HTTP语义的分类登记表；typed-nil/Is/As/聚合和负向编译测试。原生status本身保留权威RPC Code，不经兼容码重新正向映射；不得从状态反推领域原因。
- 准出：`go test ./foundation/apperr -count=1`、`make error-codes-check`；新增control/valueerr包完整测试与无环检查；负向编译夹具确定失败；成功nil不变接口错误，已选分组/责任分类通过协议回归，具体序号与容量完成登记核验。
- 状态：2026-11-09实现及Leader技术准出完成，待用户审阅；追加19项，保留旧65项。P1通过不自动启动P2，整计划仍为in-progress。

#### P1开工准备包（2026-11-09，已获实施授权）

用户本轮选择先完成开工准备，并确认目前没有仓外V2分支、伪版本或本地replace消费者。已知受控范围为仓内C01–C06，名单沿用[P0消费者台账](../engineering/research/0005-foundation-strong-error-p0-inventory.md)。该范围确认最初仅解除“仓外未知”的等待；同日用户另行明确授权P1实施和Leader派发子代理，计划据此进入in-progress，不等于仓内候选验收已通过。发现新增消费者时补登记并重评影响。

**本轮批准的实施边界：**只做P1错误核心、19项已批准定义和分类契约测试。19项已批准码/Message与16种原生RPC兼容分类见[spec0002开工候选码](../engineering/spec/0002-foundation-error-code-vocabulary.md#p1登记记录2026-11-09)及[原生RPC兼容提案](../engineering/spec/0002-foundation-error-code-vocabulary.md#原生grpc缺信封时的p1兼容分类)。开工已复核active/retired无冲突，按E04追加登记，禁止挤掉既有号码。

#### P1公开原语契约

- `apperr.IsNil(err error) bool`：只识别nil与typed-nil原因，不调用Error方法。
- `apperr.Find(cause error) *Error`：只提取已有非nil统一错误；遵循节点类型、标准As、子原因的深度优先顺序，跳过nil投影，不以固定深度截断合法链。
- `apperr.From(code *ErrorCode, msg ErrMessage, cause error) *Error`：无原因返回nil，已有统一错误保留Code/安全描述/Data与整条根因链；未知原因按显式所属码包装，不把原始文本复制为安全描述。
- `apperr.Join(code *ErrorCode, msg ErrMessage, errs ...*Error) *Error`：零项nil、单项原对象、多项由责任层显式选总码，全部原因保持输入顺序；不自动应用框架总码。`New`仍始终非nil，原生原因协议不改。
- `control.Classify(cause error, phase Phase) *apperr.Error`：只识别已有统一错误或真实取消/超时。nil也可能表示未知普通原因，调用方必须用所属域apperr.From兜底，不能把未识别当成功；`PhaseOperation`默认504，`PhaseRequestRead`明确读取408。
- `valueerr.From(kind Kind, msg apperr.ErrMessage, cause error) *apperr.Error`：按显式值类别区分400/500，已有统一错误不重分类；服务自身配置需所属域apperr.New显式重包。
- `grpcresult.ClassifyStatus(st *status.Status, phase control.Phase) *apperr.Error`：只取16态兼容分类并保存原status根因，不修改原状态/文案/Details；不能将分类结果经ToStatusError回转后替代原status，真实缺信封追加仍留P6。

| 顺序 | 允许修改的位置与动作                                                                                            | 交付与停止边界                                                                                                   |
| ---- | --------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| P1.1 | spec0002、各归属error_code.go、现有check-error-codes.go及其测试：按批准提案追加定义和control/valueerr归属白名单 | 保留旧65项；新域在原foundation模块内，不增模块，不改go.mod/go.work或增加运行时依赖                               |
| P1.2 | foundation/apperr：补typed-nil过滤、nil安全转接与强类型聚合辅助；保留New、Err和Unwrap原生原因协议               | New仍是显式非nil构造器，不能因cause为空改为返回nil；成功返回nil由转接/聚合路径独立保证，禁止用空错误冒充成功     |
| P1.3 | 新foundation/control、foundation/valueerr：建立控制原因和值校验的强类型核心与分类测试                           | 取消/超时只依据真实cause及明确阶段；值源非法400与目标/接收器契约非法500分开；不迁移现有typed/logging等全部调用点 |
| P1.4 | plugins/grpc/grpcresult及协议测试：追加已批RPC码，建立原生状态到兼容信封的纯分类契约与表驱动测试                | Aborted/OutOfRange/DataLoss保留原状态；不修改plugins/grpc/plugin.go归一链或拦截器，实际补Details接入留P6         |
| P1.5 | foundation/apperr内持久正反编译夹具与测试；受影响包协议/错误链测试                                              | 验证原生error不能进入自有强类型结果/字段/回调、标准薄适配确能返回真实nil；不宣称P7全仓类型守卫已存在             |

**不包含：**修复Worker丢链、切换Plugin/RunHandle/Activator及全部实现者、重写RunResult/停机账本、迁移五驱动或全部插件、实现HTTP/RPC完整接入、接入P7全局类型门禁、RC或稳定发布。现有Worker缺陷仍明确留P2；不能用“P1完成”替代整计划完成。后续可按依赖连续推进，但每次越过本轮批准范围须有用户指示。

#### P1必须落到持久测试的行为

| 场景                                                        | 预期断言                                                                                                                          |
| ----------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| 原生nil、装入error接口的nil统一错误指针、nil第三方指针原因  | 薄适配返回的error必须真实等于nil；根因集合/通道/聚合不残留typed-nil；不调用nil原因的Error方法                                     |
| New只收到nil原因                                            | 显式New仍返回携带所选Code的非nil对象；不能把New当成功转接器                                                                       |
| 单个自有失败、多个独立失败                                  | 单项保留原码；多项由责任层显式选总码，保留全部cause的Is/As；错误顺序不依赖goroutine完成先后                                       |
| Factory失败/未派发取消、Wait等待者取消、组件终态失败        | 先保留现有优先级与“等待取消不等于组件终态”语义；P1只验证聚合原语，不提前接管P3结果选择逻辑                                        |
| HookPanic/ShutdownIncomplete/WorkerUnexpectedExit的规划包装 | 原诊断对象可As；所有成员cause可遍历；ShutdownIncomplete含成员Err不能仅保留可As的外壳；实现接入与异步终态验收留P2/P3               |
| control及valueerr分类                                       | 499/408/504与400/500逐项断言；已有结构化错误不能被深层context原因随意重分类；服务自身配置由所属500码再包并保留值原因              |
| 19项新码经过现有HTTP/RPC统一错误出口                        | Code/Output/Message及Code分组不漂移，FRAMEWORK仍500/Internal；该正向映射不同于原生状态保真                                        |
| 16种原生非OK状态的兼容分类                                  | 表完整且无重复；尤其Aborted≠AlreadyExists、OutOfRange≠InvalidArgument、DataLoss≠Internal；ResourceExhausted不等于请求过多         |
| 外部标准签名与自有结果正反例                                | 固定Unwrap/error链及外部方法集保留；自有结果返回errors.New、字段/回调装原生error的负例必须编译失败，对应*apperr.Error正例必须通过 |

聚合辅助不自动把所有域包装成FrameworkMultipleFailures；单个失败外层已有明确责任码时保留，多个独立框架阶段才选框架聚合码。HookPanic/ShutdownIncomplete/WorkerUnexpectedExit等明确场景优先用自己的场景码并保全其余cause；Factory及未派发取消的既有调度优先级不因通用聚合辅助改变。日志双路失败使用日志所属码，缓存关闭失败使用缓存所属码，不为借用总码引入跨插件依赖。公开原语的精确签名由持久`positive_api`与RPC包编译断言约束；真实Join拒绝原生成员另有`negative_join`反例，防止只验证虚构的强类型样例而遗漏实际API。

#### P1准出命令（实施完成后记录实际结果）

- `go test ./foundation/apperr/... ./foundation/control/... ./foundation/valueerr/... -count=1`：包含持久负向编译测试驱动；负例失败必须由预期类型位置引发，不把依赖下载失败当负例成功。
- `go test -race ./foundation/apperr/... ./foundation/control/... ./foundation/valueerr/... -count=1`：核心并发与nil/根因回归。
- `go test -race ./plugins/web-gin/httpresult/... ./plugins/grpc/grpcresult/... -count=1`：19项码的正向出口及RPC分类契约；P6再测实际拦截器全链。
- `make error-codes-check`、`make test-scripts`：定义、归属、全库唯一/连续追加、守卫正反例通过。
- `GOTOOLCHAIN=go1.25.11 GOFLAGS=-mod=readonly go list -deps ./foundation/...`：逐项确认无application/plugins反向依赖；不是只凭命令退出0判无反向依赖。
- `make test`、`make vet`、`make lint-go`、`make lint-docs`、`git diff --check`：覆盖现有14模块及新增包、文档与格式门禁，实际结果按准出记录，不沿用开工前历史结果。

每项保存准确命令、SHA及dirty状态；未跑/跳过与通过分开。P1夹具只证明已测试API和语言契约，不覆盖全部517条记录；全仓类型守卫仍属P7。P1通过后先交用户审阅，不自动启动P2、提交推送、创建卡或发布。若准出失败，停在P1保留证据并修复，不为了变绿顺带重写生命周期或放宽例外。不得用清理工作树的方式删掉本次既有文档。

#### P1准出记录（2026-11-09）

实现基线为 `9a2efca4ad13544f8bd7c9634eccc8b6c3899626`，本轮源码与测试均在该提交的未提交工作树上验证（dirty=true），没有新提交、推送或发布。3个限定实现子任务与1个只读复核均按用户要求使用 `minimax-m3.1-flash`、max；Leader实际复读差异并执行下列检查，不直接将子代理总结视为验收。

- 旧65项六列保持原值，追加19项后共84项；新域仍在既有foundation模块内，无go.mod/go.work/go.sum或生成pb修改。
- 交付 `IsNil/Find/From/Join`、control/valueerr及16态原生RPC纯分类；标准As短路、深层包装、动态不可比较值、nil投影与全兄弟cause已补持久回归。无cause失败用New的示例已编译执行。
- 4个正向/4个负向持久编译夹具覆盖实际公开API签名、真实Join成员和返回/字段/回调位置；先验证环境可编译正例，再要求反例命中具体类型拒绝。它们不是P7全库类型扫描。
- 19项新码两端出口分别验证状态、三元组和脱敏；RPC纯分类保存原status/文案/Any顺序与字节，不伪造context cause，不接入真实拦截器。

下表记录本轮实际执行范围；所有Go检查使用 `GOTOOLCHAIN=go1.25.11`。完整运行日志与SHA/dirty证据同步保存在卡片workpad，临时日志目录为 `/tmp/opencode/v2-p1/`；文档/范围检查在写回记录后再复验，以workpad最后一轮为准。

| 验证范围                 | 执行命令                                                                                                                    | 实际结果                                           |
| ------------------------ | --------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------- |
| 错误核心与编译夹具       | `go test ./foundation/apperr/... ./foundation/control/... ./foundation/valueerr/... -count=1`，以及相同范围 `go test -race` | 通过                                               |
| HTTP/RPC新增码与原生分类 | `go test -race ./plugins/web-gin/httpresult/... ./plugins/grpc/grpcresult/... -count=1`                                     | 通过                                               |
| 14模块普通/竞态          | `make test`、`make race`                                                                                                    | 通过                                               |
| 14模块静态与Go规范       | `make vet`、`make lint-go`                                                                                                  | 通过                                               |
| 词表与脚本自身           | `make error-codes-check`、`make test-scripts`                                                                               | 通过                                               |
| 示例与独立消费           | `make check-examples`、`bash .github/scripts/consumer/check-v2-consumers.sh`                                                | 通过；不能等同全部自定义业务/真实外部环境验收      |
| 文档                     | `make lint-docs`                                                                                                            | 通过；记录写回后复验                               |
| foundation依赖           | `GOTOOLCHAIN=go1.25.11 GOFLAGS=-mod=readonly go list -deps ./foundation/...`                                                | 通过，并显式检查无application/plugins反向依赖      |
| P1补丁范围与链接         | `python3 /tmp/opencode/v2-p1/verify.py`                                                                                     | 通过；旧65项与P0历史快照未漂移，最终记录写回后复验 |
| 差异与Go格式             | `git diff --check && git status --short`、已修改/新增Go文件的 `gofmt -l`                                                    | 通过，格式输出为空                                 |

真实Redis/SMTP/专属数据库环境集成仍由测试开关默认跳过，不因本轮单测、SQLite或独立消费通过而标为真实服务验收。Worker丢链及其真实诊断对象/终态、生命周期强类型切换、typed/logging等调用点、原生RPC实际补Details和P7完整类型守卫仍留P2–P7。计划保持in-progress、规范保持draft；P1技术准出待用户审阅，不将长期细节优化整卡或整计划标完成。

### P2：修复Worker丢链并迁移foundation强类型核心

- 动作：先把 `application/app.go:948` 的Worker原错误丢失变成持久回归，再按规范修复保留cause；迁移typed解析、snowflake、logging、toolkit自有出口。编解码、SQL、flag、slog、io方法用强类型内部核心及nil安全薄适配。
- 交付：Worker返回自定义错误后，`errors.As`、`errors.Is(original)`、`errors.Is(ErrWorkerUnexpectedExit)`均保留的测试；标准接口编译断言；成功编码、日志写入和关闭的真实nil测试。
- 准出：`go test ./foundation/... ./application/... -count=1` 及相同范围 `-race`；`go list -deps ./foundation/...`无application/plugins依赖。此阶段可用错误链先行修复，但不能以此宣称整个application已经强类型化。
- 状态：2026-11-09实现及Leader准出完成，待用户审阅；Worker丢链已修复，词表追加7项至91项。P3及以后未执行。

#### P2开工准备包（2026-11-09）

P1已提交为 `7c4f7fed106ad36d6127e5f19d212b7bbc12897c`（`feat: 实现 V2 强类型错误核心与协议分类`），候选树与最终提交树一致，未推送。该提交是P2的起点。用户本轮只要求提交和准备后续内容，不将准备视为P2实现或派工授权；本节及索引在P1提交后补写，尚未提交。

**先后关系：**先写Worker丢链失败回归并修复，再迁移foundation。Leader先统一公共签名、错误归属和调用方适配约定，再允许限定目录并行。以下是拟议分工，不是已派发任务；后续获准开工时沿用 `minimax-m3.1-flash`、max。

| 工作包          | 改动范围                                                  | 必须交付                                                                                               | 依赖与边界                                                                    |
| --------------- | --------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------- |
| P2.1 Worker丢链 | `application/app.go` 看门狗分支及相邻持久测试             | 先红后绿：保留原始统一错误/诊断对象、原始cause和 `ErrWorkerUnexpectedExit`，三类Is/As均可达            | 先完成；不切换Plugin/RunHandle/Activator、RunResult与停机账本签名             |
| P2.2 值转换     | `foundation/typed`、`foundation/ctxkey`、`foundation/env` | 自有转换核心返回统一指针，真实输入分支映射valueerr；13种标准编解码/SQL/flag方法保留原签名并显式nil适配 | 根据P0逐符号台账重新核对当前调用方；不把标准方法当自有接口迁移                |
| P2.3 雪花与工具 | `foundation/snowflake`、`foundation/toolkit`              | 自有构造/生成/解析等失败强类型；SnowflakeID的JSON/Binary/SQL方法保持固定签名与原格式                   | 与值转换公共签名协调；拒绝目标/API契约和非法输入不能混为一类                  |
| P2.4 日志       | `foundation/logging`                                      | 构造、双路写入、轮转/归档与延迟错误账本强类型；slog/io方法保留薄适配                                   | 不新增Info/Error等无返回方法的失败API；保留部分写入字节数及Close幂等/真实收尾 |
| P2.L Leader集成 | spec0002词表、调用方桥接、协议/编译回归及阶段文档         | 全库编译、旧码不漂移、跨层nil真实为nil、依赖无环、完整准出证据                                         | 新增码集中登记；不让多个实现者各自抢号或修改另一人的目录                      |

**必须守住的行为：**

- Worker在Running且非预期停止时，即使Run返回nil也仍属意外退出。不能只修 `%v` 文本包装而漏掉nil分支；不能因为根因包含context.Canceled就吞掉真实意外退出。关机/预期停止分支原行为保持。
- 所有异步断言用完成通道或确定性屏障同步；不使用sleep，也不在释放屏障后立即读取尚未发布的计数。真实成员cause、原统一Code及诊断对象必须保留；框架意外退出的外层责任与内层原码分开断言。
- 保留typed/ctxkey/env现有13种标准方法集、失败不写回原值及各自SQL NULL处理。SnowflakeID仍以十进制字符串传JSON、十进制文本字节传Binary，Scan(nil)仍置零成功，不借错误迁移改变格式或容忍输入。
- `error` 固定签名的适配层必须在强类型结果为nil时显式返回原生nil。builtin、GORM hook、示例及其它调用方若引用了已改函数，同一补丁只补必需桥接，不提前整体切换其生命周期或插件接口。
- 日志一路失败仍尝试另一路，聚合保留全部cause；关闭后ErrClosed仍可Is。Close必须等待原有scheduler/归档真实完成，保留启动归档与后台延迟错误，不能只把账本类型改成指针而漏掉异步发布。

**错误码准备与容量：**现有84项不动，复用已批准control/valueerr及框架意外退出码。logging/snowflake等尚无足够具体所属码的分支，开工先按实际失败列归属、分组、安全Message和原因类型，依E04核算新增容量并集中登记；当前500组26项、400组17项，分别余24/33个SDK槽位。这里不抢占新号，不将日志/雪花故障一律塞入FRAMEWORK内部兜底，不按错误文本猜400/500责任。

**准出命令（全部尚未作为P2执行）：**

- `go test ./foundation/... ./application/... -count=1` 与 `go test -race ./foundation/... ./application/... -count=1`：Worker、值转换、标准方法集、日志异步收尾与失败原因保留。
- `go test -race ./plugins/web-gin/httpresult/... ./plugins/grpc/grpcresult/... -count=1`：新增或复用码的状态与安全信封回归；不冒充P6拦截器接入。
- `make error-codes-check`、`make test-scripts`、`make test`、`make race`、`make vet`、`make lint-go`、`make lint-docs`：定义、根因、调用方及全部14模块不受破坏。
- `make check-examples`、`bash .github/scripts/consumer/check-v2-consumers.sh`：固定签名桥接与独立消费，不用本地replace掩盖调用方不匹配。
- `GOTOOLCHAIN=go1.25.11 GOFLAGS=-mod=readonly go list -deps ./foundation/...`：另检输出中无application/plugins反向依赖；`git diff --check` 及受影响Go文件格式输出为空。

任一包无法同时保留根因、成功nil、标准方法集或确定性收尾时，停在对应包修复，不把失败留给P3。必要回退以P1提交为锚点，只在另获授权后执行完整补丁回退；不自动reset、变基或重写历史。P3–P7、真实外部环境和发布仍不在本次准备的实施范围内。

#### P2准出记录（2026-11-09，待用户审阅）

用户于2026-11-09批准P2开工并裁决7项新码（50026–50032，归属snowflake/logging/toolkit），子代理按 `minimax-m3.1-flash`（effort: medium）并行实施，Leader统一集成。以P1提交 `7c4f7fed106ad36d6127e5f19d212b7bbc12897c` 为基线，39个改动文件全部落在授权边界内（`/tmp/opencode/v2-p2/verify.py` 校验通过）。

**各工作包结果：**

- P2.1 Worker丢链：`application/app.go` 看门狗Running非预期分支改 `apperr.New(FrameworkWorkerUnexpectedExit, …, ErrWorkerUnexpectedExit[, runErr])` 双根因；nil早退同样判失败。`application/worker_chain_test.go` 三条持久回归（原始哨兵Is、外层50023 As、内层统一错误保链）先红后绿；账本与通道签名未动。
- P2.2 值转换：typed 14个解析方法改返回统一指针（语法非法40013、strconv范围40014、非有限值/非布尔词40013）；13种标准方法保留原签名、内部强类型核心显式nil适配（键语法40013、目标类型40015、nil接收器50019）；ctxkey/env同构迁移并保留各自哨兵根因；编码器I/O失败包FrameworkInternalError保留原错误。
- P2.3 雪花与工具：NewNode参数非法50026、时钟回退50027（根因ErrClockBackwards）；nil node/receiver复用VALUE_INVALID_RECEIVER并保留哨兵；基因越界40014、ID解析40013、Scan类型40015；SnowflakeID六方法固定签名薄适配，2^53精度与Scan(nil)置零格式固化。toolkit密码三函数50032（bcrypt根因、消息不含密码）、time.Parse 40013（*time.ParseError根因）；Must*与(值,bool)签名不动。
- P2.4 日志：New 50028、配置校验50029、双路写入失败单路From/双路Join(WriteFailed 50030)保留全部cause、IO/轮转失败50031；Handle/Write/Close保持slog/io签名薄适配；closeErr/bgErrors账本强类型；Close幂等并等待schedulerDone+archiveWg真实收尾；rotator测试去sleep改完成通道。
- P2.L Leader集成：spec0002登记7码（84→91项）与检查器白名单；builtin调用点经指针nil判定天然兼容未改；`plugins/database/models/models_test.go` GORM hook返回值改独立变量适配；本准出记录与planREADME/spec0002同步。

**准出结果（基线7c4f7fe + dirty，日志 `/tmp/opencode/v2-p2/gate-*.log`）：**

| 检查                                                                           | 结果                                   |
| ------------------------------------------------------------------------------ | -------------------------------------- |
| `make test` / `make race` / `make vet`（14模块）                               | 通过                                   |
| `make lint-go` / `make lint-docs`                                              | 通过（spec0002表格列宽经Prettier重排） |
| `make error-codes-check`（91项全库唯一且与定义精确对应）                       | 通过                                   |
| `make test-scripts`（含CI守卫race）                                            | 通过                                   |
| `make check-examples` / `bash .github/scripts/consumer/check-v2-consumers.sh`  | 通过                                   |
| `go list -deps ./foundation/...` 无application/plugins反向依赖                 | 通过                                   |
| `gofmt -l application foundation plugins` / `git diff --check`                 | 空输出 / 通过                          |
| `python3 /tmp/opencode/v2-p2/verify.py`（39文件边界+91项归属计数+P1/P2码完整） | 通过                                   |

外部Redis/SMTP/数据库集成仍默认跳过；P3生命周期切换、P4–P7与发布验收未执行。本记录不将P2升级为已验收完成；P2改动已于2026-11-10按用户授权提交为 `fad7597c62496ea5c0e7f280227bb6a2de977782`，未推送。

### P3：原子切换application生命周期契约与全部实现者

- 动作：同步改自有Plugin/RunHandle/Activator等接口、内核辅助返回、错误账本/通道、builtin、全部上层插件模块的实现方法、所有测试替身及示例消费者。不能仅改接口后单独交付。
- 交付：直接返回原生error不再编译的契约断言；RunResult字段、错误聚合及StopIncomplete/HookPanic诊断保留测试；启动、回滚、自然排空和清理语义等价证据。
- 准出：`make test`、`make race`、`make vet`、`make check-examples`，以及 `bash .github/scripts/consumer/check-v2-consumers.sh` 全部通过。已确认的控制语义若未在实现和测试中保留，或任何实现者仍使用旧方法集，本阶段不能准出。
- 状态：2026-11-10实现及Leader准出完成，待用户审阅；内核与全部实现者一次原子切换，无新码。P4–P7未执行。

#### P3开工准备包（2026-11-10，尚未实施）

用户于2026-11-10授权提交P2并要求制定下一步工作任务；本节为准备文档，不构成P3开工或派工授权。P2已提交为 `fad7597c62496ea5c0e7f280227bb6a2de977782`（未推送），是P3的起点。

**范围（一次原子切换，不留接口已改而实现者未跟的中间态）：**

- 内核契约：自有 `Plugin.Start/Stop`、`RunnablePlugin.Run`、`Quiescer.Quiesce`、`Activator.Activate`、`RunHandle.Wait`、`App.Run/Shutdown` 及内部辅助改返回 `*apperr.Error`；`RunResult` 七阶段错误字段、`StuckInstance.Err`、`runErrs` 等账本同步强类型。
- 实现者：`application/builtin`、web-gin、grpc、database、cache、email、cron、async 的生命周期方法、全部测试替身、示例与独立消费测试同一补丁切换。

**拟议分工（获准开工时沿用 `minimax-m3.1-flash`，effort届时确认）：**

| 工作包              | 改动范围                                                 | 必须交付                                                                                                                                     | 依赖与边界                             |
| ------------------- | -------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------- |
| P3.1 内核契约与账本 | `application` 接口、App、supervisor、shutdown、context   | 契约与账本原子改型；computeWaitError 聚合改强类型Join/From；HookPanic/StopIncomplete诊断对象保留errors.As；context哨兵经control.Classify归因 | 最先完成并冻结契约；先写编译负例夹具   |
| P3.2 builtin实现者  | `application/builtin`                                    | logger等内置组件Start/Stop适配新契约，根因与真实收尾保留                                                                                     | 依赖P3.1契约                           |
| P3.3 协议插件实现者 | `plugins/web-gin`、`plugins/grpc` 生命周期方法           | Start/Stop/Run适配；不改本阶段归属的HTTP/RPC归一链                                                                                           | 依赖P3.1契约；与P3.4并行               |
| P3.4 领域插件实现者 | `plugins/{database,cache,email,cron,async}` 生命周期方法 | 各自Start/Stop/Run适配；外部服务集成仍默认跳过                                                                                               | 依赖P3.1契约；与P3.3并行               |
| P3.L Leader集成     | 编译/词表/门禁/准出记录                                  | 全库编译、示例与独立消费、全量门禁与范围守卫                                                                                                 | 新码按E04集中裁决，预期复用50020–50023 |

**行为红线：**RunResult各阶段字段语义与聚合规则不变；StopIncomplete成员Err经结构化聚合逐个可达（spec R06）；Quiesce排空与等待语义不变；nil核心显式真nil；异步断言用通道屏障不用sleep；不借切换改变关机预算、panic隔离或平滑关机顺序。

**准出命令（沿用P2口径）：**`make test/race/vet/lint-go/lint-docs`、`make error-codes-check`、`make test-scripts`、`make check-examples`、`bash .github/scripts/consumer/check-v2-consumers.sh`、`go list -deps ./foundation/...` 反向依赖检查、受影响文件 `gofmt -l`、`git diff --check` 及新写P3范围verify脚本。

#### P3准出记录（2026-11-10，待用户审阅）

用户于2026-11-10批准P3开工。P3.1（minimax-m3.1-flash/max）先冻结内核契约，P3.3/P3.4（medium）并行适配，builtin 由 P3.1 一并完成（application 模块编译闭环），Leader 统一修复 examples/test 消费者并跑全量门禁。以P2提交 `fad7597c62496ea5c0e7f280227bb6a2de977782` 为基线，全部改动落在授权边界内（`/tmp/opencode/v2-p3/verify.py` 校验通过）。

**各工作包结果：**

- P3.1 内核契约与账本：`Plugin.Start/Stop`、`RunnablePlugin.Run`、`Quiescer.Quiesce`、`RunHandle.Wait`、`Activator.Activate`、`App.Run/Shutdown/Register*` 及 `RunResult` 七阶段错误字段、`StuckInstance.Err`、`runErrs`、`watchdogErrChan` 等全部 `*apperr.Error`（14生产文件+3新测试）。HookPanic 以根因保留可 As；StopIncompleteError 新增 `Unwrap() []error` 成员逐个可达；40+哨兵保留 `errors.Is`，`sentinelCodes` 仅登记分类码不新增码；context哨兵经 control.Classify 归因（取消49900/超时50401）；聚合外层沿用首个成员码保住P2外层50023断言。红→绿证据与 application test/race/vet/gofmt/golangci 全过。
- P3.3 协议插件：web-gin/grpc 生命周期方法改型；归一顺序与内核 normalizeErr 一致（已有统一错误→控制归因→所属码兜底）；重复启动用 InvalidState、未启动即Run用 ServiceUnavailable+新增哨兵 ErrNotStarted（Leader裁定保留导出）；http.ErrServerClosed/grpc.ErrServerStopped 正常退出真nil。
- P3.4 领域插件：database/cache/email/cron/async 生命周期方法改型；database 50014/cache 50016/email 50017 保留原生根因；cron/async 暂用 FrameworkInternalError(50000) 单点兜底（专属码 CRON_SCHEDULER_ERROR/ASYNC_POOL_ERROR 列为P5准备项，届时按E04裁决）；Redis拨号超时按冻结归因顺序落50401并保留redis根因；修复 `chan error` 收 typed-nil 的假失败（通道改 `chan *apperr.Error`）。
- P3.L Leader集成：examples/v2 三示例与 test/consumer 两夹具适配新签名（Start/Run/Stop/Quiesce 改型、错误透传或 apperr.From 包装、`var runErr error` 改 `*apperr.Error` 消除 typed-nil 假失败）；P2 Worker链断言原样通过。

**准出结果（基线fad7597 + dirty，日志 `/tmp/opencode/v2-p3/gate-*.log`）：**

| 检查                                                                          | 结果                 |
| ----------------------------------------------------------------------------- | -------------------- |
| `make test` / `make race` / `make vet`（14模块）                              | 通过                 |
| `make lint-go` / `make lint-docs`                                             | 通过                 |
| `make error-codes-check`（91项，P3零新增）                                    | 通过                 |
| `make test-scripts`（含CI守卫race）                                           | 通过                 |
| `make check-examples` / `check-v2-consumers.sh`（14模块）                     | 通过                 |
| `check-v2-examples.sh --fixture http-db-cache / sqlite-async`                 | 通过                 |
| `go list -deps ./foundation/...` 无反向依赖 / `gofmt -l` / `git diff --check` | 通过 / 空输出 / 通过 |
| `python3 /tmp/opencode/v2-p3/verify.py`（边界+词表91项逐字不变）              | 通过                 |

外部Redis/SMTP/数据库集成仍默认跳过；P4–P7与发布验收未执行。本记录不将P3升级为已验收完成；P3已按用户授权提交为 `c8abf5775f189fc5b144eae210d94488fe35cd0f`，候选树核对一致，未推送。

### P4：迁移数据库及五适配器的完整错误边界

- 动作：同时修改dialect.Adapter、5个Open/DSN校验/延迟错误字段及database调用方；自有模型辅助强类型。GORM hooks仍为固定签名，显式nil适配，保留雪花和GORM原因。
- 交付：5驱动空DSN和格式拒绝测试、成功钩子确实执行且无假失败的SQLite回归、原生gorm句柄边界说明；事务适配如获授权单列门面，不误称已有。
- 准出：`go test ./plugins/database/... ./plugins/database/adapter/mysql/... ./plugins/database/adapter/postgres/... ./plugins/database/adapter/sqlite/... ./plugins/database/adapter/sqlserver/... ./plugins/database/adapter/oracle/... -count=1`；复跑独立消费者。
- 状态：2026-11-12实现及Leader准出完成，待用户审阅；六模块强类型边界、五新码与回归已交付，真实外部引擎验收仍未执行。

#### P4施工清单（2026-11-12，仅准备，尚未实施）

> 以下保留获准前的准备快照；后续用户于2026-11-12明确批准实际实施，五项编号已集中登记为50033–50037，旧91项未重排。准备态的“未获授权”不再约束本次P4；P5–P7、提交推送与发布仍须另行授权。

本轮只核对源码并编写清单，不修改Go、依赖、CI或错误码词表，不派发实施任务，不提交推送。实施基线为P3提交 `c8abf5775f189fc5b144eae210d94488fe35cd0f`；P3提交后树一致且工作区曾干净。总计划保持in-progress，P4仍未开工；2026-11-12用户已确认五项错误归属与评审边界，仅授权写回文档；编号登记和Go实施仍待另行授权。

##### P4当前缺口与接口改动

| 位置与现状依据                                                                                                                                                     | P4拟议改动                                                                                                                                                                                           | 保留的契约                                                                                                                                                                                                                                 |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| [dialect/dialect.go](../../plugins/database/dialect/dialect.go)：`Adapter.Open() (gorm.Dialector, error)`；`Lookup`、`registry.lookup`、`validateDriver` 返回error | 分别改为 `Open() (gorm.Dialector, *apperr.Error)`、`Lookup(driver Driver) (Adapter, *apperr.Error)`、`lookup(driver Driver) (Adapter, *apperr.Error)`、`validateDriver(driver Driver) *apperr.Error` | `Name() string`、`Factory func() Adapter`、`Register(driver Driver, factory Factory)` 不增加失败返回；Register非法声明仍panic，库自产的panic错误按新码构造；工厂在锁外调用、未知驱动不伪造适配器                                           |
| 五个 `adapter/<driver>/<driver>.go`：均有 `adapter.err error` 和原生error版Open                                                                                    | 延迟错误字段改 `*apperr.Error`，五个Open同步实现新接口；MySQL/Postgres/SQLServer/Oracle的 `validateDSN(raw string)` 改统一指针                                                                       | `New(dsn string) dialect.Adapter`、`FromEnv() dialect.Adapter`、init注册不改签名；New/FromEnv/Open不拨号，配置失败仍延迟至Open；SQLite没有validateDSN，不为对齐形式虚造校验器                                                              |
| [plugin.go](../../plugins/database/plugin.go)：Start/Stop已在P3强类型化；Open之后复用err接收 `gorm.Open` / `db.DB` 的原生error                                     | 切换Lookup/Open调用，分开 `adapterErr` / `gormErr` / `sqlErr`，避免把原生error赋给已推断为统一指针的变量；同步fakeAdapter及函数值断言                                                                | Start/Ping与Stop的控制归因、关闭顺序、共享句柄与P3账本不重写；第三方失败仍保留原生原因，必要清理失败不能文本化或无声吞掉                                                                                                                   |
| [models/base_entity.go](../../plugins/database/models/base_entity.go)：`generateEntityID(tx *gorm.DB) (snowflake.SnowflakeID, error)`，含裸哨兵和fmt包装           | generateEntityID改 `(snowflake.SnowflakeID, *apperr.Error)`；自有BeforeCreate/BeforeUpdate内部核心如抽出也返回统一指针                                                                               | BaseEntity与[BaseEntityWithSoftDelete](../../plugins/database/models/base_entity_soft_delete.go)共四个GORM hook仍为 `func(tx *gorm.DB) error`；非nil核心再装入error，成功显式返回nil；非零ID保留、不强制取雪花节点、时间戳及软删除语义不变 |
| [context.go](../../plugins/database/context.go)、[gormlog/gormlog.go](../../plugins/database/gormlog/gormlog.go)                                                   | Get/MustGet只做回归；GORM logger、driver测试替身仅核对固定方法集，不批量改型                                                                                                                         | Get已返回 `(*gorm.DB, *apperr.Error)`；原生gorm `.Error` / Transaction、`gorm.Dialector.Initialize(*gorm.DB) error`、gorm logger Trace原生error输入均属第三方边界；本次不新增事务门面                                                      |

同补丁更新六个模块（database核心及五个adapter）的接口实现、测试替身、函数值、GoDoc/示例和独立消费者。Go没有返回协变；正例证明五个adapter满足新接口，负例证明旧 `Open() (gorm.Dialector, error)` 不再满足接口，并排除环境/下载失败冒充预期编译失败。构造函数没有失败返回，不为统一类型新增返回值；测试故障注入与标准库方法不作为生产API整批迁移。

##### P4错误归属与评审结论

根包已拥有8项 `DATABASE_` 码，而根包依赖dialect。dialect不能反向导入database根包取 `DatabaseError`；models若导入根包也可能经根包内部测试形成测试导入环。已确认在既有database模块的低层子包内拥有各自语义，不新增module、不把领域错误搬进foundation，也不复制或重导出现有50014。

**2026-11-12用户已确认以下5项的归属、符号、Output、500分组及评审边界；尚未登记或占号。确认仅授权写回文档，不包含编号登记、Go实施或派工。**

| 已确认归属 / Go符号 / Output                                                                    | 负责的失败                                                           | 已确认分组与边界                                                                                                                    |
| ----------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| `plugins/database/dialect` / `DriverInvalid` / `DATABASE_DIALECT_DRIVER_INVALID`                | Register的空白/none驱动名、重复注册等SDK自产的非法驱动声明           | 500；保留panic控制流，不改为静默忽略或返回成功                                                                                      |
| `plugins/database/dialect` / `DriverNotLoaded` / `DATABASE_DIALECT_DRIVER_NOT_LOADED`           | Lookup未注册驱动                                                     | 500；保留 `ErrDriverNotLoaded` 根因，不把驱动缺失当业务404                                                                          |
| `plugins/database/dialect` / `AdapterInvalid` / `DATABASE_DIALECT_ADAPTER_INVALID`              | nil注册工厂、工厂返回nil/typed-nil、适配器实例/返回值违反契约        | 500；保留 `ErrNilAdapter`；显式WithAdapter的typed-nil在调用Open前防护；第三方nil Dialector不传给gorm.Open                           |
| `plugins/database/dialect` / `AdapterConfigInvalid` / `DATABASE_DIALECT_ADAPTER_CONFIG_INVALID` | 五适配器启动/部署用DSN、host/port/service/path缺失或现有格式校验失败 | 500；五驱动共用该配置语义，安全描述注明驱动但不复制DSN；确有valueerr原因时保留为内层，不因含400值错误直接把服务自身配置当客户端失败 |
| `plugins/database/models` / `ModelInvalid` / `DATABASE_MODEL_INVALID`                           | 自有模型钩子的tx/Statement/Context或雪花节点实例契约缺失             | 500；保留模型哨兵；已有application.Get、snowflake.Generate统一错误仍保留原码与完整链，不强制全部换成ModelInvalid                    |

安全Message沿用清单，依次为“数据库驱动声明非法”“数据库驱动未加载”“数据库适配器无效”“数据库适配器配置非法”“数据库模型契约无效”。已确认采用dialect的 `DATABASE_DIALECT_` 与models的 `DATABASE_MODEL_` 归属前缀，按更长前缀优先匹配；根包原有 `DATABASE_` 的8项定义不变。前缀及声明仍未写入受控词表、检查器或Go源码。

这5项的语义与边界已确认，具体编号尚未分配。另获登记与实施授权后，必须重读active/retired词表并按spec0002 E01–E05取组内全局最大号追加。本轮不设置具体编号，不修改91项active表。按准备时读取的50000–50032计算，SDK的00–49共50槽，剩17槽；5项未来登记后余12槽。这是容量核算，不是预留号或Go实施授权。

原有 `DatabaseError` 50014继续负责GORM打开、连接池、Ping、关闭等未知数据库运行失败；`TransactionFailed`及404/409约束码不自动套用到所有驱动错误。context取消/内部超时仍复用CONTROL_的499/504，并保留原生哨兵；模型发号错误保留已注册SNOWFLAKE_/VALUE_语义。管理API校验用户提交配置若属于请求输入，按R11另用输入码；不修改这里New/FromEnv启动配置的责任。

已有错误/根因用From或New保链，不用字符串替代；需要明确把服务配置重包成500时用New而不是会沿用原码的From。现有SQLServer URL解析等已脱敏的错误，不重新把含密码的原始URL解析错误挂入Err；同时断言 `ErrorMessage`、JSON以及现有 `Error()` 的密码不泄漏约束。第三方运行错误保留原生根因，但不把其文本复制进对外安全文案或Data。

**本轮确认的实施边界（2026-11-12）：**

- 五项对应五类失败责任，不是五个数据库驱动分别取一个码。DriverInvalid仅负责非法驱动声明，DriverNotLoaded负责Lookup未加载；二者都不代表数据库网络连接失败。Register既有panic方式不改，Lookup保留原哨兵根因。
- AdapterInvalid表示工厂/实例/返回值不满足对象契约，AdapterConfigInvalid表示已有校验拒绝的配置内容；不得把两类合成通用适配器失败，也不为迁移收紧五驱动的DSN接受集合。SQLite没有通用字符串格式拒绝；Open仍不建立物理连接。
- 启动/部署用DSN和内部调用契约归服务端500；管理API接收用户提交配置的请求输入错误按R11另判。只有责任确属服务自身配置时才明确重包为500，不按字符串内容猜责任。
- ModelInvalid只处理模型自身缺少必要tx/Statement/Context或节点实例的失败分支。application.Get已有的容器/组件错误及snowflake.Generate已有的范围/时钟错误保留原码和完整原因；若同时带模型哨兵，保留该哨兵和原统一错误，不能统一洗成ModelInvalid。
- 非零ID路径仍不获取或强制要求雪花节点；BeforeUpdate不因迁移额外增加tx/context校验。两模型共四个GORM hook保留原生error签名，内部核心非nil时再转接，成功显式返回字面量nil。
- 沿用既有Is/As与脱敏约束。登记前再次复核剩余容量，禁止提前占具体号；现有91项词表、根包8项DATABASE_以及所有Go文件保持原样。确认本节不自动批准P4接口切换、Go实施、派工或发布。

本轮已选择低层归属方案，未选择抽取公共码子包并迁移既有定义的备选，以避免额外归属/API迁移；禁止用根包变量别名或FrameworkInternalError掩盖取码依赖问题。**P4.0的语义评审已确认，但登记及实施仍未获授权：保持准备态，不分号、不切换接口、不派Agent。**

##### P4五适配器行为矩阵

| 适配器                                                              | 按现有代码覆盖的拒绝与成功分支                                                     | 不借本次迁移改变的行为                                                                                                   |
| ------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| [MySQL](../../plugins/database/adapter/mysql/mysql.go)              | 空DSN；缺@、缺/、非tcp地址、空tcp地址；FromEnv空host/port；合法串与默认/显式DSN    | 保持当前基本结构检查与拼装格式，不追加新的端口范围或驱动级完整语法校验                                                   |
| [PostgreSQL](../../plugins/database/adapter/postgres/postgres.go)   | 空DSN；URL无host、非法纯文本；FromEnv空port；postgres/postgresql URL与含等号串     | 现有key=value检查较宽松，不借迁移严格解析全部语法；转义、缺省值与优先级保持                                              |
| [SQLite](../../plugins/database/adapter/sqlite/sqlite.go)           | New空/空白DSN；FromEnv空白DATABASE_PATH；无配置默认 `:memory:`；DSN优先            | 没有任意字符串“格式非法”通用拒绝分支；路径不存在不是Open失败，Open不触碰文件；真实建连/文件/SQL问题由Start或gorm执行验证 |
| [SQL Server](../../plugins/database/adapter/sqlserver/sqlserver.go) | 空DSN；URL解析失败、非sqlserver scheme、无host；FromEnv空host/port；IPv6与凭据转义 | 保持URL编码、net.JoinHostPort行为与现有接受集合                                                                          |
| [Oracle](../../plugins/database/adapter/oracle/oracle.go)           | 空DSN；URL无host、非法纯文本；FromEnv空host/port/service；URL/logfmt/Easy Connect  | 现有logfmt/Easy Connect检查较宽松；libDir只拼装不探测客户端；Open构造成功不证明Oracle客户端可用或连通                    |

五驱动都验证失败时Dialector为nil、成功时错误指针真nil、延期错误可重复读取且Code/Output稳定；无网络单测不依赖真实服务。凭据测试同时覆盖显式DSN与环境变量，并用t.Setenv/清理恢复避免跨用例污染。真实SQLite回归需验证两个模型的Create/Update钩子确实执行、非零ID保留、失败不写入半成品，以及缺节点/无效gene/时钟回退的原因可达；不凭直接调用hook通过就宣称GORM已执行hook。

##### P4拟议分工与交付顺序

| 工作包                 | 唯一写入范围                                                      | 必须交付 / 准入                                                                                                                              |
| ---------------------- | ----------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| P4.0 Leader确认归属    | 计划、spec0002、词表守卫白名单（后两项仅实施获准后修改）          | 2026-11-12归属与语义已确认；另获登记/实施授权后检查91旧项保真与容量，集中分号；当前停在准备态，未登记或派工                                  |
| P4.1 公共契约          | `plugins/database/dialect/**`及其编译夹具                         | 冻结Adapter.Open/Lookup/内部辅助强类型及哨兵/panic口径；旧签名负例先红，正例方法集可验证；契约冻结后才能并行。主干最终交付必须六模块原子通过 |
| P4.2A MySQL/PostgreSQL | 对应两个adapter模块（含测试/GoDoc）                               | Open/validateDSN/err字段同步；现有校验矩阵、DSN脱敏、构造无网络、成功真nil与根因测试                                                         |
| P4.2B SQLServer/Oracle | 对应两个adapter模块（含测试/GoDoc）                               | 同上；Oracle客户端/数据库实测缺失单列，不让环境缺失冒充语法拒绝通过                                                                          |
| P4.3 SQLite/模型       | SQLite模块及 `plugins/database/models/**`                         | SQLite原有接受集合；generateEntityID强类型；四固定hook薄适配；持久SQLite真实GORM钩子回归与Is/As                                              |
| P4.L Leader集成        | database根包、共享fakeAdapter、examples、test/consumer、文档/守卫 | 处理Open之后原生error变量分离；检查P3生命周期语义不漂移、循环依赖、编译正反例、词表及全部准出证据                                            |

P4.1冻结后，P4.2A/P4.2B/P4.3写入范围互不重叠，可并行；database根包与公共词表/CI仅Leader写。模型码和dialect码由各归属维护者在Leader登记之后定义，不能多个实现者分别分号。模型回归若与root测试重叠，由Leader协调归属，不同时改同一文件。此处只是拟议角色，不创建卡、不派Agent；模型/effort在用户实际授权实施时再确认。

##### P4实施验收门禁（本轮均未运行）

从仓库根执行，固定 `GOTOOLCHAIN=go1.25.11`；六模块完整范围必须显式列出，`./plugins/database/...` 不跨越五个独立adapter模块。P4.1中间态允许暂时无法编译，但不作为可提交/可发布成果。

```bash
export GOTOOLCHAIN=go1.25.11
go test ./plugins/database/... ./plugins/database/adapter/mysql/... ./plugins/database/adapter/postgres/... ./plugins/database/adapter/sqlite/... ./plugins/database/adapter/sqlserver/... ./plugins/database/adapter/oracle/... -count=1
go test -race ./plugins/database/... ./plugins/database/adapter/mysql/... ./plugins/database/adapter/postgres/... ./plugins/database/adapter/sqlite/... ./plugins/database/adapter/sqlserver/... ./plugins/database/adapter/oracle/... -count=1
go vet ./plugins/database/... ./plugins/database/adapter/mysql/... ./plugins/database/adapter/postgres/... ./plugins/database/adapter/sqlite/... ./plugins/database/adapter/sqlserver/... ./plugins/database/adapter/oracle/...
make error-codes-check
make test-scripts
make test
make race
make vet
make lint-go
make lint-docs
make check-examples
go test -race ./plugins/web-gin/httpresult/... ./plugins/grpc/grpcresult/... -count=1
bash .github/scripts/consumer/check-v2-consumers.sh
bash .github/scripts/consumer/check-v2-examples.sh --fixture http-db-cache --race
bash .github/scripts/consumer/check-v2-examples.sh --fixture sqlite-async --race
GOFLAGS=-mod=readonly go list -deps ./foundation/...
gofmt -l plugins/database examples/v2 test/consumer
git diff --check
```

另外检查依赖输出：foundation不得出现application/plugins，dialect/models不得反向依赖database根包；错误码新归属须通过AST白名单及HTTP/RPC Code/Output/状态回归。静态编译夹具必须覆盖旧方法集、Open函数值、延迟字段及四个hook固定签名；完整全库类型扫描仍属P7，不能把P4夹具称为全仓守卫。

先通过六模块与SQLite行为门禁，再通过全量门禁和两个独立消费者fixture；记录命令、SHA、dirty状态和跳过项。MySQL/PostgreSQL真实集成按显式环境开关另跑；Oracle/SQLServer无专属环境时标未运行/受阻，不算已验收连通。若新增依赖或go.mod变化确有必要，先解释并检查独立消费候选模块图，不自动升级驱动或新增SDK模块。

**本轮准备验收：**只运行 `make lint-docs`、`make error-codes-check`、`python3 /tmp/opencode/v2-p4-prep/verify.py`、`git diff --check`。verify核对仅plan0002与planREADME两文档变化、P3基线、91项active机器区逐字未变、现有Go/依赖/CI文件未改、文档相对链接与新标题锚有效；它是本轮临时准备检查，不替代P4持久回归。真实测试结果在执行后回报，不把上面的未来命令标为通过。

##### P4停工与回滚条件

发现nil假失败、接口实现者缺漏、原有DSN接受集合变化、敏感凭据外泄、Is/As原因丢失或子包导入环时，停在该工作包修复，不把失败留给P5。不实现事务封装、不重写第三方gorm行为、不提前开展P5–P7。需要回退时以 `c8abf577` 为已提交锚点，连同新接口、五实现者、根调用方、测试与获批码表调整整批回退；未经用户授权不reset、不改历史、不移动标签或推送。P4开工/新增码确认不等于整计划或看板卡验收完成。

#### P4准出记录（2026-11-12，待用户审阅）

用户于2026-11-12批准实际实施。Leader先冻结dialect契约及集中登记，再将MySQL/PostgreSQL、SQLServer/Oracle、SQLite/模型分为互不重叠的工作包，最后复读关键diff并集成根调用方。基线为 `c8abf5775f189fc5b144eae210d94488fe35cd0f`，以下检查均基于该HEAD加本次未提交改动（dirty=true）；没有提交、推送、PR或发布，P5–P7未启动。

本次交付：

- `Adapter.Open`、Lookup/内部lookup与validateDriver改为统一指针；五适配器延期错误字段/Open/已有validateDSN同步，New/FromEnv/Factory/Register不增加失败返回，非法注册仍panic。新码50033–50037在dialect/models归属中定义，机器词表91→96，旧91项六列逐项保真；根包8项DATABASE_不变。
- 五驱动原DSN接受集合、默认值、转义与优先级不收紧；New/FromEnv/Open不建连，SQLite不新增通用格式拒绝。nil适配器、typed-nil及nil/typed-nil Dialector在调用gorm.Open前拒绝；配置错误延期交付且重复Open稳定。
- 根包分离adapterErr/gormErr/sqlErr，保既有Code及Is/As；Ping失败后的Close错误不再吞掉，聚合保首失败码和两条根因。原有驱动未加载测试从通用50014改断言50034，并补Output/ErrDriverNotLoaded链断言。
- generateEntityID返回统一指针；tx/Statement/Context缺失使用ModelInvalid，已有application/Get及snowflake错误保原码和全部原因。四个GORM hook仍返回error，成功显式nil；非零ID和BeforeUpdate原行为不变。
- 持久编译正反例覆盖旧Adapter方法集及普通error拒绝、函数值/延期字段与四hook固定签名；五新增码经HTTP500/gRPC Internal及信封三元组/脱敏回归。协议测试使用与词表一致的输入夹具，不新增插件间运行依赖。
- 真实SQLite文件库覆盖两个模型Create/Update钩子、SkipHooks对照、非零ID保留、缺节点失败不落库；补充GeneProvider非法gene经GORM仍保VALUE_OUT_OF_RANGE、根因与零ID且无半成品记录。GORM自动更新时间会覆盖落库时间，Update通过gorm:update前的观察回调验证，不以自动字段更新冒充钩子执行。

SQLite新回归直接使用application/v2，已在该子模块补同版本v2.0.0的直接require（此前仅传递依赖），不新增SDK模块、不升级第三方库，其余go.mod/go.sum不变；独立消费候选图验证通过。

实际准出命令与结果（`GOTOOLCHAIN=go1.25.11`）：

| 命令                                                                                                                                                                                                                                         | 结果与边界                                                                                                        |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| 本节施工清单所列六模块 `go test ... -count=1`、同范围 `go test -race ... -count=1` 与 `go vet ...`                                                                                                                                           | 通过；根包及dialect/gormlog/models三个子包、五独立adapter均显式覆盖；真实外部引擎按原开关跳过                     |
| `make test`、`make race`、`make vet`                                                                                                                                                                                                         | 最终复跑通过，14模块；包含现有snowflake时钟回退测试                                                               |
| `make lint-go`、`make lint-docs`                                                                                                                                                                                                             | 最终通过；新增两测试曾触发ST1023/QF1011，已把精确Open方法断言移至包级而非关闭lint                                 |
| `make error-codes-check`、`make test-scripts`                                                                                                                                                                                                | 通过；新子域最长前缀、点导入/别名反例及词表-源码双向核对                                                          |
| `make check-examples`、`bash .github/scripts/consumer/check-v2-consumers.sh`                                                                                                                                                                 | 通过；独立候选无本地replace，覆盖仓内受控消费范围，不等于发布验收                                                 |
| `bash .github/scripts/consumer/check-v2-examples.sh --fixture http-db-cache --race`                                                                                                                                                          | 通过，独立fixture竞态验证                                                                                         |
| `bash .github/scripts/consumer/check-v2-examples.sh --fixture sqlite-async --race`                                                                                                                                                           | 通过，独立fixture竞态验证                                                                                         |
| `go test -race ./plugins/web-gin/httpresult/... ./plugins/grpc/grpcresult/... -count=1`                                                                                                                                                      | 通过，包含五新增码协议映射                                                                                        |
| `GOFLAGS=-mod=readonly go list -deps ./foundation/...` 与 `GOFLAGS=-mod=readonly go list -deps ./plugins/database/dialect ./plugins/database/models`                                                                                         | 通过并检查输出：foundation无application/plugins，dialect/models无database根包反向依赖                             |
| `gofmt -l plugins/database plugins/web-gin/httpresult/p4_error_mapping_test.go plugins/grpc/grpcresult/p4_error_mapping_test.go .github/scripts/ci/check-error-codes.go .github/scripts/ci/check-error-codes_p4_test.go`、`git diff --check` | 输出空，无格式或空白问题                                                                                          |
| `python3 /tmp/opencode/v2-p4/verify.py`                                                                                                                                                                                                      | 通过：96项、旧91项六列保真、P4路径范围及三份文档相对链接/锚、仅SQLite既有内部模块直接依赖变化；不是P7全仓类型守卫 |

日志位于本机会话 `/tmp/opencode/v2-p4/gate-*.log`，机器检查汇总为 `gates.json` / `scope.json`。这些临时文件不是仓库内持久门禁；回归源码与编译夹具已持久落库。文档结果在更新本记录后重跑，不将先前失败冒称一次通过。

未执行/保留边界：MySQL/PostgreSQL真实连接仍按既有显式环境开关跳过；无专属Oracle/SQLServer环境与Oracle客户端运行期验收，Open构造成功不等于连通。模型链路未注入真实时钟回退（无公开注入点，不使用unsafe或改foundation私有状态）；发号器现有时钟回退用例已在本次全量test/race中执行，模型Generate失败分支直接透传并经非法gene回归覆盖。ErrNilSnowflakeNode防御分支因容器禁止暴露nil节点无法经公开API触发，保留不变量测试；不伪造为已跑通。以上不是整计划或看板卡完成，P4待用户审阅。

### P5：迁移缓存、邮件、Cron和Async自有操作

- 动作：同时改cache自有Codec与四种泛型接口/双后端，保留键缺失非失败语义；改邮件发送、模板、附件及生命周期；改Cron与Async的状态/准入/取消/等待，保留原因和并发终态契约。

- P5准备注记（2026-11-10）：P3.4 将 cron/async 生命周期失败暂以 FrameworkInternalError(50000) 单点兜底（`lifecycle_error.go` 末行）；P5 迁移其自有操作时应按 E04 向用户提案专属码（候选 CRON_SCHEDULER_ERROR / ASYNC_POOL_ERROR，500组），获批后替换兜底码并登记词表，不提前占号。
- 交付：编解码失败的模块码、发送与模板失败的模块码、pool准入/状态/超时类型测试；外部拨号和net.Conn适配的成功nil/关闭测试。标准context分支不误映射到Internal。
- 准出：`go test -race ./plugins/cache/... ./plugins/email/... ./plugins/cron/... ./plugins/async/... -count=1`。Redis/SMTP真实集成另记环境与实际执行结果；所有异步断言用通道屏障，不用sleep代替同步。
- 状态：2026-11-15实现及Leader准出完成，待用户审阅；四插件强类型边界、五新码与消费者回归已交付，外部Redis/SMTP集成仍按开关跳过。

#### P5施工清单（2026-11-15，仅准备，尚未实施）

> 以下保留获准前的准备快照；用户已于2026-11-15确认主推提案并批准实际实施，五项编号已集中登记为 `42903` 与 `50038–50041`（词表96→101），准备态的“未占号/未实施”不再约束本次P5；P6–P7、提交推送与发布仍须另行授权。

本轮只核对源码、调用方与号段容量并编写清单和错误码提案，不修改Go源码、`go.mod`/CI或受控词表（96项active表保持不变），不派发实施子任务，不提交推送。实施基线为P4提交 `e4930971ac9155bbf31429b36b66fcc062e58590`（未推送）；具体编号分配与Go实施均待用户确认错误码提案并另行授权。

##### P5当前缺口与接口改动

| 模块与现状依据                                                                                                                                                                                                                                                                                                                   | P5拟议改动                                                                                                                                                                                                                                                                   | 保留的契约与边界                                                                                                                                                                                                                                                    |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [cache/driver](../../plugins/cache/driver/interface.go)：`Codec` 2法与 `KeyCache`/`HashCache`/`SetCache`/`ListCache` 共28个泛型接口方法均返回 `error`                                                                                                                                                                            | 全部30个自有方法统一改为返回 `*apperr.Error`；`JSONCodec` 包装原生 `encoding/json` 失败为统一错误并保留根因                                                                                                                                                                  | `KeyEncoder`、`EncodeKey`、`SetOption`、`ApplySet`、`CacheType` 与 `*CacheOf` 工厂无失败返回不加错；`Get` 未命中仍 `(nil, false, nil)`、`Index/Pop/PopLast` 空或越界 `(nil, nil)`、`Delete/Remove` 不存在幂等 `nil`                                                 |
| [cache/memory](../../plugins/cache/memory/key.go) 与 [cache/redis](../../plugins/cache/redis/key.go)：四类实现共56个导出方法及 `encodeMember`/`encodeValues`/`encodeMembers` 返回 `error`；`memory/list.go` 的 `Pop`/`PopLast` 在 `Store.Update` 闭包内吞掉反序列化错误                                                          | 同步切换为 `*apperr.Error`；`memory/list.go` `Pop`/`PopLast` 捕获闭包内解码失败并原样返回（不弹出元素，对齐Redis）；`redis/*` 区分 `goredis.Nil`（非失败）、`context` 控制错误（49900/50401）与Redis运行故障                                                                 | 不修改 `memory.Store` 无失败方法与 `Manager.Redis()/Memory()/Close()` 逃生口；保留条件写入 `NX/XX/KeepTTL/NoSlide` 语义；`redis/hash.go` `GetAll` 非string `F` 契约错误不静默返回空map；自定义测试 `failingCodec` 同步切换                                          |
| [cache 根包](../../plugins/cache/plugin.go) 与 [config.go](../../plugins/cache/config.go)：`pluginImpl.initErr` 与 `Config.validationErr` 为 `error`；`applyFromEnv` 使用 `fmt.Errorf` 拼接 `raw.Int()` 错误                                                                                                                     | `initErr` 与 `validationErr` 改为 `*apperr.Error`，多条配置错误用 `apperr.Join`；按R11将服务启动配置故障归一为 `CacheError`（50016）并保留内层 `VALUE_*` 原因                                                                                                                | `Plugin.Start/Stop` 已在P3强类型化，`Get/MustGet` 已强类型，仅回归；不将 `NoSQLPass` 密码写入安全描述或JSON；不要求 `driver/memory/redis` 反向导入根包                                                                                                              |
| [email](../../plugins/email/client.go)、[message.go](../../plugins/email/message.go) 与 [template.go](../../plugins/email/template.go)：`Client.Send/SendHTML/SendTemplate/AddTemplateDir/AddTemplateFS`、`Message.AttachFile`、`TemplateManager.Render/AddDir/AddFS`、`newTemplateManager/acquire/buildMailClient` 返回 `error` | 全部12个自有函数/方法改为返回 `*apperr.Error`；`Send/SendTemplate` 对 `context` 取消/超时经 `control.Classify` 归为49900/50401；`Start` 中 `EMAIL_PORT` 非法同时保留 `ErrInvalidPort` 与 `valueerr` 原因                                                                     | 固定外部边界（W07）：`watchedConn.Close() error`（`net.Conn`）与 `mail.WithDialContextFunc` 回调保留原生 `error` 签名，内部强类型核心+nil安全薄适配（成功显式 `nil`）；保留全部10个导出哨兵；严禁向 `ErrorMessage`/JSON 泄漏 `cfg.Password`；不新增 `email.Get` API |
| [cron/lifecycle_error.go](../../plugins/cron/lifecycle_error.go) 与 [plugin.go](../../plugins/cron/plugin.go)：生命周期方法已为 `*apperr.Error`，非控制失败暂用 `50000` 兜底                                                                                                                                                     | 移除 `FrameworkInternalError`（50000）临时兜底，接入获批 `CRON_*` 专属码；核对 `ErrNoActivator`/`ErrInvalidJobTarget`/`ErrInvalidJobSchedule`/`errSchedulerNotReady` 与 `gocron` 原因链                                                                                      | 不改 `gocron` 内部调度语义，`Quiesce`/`Stop` 排空等待与幂等逻辑不变；`ctx.Err()` 仍走49900/50401                                                                                                                                                                    |
| [async/plugin.go](../../plugins/async/plugin.go)、[task.go](../../plugins/async/task.go) 与 [lifecycle_error.go](../../plugins/async/lifecycle_error.go)：`Pool.Submit`、`Plugin.Submit`、`async.Submit`、`Task.Wait/WaitFor` 返回 `error`，生命周期暂用 `50000` 兜底                                                            | 5个自有入口统一改为 `*apperr.Error`；`Task.Wait/WaitFor` 的 `ctx.Err()` 经 `control.Classify(..., PhaseOperation)` 归为49900/50401；nil `*Task` 返回结构化错误而非panic；`async.Submit` 保留 `application.Get` 原码与 `ErrPoolClosed` 双cause；替换 `50000` 兜底为 `ASYNC_*` | 不改 `Task.Done/Ctx/Cancel/Panicked` 契约；保留 `ErrPoolClosed/ErrPoolFull/ErrNilTaskFunc/ErrNotInitialized/ErrInvalidPoolCapacity/ErrInvalidContext` 全部哨兵；异步断言全用通道屏障，不用sleep                                                                     |
| 仓内示例与消费者：[grpc-cron-async](../../examples/v2/grpc-cron-async/main.go)、[v2-http-db-cache](../../test/consumer/v2-http-db-cache/lifecycle_test.go)、[v2-sqlite-async](../../test/consumer/v2-sqlite-async/lifecycle_test.go)                                                                                             | `grpc-cron-async` 移除 `50000` 临时包装改为直接透传或 `apperr.From(&err.Code, ...)`；`v2-http-db-cache` 缓存错误分岐改 `apperr.From` 避免掩盖底层码；核对 `v2-sqlite-async` 断言与竞争安全                                                                                   | 不新增SDK模块，不升级第三方驱动或修改 `.pb.go`；无仓外V2消费者                                                                                                                                                                                                      |

##### P5错误码归属与容量提案（未占号，待用户裁决）

按2026-11-15核对的受控词表（96项active）：`500xx` 已用 `50000–50037`（38项），SDK `00–49` **仅余12个槽位**（候选从 `50038` 起）；`429xx` 余47槽（候选从 `42903` 起）；`503xx` 余47槽（候选从 `50303` 起）；`400xx` 余33槽（候选从 `40017` 起）。

依赖约束：`plugins/cache/memory` 与 `plugins/cache/redis` 均导入 `plugins/cache/driver`，而根包 `plugins/cache` 导入上述三子包；`driver`/`memory`/`redis` **不能反向导入 `plugins/cache` 读取 `CacheError`（50016）**，也不能复制或别名50016或借用50000。因此需仿照P4的 `plugins/database/dialect` 模式，为 `plugins/cache/driver` 设立子域前缀 `CACHE_DRIVER_`（由检查器按最长前缀匹配）。

以下为P5提案（均未登记，实施前必须再按spec0002 E01–E05核对全局最大号）：

| 方案档位                        | 拟定归属 / 前缀 / Go符号                                     | 拟定Output与安全Message                        | 拟定分组                                        | 负责失败与设计理由                                                                                                                                                  |
| ------------------------------- | ------------------------------------------------------------ | ---------------------------------------------- | ----------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **主推（4项500 + 1项可选429）** | `plugins/cache/driver` / `CACHE_DRIVER_` / `CodecFailed`     | `CACHE_DRIVER_CODEC_FAILED` / 缓存编解码失败   | 500（候选 `50038`）                             | `Codec.Marshal/Unmarshal`、`JSONCodec`、结构体/列表/集合编解码故障；与传输/存储故障分离                                                                             |
| 同上                            | `plugins/cache/driver` / `CACHE_DRIVER_` / `OperationFailed` | `CACHE_DRIVER_OPERATION_FAILED` / 缓存操作失败 | 500（候选 `50039`）                             | Redis命令异常、`GetAll` 非法泛型约束等数据面操作失败（`goredis.Nil` 不触发）                                                                                        |
| 同上                            | `plugins/cron` / `CRON_` / `SchedulerError`                  | `CRON_SCHEDULER_ERROR` / 定时调度器错误        | 500（候选 `50040`）                             | 替换 `cron/lifecycle_error.go` 50000 兜底，承接调度器配置/启停/未就绪失败，保留4哨兵                                                                                |
| 同上                            | `plugins/async` / `ASYNC_` / `PoolError`                     | `ASYNC_POOL_ERROR` / 异步任务池错误            | 500（候选 `50041`）                             | 替换 `async/lifecycle_error.go` 50000 兜底，承接池容量/关闭/未初始化/无效上下文失败                                                                                 |
| **待拍板项（`ErrPoolFull`）**   | `plugins/async` / `ASYNC_` / `PoolFull`                      | `ASYNC_POOL_FULL` / 异步任务池已满             | **429（推荐，候选 `42903`）** 或 503（`50303`） | 非阻塞 `ants.ErrPoolOverload` 并发削峰拒绝：归429可直映HTTP 429/gRPC ResourceExhausted且不占500稀缺槽位；备选可归503或并入 `PoolError`（500）                       |
| **复用不占号**                  | `plugins/email`（已有 `EMAIL_`）                             | 沿用 `EMAIL_ERROR`（50017）                    | 500（已有）                                     | 承接SMTP构建/发信、模板加载渲染与附件读取失败；`EMAIL_PORT` 等配置错保留 `valueerr` 原因；备选可另增 `EMAIL_TEMPLATE_ERROR`（500）或 `EMAIL_MESSAGE_INVALID`（400） |

按主推方案落地后，500组SDK `00–49` 将占用42槽，仍余8个槽位（`50042–50049`）留给P6或后续收口；若把 `PoolFull` 归429/503，还可在协议层避免把受控并发限流误报成500内部故障。

##### P5拟议分工与交付顺序

| 工作包                | 唯一写入范围                                                                          | 必须交付 / 准入                                                                                                                                                      |
| --------------------- | ------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| P5.0 Leader登记与守卫 | `spec/0002`、`.github/scripts/ci/check-error-codes.go` 及单测、各归属 `error_code.go` | 获批后集中分配正式编号；补齐 `plugins/cache/driver`、`plugins/cron`、`plugins/async` 白名单及 `CACHE_DRIVER_` 最长前缀正反例测试；先冻结码表再并行                   |
| P5.1 Cache 全模块     | `plugins/cache/**`（含 `driver`、`memory`、`redis`、根包）                            | 30个接口方法与56个实现方法原子切换；修复 `memory/list.go` `Pop/PopLast` 吞错；`failingCodec`、未命中真nil、`goredis.Nil` 非失败、`control` 499/504、脱敏及编译正反例 |
| P5.2 Email 模块       | `plugins/email/**`                                                                    | 12个自有出口切 `*apperr.Error`；`watchedConn.Close` 与 `dialCtxFunc` 固定签名薄适配及真nil/关闭回归；10哨兵、`EMAIL_PORT` 双cause、密码脱敏与 `control` 499/504      |
| P5.3 Cron 与 Async    | `plugins/cron/**`、`plugins/async/**`                                                 | `cron` 替换50000兜底并补GoDoc/映射；`async` 5入口切 `*apperr.Error`、`Task.Wait/WaitFor` 499/504、`async.Submit` 保留应用层原码、通道屏障与编译夹具                  |
| P5.L Leader集成与准出 | `examples/v2/grpc-cron-async/**`、`test/consumer/**`、协议映射测试、计划准出          | 移除示例50000临时包装、补P5新码HTTP/RPC协议回归、跑通全部单测/竞态/lint/独立消费者与 `check-v2-examples.sh` 两fixture，并记录外部Redis/SMTP未运行边界                |

注：P5.1、P5.2、P5.3（或拆分cron/async）属于不同Go module且无跨插件依赖，可在P5.0冻结词表后并行；不创建卡或提前派工。

##### P5实施验收门禁（本轮未运行）

实施阶段从仓库根执行（固定 `GOTOOLCHAIN=go1.25.11`）：

```bash
export GOTOOLCHAIN=go1.25.11
go test ./plugins/cache/... ./plugins/email/... ./plugins/cron/... ./plugins/async/... -count=1
go test -race ./plugins/cache/... ./plugins/email/... ./plugins/cron/... ./plugins/async/... -count=1
go vet ./plugins/cache/... ./plugins/email/... ./plugins/cron/... ./plugins/async/...
make error-codes-check
make test-scripts
make test
make race
make vet
make lint-go
make lint-docs
make check-examples
go test -race ./plugins/web-gin/httpresult/... ./plugins/grpc/grpcresult/... -count=1
bash .github/scripts/consumer/check-v2-consumers.sh
bash .github/scripts/consumer/check-v2-examples.sh --fixture http-db-cache --race
bash .github/scripts/consumer/check-v2-examples.sh --fixture sqlite-async --race
GOFLAGS=-mod=readonly go list -deps ./foundation/... ./plugins/cache/driver
gofmt -l plugins/cache plugins/email plugins/cron plugins/async examples/v2 test/consumer
git diff --check
```

依赖方向校验必须核实：`foundation` 不含 `application/plugins`；`plugins/cache/driver` 不含 `plugins/cache` 根包；四插件间无跨插件导入。Redis与Mailpit/SMTP真实集成用例仅按既有环境变量开关执行，未配置外部服务时如实记为默认跳过，严禁宣称通过外部集成验收；所有异步/并发回归禁用 `time.Sleep`，必须使用通道屏障同步。

##### P5停工与回滚条件

发现 `Codec` 或缓存泛型接口遗漏、`memory` `Pop/PopLast` 继续吞掉反序列化错误、`Task.Wait/WaitFor` 将 `context` 取消/超时误转为500、`watchedConn`/`dialCtxFunc` 成功返回非nil、邮件密码泄漏或 `driver` 反向导入根包时，立即停工修复，不留给P6。回退锚点为 `e4930971`；未经授权不执行 `git reset`、不修改历史或推送。

#### P5准出记录（2026-11-15，待用户审阅）

用户于2026-11-15确认主推错误码提案并批准P5实施。Leader先集中登记5项新码（`42903` 与 `50038–50041`）并更新词表守卫，再并行分派 `plugins/cache`、`plugins/email` 与 `plugins/cron`+`plugins/async` 三个互不重叠工作包，最后由Leader收尾 `email` 测试与类型检查、同步示例与消费者、编写协议映射回归并执行全量门禁。基线为 `e4930971ac9155bbf31429b36b66fcc062e58590`，以下检查均基于该HEAD加本次未提交改动（dirty=true）；没有提交、推送、PR或发布，P6–P7未启动。

本次交付：

- 词表96→101（旧96项六列逐项保真）：新增 `ASYNC_POOL_FULL`（42903）、`CACHE_DRIVER_CODEC_FAILED`（50038）、`CACHE_DRIVER_OPERATION_FAILED`（50039）、`CRON_SCHEDULER_ERROR`（50040）、`ASYNC_POOL_ERROR`（50041）；`email` 沿用 `EmailError`（50017）。500组SDK槽位（00–49）余8个（`50042–50049`）。
- `plugins/cache`：`Codec` 2法及 `JSONCodec`、4类泛型缓存接口28法与 `memory`/`redis` 56实现方法及内部编解码辅助切换为 `*apperr.Error`；修复 `memory/list.go` `Pop`/`PopLast` 闭包吞错（解码失败保留列表并原样返回错）；`redis` 保留 `goredis.Nil` 非失败语义、取消/超时走49900/50401、`GetAll` 非string键类型返回 `OperationFailed`；根包 `initErr`/`validationErr` 与 `applyFromEnv` 按R11保留 `VALUE_*` 原因且不泄 `NoSQLPass`。
- `plugins/email`：12个自有失败出口切换为 `*apperr.Error`；W07固定签名 `watchedConn.Close() error` 与 `mail.WithDialContextFunc` 保留原生签名+内部强类型与nil安全薄适配；保留10导出哨兵；`EMAIL_PORT` 解析失败保留 `ErrInvalidPort` 与 `valueerr`（40013）；`Send`/`SendTemplate` 取消/超时归49900/50401；严禁密码泄漏；同步 `client_test.go` 独立 `sendErr` 消除 `SA4023` typed-nil 比较隐患。
- `plugins/cron` 与 `plugins/async`：移除 `lifecycle_error.go` 50000 临时兜底（分别替换 `SchedulerError` 50040 / `PoolError` 50041）；`Pool.Submit`/`Plugin.Submit`/`async.Submit` 与 `Task.Wait`/`Task.WaitFor` 切换为强类型指针；池满映射 `PoolFull`（42903）；`async.Submit` 组件停止/不可用保留50005/50006与 `ErrPoolClosed` 双cause；nil `*Task` 返回50019，等待期取消/超时走49900/50401。
- 仓内调用方与协议回归：同步 `examples/v2/grpc-cron-async/main.go` 与 `test/consumer/v2-http-db-cache/lifecycle_test.go`（保留内层码）；修复 `test/consumer/v2-sqlite-async/lifecycle_test.go` `Submit` 复用外层 `err error` 造成的 typed-nil 接口真值假失败；新增HTTP/gRPC五码映射回归与 `TestP5PluginDomainOwnership` 守卫单测。

实际准出命令与结果（`GOTOOLCHAIN=go1.25.11`）：

| 命令                                                                                                                | 结果与边界                                                                             |
| ------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| `go test -race ./plugins/cache/... ./plugins/email/... ./plugins/cron/... ./plugins/async/... -count=1` 与 `go vet` | 最终复跑通过；外部Redis与SMTP/Mailpit测试按原环境开关跳过                              |
| `go test -race ./plugins/web-gin/httpresult/... ./plugins/grpc/grpcresult/... -count=1`                             | 最终复跑通过，包含五新增码（含42903 HTTP 429 / gRPC ResourceExhausted）                |
| `make test`、`make race`、`make vet`、`make lint-go`                                                                | 最终复跑通过，14模块；`client_test.go` 曾触发 `SA4023`，修复后全部重跑通过             |
| `make error-codes-check`、`make test-scripts`、`make lint-docs`                                                     | 文档写回后重跑通过；101项词表与源码双向一致                                            |
| `make check-examples`、`bash .github/scripts/consumer/check-v2-consumers.sh`                                        | 最终复跑通过；独立候选无本地replace，覆盖仓内受控消费范围                              |
| `bash .github/scripts/consumer/check-v2-examples.sh --fixture http-db-cache --race`                                 | 最终复跑通过，独立fixture竞态验证                                                      |
| `bash .github/scripts/consumer/check-v2-examples.sh --fixture sqlite-async --race`                                  | 最终复跑通过（捕获并修复 `Submit` typed-nil 假失败后稳定全绿）                         |
| `GOFLAGS=-mod=readonly go list -deps ./foundation/...` 与 `./plugins/cache/driver`                                  | 通过并检查输出：foundation无上层依赖，`cache/driver` 无 `cache` 根包反向依赖           |
| `gofmt -l`（P5变更目录/文件）、`git diff --check`                                                                   | 输出空，无格式或空白问题                                                               |
| `python3 /tmp/opencode/v2-p5/verify.py`                                                                             | 文档写回后通过：101项词表、旧96项保真、P5路径范围及相对链接/锚有效；不是P7全仓类型守卫 |

日志位于本机会话 `/tmp/opencode/v2-p5/gate-*.log`，汇总为 `gates.json` / `scope.json`。未运行：外部真实 Redis 与真实 SMTP/Mailpit 服务集成（按原开关默认跳过），不能宣称真实外部连接验收。以上不是整计划或看板卡完成，P5待用户审阅。

### P6：迁移HTTP/gRPC并验证对外语义

- 动作：强类型化Header/Binder/validator及自有解析辅助；生命周期已在P3同步切换。拆清自有强类型归一核心与最终status边界，按spec0003 R12为原生context/native status补齐信封；保留原生状态、显式文案和全部已有Details，已有BaseResponse不覆盖、不重复追加。
- 交付：SDK自有错误经HTTP/RPC保持Code/Output/安全文案；原生各非OK状态均覆盖缺信封补齐、已有信封去重、未知/冲突/不可解码Details原载荷保留、包装层文本不外泄、取消/超时与typed-nil。补充编码失败须可观测降级并保留原状态，不伪称信封已补齐。
- 准出：`go test -race ./plugins/web-gin/... ./plugins/grpc/... -count=1`；测试实际插件请求链，不复制适配实现。将上轮/tmp探针的有效场景转成仓库内持久测试，不依赖/tmp残留文件。
- 状态：已由用户批准完成并提交为 `7e456a7`（未推送）：`plugins/web-gin` 自有强类型出口收敛（零新增码）、`plugins/grpc` R12 原生状态与兼容信封补齐（`extractExplicitGRPCStatus` 零包装文本泄漏）与实链测试均已落地。

### P7：接入类型门禁、全量消费者及发布验收

- 动作：在独立 `.github/scripts/ci/error-contract/` 子目录实现类型与例外守卫，避免与现有单文件main脚本混编；接入规划目标 `make error-contract-check`、脚本测试驱动和quality流程。另行验证RC与正式发布链路的依赖版本、不可变tag/commit、Manifest及预发布标记；当前Release Action有写入/推送副作用，未经另行授权不直接触发。
- 交付：自有error返回/别名/匿名回调/错误字段/方法集绕过反例、白名单超范围/过期反例、原生成功nil与失败cause测试。检查器须基于实际类型；若引入go/packages等新工具依赖，先评审根工具模块变更，不能暗加SDK运行时依赖。
- 准出：规划的 `make error-contract-check`、`GOWORK=off go test -race ./.github/scripts/ci/error-contract` 通过；以及下述全量门禁。
- 状态：已完成 P7.1–P7.3 实施（提交为 `3db459a`，未推送），并完成 P7.4 发布离线沙盒演练与四容器真引擎集成验收；公开 RC / 正式打 tag 发布待另行授权。

#### P7准出记录（2026-11-19，待用户审阅）

经用户确认采用 `.github/scripts/ci/error-contract/go.mod` 独立工具模块引入 `golang.org/x/tools v0.44.0`（`go/packages`），不污染 14 个业务 SDK 模块与根 `go.mod`；完成 P7.1–P7.3 后按用户授权本地提交为 `3db459a61690244b02401768608dbacdfc0e6e75`（`XiaoLFeng <gm@x-lf.cn>`，未推送），随后基于该 `HEAD` 完成 P7.4 发布离线沙盒演练与四容器真引擎集成验收。

本次交付：

- 独立契约守卫 `.github/scripts/ci/error-contract/`（`go.mod`/`go.sum`/`main.go`/`checker.go`/`whitelist.go`/`checker_test.go`）：基于 `go/packages`（`GOWORK=<repo>/go.work`）覆盖全部 14 模块 / 41 包，检查函数/方法/接口/类型别名/自定义 error 接口/结构体字段/包变量/参数回调与匿名函数字面量；内置 `W01–W11` 与 `STRUCT-CAUSE` 共 80 项精确符号台账（`ValidateWhitelist` 显式拒绝 glob 通配符、重复条目与空说明）。
- 守卫自身 4 组 `-race` 单测（`2.377s–2.602s`）：`TestLiveRepoContractAndWhitelist`、`TestWhitelistValidationRejectsGlobsAndDuplicates`、`TestCheckerRejectsBypassPatterns`（14 类绕过与未使用白名单反例）及 `TestCompileNegativeUntypedErrorCannotSatisfyStrongSignature`。
- 源码顺手清零（3 处内部 `error` 助函数）：`foundation/logging/rotator.go` `(*RotatingWriter).addFileToTar` 迁移为 `*apperr.Error`；`plugins/web-gin/validator/custom.go` `addBuiltinTranslation` 迁移为 `*apperr.Error` 且 9 处 `W11` 翻译回调成功直返字面量 `nil`；`plugins/email/error.go` 移除 `controlSentinel(error)` 包装与 `"errors"` 导入，直走 `control.Classify`。
- CI 与脚本接线：`Makefile` 新增 `error-contract-check`；`.github/scripts/tests/run-all-tests.sh` `go` 目标同步以 `-race` 执行 `error-contract` 单测；`.github/workflows/quality.yml` 新增 `Check strong-error contract and whitelist` 步骤。

实际准出命令与结果（基于提交 `3db459a`，`GOTOOLCHAIN=go1.25.11`）：

| 命令 / 范围                                                                                                        | 实测结果与边界                                                                                                                  |
| ------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------- |
| `make error-codes-check` & `make error-contract-check`                                                             | 通过；词表 101 项一致；全库扫描 14 模块 / 41 包 \| 命中白名单 80 \| 未使用 0 \| 违规 0                                          |
| `(cd .github/scripts/ci/error-contract && GOWORK=off go test -race -v . && GOWORK=off golangci-lint run ...)`      | 通过；4 组 `-race` 测试全绿（2.602s），`golangci-lint` 0 issues                                                                 |
| `make vet`、`make lint-go`、`make lint-go-oracle`（等效 `make lint-go-all`）                                       | 通过；全部 14 模块 + `error-contract` 0 issues                                                                                  |
| `make race`（14 模块）、`make proto-check`、`make lint-docs`（21 文件）、`git diff --check`、foundation 反向依赖   | 通过；无竞态、Proto 一致、Docs 0 issues、foundation 零反向依赖                                                                  |
| `make test-scripts`（7 套件）、`make check-examples`（`readme`/`worker-tool`/`http-db-cache`/`grpc-cron-async`）   | 通过；7 个脚本套件 44s 全绿（含 `error-contract` `-race` 2.377s）；4 个示例 `GOWORK=off` 独立编译通过                           |
| `bash .github/scripts/consumer/check-v2-consumers.sh --flags '-race -count=1'` 与 `check-v2-examples.sh --fixture` | 通过；14/14 离线模块消费（`-race`）与 `sqlite-async`（1.661s）/ `http-db-cache`（1.627s）生命周期夹具全部通过，无本地 replace   |
| `bash .github/scripts/release/test-release-offline.sh`                                                             | 通过（基于 `3db459a`）；6/6 离线发布演练用例通过（14 模块清单、依赖替换、Manifest 准入、14 tag、防漂移重入、按需发版）          |
| `integration-env.sh up` → 真引擎 `-race` 集成 → `integration-env.sh down`                                          | 通过并零残留销毁：MySQL 8.4.4、PostgreSQL 17.4、Redis 7.4.2（`BAMBOO_TEST_REDIS_ENABLED=1`）、Mailpit v1.22.3 全部 `-race` PASS |

未验收 / 显式保留边界：

- **SQLServer**：按 `test/integration/README.md` 约定，ARM64 架构下保持隔离未验收状态。
- **Oracle**：无真实独立商业/企业测试环境，按约定保持显式 `SKIP`，未用 SQLite 或其它容器降级替代。
- **公开 RC 与正式发布**：`test-release-offline.sh` 已验证离线发布链路，但未向远端推送或触发 GitHub Release；实际 RC / 正式 tag 与外部下游验证仍待另行授权。

## 迁移与回滚

- 本次属于Go API破坏性迁移。保留旧 `error` 接口并仅写“返回值实际是统一错误”不算完成；若业务需要暂时兼容，必须在明确命名的边界适配器与限时移除计划中登记，不保留双套自有接口长期分叉。
- 以基线提交、完整测试证据及当前词表为回滚锚点。P3切换必须连同所有实现者、测试和示例回退，不能只回退application接口或只回退一个插件。执行Git回滚前另行获得授权，不默认reset、变基、强推或重写远程历史。
- 若新增错误码尚未发布，可随未交付补丁一起撤回；一旦对外发布，按spec0002保留占号并退役，不能删除台账后重新使用。
- 发现typed-nil、标准方法集丢失、取消变Internal或根因丢失时，停止该阶段并保留失败证据；修复后重跑完整相关范围。禁止跳过hook、删测试、关门禁或声称“类型能编译就完成”。
- 发布窗口已由R13确定：首个稳定V2前集中迁移全部已知V2试用消费者，公开RC验证后首批14模块v2.0.0。保持V1业务固定旧版的选择，不长期保留双套自有接口；每次RC/正式发布动作仍须另获授权，本文不授权提交、推送、建PR或发布。

## 验收与归档标准

- [x] P0技术核对：逐符号分类、固定方法集、仓内引用/实现者和受控消费者模板已展开；方向性问题已有用户裁决。
- [x] P0整体准出：2026-11-09用户明确批准P1开工，已确认无仓外V2消费者；基线技术台账准出，仓内C01–C06候选图与行为仍随阶段验收。
- [x] 自有错误结果全部为 `*apperr.Error`，固定签名只保留已核实薄适配；Code/Output不是统一通用500（`make error-contract-check` 14 模块 / 41 包 0 违规）。
- [x] Worker原错误、所有子cause、Is/As、成功nil、标准控制流及诊断账本全部有持久回归。
- [x] `make error-contract-check`、`make error-codes-check`、`make test-scripts`通过。
- [x] `make test`、`make race`、`make vet`、`make lint-go-all`、`make lint-docs`通过；缺少特殊驱动环境时记录实际范围和阻塞，不虚报全绿（SQLServer/Oracle 已显式记录未验收边界）。
- [x] `make proto-check`通过或确认生成源未改且按发布流程完成一致性验证；不手改生成文件。
- [x] `make check-examples`及 `bash .github/scripts/consumer/check-v2-consumers.sh`通过，独立消费者没有本地replace补救。
- [x] 真实外部环境验收与默认跳过清单分离，所有未通过项有明确去向（MySQL / PG / Redis / Mailpit 四容器 `-race` 已验收；SQLServer / Oracle 显式保留 SKIP）。
- [ ] 全部已知V2试用消费者在最终RC清单完成同步验证，正式稳定候选组合与公开独立取包通过，发布链路经人工准出且每次实际发布另获授权。
- [ ] 用户完成规范/实现及首发兼容性验收后，才可将计划改done并归档；方向性待确认清零和文档检查通过均不勾选以上实施项。

## 文件级排查台账

以下记录早期基线的60个词法候选文件及入口族，保留作为历史定位输入，不替代已经展开的P0逐符号类型台账，也不把只含error参数的函数计为待迁移返回。P0补齐了类型载体、跨行/嵌入方法、字段/通道、函数值和方法集；SnowflakeID.Scan等早期遗漏以P0现状表为准。

<!-- strong-error-candidates:start -->

| 基线文件                                             | error相关入口族（含原因入参/固定边界）                                                                                                                                                                                                                                   | 对应阶段 |
| ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------- |
| `application/activator.go`                           | `closeParent`、`waitParent`、`requestStopAll`、`cancelUndispatchedItem`、`Activate`                                                                                                                                                                                      | P2/P3    |
| `application/app.go`                                 | `RegisterTool`、`RegisterWorker`、`RegisterComponent`、`Run`、`recordRunErr`、`getRunErr`、`startStaticLayers`、`createPluginInstance`、`safeCallStart`、`safeCallStop`、`publishExposedKeys`、`launchWorkers`、`Shutdown`、`executeShutdownTopology`、`safeCallQuiesce` | P2/P3    |
| `application/builtin/logging.go`                     | `Start`、`Stop`                                                                                                                                                                                                                                                          | P2/P3    |
| `application/builtin/snowflake.go`                   | `Start`、`Stop`                                                                                                                                                                                                                                                          | P2/P3    |
| `application/component.go`                           | `isContextCanceledError`、`safeCallComponentStop`、`handleStopPanic`                                                                                                                                                                                                     | P2/P3    |
| `application/container.go`                           | `publishBatch`、`getEntry`                                                                                                                                                                                                                                               | P2/P3    |
| `application/context.go`                             | `Err`、`newAppErr`                                                                                                                                                                                                                                                       | P2/P3    |
| `application/errors.go`                              | `Is`、`Unwrap`                                                                                                                                                                                                                                                           | P2/P3    |
| `application/graph.go`                               | `buildGraph`、`validateComponentOverlap`                                                                                                                                                                                                                                 | P2/P3    |
| `application/option.go`                              | `resolveConfig`、`resolveField`                                                                                                                                                                                                                                          | P2/P3    |
| `application/plugin.go`                              | `Start`、`Stop`、`Run`、`Quiesce`、`Wait`                                                                                                                                                                                                                                | P2/P3    |
| `application/supervisor.go`                          | `Wait`、`computeWaitError`                                                                                                                                                                                                                                               | P2/P3    |
| `foundation/apperr/error.go`                         | `Unwrap`                                                                                                                                                                                                                                                                 | P1/P2    |
| `foundation/apperr/new.go`                           | `New`                                                                                                                                                                                                                                                                    | P1/P2    |
| `foundation/ctxkey/key.go`                           | `MarshalText`、`UnmarshalText`、`MarshalBinary`、`UnmarshalBinary`、`MarshalJSON`、`UnmarshalJSON`、`MarshalXML`、`UnmarshalXML`、`MarshalXMLAttr`、`UnmarshalXMLAttr`、`Value`、`Scan`、`Set`                                                                           | P1/P2    |
| `foundation/env/key.go`                              | `MarshalText`、`UnmarshalText`、`MarshalBinary`、`UnmarshalBinary`、`MarshalJSON`、`UnmarshalJSON`、`MarshalXML`、`UnmarshalXML`、`MarshalXMLAttr`、`UnmarshalXMLAttr`、`Value`、`Scan`、`Set`                                                                           | P1/P2    |
| `foundation/logging/command.go`                      | `New`                                                                                                                                                                                                                                                                    | P1/P2    |
| `foundation/logging/handler.go`                      | `Handle`、`writeConsole`、`writeFile`                                                                                                                                                                                                                                    | P1/P2    |
| `foundation/logging/rotator.go`                      | `NewRotatingWriter`、`Write`、`Close`、`openFileLocked`、`renameCurrentLogByDateLocked`、`rotateLocked`、`rotateForNewDayLocked`、`createTarGz`、`addFileToTar`、`checkStartupState`、`recordBgError`、`archiveOldFiles`                                                 | P1/P2    |
| `foundation/snowflake/snowflake.go`                  | `NewNode`、`Generate`、`MarshalJSON`、`UnmarshalJSON`、`MarshalBinary`、`UnmarshalBinary`、`Value`、`Scan`                                                                                                                                                               | P1/P2    |
| `foundation/toolkit/password.go`                     | `EncryptPassword`、`VerifyPassword`、`EncryptPasswordString`                                                                                                                                                                                                             | P1/P2    |
| `foundation/toolkit/time.go`                         | `Parse`                                                                                                                                                                                                                                                                  | P1/P2    |
| `foundation/typed/convert.go`                        | `MarshalText`、`UnmarshalText`、`MarshalBinary`、`UnmarshalBinary`、`MarshalJSON`、`UnmarshalJSON`、`MarshalXML`、`UnmarshalXML`、`MarshalXMLAttr`、`UnmarshalXMLAttr`、`Value`、`Scan`、`Set`                                                                           | P1/P2    |
| `foundation/typed/parse.go`                          | `Int`、`Int8`、`Int16`、`Int32`、`Int64`、`Uint`、`Uint8`、`Uint16`、`Uint32`、`Uint64`、`Float32`、`Float64`、`Bool`、`Duration`                                                                                                                                        | P1/P2    |
| `plugins/async/plugin.go`                            | `Submit`、`Start`、`Stop`、`doStop`                                                                                                                                                                                                                                      | P3/P5    |
| `plugins/async/task.go`                              | `Wait`、`WaitFor`                                                                                                                                                                                                                                                        | P3/P5    |
| `plugins/cache/driver/codec.go`                      | `Marshal`、`Unmarshal`                                                                                                                                                                                                                                                   | P3/P5    |
| `plugins/cache/driver/interface.go`                  | `Get`、`Set`、`Exists`、`Delete`、`GetAll`、`GetAllStruct`、`SetAll`、`SetAllStruct`、`Remove`、`Add`、`Members`、`IsMember`、`Count`、`Prepend`、`Append`、`Range`、`Index`、`Len`、`Pop`、`PopLast`                                                                    | P3/P5    |
| `plugins/cache/memory/hash.go`                       | `Get`、`Set`、`GetAll`、`GetAllStruct`、`SetAll`、`SetAllStruct`、`Exists`、`Remove`、`Delete`                                                                                                                                                                           | P3/P5    |
| `plugins/cache/memory/key.go`                        | `Get`、`Set`、`Exists`、`Delete`                                                                                                                                                                                                                                         | P3/P5    |
| `plugins/cache/memory/list.go`                       | `Prepend`、`Append`、`Range`、`Index`、`Len`、`Pop`、`PopLast`、`Remove`、`Delete`                                                                                                                                                                                       | P3/P5    |
| `plugins/cache/memory/set.go`                        | `encodeMember`、`Add`、`Members`、`IsMember`、`Count`、`Remove`、`Delete`                                                                                                                                                                                                | P3/P5    |
| `plugins/cache/plugin.go`                            | `Start`、`Stop`                                                                                                                                                                                                                                                          | P3/P5    |
| `plugins/cache/redis/hash.go`                        | `Get`、`Set`、`GetAll`、`GetAllStruct`、`SetAll`、`SetAllStruct`、`Exists`、`Remove`、`Delete`                                                                                                                                                                           | P3/P5    |
| `plugins/cache/redis/key.go`                         | `Get`、`Set`、`Exists`、`Delete`                                                                                                                                                                                                                                         | P3/P5    |
| `plugins/cache/redis/list.go`                        | `encodeValues`、`Prepend`、`Append`、`Range`、`Index`、`Len`、`Pop`、`PopLast`、`Remove`、`Delete`                                                                                                                                                                       | P3/P5    |
| `plugins/cache/redis/set.go`                         | `encodeMembers`、`Add`、`Members`、`IsMember`、`Count`、`Remove`、`Delete`                                                                                                                                                                                               | P3/P5    |
| `plugins/cron/plugin.go`                             | `Start`、`doStart`、`Run`、`Quiesce`、`Stop`、`doStop`                                                                                                                                                                                                                   | P3/P5    |
| `plugins/database/adapter/mysql/mysql.go`            | `Open`、`validateDSN`                                                                                                                                                                                                                                                    | P3/P4    |
| `plugins/database/adapter/oracle/oracle.go`          | `Open`、`validateDSN`                                                                                                                                                                                                                                                    | P3/P4    |
| `plugins/database/adapter/postgres/postgres.go`      | `Open`、`validateDSN`                                                                                                                                                                                                                                                    | P3/P4    |
| `plugins/database/adapter/sqlite/sqlite.go`          | `Open`                                                                                                                                                                                                                                                                   | P3/P4    |
| `plugins/database/adapter/sqlserver/sqlserver.go`    | `Open`、`validateDSN`                                                                                                                                                                                                                                                    | P3/P4    |
| `plugins/database/dialect/dialect.go`                | `Open`、`validateDriver`、`lookup`、`Lookup`                                                                                                                                                                                                                             | P3/P4    |
| `plugins/database/gormlog/gormlog.go`                | `Trace`                                                                                                                                                                                                                                                                  | P3/P4    |
| `plugins/database/models/base_entity.go`             | `BeforeCreate`、`BeforeUpdate`、`generateEntityID`                                                                                                                                                                                                                       | P3/P4    |
| `plugins/database/models/base_entity_soft_delete.go` | `BeforeCreate`、`BeforeUpdate`                                                                                                                                                                                                                                           | P3/P4    |
| `plugins/database/plugin.go`                         | `Start`、`Stop`、`closeSQLDB`                                                                                                                                                                                                                                            | P3/P4    |
| `plugins/email/client.go`                            | `Start`、`Stop`、`acquire`、`buildMailClient`、`Close`、`Send`、`SendHTML`、`SendTemplate`、`AddTemplateDir`、`AddTemplateFS`                                                                                                                                            | P3/P5    |
| `plugins/email/message.go`                           | `AttachFile`                                                                                                                                                                                                                                                             | P3/P5    |
| `plugins/email/template.go`                          | `newTemplateManager`、`Render`、`AddDir`、`AddFS`                                                                                                                                                                                                                        | P3/P5    |
| `plugins/grpc/grpcresult/response.go`                | `ToStatusError`                                                                                                                                                                                                                                                          | P3/P6    |
| `plugins/grpc/plugin.go`                             | `waitDrained`、`Start`、`Run`、`Quiesce`、`Stop`、`doStop`、`normalizeRPCError`                                                                                                                                                                                          | P3/P6    |
| `plugins/web-gin/bind/binder.go`                     | `New`                                                                                                                                                                                                                                                                    | P3/P6    |
| `plugins/web-gin/bind/binding.go`                    | `handleBindError`、`readBodyBytes`、`mapValuesToStruct`、`fillStructFields`、`setFieldValue`、`setSingleScalar`、`parseTimeValue`                                                                                                                                        | P3/P6    |
| `plugins/web-gin/header/util.go`                     | `GetAuthorization`                                                                                                                                                                                                                                                       | P3/P6    |
| `plugins/web-gin/plugin.go`                          | `WithValidatorRegistrar`、`waitDrained`、`Start`、`Run`、`Quiesce`、`Stop`、`doStop`                                                                                                                                                                                     | P3/P6    |
| `plugins/web-gin/validator/custom.go`                | `registerBuiltinCustomRules`、`第三方翻译注册回调`                                                                                                                                                                                                                       | P3/P6    |
| `plugins/web-gin/validator/engine.go`                | `New`、`Validate`、`RegisterValidation`、`TranslateError`、`HandleError`                                                                                                                                                                                                 | P3/P6    |
| `plugins/web-gin/validator/error.go`                 | `handleErrorInternal`、`parseBindingErrorDetail`                                                                                                                                                                                                                         | P3/P6    |

<!-- strong-error-candidates:end -->
