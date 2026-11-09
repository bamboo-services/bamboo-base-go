# v2.0.0 错误码分配与受控词表规范

> 状态：draft · 日期：2026-11-06 · Scope：foundation
> 承接：[foundation 能力与测试迁移规范](./0001-foundation-capability-and-test-inventory.md)

## 适用范围与背景

V2 SDK 的各插件曾独立选择错误码，HTTP 与 gRPC 出现相同 Code 和 Output；application 又把组件装配失败表达为客户端 400/403/404。统一分配编号不代表把所有定义搬进 foundation。

本规范管辖 `foundation/apperr` 与 `plugins/` 中对外导出的预置结构化错误码、协议转换及其检查门禁。本文不单独规定全部SDK函数的返回签名；自有错误返回强类型要求及固定外部接口例外见[强类型错误返回规范](./0003-foundation-strong-error-contract.md)。原生错误可作为根因保留并支持errors.Is，但不能据此认为自有出口继续裸返回error已符合新规范。本规范不替业务应用自动解释身份认证或生成支付、验证码等领域错误，尚未冻结为ADR。

### 工作树现状与边界

| 项目     | 当前工作树实施内容                                          | 验证与非目标                                                     |
| -------- | ----------------------------------------------------------- | ---------------------------------------------------------------- |
| 归属     | 框架12码、控制3码、值5码，协议及领域插件合计84项            | 以 make error-codes-check 和所属包测试为准，不以文档状态代替测试 |
| 唯一性   | 本文有序词表 + 源码 AST 守卫，接入 Go quality job           | 检查本仓预置码，不宣称能扫描仓外全部服务                         |
| 生命周期 | 定义装配、钩子、关机及意外退出的框架诊断码；流程接入留P2/P3 | 不把业务资源404或用户无权限403改称框架错误                       |
| 发布     | 本次允许删除旧名并重排 Code/Output                          | 下游须同步升级；不提供旧编号别名或隐式双协议兼容                 |
| 未来     | 新域和新错误按追加/退役规则评审登记                         | 草案不意味着后续重编已获授权                                     |

## 规范条款

### E01：必须先登记，再定义

所有 SDK 预置错误必须先登记本文“错误码受控词表”，且只能在归属包的 error_code.go 中声明。词表是编号、Output、中文 Message、归属及符号名的唯一分配台账；源码是被检查的实现，禁止在第二份手写 JSON 或独立服务表中再次分配同一编号，以免出现多源分叉。新增包必须先在“归属词表”与检查器白名单登记，不得绕过检查。

### E02：框架、协议、领域各自拥有语义

foundation/apperr 的预置码必须使用 Framework 开头的导出名、FRAMEWORK_ 开头的 Output，且只能处于500分组。容器未挂载、依赖未声明、组件不存在/类型不符/停止/不可用属于服务装配与运行故障，禁止以通用参数缺失、权限不足或业务资源404代替，因为责任在服务端而非请求用户。

HTTP 请求、绑定、校验、无响应兜底归 web-gin 的对应子包；RPC metadata、handler 返回值及错误兜底归 grpcresult；数据库、缓存、邮件的错误归各自插件。foundation 不得导入 application/plugins，插件不得为复用错误码横向依赖其他插件。通用“内容不存在”不保留模糊 NotExist 别名：业务资源选所属协议的 NotFound，组件缺失选 FrameworkComponentNotFound。

### E03：Code 必须全库唯一，不能只保证包内唯一

SDK 错误码必须使用恰好5位、无前导零及无分隔符的十进制字面量，禁止八进制/十六进制写法：Code = HTTP状态分组 × 100 + 序号。分组必须处于400..599；序号00..49保留给本 SDK，50..99留给下游业务。HTTP 与 RPC 即使语义相近，也必须取得不同 Code，并独立使用 `HTTP_` / `GRPC_` Output，禁止重新声明相同三元组，以便仅凭信封也可定位错误来源。

成功信封的200不参与错误词表；负数、零和非400..599分组不得分配为SDK错误码。协议对非法自定义值的既有安全兜底仍保留，不因词表规则把 apperr.New 改成有状态注册器或拒绝构造。

### E04：分组内连续追加，禁止插号重排

词表必须按 Code 数值升序排列。本次首次编排按 FRAMEWORK → HTTP → HTTP_BIND → HTTP_VALIDATOR → GRPC → DATABASE → CACHE → EMAIL 的顺序，在每个分组从00起连续分配；该顺序只用于本次初始化，不是以后重新生成编号的排序键。

后续新增错误必须取目标分组中“包括退役项在内的最大序号 +1”，禁止按包内最大值取号、在中间插号、因排序变动重编旧条目，或从其他分组借号伪装状态。一个分组达到49时必须先重新评审容量与协议格式；不得侵占业务50..99号段。

删除定义必须保留原词表行并将状态改为 retired，源码删除相应声明；Code/Output 永不复用。重命名 Output、改变含义或跨分组改变状态必须分配新条目并退役旧项。Message 的兼容性由评审确认，不得用同一标识承载另一种失败原因。本次重构前目录尚未按此规则冻结，用户明确授权的首次整体重编是唯一历史豁免；发布后不得援引本次豁免再次重排。

### E05：Output 必须带唯一所属域前缀

Output 必须匹配大写蛇形格式：所属域前缀 + 具体失败原因。完整 Output 必须全库唯一，前缀按下面的归属词表匹配（更长前缀优先）；禁止把 `HTTP_BIND_` 借给 httpresult、以拼写变体 `FARMWORK_` 另造框架域，或新增无前缀的 NOT_FOUND。

Go 符号在所属包内唯一即可；不同包允许同名 NotFound。Message 必须为非空中文说明，不承担机器匹配职责。声明使用具名 Code/Output/Message 字段及直接说明内容的行尾注释，禁止在注释重复变量名，禁止修改共享预置指针所指的值。 生产源码禁止点导入受控错误码包，也禁止在目录外用包级变量重导出预置码；普通命名导入别名不受限制。

### E06：状态来自分组，不能靠新序号猜测

HTTP 状态取 Code / 100；非400..599的自定义错误退为500。RPC 沿用 grpcresult 的明确映射：400→InvalidArgument，401→Unauthenticated，403→PermissionDenied，404→NotFound，408/504→DeadlineExceeded，409→AlreadyExists，412→FailedPrecondition，429→ResourceExhausted，499→Canceled，500→Internal，501→Unimplemented，502/503→Unavailable，其余分组→Unknown。禁止用简单全局整数递增使除100后的状态改变。
上述是SDK统一错误到RPC的正向映射，不是从原生RPC状态反推业务原因的一一对应关系。2026-11-08已确认 [spec0003 R12](./0003-foundation-strong-error-contract.md)：原生合法非OK status补BaseResponse时保留原RPC状态/显式文案/已有Details；已有信封不重写，缺信封只追加兼容信封。P1已登记完整标准状态分类及其号码，不把兼容业务Code重新正向映射后覆盖原RPC状态。既有正向映射不变；新增分类不等于P6原生信封链路已接入。

所有 `FRAMEWORK_` 故障经 HTTP 返回500、经 RPC 返回 Internal；HTTP/RPC 自身的 InternalError、ResponseMissing 必须使用各自协议定义。RPC 无 metadata 或键无有效值使用400组的 GRPC_METADATA_MISSING，不得推断为未认证；nil RPC context 属于内部调用契约错误，使用500组的 GRPC_CONTEXT_MISSING。SQL事务整体失败使用 DATABASE_TRANSACTION_FAILED 的500分组，具体并发/约束冲突使用409分组，不把全部驱动原始错误自动改写为这些码。

### E07：下游业务号段不是跨服务唯一性的自动保证

业务自定义码使用状态分组中的50..99，并使用业务域前缀（例如 SHOP_ORDER_CLOSED=40950）。同一产品中的所有服务必须共享它们自己的中央台账，禁止每个服务各自从50开始而宣称全局唯一；服务数量或容量超出预留范围时必须升级格式而非继续碰撞。本仓守卫不检查仓外台账，也不提供分布式编号服务。

单元测试中为覆盖非法值回退而构造的临时字面量不算SDK预置码；这仅豁免 _test.go 与测试夹具，不豁免生产包级临时变量。面向用户的示例必须使用业务号段和业务前缀，不应鼓励复制SDK编号。

### E08：变更必须同时更新调用方和证据

修改词表或定义必须在同一补丁更新生产调用、快照、示例、对应状态映射测试及现行说明；必须执行 make error-codes-check、make test、make vet 和受影响竞态回归。守卫必须拒绝重复 Code/Output、缺项、多项、错误归属、非具名字面量、非法序号和词表漂移，不得只在每个包内单独检查唯一性。

errors.Is/errors.As 根因、JSON不含底层Err、HTTP信封/RPC Details脱敏不随重编号改变。禁止为使测试通过删除状态断言或吞掉原始错误。历史已归档计划与旧版测试记录不回写为本次新值；现行说明必须明确引用本词表。

### 归属词表

| 归属包                       | Output前缀        | 当前active数量 |
| ---------------------------- | ----------------- | -------------- |
| `foundation/apperr`          | `FRAMEWORK_`      | 12             |
| `foundation/control`         | `CONTROL_`        | 3              |
| `foundation/valueerr`        | `VALUE_`          | 5              |
| `plugins/web-gin/httpresult` | `HTTP_`           | 26             |
| `plugins/web-gin/bind`       | `HTTP_BIND_`      | 1              |
| `plugins/web-gin/validator`  | `HTTP_VALIDATOR_` | 2              |
| `plugins/grpc/grpcresult`    | `GRPC_`           | 25             |
| `plugins/database`           | `DATABASE_`       | 8              |
| `plugins/cache`              | `CACHE_`          | 1              |
| `plugins/email`              | `EMAIL_`          | 1              |
| `foundation/snowflake`       | `SNOWFLAKE_`      | 2              |
| `foundation/logging`         | `LOGGING_`        | 4              |
| `foundation/toolkit`         | `TOOLKIT_`        | 1              |

### P1登记记录（2026-11-09）

用户于2026-11-09明确授权按P1开工准备开展实施。本补丁按E04追加19项，当前active由65项变为84项，旧65项的Code/Output/所属包/符号/Message全部保留。500组从18项变为26项，SDK槽位余24个；400组从13项变为17项，余33个。不占业务50..99，不把原生RPC状态反向猜成领域故障。

新域 `foundation/control`/`CONTROL_` 与 `foundation/valueerr`/`VALUE_` 均属于原foundation模块，登记下方机器词表及检查器白名单，不新增module或反向依赖。取消499→Canceled；明确请求接收/读取超时408、内部deadline504→DeadlineExceeded。输入值非法400→InvalidArgument；转换目标/接收器契约非法500→Internal；服务自身配置仍由调用方所属500码包装并保留值错误原因。不采用422方案，不改变既有422正向映射。

499为非标准扩展，内部deadline默认504是本项目约定，连接断开时不承诺能够写回。VALUE_INVALID_TYPE指输入源类型不合法；VALUE_UNSUPPORTED_TARGET和VALUE_INVALID_RECEIVER分别表示目标/API与接收器契约错误，不能靠错误文本猜责任。FrameworkMultipleFailures只作框架多阶段失败显式总码，不成为所有插件的通用失败桶。

19项的完整三元组只保留在下方机器词表，不再维护第二份候选分配表。4项框架新码在P1建立定义和聚合诊断原语测试；Worker/钩子/停机流程接入仍留P2/P3。其他日志、雪花、调度、任务池细分类仍按对应阶段登记，本次84项不是全部SDK失败出口已经归一化的证明。

### P2登记记录（2026-11-09）

用户于2026-11-09批准P2按提案追加7项，当前active由84项变为91项；旧84项逐字保留，500组从26项变为33项，SDK槽位余16个。新增 `foundation/snowflake`（2项）、`foundation/logging`（4项）、`foundation/toolkit`（1项）三个归属域及检查器白名单，均在既有foundation模块内，不新增module。

值转换失败复用已批准的valueerr码（格式/范围/类型非法归400组，不支持目标与无效接收器归500组），不重复占号；snowflake的nil node/nil receiver调用按无效接收器（50019）归一并保留原哨兵根因，时钟回退与NewNode参数非法使用SNOWFLAKE_码；logging处理器缺失、配置非法、写入与轮转失败使用LOGGING_码；密码加密或校验失败使用TOOLKIT_PASSWORD_FAILED并保留bcrypt原始错误为根因。各出口的强类型迁移已于2026-11-09按[plan0002 P2准出记录](../../plan/0002-foundation-strong-error-migration.md#p2准出记录2026-11-09待用户审阅)完成并通过全量门禁（91项词表、14模块测试、示例与独立消费者全绿）；开工依据与分工见[plan0002 P2开工准备包](../../plan/0002-foundation-strong-error-migration.md#p2开工准备包2026-11-09)。

### 原生gRPC缺信封时的P1兼容分类

下表覆盖16种标准非OK状态，只在缺BaseResponse时选择兼容码。取码只引用当前机器词表；P1仅实现纯分类契约，不接管真实拦截器。原生RPC状态、显式GRPCStatus文案和原Any Details仍按spec0003 R12保留，禁止把兼容Code重新正向映射覆盖原RPC状态。已有BaseResponse（包括重复、冲突或不可解码条目）不重写、不追加；问题进入受控诊断，不冒称有效信封。

| 原生RPC状态        | 兼容Output                                              | HTTP分组  | 取码依据与边界                                              |
| ------------------ | ------------------------------------------------------- | --------- | ----------------------------------------------------------- |
| Canceled           | `CONTROL_CANCELED`                                      | 499       | 本轮登记；原生status不等于context.Canceled原因，不伪造cause |
| Unknown            | `GRPC_UNKNOWN_ERROR`                                    | 500       | 本轮登记；不猜具体业务原因                                  |
| InvalidArgument    | `GRPC_PARAMETER_ERROR`                                  | 400       | 当前active；不猜具体字段                                    |
| DeadlineExceeded   | `CONTROL_DEADLINE_EXCEEDED` / `CONTROL_REQUEST_TIMEOUT` | 504 / 408 | 默认504；实际请求接收/读取阶段证据才408                     |
| NotFound           | `GRPC_NOT_FOUND`                                        | 404       | 当前active；不反推数据库记录缺失                            |
| AlreadyExists      | `GRPC_CONFLICT`                                         | 409       | 当前active；保留AlreadyExists                               |
| PermissionDenied   | `GRPC_PERMISSION_DENIED`                                | 403       | 当前active；不猜认证失败                                    |
| ResourceExhausted  | `GRPC_RESOURCE_EXHAUSTED`                               | 429       | 本轮登记；不等于只表示请求过多的TooManyRequests             |
| FailedPrecondition | `GRPC_PRECONDITION_FAILED`                              | 412       | 本轮登记；不把所有前置条件窄化成InvalidState                |
| Aborted            | `GRPC_ABORTED`                                          | 409       | 本轮登记；保留Aborted，不改成AlreadyExists                  |
| OutOfRange         | `GRPC_OUT_OF_RANGE`                                     | 400       | 本轮登记；保留OutOfRange，不改成InvalidArgument             |
| Unimplemented      | `GRPC_UNIMPLEMENTED`                                    | 501       | 本轮登记；保留Unimplemented                                 |
| Internal           | `GRPC_INTERNAL_ERROR`                                   | 500       | 当前active；不复制未知包装层文本                            |
| Unavailable        | `GRPC_SERVICE_UNAVAILABLE`                              | 503       | 当前active；不猜具体网络故障                                |
| DataLoss           | `GRPC_DATA_LOSS`                                        | 500       | 本轮登记；保留DataLoss，不改成Internal                      |
| Unauthenticated    | `GRPC_UNAUTHORIZED`                                     | 401       | 当前active；与PermissionDenied分开                          |

OK不制造失败信封；非标准/非法RPC状态不在这16项承诺内，需受控兜底与诊断。原生status.Message保留；未知包装层文本不得冒充该文案。若补充Details编码失败，保留原状态与原Details并记录受控降级，不能宣称已成功补齐。该表定义P1纯分类必须验证的语义；真实normalizeRPCError拆分及拦截器接入留P6，不把文档检查当作协议链路验收。

### 错误码受控词表

本表的 Code/Output 在 active 与 retired 两种状态下均占号。检查器只要求 active 项存在且与源码精确对应；retired 项必须不存在于预置目录。序号在每个状态分组中单独连续，整个表按 Code 排序。

<!-- error-code-vocabulary:start -->

| Code  | Output                              | 归属包                       | Go符号                           | Message                | 状态   |
| ----- | ----------------------------------- | ---------------------------- | -------------------------------- | ---------------------- | ------ |
| 40000 | `HTTP_BAD_REQUEST`                  | `plugins/web-gin/httpresult` | `BadRequest`                     | 错误请求               | active |
| 40001 | `HTTP_DISABLED`                     | `plugins/web-gin/httpresult` | `Disabled`                       | 内容已禁用             | active |
| 40002 | `HTTP_INVALID_STATE`                | `plugins/web-gin/httpresult` | `InvalidState`                   | 无效状态               | active |
| 40003 | `HTTP_PARAMETER_EMPTY`              | `plugins/web-gin/httpresult` | `ParameterEmpty`                 | 参数缺失               | active |
| 40004 | `HTTP_PARAMETER_TYPE`               | `plugins/web-gin/httpresult` | `ParameterType`                  | 参数类型错误           | active |
| 40005 | `HTTP_BIND_PARAMETER_ERROR`         | `plugins/web-gin/bind`       | `ParameterError`                 | 参数错误               | active |
| 40006 | `HTTP_VALIDATOR_BODY_ERROR`         | `plugins/web-gin/validator`  | `BodyError`                      | 请求体错误             | active |
| 40007 | `HTTP_VALIDATOR_VALIDATION_ERROR`   | `plugins/web-gin/validator`  | `ValidationError`                | 验证错误               | active |
| 40008 | `GRPC_PARAMETER_ERROR`              | `plugins/grpc/grpcresult`    | `ParameterError`                 | 参数错误               | active |
| 40009 | `GRPC_PARAMETER_EMPTY`              | `plugins/grpc/grpcresult`    | `ParameterEmpty`                 | 参数缺失               | active |
| 40010 | `GRPC_PARAMETER_TYPE`               | `plugins/grpc/grpcresult`    | `ParameterType`                  | 参数类型错误           | active |
| 40011 | `GRPC_DISABLED`                     | `plugins/grpc/grpcresult`    | `Disabled`                       | 内容已禁用             | active |
| 40012 | `GRPC_METADATA_MISSING`             | `plugins/grpc/grpcresult`    | `MetadataMissing`                | 请求元数据缺失         | active |
| 40013 | `VALUE_INVALID_FORMAT`              | `foundation/valueerr`        | `InvalidFormat`                  | 值格式非法             | active |
| 40014 | `VALUE_OUT_OF_RANGE`                | `foundation/valueerr`        | `OutOfRange`                     | 值超出范围             | active |
| 40015 | `VALUE_INVALID_TYPE`                | `foundation/valueerr`        | `InvalidType`                    | 值类型非法             | active |
| 40016 | `GRPC_OUT_OF_RANGE`                 | `plugins/grpc/grpcresult`    | `OutOfRange`                     | 参数超出范围           | active |
| 40100 | `HTTP_UNAUTHORIZED`                 | `plugins/web-gin/httpresult` | `Unauthorized`                   | 未授权                 | active |
| 40101 | `GRPC_UNAUTHORIZED`                 | `plugins/grpc/grpcresult`    | `Unauthorized`                   | 未授权                 | active |
| 40300 | `HTTP_FORBIDDEN`                    | `plugins/web-gin/httpresult` | `Forbidden`                      | 禁止访问               | active |
| 40301 | `HTTP_PERMISSION_DENIED`            | `plugins/web-gin/httpresult` | `PermissionDenied`               | 权限不足               | active |
| 40302 | `HTTP_OPERATION_DENIED`             | `plugins/web-gin/httpresult` | `OperationDenied`                | 操作被拒绝             | active |
| 40303 | `GRPC_FORBIDDEN`                    | `plugins/grpc/grpcresult`    | `Forbidden`                      | 禁止访问               | active |
| 40304 | `GRPC_PERMISSION_DENIED`            | `plugins/grpc/grpcresult`    | `PermissionDenied`               | 权限不足               | active |
| 40305 | `GRPC_OPERATION_DENIED`             | `plugins/grpc/grpcresult`    | `OperationDenied`                | 操作被拒绝             | active |
| 40400 | `HTTP_NOT_FOUND`                    | `plugins/web-gin/httpresult` | `NotFound`                       | 未找到                 | active |
| 40401 | `GRPC_NOT_FOUND`                    | `plugins/grpc/grpcresult`    | `NotFound`                       | 未找到                 | active |
| 40402 | `DATABASE_RECORD_NOT_FOUND`         | `plugins/database`           | `RecordNotFound`                 | 记录未找到             | active |
| 40500 | `HTTP_METHOD_NOT_ALLOWED`           | `plugins/web-gin/httpresult` | `MethodNotAllowed`               | 方法不允许             | active |
| 40600 | `HTTP_NOT_ACCEPTABLE`               | `plugins/web-gin/httpresult` | `NotAcceptable`                  | 不可接受               | active |
| 40800 | `HTTP_TIMEOUT`                      | `plugins/web-gin/httpresult` | `Timeout`                        | 请求超时               | active |
| 40801 | `GRPC_TIMEOUT`                      | `plugins/grpc/grpcresult`    | `Timeout`                        | 请求超时               | active |
| 40802 | `CONTROL_REQUEST_TIMEOUT`           | `foundation/control`         | `RequestTimeout`                 | 请求接收超时           | active |
| 40900 | `HTTP_CONFLICT`                     | `plugins/web-gin/httpresult` | `Conflict`                       | 冲突                   | active |
| 40901 | `GRPC_CONFLICT`                     | `plugins/grpc/grpcresult`    | `Conflict`                       | 冲突                   | active |
| 40902 | `DATABASE_OPTIMISTIC_LOCK`          | `plugins/database`           | `OptimisticLock`                 | 乐观锁冲突             | active |
| 40903 | `DATABASE_DUPLICATE_ENTRY`          | `plugins/database`           | `DuplicateEntry`                 | 重复条目               | active |
| 40904 | `DATABASE_UNIQUE_CONSTRAINT`        | `plugins/database`           | `UniqueConstraint`               | 唯一约束冲突           | active |
| 40905 | `DATABASE_FOREIGN_KEY_ERROR`        | `plugins/database`           | `ForeignKeyError`                | 外键约束错误           | active |
| 40906 | `DATABASE_INTEGRITY_ERROR`          | `plugins/database`           | `IntegrityError`                 | 完整性约束错误         | active |
| 40907 | `GRPC_ABORTED`                      | `plugins/grpc/grpcresult`    | `Aborted`                        | 操作已中止             | active |
| 41000 | `HTTP_GONE`                         | `plugins/web-gin/httpresult` | `Gone`                           | 资源已删除             | active |
| 41200 | `GRPC_INVALID_STATE`                | `plugins/grpc/grpcresult`    | `InvalidState`                   | 无效状态               | active |
| 41201 | `GRPC_PRECONDITION_FAILED`          | `plugins/grpc/grpcresult`    | `PreconditionFailed`             | 前置条件未满足         | active |
| 41300 | `HTTP_PAYLOAD_TOO_LARGE`            | `plugins/web-gin/httpresult` | `PayloadTooLarge`                | 请求体过大             | active |
| 41500 | `HTTP_UNSUPPORTED_MEDIA`            | `plugins/web-gin/httpresult` | `UnsupportedMedia`               | 不支持的媒体类型       | active |
| 42200 | `HTTP_UNPROCESSABLE_ENTITY`         | `plugins/web-gin/httpresult` | `UnprocessableEntity`            | 无法处理的实体         | active |
| 42900 | `HTTP_TOO_MANY_REQUESTS`            | `plugins/web-gin/httpresult` | `TooManyRequests`                | 请求过多               | active |
| 42901 | `GRPC_TOO_MANY_REQUESTS`            | `plugins/grpc/grpcresult`    | `TooManyRequests`                | 请求过多               | active |
| 42902 | `GRPC_RESOURCE_EXHAUSTED`           | `plugins/grpc/grpcresult`    | `ResourceExhausted`              | 资源耗尽               | active |
| 49900 | `CONTROL_CANCELED`                  | `foundation/control`         | `Canceled`                       | 操作已取消             | active |
| 50000 | `FRAMEWORK_INTERNAL_ERROR`          | `foundation/apperr`          | `FrameworkInternalError`         | 框架内部错误           | active |
| 50001 | `FRAMEWORK_CONTAINER_NOT_MOUNTED`   | `foundation/apperr`          | `FrameworkContainerNotMounted`   | 应用容器未挂载         | active |
| 50002 | `FRAMEWORK_DEPENDENCY_UNDECLARED`   | `foundation/apperr`          | `FrameworkDependencyUndeclared`  | 组件依赖未声明         | active |
| 50003 | `FRAMEWORK_COMPONENT_NOT_FOUND`     | `foundation/apperr`          | `FrameworkComponentNotFound`     | 组件未找到             | active |
| 50004 | `FRAMEWORK_COMPONENT_TYPE_MISMATCH` | `foundation/apperr`          | `FrameworkComponentTypeMismatch` | 组件类型不匹配         | active |
| 50005 | `FRAMEWORK_COMPONENT_STOPPED`       | `foundation/apperr`          | `FrameworkComponentStopped`      | 组件已停止             | active |
| 50006 | `FRAMEWORK_COMPONENT_UNAVAILABLE`   | `foundation/apperr`          | `FrameworkComponentUnavailable`  | 组件不可用             | active |
| 50007 | `FRAMEWORK_COMPONENT_INVALID_STATE` | `foundation/apperr`          | `FrameworkComponentInvalidState` | 组件状态无效           | active |
| 50008 | `HTTP_INTERNAL_ERROR`               | `plugins/web-gin/httpresult` | `InternalError`                  | 服务器内部错误         | active |
| 50009 | `HTTP_RESPONSE_MISSING`             | `plugins/web-gin/httpresult` | `ResponseMissing`                | 处理器未输出响应       | active |
| 50010 | `HTTP_UNKNOWN_ERROR`                | `plugins/web-gin/httpresult` | `UnknownError`                   | 未知错误               | active |
| 50011 | `GRPC_INTERNAL_ERROR`               | `plugins/grpc/grpcresult`    | `InternalError`                  | 服务器内部错误         | active |
| 50012 | `GRPC_RESPONSE_MISSING`             | `plugins/grpc/grpcresult`    | `ResponseMissing`                | 处理器未返回响应       | active |
| 50013 | `GRPC_CONTEXT_MISSING`              | `plugins/grpc/grpcresult`    | `ContextMissing`                 | RPC上下文缺失          | active |
| 50014 | `DATABASE_ERROR`                    | `plugins/database`           | `DatabaseError`                  | 数据库错误             | active |
| 50015 | `DATABASE_TRANSACTION_FAILED`       | `plugins/database`           | `TransactionFailed`              | 事务失败               | active |
| 50016 | `CACHE_ERROR`                       | `plugins/cache`              | `CacheError`                     | 缓存错误               | active |
| 50017 | `EMAIL_ERROR`                       | `plugins/email`              | `EmailError`                     | 邮件服务错误           | active |
| 50018 | `VALUE_UNSUPPORTED_TARGET`          | `foundation/valueerr`        | `UnsupportedTarget`              | 不支持的转换目标       | active |
| 50019 | `VALUE_INVALID_RECEIVER`            | `foundation/valueerr`        | `InvalidReceiver`                | 值接收器无效           | active |
| 50020 | `FRAMEWORK_MULTIPLE_FAILURES`       | `foundation/apperr`          | `FrameworkMultipleFailures`      | 多个框架阶段失败       | active |
| 50021 | `FRAMEWORK_HOOK_PANIC`              | `foundation/apperr`          | `FrameworkHookPanic`             | 框架生命周期钩子异常   | active |
| 50022 | `FRAMEWORK_SHUTDOWN_INCOMPLETE`     | `foundation/apperr`          | `FrameworkShutdownIncomplete`    | 框架关闭未完成         | active |
| 50023 | `FRAMEWORK_WORKER_UNEXPECTED_EXIT`  | `foundation/apperr`          | `FrameworkWorkerUnexpectedExit`  | 后台服务意外退出       | active |
| 50024 | `GRPC_UNKNOWN_ERROR`                | `plugins/grpc/grpcresult`    | `UnknownError`                   | RPC未知错误            | active |
| 50025 | `GRPC_DATA_LOSS`                    | `plugins/grpc/grpcresult`    | `DataLoss`                       | 数据丢失               | active |
| 50026 | `SNOWFLAKE_NODE_INVALID`            | `foundation/snowflake`       | `NodeInvalid`                    | 数据中心或节点编号非法 | active |
| 50027 | `SNOWFLAKE_CLOCK_BACKWARDS`         | `foundation/snowflake`       | `ClockBackwards`                 | 系统时钟回退           | active |
| 50028 | `LOGGING_HANDLER_INVALID`           | `foundation/logging`         | `HandlerInvalid`                 | 日志处理器缺失         | active |
| 50029 | `LOGGING_CONFIG_INVALID`            | `foundation/logging`         | `ConfigInvalid`                  | 日志配置非法           | active |
| 50030 | `LOGGING_WRITE_FAILED`              | `foundation/logging`         | `WriteFailed`                    | 日志写入失败           | active |
| 50031 | `LOGGING_ROTATION_FAILED`           | `foundation/logging`         | `RotationFailed`                 | 日志轮转失败           | active |
| 50032 | `TOOLKIT_PASSWORD_FAILED`           | `foundation/toolkit`         | `PasswordFailed`                 | 密码加密或校验失败     | active |
| 50100 | `GRPC_UNIMPLEMENTED`                | `plugins/grpc/grpcresult`    | `Unimplemented`                  | 功能未实现             | active |
| 50200 | `HTTP_GATEWAY_ERROR`                | `plugins/web-gin/httpresult` | `GatewayError`                   | 网关错误               | active |
| 50300 | `HTTP_SERVICE_UNAVAILABLE`          | `plugins/web-gin/httpresult` | `ServiceUnavailable`             | 服务不可用             | active |
| 50301 | `HTTP_RESOURCE_EXHAUSTED`           | `plugins/web-gin/httpresult` | `ResourceExhausted`              | 资源耗尽               | active |
| 50302 | `GRPC_SERVICE_UNAVAILABLE`          | `plugins/grpc/grpcresult`    | `ServiceUnavailable`             | 服务不可用             | active |
| 50400 | `HTTP_GATEWAY_TIMEOUT`              | `plugins/web-gin/httpresult` | `GatewayTimeout`                 | 网关超时               | active |
| 50401 | `CONTROL_DEADLINE_EXCEEDED`         | `foundation/control`         | `DeadlineExceeded`               | 操作超过截止时间       | active |

<!-- error-code-vocabulary:end -->

## 正反例与边界

- 合规：HTTP NotFound=40400、RPC NotFound=40401、数据库 RecordNotFound=40402；语义相近也有独立编号和前缀，映射仍为404/NotFound。
- 合规：application.Get 发现未声明依赖，返回50002/FRAMEWORK_DEPENDENCY_UNDECLARED，并保留 ErrUndeclaredDependency 根因。
- 违规：在 grpcresult 再定义40400/NOT_FOUND，或只修改Go目录却不登记词表。
- 违规：把409分组用完后改用500分组装“用户重复提交”，导致 RPC 错误从 AlreadyExists 变成 Internal。
- 合法边界：在测试中构造60000验证HTTP500兜底；这不是允许生产预置码占用600分组。
- 合法边界：业务应用 apperr.New(&apperr.ErrorCode{Code:40950, Output:"SHOP_ORDER_CLOSED", Message:"订单已关闭"}, "")；业务多服务仍须维护自己的共享台账。

## 合规与后果

### 校验与暴露方式

- make error-codes-check：读取本文标记区与application/foundation/plugins全部非测试Go源码的AST，不执行运行时代码、不引入跨模块依赖。词表与error_code.go双向精确核对，任一违规退出非0。
- make test-scripts：包含守卫正反例（重复编号/标识、错前缀、非法组号、乱序/缺号、缺定义/多定义、别名/非字面量、退役占号与生产绕过）。质量工作流在Go检查阶段执行词表门禁。
- make test / make race / make vet：覆盖14个模块，检查迁移、映射和根因行为；外部Redis/SMTP/数据库集成用例仍按各模块环境开关执行，不把默认跳过写成外部验收通过。
- make lint-docs：检查文档格式与Markdown规则。校验器与词表在本补丁交付；具体运行结果记录在本卡workpad，不以本文草案冒充CI已合入。

### 收益与代价

同一编号只表达一个所属域的错误；源码可保持分包且不反向依赖。代价是新增预置错误必须协调中央词表，每个状态分组只有50个SDK槽位。包级变量仍是Go可变指针，守卫不是运行时防篡改机制；调用方必须遵守只读约定。

### 重评条件

任一状态分组达到50项、多个独立SDK需要共享机器码空间、或业务集群无法在50个预留号内维护唯一性时，必须重评位数/域段及显式状态映射。不得静默扩大本规则或把整数溢出/Protocol Buffers数值范围当作无限容量。已发布编号的兼容性需要版本及发布评审，当前守卫只检查当前树，不证明历史号从未修改。

## 否决项

- 各插件各维护一份从00开始的编号：维护方便，但不能保证组合使用时Code唯一；本次已出现碰撞。
- 相同语义跨HTTP/RPC共享编号或指针别名：可减少条目，但无法仅凭错误信封区分来源；跨插件别名还破坏依赖独立性。
- 全部定义集中进foundation：便于直接查表，却把协议/领域语义带回基础层，违背本次边界要求。
- 删除状态分组直接全局1、2、3递增：空间充裕，但需要重做HTTP/RPC状态映射契约；本轮不扩大这个API变更。
- 在apperr.New运行时抢占号码或注册插件全局状态：无法协调未加载插件与仓外服务，还破坏纯构造语义。

## 待确认

规则保持草案；首批65项和本轮P1追加19项的实施授权不等于ADR冻结。用户已确认FRAMEWORK_拼写、框架内部以500系列表达，并授权本次首次重编。将本文从draft升级为rfc/adr仍需人工指示；已实施与验证不等于长期约束自动冻结。
