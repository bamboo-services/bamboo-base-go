# v2.0.0 强类型错误返回迁移计划

> 状态：in-progress · 日期：2026-11-09 · Scope：foundation
> 承接：[强类型错误返回与协议适配规范](../engineering/spec/0003-foundation-strong-error-contract.md) · [错误码受控词表](../engineering/spec/0002-foundation-error-code-vocabulary.md)

## 目标与范围

从基线 `aaf73c684d2d108c0916bba1733ef8c608024831` 迁移到：SDK自有公开及内部失败结果的静态类型均为 `*apperr.Error`；固定外部方法集保留薄适配；HTTP/gRPC转换前已有明确的Code/Output，不能把通用500当作统一完成。

**2026-11-09用户已明确授权P1开工，并要求Leader派发minimax-m3.1-flash/max子代理。P0技术台账据此准出；P1实现及Leader准出检查已完成，只覆盖错误核心、19项新增码和纯分类/编译契约测试。P2–P7未执行，Worker丢链、生命周期与真实原生RPC信封接入仍未修。**不自动创建工作卡、提交推送或发布，不以P1代替整计划完成。

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
- 状态：未执行；Worker目前仍有基线缺陷。

### P3：原子切换application生命周期契约与全部实现者

- 动作：同步改自有Plugin/RunHandle/Activator等接口、内核辅助返回、错误账本/通道、builtin、全部上层插件模块的实现方法、所有测试替身及示例消费者。不能仅改接口后单独交付。
- 交付：直接返回原生error不再编译的契约断言；RunResult字段、错误聚合及StopIncomplete/HookPanic诊断保留测试；启动、回滚、自然排空和清理语义等价证据。
- 准出：`make test`、`make race`、`make vet`、`make check-examples`，以及 `bash .github/scripts/consumer/check-v2-consumers.sh` 全部通过。已确认的控制语义若未在实现和测试中保留，或任何实现者仍使用旧方法集，本阶段不能准出。
- 状态：未执行；允许与P4/P5/P6在同一兼容性切换补丁中落地，阶段只是验证顺序，不要求发布不可编译的中间提交。

### P4：迁移数据库及五适配器的完整错误边界

- 动作：同时修改dialect.Adapter、5个Open/DSN校验/延迟错误字段及database调用方；自有模型辅助强类型。GORM hooks仍为固定签名，显式nil适配，保留雪花和GORM原因。
- 交付：5驱动空DSN和格式拒绝测试、成功钩子确实执行且无假失败的SQLite回归、原生gorm句柄边界说明；事务适配如获授权单列门面，不误称已有。
- 准出：`go test ./plugins/database/... ./plugins/database/adapter/mysql/... ./plugins/database/adapter/postgres/... ./plugins/database/adapter/sqlite/... ./plugins/database/adapter/sqlserver/... ./plugins/database/adapter/oracle/... -count=1`；复跑独立消费者。
- 状态：未执行；真实Oracle/SQL Server等集成条件不足时记录阻塞，不能标记真实服务验收通过。

### P5：迁移缓存、邮件、Cron和Async自有操作

- 动作：同时改cache自有Codec与四种泛型接口/双后端，保留键缺失非失败语义；改邮件发送、模板、附件及生命周期；改Cron与Async的状态/准入/取消/等待，保留原因和并发终态契约。
- 交付：编解码失败的模块码、发送与模板失败的模块码、pool准入/状态/超时类型测试；外部拨号和net.Conn适配的成功nil/关闭测试。标准context分支不误映射到Internal。
- 准出：`go test -race ./plugins/cache/... ./plugins/email/... ./plugins/cron/... ./plugins/async/... -count=1`。Redis/SMTP真实集成另记环境与实际执行结果；所有异步断言用通道屏障，不用sleep代替同步。
- 状态：未执行。

### P6：迁移HTTP/gRPC并验证对外语义

- 动作：强类型化Header/Binder/validator及自有解析辅助；生命周期已在P3同步切换。拆清自有强类型归一核心与最终status边界，按spec0003 R12为原生context/native status补齐信封；保留原生状态、显式文案和全部已有Details，已有BaseResponse不覆盖、不重复追加。
- 交付：SDK自有错误经HTTP/RPC保持Code/Output/安全文案；原生各非OK状态均覆盖缺信封补齐、已有信封去重、未知/冲突/不可解码Details原载荷保留、包装层文本不外泄、取消/超时与typed-nil。补充编码失败须可观测降级并保留原状态，不伪称信封已补齐。
- 准出：`go test -race ./plugins/web-gin/... ./plugins/grpc/... -count=1`；测试实际插件请求链，不复制适配实现。将上轮/tmp探针的有效场景转成仓库内持久测试，不依赖/tmp残留文件。
- 状态：未执行。

### P7：接入类型门禁、全量消费者及发布验收

- 动作：在独立 `.github/scripts/ci/error-contract/` 子目录实现类型与例外守卫，避免与现有单文件main脚本混编；接入规划目标 `make error-contract-check`、脚本测试驱动和quality流程。另行验证RC与正式发布链路的依赖版本、不可变tag/commit、Manifest及预发布标记；当前Release Action有写入/推送副作用，未经另行授权不直接触发。
- 交付：自有error返回/别名/匿名回调/错误字段/方法集绕过反例、白名单超范围/过期反例、原生成功nil与失败cause测试。检查器须基于实际类型；若引入go/packages等新工具依赖，先评审根工具模块变更，不能暗加SDK运行时依赖。
- 准出：规划的 `make error-contract-check`、`GOWORK=off go test -race ./.github/scripts/ci/error-contract` 通过；以及下述全量门禁。两个规划命令本轮不存在，不能以当前执行失败或未执行当作实施成果。
- 状态：未执行，自动扫描器与白名单清单均未创建。

## 迁移与回滚

- 本次属于Go API破坏性迁移。保留旧 `error` 接口并仅写“返回值实际是统一错误”不算完成；若业务需要暂时兼容，必须在明确命名的边界适配器与限时移除计划中登记，不保留双套自有接口长期分叉。
- 以基线提交、完整测试证据及当前词表为回滚锚点。P3切换必须连同所有实现者、测试和示例回退，不能只回退application接口或只回退一个插件。执行Git回滚前另行获得授权，不默认reset、变基、强推或重写远程历史。
- 若新增错误码尚未发布，可随未交付补丁一起撤回；一旦对外发布，按spec0002保留占号并退役，不能删除台账后重新使用。
- 发现typed-nil、标准方法集丢失、取消变Internal或根因丢失时，停止该阶段并保留失败证据；修复后重跑完整相关范围。禁止跳过hook、删测试、关门禁或声称“类型能编译就完成”。
- 发布窗口已由R13确定：首个稳定V2前集中迁移全部已知V2试用消费者，公开RC验证后首批14模块v2.0.0。保持V1业务固定旧版的选择，不长期保留双套自有接口；每次RC/正式发布动作仍须另获授权，本文不授权提交、推送、建PR或发布。

## 验收与归档标准

- [x] P0技术核对：逐符号分类、固定方法集、仓内引用/实现者和受控消费者模板已展开；方向性问题已有用户裁决。
- [x] P0整体准出：2026-11-09用户明确批准P1开工，已确认无仓外V2消费者；基线技术台账准出，仓内C01–C06候选图与行为仍随阶段验收。
- [ ] 自有错误结果全部为 `*apperr.Error`，固定签名只保留已核实薄适配；Code/Output不是统一通用500。
- [ ] Worker原错误、所有子cause、Is/As、成功nil、标准控制流及诊断账本全部有持久回归。
- [ ] `make error-contract-check`（规划）、`make error-codes-check`、`make test-scripts`通过。
- [ ] `make test`、`make race`、`make vet`、`make lint-go-all`、`make lint-docs`通过；缺少特殊驱动环境时记录实际范围和阻塞，不虚报全绿。
- [ ] `make proto-check`通过或确认生成源未改且按发布流程完成一致性验证；不手改生成文件。
- [ ] `make check-examples`及 `bash .github/scripts/consumer/check-v2-consumers.sh`通过，独立消费者没有本地replace补救。
- [ ] 真实外部环境验收与默认跳过清单分离，所有未通过项有明确去向。
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
