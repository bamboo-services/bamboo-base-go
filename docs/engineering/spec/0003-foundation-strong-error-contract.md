# v2.0.0 强类型错误返回与协议适配规范

> 状态：draft · 日期：2026-11-07 · Scope：foundation
> 承接：[错误码分配与受控词表规范](./0002-foundation-error-code-vocabulary.md)
> 实施：[强类型错误返回迁移计划](../../plan/0002-foundation-strong-error-migration.md)

## 适用范围与背景

SDK 自有失败出口必须以 `*apperr.Error` 表达，不能只依赖 `error` 接口和调用方约定。错误码目录唯一，不代表每个失败出口已经携带错误码。

本规范覆盖14个工作区模块中自有函数、方法、接口、回调及错误结果账本。Go 标准库和第三方固定方法集只允许下述薄适配例外；工具脚本、测试用故障注入及第三方实现不作为 SDK API。既有无错误返回的函数不因此新增返回值。原生对象逃生口不承诺改造第三方对象的方法。

### 现状、目标与证据

| 项目       | 基线 `aaf73c6` 的现状                                       | 本规范要求的目标                               |
| ---------- | ----------------------------------------------------------- | ---------------------------------------------- |
| 构造与 Get | `apperr.New`、`application.Get` 已返回统一指针              | 保持构造快照及根因语义                         |
| 自有出口   | 已实测22个无统一错误结构的公开反例                          | 自有错误结果位置静态声明为 `*apperr.Error`     |
| 协议识别   | HTTP/gRPC 的 `errors.As` 能识别链中已有结构；未知错误只兜底 | SDK失败在协议转换前就具有明确的 Code/Output    |
| Worker     | `application/app.go:948` 用 `%v` 将原始错误变成文字         | 归一化及汇总全程保留原错误和根因               |
| 标准接口   | 编解码、SQL、日志、网络和gRPC存在固定签名                   | 固定签名薄适配，防止 typed-nil，保留标准控制流 |
| 门禁       | 现有词表检查器不检查错误返回签名                            | 增加全库类型契约、例外台账和运行行为检查       |

上述是源码核查结论，不代表目标已经实现。本文只起草规范；Go实现、返回签名、错误码定义、CI均未因本规范而迁移。P0已展开[逐符号分类/固定签名/消费者模板](../research/0005-foundation-strong-error-p0-inventory.md)和[引用/接口实现者](../research/0006-foundation-strong-error-p0-callers.md)；临时探针不作为长期验收依赖。

2026-11-09仅补P1开工准备：19项候选码与16种原生RPC状态兼容提案统一见[spec0002](./0002-foundation-error-code-vocabulary.md#p1开工候选码2026-11-09未登记)，范围、持久测试和命令见[plan0002 P1](../../plan/0002-foundation-strong-error-migration.md#p1先稳定错误核心码映射和负向编译测试)。候选未占号，用户尚未批准具体提案或Go开工，规范继续draft。

## 规范条款

### R01：自有错误结果位置必须使用统一指针

SDK 自己控制的函数和方法只要返回失败结果，该结果位置就必须声明为 `*apperr.Error`，包括多返回值与具名返回值；禁止用 `error`、自定义通用错误接口、`any` 或泛型占位符退回最低公共类型。内部自有辅助函数、应用生命周期接口、组件激活/等待、缓存驱动接口与插件自有回调也适用，不能仅改公开函数最外层。

例如 `func Start(context.Context) *apperr.Error`、`func Get(...) (T, *apperr.Error)` 是目标形态。`application.Plugin`、`RunnablePlugin`、`Quiescer`、`RunHandle`、`plugins/async.Pool`、`plugins/cache/driver` 的缓存与 Codec 接口、`plugins/database/dialect.Adapter` 都是本库拥有的契约，不能因“像第三方接口”而豁免。接口、实现者、函数值和调用方必须同步迁移，因为 Go 返回类型没有协变。

### R02：签名强类型不能退化成统一的通用500

每个自有失败出口必须按所属域选择已登记的 Code/Output，并由统一错误构造器返回。禁止所有失败都套 `FrameworkInternalError` 以通过类型检查；禁止用裸 `&apperr.Error{}`、错误码零值或把原始文本当作机器语义。预期字段、参数、资源、配置、状态、依赖与操作错误必须分别确认责任和协议含义。

错误码继续由 [0002](./0002-foundation-error-code-vocabulary.md) 分配。当前65项不重新排序或复用，新增语义先评审归属、状态与容量，再登记；本轮不预占未批准的编号。`foundation/apperr` 仍不得重新成为协议/领域码全集，基础层不得反向依赖 application/plugins，插件不得为取码横向依赖。

### R03：原始错误是原因，不是自有失败的最终返回

`apperr.Error.Err`、构造器 `errs ...error`、`Unwrap() error` 及标准 `Is/As` 钩子必须继续容纳原生错误，禁止把它们的参数/结果一律替换为统一指针。这里的 `error` 是根因和语言互操作协议，不是自有API失败返回的豁免。

第三方失败、原生哨兵、context错误和其他诊断类型必须保留为可遍历根因，保证 `errors.Is` 与 `errors.As`。SDK自有调用链必须逐层传播已有统一错误；确需补充上层语义时，用新的统一错误保留原错误为原因。禁止 `%v`、字符串拼接或 `errors.New(err.Error())` 代替因果链。既有直接比较哨兵的调用方必须迁移到 `errors.Is`，不能以保留 `err == sentinel` 为由继续裸返回。

### R04：聚合结果与账本也必须具有明确错误结构

自有汇总函数、错误通道及面向调用方的错误结果字段，必须纳入强类型迁移。`RunResult` 的 Cancel/Factory/Start/Expose/Quiesce/Run/Stop 错误字段、`StuckInstance.Err` 与内部错误账本不能成为原生失败外泄的旁路。

多个失败可以用 `errors.Join` 存在统一错误的根因中，但自有函数不能直接以 Join 的原生结果作为最终返回。聚合必须有明确的总错误码，并保留每个子错误可遍历；禁止任意挑一个子错误的码掩盖“多个阶段失败”。保留 `HookPanicError`、`StopIncompleteError` 等诊断类型的 `errors.As` 可用性，不把诊断账本自动塞入对外 `Data`，以免泄漏堆栈、配置或原始错误文本。

### R05：固定签名例外必须按符号登记

只有真实实现标准库/第三方接口，或实际传给外部API的函数值，才能保留其固定 `error` 签名。例外必须登记“文件 + 接收器/符号 + 完整签名或精确方法集 + 外部契约 + 原因 + 动态返回策略 + 验证方法”；禁止按目录、方法名、`*_adapter.go` 文件名或所有导出函数批量豁免。

固定签名适配器必须把工作委托给强类型内部实现，或只负责最终的协议编码。除下述标准控制流与原生句柄边界外，其非 nil 失败动态值也必须是可提取的统一错误。不能将整个业务流程藏进例外函数，继续返回裸错误。新增外部集成须先登记和验证，不得靠未核实的“可能兼容第三方”申请例外。

### R06：统一指针跨入原生error时必须显式处理nil

强类型成功结果必须返回真正的 nil 指针。固定签名包装器或原生回调在返回、赋值、追加 `[]error`、写入 `chan error` 或传递给任何 `error` 类型入口（包括构造器 `errs ...error`）前，必须判断该指针是否为 nil；成功时返回或传递字面量 nil。禁止直接把可能为 nil 的 `*apperr.Error` 装入 `error`，因为接口的动态类型非空会导致 `err != nil`。

错误对象的 `Error()` 方法对 nil 安全，不能修复 typed-nil。测试必须覆盖编码成功不误报、GORM成功钩子不阻断、事务成功不误回滚、HTTP/gRPC成功不变成内部错误，以及聚合不产生只有 nil 指针的假失败。

### R07：标准控制流必须在错误归一前判定

`context.Context.Err()`、`Unwrap()` 以及标准库要求的原生控制结果保持其外部协议。`io.EOF`、正常停止时的 `http.ErrServerClosed`/`grpc.ErrServerStopped`、缓存键不存在等非失败结果，不得因统一错误而变成500，也不得包装成假失败。

自有 `Wait/Activate/Submit` 的取消或超时若需要返回失败，结果类型仍必须为统一指针，原生context原因保留供 `errors.Is`。已确认采用R11的独立 `foundation/control` 归属与499/408/504分组，不把控制语义包装成框架内部500。正常停服和其他非失败控制结果仍返回真正nil；业务“取消订单”等状态不等同于执行上下文取消。Go实现尚未迁移，不能把本条当作现状已通过。

### R08：协议层转换保留语义，不要求传输层仍是Go指针

HTTP 的 `httpresult.Error/AbortError` 和gRPC的转换入口接受统一错误，输出 Code/Output/安全文案并隔离底层原因。原生gRPC handler/interceptor 的非nil失败输出必须是 `status.Error` 等协议表达，不要求序列化后仍能 `errors.As` 为原Go指针；应验证 `BaseResponse` Details 的业务字段。

外部业务代码交给Gin的普通 `error`、gRPC原生status或context结果，必须经过明确的输入兼容策略。已有通用500兜底只保护未知输入，不能作为SDK自有出口合规证据。已确认按R12为原生gRPC status/context补齐BaseResponse，并保留原状态与显式文案；当前Go实现仍未迁移，不能把所选策略当作现有信封保证。

### R09：原生对象逃生口与SDK门面不能混淆

`database.Get` 返回原生 `*gorm.DB`、缓存管理器返回原生Redis客户端等逃生口时，获取操作的SDK失败必须强类型化，但对象后续第三方方法仍遵循其原生契约。禁止宣称一个 `Get` 签名就能保证所有GORM `.Error`、`Transaction` 或Redis命令结果自动统一。

需要统一保证的操作必须提供或明确使用SDK门面。强类型业务回调转为原生GORM事务回调时，也必须通过R06的显式nil适配；当前没有生产事务包装器，不以测试中的 `Transaction` 调用冒充现成功能。

### R10：合规必须同时检查签名、动态值和协议结果

迁移完成必须通过以下三层检查，任何一层不能代替另一层：自有API/接口/函数值/结果字段的类型守卫；固定签名例外、成功nil和失败根因的运行测试；HTTP信封/gRPC Details及状态映射测试。只搜索 `fmt.Errorf` 或只通过词表门禁不足以验收。

生产新增自有失败签名不得绕过守卫；编译测试必须证明普通 `errors.New` 无法作为自有失败结果返回，并检查返回别名、匿名回调和方法集的绕过。反例夹具可保留 `error`，但只能在测试夹具范围内；测试中的示例插件和消费者实现仍必须按新接口迁移。

### R11：取消、超时与值转换按已确认的归属和分组表达

2026-11-07用户已逐项确认以下规则。下表是目标契约，不是现有定义清单：`foundation/control` 和 `foundation/valueerr` 均为规划新增基础子包，本轮未创建；`xx` 仅表示以后按全局词表分配的SDK组内序号，不是已占用的Code。

| 场景                                             | 规划归属 / Output                                   | Code分组 | HTTP | gRPC             |
| ------------------------------------------------ | --------------------------------------------------- | -------- | ---- | ---------------- |
| 执行上下文取消，且自有API需要以失败返回          | `foundation/control` / `CONTROL_CANCELED`           | `499xx`  | 499  | Canceled         |
| 请求接收或读取超时                               | `foundation/control` / `CONTROL_REQUEST_TIMEOUT`    | `408xx`  | 408  | DeadlineExceeded |
| 应用内部操作的deadline到期                       | `foundation/control` / `CONTROL_DEADLINE_EXCEEDED`  | `504xx`  | 504  | DeadlineExceeded |
| 输入内容格式非法、数值越界、布尔值或枚举不可接受 | `foundation/valueerr` / `VALUE_*`                   | `400xx`  | 400  | InvalidArgument  |
| 不支持的目标类型或违反API调用契约                | `foundation/valueerr` / `VALUE_*`                   | `500xx`  | 500  | Internal         |
| 服务配置入口确认的配置错误                       | 所属框架或插件的配置错误码，保留底层 `VALUE_*` 原因 | `500xx`  | 500  | Internal         |

取消采用HTTP499是本项目明确选择的非标准扩展，不宣称它是IETF标准状态，也不据此判定取消一定由客户端断连触发；连接已断时不保证还能写回HTTP响应。内部deadline使用HTTP504是本项目跨协议约定，不宣称任意内部操作都具有HTTP网关角色；请求接收/读取超时不得与其混用。不得把所有5xx、网络失败或普通服务故障推断为超时。

转换底层只陈述“值格式/范围/目标类型”等事实，不凭字符串内容推断来自请求还是服务配置。请求边界直接传播400类值错误；部署/启动时读取服务自身配置的入口必须按服务配置责任使用所属500码，并把原始统一值错误保留为原因。只有该明确的责任边界可以重分类，不能沿调用栈每层随意改码，也不能把控制取消/超时误归为配置非法。
管理API提交配置时的请求校验仍按输入责任400处理，不因字段名含“配置”就归500；只有确认是在读取/建立服务自身配置的责任边界，才作配置500包装。

`VALUE_INVALID_FORMAT`、`VALUE_OUT_OF_RANGE`、`VALUE_UNSUPPORTED_TARGET` 是待词表评审的候选标识，不是本轮完成登记的变量名。错误分类须使用实际校验分支及可识别的原始类型/哨兵，禁止解析人读错误文本来猜分类；安全Message/ErrorMessage不得直接复制可能包含凭据的原始输入。

规划依赖为 `control/valueerr → apperr`，以及 `typed/ctxkey/env/toolkit → valueerr`（按实际需要导入）；不得出现 `apperr → control/valueerr/typed` 的反向依赖，不新增Go模块、不引入插件横向依赖。两个新归属在实施前必须同步登记spec0002、源码目录和检查器白名单，当前65项不重排、不占具体新号。

当前gRPC分组映射已支持400/408/499/500/504；本次没有选择422，不新增422到InvalidArgument的映射。本结论只说明分组规则可沿用，构造、归一化、错误链和协议Details仍必须在后续迁移中实现并测试。

### R12：原生gRPC输入补齐信封，但不重写原生语义

2026-11-08用户选择“全部补齐，保留原生语义”。该规则适用于SDK错误经桥接归一后的输出，以及错误链中可识别的原生context原因和合法非OK的原生gRPC status。nil成功仍原样返回nil，不因信封规则生成错误。框架内被显式处理的正常停服等非失败控制结果不进入此错误适配规则。

| 输入分支                                      | 目标状态与文案                                           | BaseResponse处理                                                          |
| --------------------------------------------- | -------------------------------------------------------- | ------------------------------------------------------------------------- |
| SDK `*apperr.Error`（含错误链内可提取者）     | 按已登记Code及既定协议规则转换，使用安全文案             | 保留其明确的Code/Output，不降为原生通用兼容码                             |
| 原生 `context.Canceled`（含可由Is识别的包装） | Canceled及固定安全文案，不复制包装层文本                 | 使用R11的 `CONTROL_CANCELED`，携带请求追踪信息                            |
| 原生 `context.DeadlineExceeded`               | DeadlineExceeded及固定安全文案                           | 有明确请求接收/读取阶段才用408控制码，内部操作或缺少该阶段证据用504控制码 |
| 合法非OK的原生status，缺少BaseResponse        | 保留原生 `codes.Code`、显式status.Message及所有原Details | 仅追加一个与原生状态类别相符的兼容信封，不猜原本不存在的领域错误          |
| 原生status已经携带BaseResponse                | 保留原状态、显式文案和全部原Details                      | 不覆盖、不补字段、不再追加；不能仅为换追踪ID改写调用方已有信封            |

已有Details必须按原始Any保留其顺序和内容，不能先全部解包再重新打包导致未知类型丢失。即使原Details中含多个、内容冲突或不可解码的BaseResponse，也不擅自删除/替换或再追加一个；该异常须通过诊断和测试暴露，不把“保留原载荷”宣称成“原信封有效”。对其他第三方Details类型不建立白名单过滤，不以BaseResponse替换它们。

原生status的Message作为调用方显式选择的协议文案保留。只取显式 `GRPCStatus` 提供的文案，禁止将未知包装层的 `err.Error()` 或原始cause文本自动抄入Message、ErrorMessage或Data。SDK自有错误仍须经过安全文案边界，不能先制造含敏感信息的原生status来绕过要求。

**兼容信封的码来源必须明确。**原生status只证明RPC状态类别，不证明发生的是数据库、缓存、认证逻辑或某个业务资源。缺少原信封时，仅能复用语义完全吻合的已登记通用RPC/控制码，或在P1新增通用RPC状态兼容码；禁止把所有NotFound反推成DATABASE_RECORD_NOT_FOUND，或把所有ResourceExhausted说成“请求过多”。具体符号和数值仍待全局词表登记，本轮不分配。

原生status是该分支的权威gRPC状态。补Details不能把它重新经Code/100映射一遍：正向映射不是一一对应，例如既有409→AlreadyExists、500→Internal，不能因此把原生Aborted/DataLoss改成另一状态。兼容信封的业务Code可采用对应HTTP语义分组，但原生RPC Code保持不变；此为已有原生状态输入的明确例外，不改变SDK统一错误的既定正向映射。

补充的新信封携带可用的请求追踪ID及既有调试耗时规则，Message与保留的安全协议文案保持一致；不伪造原始领域Data、内部诊断或并不存在的context哨兵。原生status恰为Canceled不代表原始原因就是 `context.Canceled`，不能为凑Is结果往原因链中制造该哨兵。

后续实现必须在P1/P6为全部合法非OK的标准 `codes.Code` 建立确定的兼容码/HTTP语义分类表，并测出“状态不改、已有Details不丢、缺信封只追加一个”。未经分类登记不能以通用500补信封掩盖缺口。原生编码/头部大小/传输失败不能被承诺为无条件成功；若补充Details本身无法编码，不得改写原状态或泄漏内部原因，应保留原status/Details并记录受控诊断，作为显式可观测降级，不宣称该响应满足已补齐信封。

本轮不新增Go包、Code或映射实现。`plugins/grpc/plugin.go` 当前context原生分支仍不带信封、native status仍透传；R12为待实施的目标，不以文档裁决冒充代码已通过。

### R13：首发前集中迁移，下游通过RC后发布首个V2正式版

2026-11-08用户确认：全部已知V2试用消费者必须在首个稳定V2正式发布前集中完成本次强类型接口迁移；先公开RC验证，再发布首批14模块的 `v2.0.0`，module/import继续使用 `/v2`。不按固定日历天数强行放行，不把SDK测试通过当作消费者已完成迁移。

**发布事实只作有时点的证据。**本轮只读查询 `origin` 全部tag及GitHub Release列表，未观察到V2标签或V2 Release，可见Release仍为V1。该结果支持将本次调整纳入首个V2正式版，但不证明仓外没有分支/伪版本消费者。实际发布前必须再次查询，若已出现V2正式模块版本或其他公开V2候选，先重新核对已有身份和兼容责任，不覆写任何已公开内容。
核查来源为本仓 `origin` 的只读 `git ls-remote --tags origin` 和 `gh release list --repo bamboo-services/bamboo-base-go --limit 100 --json tagName,isDraft,isPrerelease,publishedAt`；当时返回的Release记录未达到100条上限。公开入口为[仓库标签](https://github.com/bamboo-services/bamboo-base-go/tags)与[Release列表](https://github.com/bamboo-services/bamboo-base-go/releases)。这些查询不证明已盘点仓外私有消费者。

#### 升级窗口与已知消费者门禁

- 建立用户确认的受控消费者清单：工程、负责人、当前依赖/提交、目标精确版本组合、自定义Plugin/Codec/Adapter/函数值及验证结果。仓库内示例和测试只是清单的一部分，不能自动冒充全部仓外消费者。
- 当前V1业务可继续固定V1，不强制同一天迁移。已使用旧V2分支或伪版本的消费者须锁定旧完整候选，在首发窗口内迁移到同一已验证候选清单；禁止新旧方法集随意混搭。
- 不长期保留两套自有 `error` / `*apperr.Error` 接口。必须存在的固定签名薄适配仍按R05处理，不能以升级窗口为由扩大永久豁免。
- 全部已知V2试用消费者按精确模块图通过后，由用户确认首发准出；未完成任何已知消费者验收时不宣布集中同步完成，也不以Agent会话结束替代验收。无法自动枚举未知或离线消费者，必须如实记录覆盖范围。

#### 候选、RC与正式版本身份

1. 未公开候选先经隔离本地模块代理与 `GOWORK=off` 验证，禁止本地replace或Workspace掩盖接口不匹配。
   RC模块的require也必须指向可取的候选或已公开稳定依赖，不能遗留尚未发布的稳定v2.0.0依赖而仅靠本地工作区通过。
2. 公开RC使用 `v2.0.0-rc.N`（例如rc.1、rc.2），子模块tag按源码目录生成，完整清单记录每个精确版本和源码提交。内容或依赖组合变化时生成新的候选身份；已公开tag不得移动、覆盖或删后重发，RC也不例外。
3. RC用于已知消费者验证。需要多轮RC时，消费者必须在最终RC组合上重新完成受影响验证，不能拿较早RC的通过记录代替最终组合。
4. 最终通过且另获发布授权后，首批14模块发布各自 `v2.0.0`，附整体Release精确清单。正式tag是新的不可变身份，不是把RC标签挪到另一个提交。
5. 若候选go.mod中的RC依赖需要改为稳定依赖，即使生产Go源码相同，正式模块包的内容也已经改变；必须验证最终稳定候选清单与有效模块图，不能声称它与RC字节完全相同。公开取包还需考虑代理缓存延迟。

正式V2发布后，兼容修复用patch、兼容新增用minor；再次破坏已发布模块API必须采用新的主版本与合法module/import路径，不能用v2.x小版本或整体Release标题掩盖。当前发布清单检查器只接受V2模块路径/版本，未来新主版本还须另行调整发布工具，不能只改tag标题。本次首发统一14模块不等于每次后续发布都必须重打全部模块tag；后续节奏与当前脚本的差异须另行核验，不在本轮改实现。

**发布动作必须另获授权。**现有Release Action会改依赖、提交bump、创建14个子模块tag并推送，不能只因为正则接受RC格式就直接触发。实际RC前必须验证候选依赖、tag/提交身份、幂等/防覆盖、Manifest、预发布标记、推送副作用与公开独立取包；正式发布还必须通过最终清单验证。本文不修改Go、go.mod、CI、tag或Release，不把选择发布策略视为执行发布授权。

## 固定签名初始白名单

以下是基线已核实的外部契约候选登记，逐符号现状已在[P0台账](../research/0005-foundation-strong-error-p0-inventory.md)展开，尚未交付自动白名单或动态适配。实施时按精确符号和外部契约验证，不能将本表的“方法族”直接转成正则豁免。

| ID  | 已有文件/符号                                                                                                               | 强制契约与保留范围                                                                                   | 动态策略与必验项                                                        |
| --- | --------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| W01 | `foundation/apperr/error.go` 的 `(*Error).Unwrap`；`application/errors.go` 的 `(*HookPanicError).Unwrap`                    | `Unwrap() error` 标准错误链                                                                          | 原始原因直出；验证多层 Is/As，不另套外壳                                |
| W02 | `application/context.go` 的 `(*bridgedContext).Err`                                                                         | `context.Context.Err() error`                                                                        | 标准取消值保持；不能改成自有返回签名                                    |
| W03 | `foundation/typed/convert.go` 的 String、`foundation/ctxkey/key.go` 与 `foundation/env/key.go` 的 Key                       | Text/Binary/JSON/XML编解码（含XML属性）、`driver.Valuer.Value`、`sql.Scanner.Scan`、`flag.Value.Set` | 强类型核心 + nil安全薄适配；`typed.Convertible`嵌入标准接口不改方法集   |
| W04 | `foundation/snowflake/snowflake.go` 的 SnowflakeID                                                                          | 当前实际实现的JSON/Binary编解码、`driver.Valuer.Value`、`sql.Scanner.Scan`                           | 只列实际通道，不泛加其他 Marshal 方法；生成/校验核心强类型              |
| W05 | `foundation/logging/handler.go` 的 `(*LogHandler).Handle`                                                                   | `slog.Handler.Handle(context.Context,slog.Record) error`                                             | I/O失败带统一原因，成功返回字面量nil；内部write辅助函数不豁免           |
| W06 | `foundation/logging/rotator.go` 的 `(*RotatingWriter).Write/Close`                                                          | `io.Writer`、`io.Closer`                                                                             | 保留字节计数及关闭语义；其内部辅助函数强类型                            |
| W07 | `plugins/email/client.go` 的 `(*watchedConn).Close`、实际传给 `mail.WithDialContextFunc` 的拨号函数                         | `net.Conn.Close() error`、`func(context.Context,string,string)(net.Conn,error)`                      | 第三方网络边界，错误应规范化；不豁免 Client.Send/模板接口               |
| W08 | `plugins/database/models/base_entity.go`、`plugins/database/models/base_entity_soft_delete.go` 的 BeforeCreate/BeforeUpdate | GORM `func(*gorm.DB) error` 钩子                                                                     | 不能改成不匹配的方法集；内部ID生成强类型，成功钩子/实际回调必须运行验证 |
| W09 | `plugins/grpc/plugin.go` 的桥接器/链内handler与 `plugins/grpc/middleware/middleware.go` 的类型别名                          | gRPC Unary/Stream interceptor及handler                                                               | 最终转换为status；自有生命周期和错误归一核心不一并豁免                  |
| W10 | `plugins/grpc/grpcresult/response.go` 的 ToStatusError                                                                      | 对外gRPC状态编码适配器                                                                               | `*apperr.Error`输入、原生error输出；单独登记，不扩成通用自有业务接口    |
| W11 | `plugins/web-gin/validator/custom.go` 中实际传给翻译注册器的回调                                                            | 第三方translator注册回调                                                                             | 逐个调用点登记，初始化成功必须为真正nil                                 |
| W12 | `plugins/database/gormlog/gormlog.go` 的 `(*Logger).Trace`                                                                  | GORM `logger.Interface.Trace(context.Context,time.Time,func()(string,int64),error)`，无返回值        | 只保留第三方原生错误输入；不新增失败返回，不豁免其他自有错误出口        |

所有条目必须通过对应标准/第三方接口的编译期断言或函数值赋值验证。原生错误类型的 `Is/As` 方法、原因参数及标准库输入参数不属于“返回失败”扫描目标。当前未发现生产迭代器API，不为假想 `yield` 回调预建豁免。

## 正反例与边界

目标态自有接口：

```go
// 规划示例；本轮未修改实际接口。
type Worker interface {
    Start(context.Context) *apperr.Error
    Run(context.Context) *apperr.Error
    Stop(context.Context) *apperr.Error
}
```

固定签名薄适配的成功路径：

```go
// 规划示例：writeCore 是未来的强类型内部实现。
func (w *Writer) Write(p []byte) (int, error) {
    n, appErr := w.writeCore(p)
    if appErr != nil {
        return n, appErr
    }
    return n, nil
}
```

- 合规：自有返回 `*apperr.Error`，`errors.Is(appErr, 原生哨兵)` 仍成立，协议转换保留已登记码。
- 违规：强类型函数内部先 `fmt.Errorf("%v", err)` 再套通用500，原链与具体语义仍丢失。
- 违规：`func Native() error { return Strong() }`，其中 `Strong()` 成功返回nil指针；这会制造非nil接口。
- 违规：把 `BeforeCreate(*gorm.DB) error` 改为同名但返回 `*apperr.Error`，GORM可能不再识别钩子。
- 合法边界：`Error.Unwrap()` 返回标准错误原因，gRPC最终返回status，SDK不因此降低自有签名要求。

## 合规与后果

### 校验与暴露方式

当前可运行的是 `make error-codes-check`、各模块测试及 `make lint-docs`，它们不提供完整的强类型签名守卫。迁移计划中的 `make error-contract-check` 和独立 `.github/scripts/ci/error-contract/` 守卫目录均为规划，未创建、未执行，禁止把其预期效果写成已通过。

未来守卫必须扫描14模块生产签名、自有接口/回调、账本字段和例外台账，基于实际类型区分内置 `error` 与 `*apperr.Error`，不得仅靠方法名判断。编译失败夹具应在隔离测试中断言失败，不把非法代码放进正常构建路径。typed-nil、Worker丢链、标准方法集和协议映射的行为测试必须独立保留。

### 收益与代价

编译器能拒绝自有API直接返回普通错误，调用方可以静态依赖Code/Output；代价是Go方法集、函数值、实现者和消费者均需破坏性迁移。强类型不能阻止代码选错错误码，也不能自动改变原生对象的方法。例外适配和运行测试仍不可省略。

### 重评条件

新增外部接口需要扩大白名单、500分组新增码超出32个当前剩余SDK槽位、取消/超时语义无法保持，或已发布消费者不能同步升级时，必须先重评契约/容量/版本窗口。不能静默放宽为 `error`，也不能占用业务号段或关闭类型门禁应付兼容。

## 否决项

- 自有API保留 `error`，只约定动态值是统一错误：比强类型容易接入，但编译器无法保证用户要求。
- 全仓文本替换 `error` 为 `*apperr.Error`：会破坏标准接口、根因与gRPC传输方法集。
- 只新增一个万能500包装函数：类型一致但语义仍降为最低兼容性，无法直接按Code适配。
- 按函数名豁免所有 Marshal/Scan/Close/Start：同名不代表同一外部契约，会留下业务旁路。
- 靠 nil安全的 Error() 掩盖 typed-nil：字符串表现正常，接口是否为nil仍然错误。

## 待确认

用户已确认：自有错误返回采用统一指针、固定外部接口保留薄适配；R11控制/值错误归属与映射；R12原生gRPC输入补齐信封且保留原生语义；R13首个V2稳定版前集中同步已知消费者，RC验证后首批14模块发布v2.0.0。上述原待确认方向均已有裁决，不再反复询问。

**原列的方向性待确认项已清零，但本文仍为draft，未冻结ADR，计划也不因本次问答自动转ready。**用户仅授权本轮写回文档，未授权修改Go、执行迁移、派工或发布。

实施前还须完成的登记/核验工作：按已确认归属细化具体符号、分配全局序号并核算容量；完成原生非OK标准gRPC状态到兼容信封码的分类表；将P0逐符号固定签名/类型核对结果实现为带失效检查的自动门禁并验证动态语义；由用户补齐仓外受控消费者真实清单并逐项验收；验证RC和正式候选发布链路。P0技术台账/模板已交付不等于这些人工、实现或发布前置完成；其他错误类别和聚合主码仍按P1评审。
