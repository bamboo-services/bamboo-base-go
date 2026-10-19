# v2.0.0 foundation/typed 强类型字符串原语与受控词汇设计

> 状态：draft · 日期：2026-10-19 · Scope：typed
> 承接 [0003 foundation 地基模块合并与包重构设计](./0003-foundation-module-consolidation.md)

## 目标与边界

在 Go 语言里，`int`、`int64`、`float64`、`bool`、`time.Duration` 这些原生值类型本身就已经够清晰了，再去给它们套一层 `typed.Int` 或 `typed.Float` 纯属画蛇添足。真正到处裸奔、最容易把代码写乱的，其实是各种各样的 `string`。

翻看现有的 v1.x 代码，`defined/context/context.go` 里的 `ContextKey`、`defined/env/env.go` 里的 `EnvKey`、`defined/http/header.go` 里的 `Header` 各自基于 `string` 定义了一套小方法（有的叫 `IsNil()`，有的叫 `IsEmpty()`，有的连去空白和格式校验都没有）；而到了解析环境变量（`defined/env/utility.go`）或者通用类型转换（`common/utility/package/parse.go`）时，又得把 `string` 传进一堆散装的工具函数里用 `strconv` 反复拆包。

因此，我们在 `foundation` 模块最底层新建 **`foundation/typed`** 包（包声明直接用最顺手的 **`package typed`**），遵循“**只做 `typed.String`，解析出口直接还原为 Go 原生类型**”的克制原则，一次性解决三件事：

1. **提供唯一的强类型字符串原语 `typed.String`**：集齐基础值操作、受控标识符语法校验、标准库六通道双向编解码（Text / JSON / XML / Binary / SQL / CLI Flag）、以及向 Go 原生数值/布尔/时长解析的完整能力。
2. **确立按职责拆分文件的包组织规范**：包内严格按 `doc.go`、`index.go`、`operate.go`、`convert.go`、`parse.go` 五类职责分文件，告别把所有东西塞进一个大文件的乱象。
3. **统一领域词汇的基底与委托规则**：`ctxkey.Key`、`env.Key` 以 `typed.String` 为底层类型，值实现 `typed.Operable`，指针实现 `typed.Convertible`。Go 不会继承基底方法，所以领域类型显式委托编解码，而不是只写一句类型定义就宣称拥有全部通道。

## 工作方式

### 1. `foundation/typed` 的五文件职责布局

`foundation/typed` 只依赖 Go 标准库，内部按职责固定拆分为 5 个文件（配套测试按同名 `*_test.go` 一一对应）：

```text
foundation/typed/           # package typed
├── doc.go                  # 包级 GoDoc、设计边界与典型示例
├── index.go                # MaxKeyBytes、String 类型、Operable/Convertible
├── operate.go              # 值读取、判空、大小写转换、相等比较与 IsValid 校验
├── convert.go              # Text/JSON/XML/Binary/SQL/Flag 六通道编解码与断言
└── parse.go                # 将原始文本解析为 Go 原生数值/布尔/Duration
```

### 2. `typed.String` 的双重角色：受控键名 vs 原始值载体

同一个 `typed.String`，在框架里扮演两个互补但边界分明的角色：

```text
                  ┌─ 角色 A：受控机器键 / 词汇 (Key & Vocabulary)
                  │   ├─ 走 operate.go 的 IsValid()：校验 1~128 字节、字母开头、无连续分隔符
typed.String ─────┤   └─ 走 convert.go 的六通道编解码：编码前校验、解码即校验（失败不改原值）
                  │
                  └─ 角色 B：原始文本值载体 (Raw Value Carrier)
                      └─ 走 parse.go 的 Int()/Float64()/Bool()/Duration()：
                         自动 TrimSpace 后直接解析成 Go 原生类型，不强制要求满足键名语法
```

为什么要把“键名语法校验（`IsValid`）”和“原始值解析（`parse.go`）”分开？
道理很实在：`typed.String("app_port").IsValid()` 校验的是它能不能当一个合法的键名；而当我们用环境变量读取器拿到它的值 `typed.String(" 8080 ")` 时，这段内容是纯数字，显然不以字母开头，但直接调 `typed.String(" 8080 ").Int()` 就能干脆利落地拿到 Go 原生的 `8080`。两边各司其职，谁也不卡谁的脖子。

### 3. 标识符校验、双向编解码与值解析细则

1. **受控标识符语法规则（`operate.go` → `IsValid()`）**：
   - 长度必须在 `1` 到 `MaxKeyBytes`（`128` 字节）之间；空串 `""` 的 `IsEmpty()` 为 `true`，`IsValid()` 为 `false`；
   - 首字符必须是 ASCII 字母（`A-Z` 或 `a-z`）；
   - 后续字符仅允许 ASCII 字母、数字（`0-9`）、下划线 `_` 与连字符 `-`；
   - 分隔符（`_` 或 `-`）不能出现在末尾，且不能连续出现（如 `a__b` 或 `a-_b` 均视为非法）。
2. **六通道双向编解码（`convert.go`，100% 标准库接口，解码即校验）**：
   既然做到底层原语，就别只管 JSON 不管别的——只要是 Go 标准库自带的序列化接口，我们一次性全给它接通，既保持零第三方依赖，又让 `typed.String` 放进任何结构体都能直接序列化：
   - **Text 通道**（`encoding.TextMarshaler` / `TextUnmarshaler`）：供支持这些接口的 YAML、TOML 等适配器复用。具体库是否自动调用要按其文档验证；例如 mapstructure 通常需要配置 Text 解码钩子，不能承诺所有反射转换器都会自动识别。
   - **JSON 通道**（`json.Marshaler` / `Unmarshaler`）：严格按 JSON 字符串编解码，遇到 JSON `null` 或非字符串类型直接报错拒绝；
   - **XML 双通道**（`xml.Marshaler` / `Unmarshaler` + `xml.MarshalerAttr` / `UnmarshalerAttr`）：做网关、银行接口或微信支付时经常碰到 XML。这里有个细节要注意：在 Go 的 `encoding/xml` 里，“作为 `<tag>val</tag>` 子元素”和“作为 `<tag attr="val">` 属性”走的是两套完全不同的接口！我们把元素编解码和 `Attr` 属性编解码四件套全部实现，不管业务贴 `xml:"key"` 还是 `xml:"key,attr"` 都能无缝校验通过；
   - **Binary 二进制通道**（`encoding.BinaryMarshaler` / `BinaryUnmarshaler`）：千万别小看这两个标准库接口——咱们缓存模块用的 **`github.com/redis/go-redis/v9`** 以及标准库 `encoding/gob` 在读写自定义类型时，认的就是 `BinaryMarshaler` / `BinaryUnmarshaler`！实现它之后，`typed.String`（及派生的 Key）就能直接塞进 Redis 命令参数或扫描结果里；
   - **SQL 通道**（`driver.Valuer` / `sql.Scanner`）：对接 GORM 与 `database/sql`，仅接受 `string` 与 `[]byte` 列值，遇到 SQL `NULL` 直接返回错误；
   - **CLI Flag 通道**（`flag.Value` 的 `Set(string) error`）：加上 `String()` 方法后，`*typed.String` 天然满足标准库 `flag.Value`，可以直接绑到命令行参数上；
   - **接收器与原子写回铁律**：所有编码方法（`MarshalText`、`MarshalJSON`、`MarshalXML`、`MarshalXMLAttr`、`MarshalBinary`、`Value`）统一用**值接收器**；所有解码方法（`UnmarshalText`、`UnmarshalJSON`、`UnmarshalXML`、`UnmarshalXMLAttr`、`UnmarshalBinary`、`Scan`、`Set`）统一用**指针接收器**，且先在临时变量 `candidate` 上完成 `IsValid()` 校验，通过后才写回 `*s`，失败时原实例的值纹丝不动。
3. **向 Go 原生类型的解析族（`parse.go`）**：
   - 所有解析方法内部先执行 `strings.TrimSpace` 去除首尾空白；
   - **整数族**（`Int`、`Int8`、`Int16`、`Int32`、`Int64`、`Uint`、`Uint8`、`Uint16`、`Uint32`、`Uint64`）：按十进制严格解析并卡死目标位宽，超出位宽范围直接返回溢出错误（其中 `Uint` 的 `bitSize` 传 `0` 跟随平台位宽，绝不先按 `uint64` 解析再强转截断）；
   - **浮点族**（`Float32`、`Float64`）：通过 `strconv.ParseFloat` 解析，并拒绝 `NaN` 与 `±Inf`；
   - **布尔值**（`Bool`）：忽略大小写与首尾空白，真值接受 `"true"`、`"1"`、`"yes"`、`"on"`、`"enabled"`，假值接受 `"false"`、`"0"`、`"no"`、`"off"`、`"disabled"`，其余一律报错；
   - **时间间隔**（`Duration`）：委托 `time.ParseDuration`，直接支持 `"30s"`、`"500ms"`、`"1h30m"` 等标准写法。

### 4. 全仓受控词汇怎么基于 `typed.String` 派生？

有了 `typed.String` 打底，`foundation` 里的核心词汇统一收编到同一套模式（后续 HTTP 插件中的 Header 等词汇也可按同样方式派生）：

| 领域词汇     | 所在包                                 | 派生定义方式            | 怎么复用 `typed.String`                                                                                                                |
| ------------ | -------------------------------------- | ----------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `ctxkey.Key` | `foundation/ctxkey` (`package ctxkey`) | `type Key typed.String` | 实现 Operable；显式委托完整编解码，使 `*Key` 满足 Convertible，删除旧 IsNil/IsExec                                                     |
| `env.Key`    | `foundation/env` (`package env`)       | `type Key typed.String` | 实现 Operable 与指针 Convertible；`env.Lookup(key)` 直接返回 `(typed.String, bool)`，调用方链式调 `.Int()` / `.Bool()` / `.Duration()` |

你看 `foundation/env` 这一改有多爽：以前为了读不同类型，得维护 `GetEnvString`、`GetEnvInt`、`GetEnvBool`、`GetEnvDuration` 一大堆函数；现在底层只需一个返回 `typed.String` 的读取入口（当然常用快捷函数仍可保留做一行委托），拿到 `typed.String` 后想转 `Int64()`、`Uint16()` 还是 `Float64()` 全凭调用方一句话搞定！

## 数据与接口

### 1. `foundation/typed/index.go` 核心类型与契约接口

```go
package typed

import (
    "database/sql"
    "database/sql/driver"
    "encoding"
    "encoding/json"
    "encoding/xml"
    "flag"
    "fmt"
)

// MaxKeyBytes 是受控标识符键名的最大字节长度上限。
const MaxKeyBytes = 128

// String 是以 Go 原生 string 为基底的强类型字符串原语。
// 既可作为受控键名/常量词汇的基底，也可作为环境变量等原始文本的解析载体。
type String string

// Operable 约束所有基于 typed.String 派生的领域词汇必须具备的基础操作能力。
type Operable interface {
    fmt.Stringer
    TypedString() String
    IsEmpty() bool
}

// Convertible 约束支持标准文本、JSON、XML（含属性）、二进制、SQL 与 CLI Flag 双向转换的完整契约。
type Convertible interface {
    encoding.TextMarshaler
    encoding.TextUnmarshaler
    encoding.BinaryMarshaler
    encoding.BinaryUnmarshaler
    json.Marshaler
    json.Unmarshaler
    xml.Marshaler
    xml.Unmarshaler
    xml.MarshalerAttr
    xml.UnmarshalerAttr
    driver.Valuer
    sql.Scanner
    flag.Value
}
```

### 2. `foundation/typed/operate.go`、`convert.go` 与 `parse.go` 方法签名

```go
package typed

import (
    "database/sql/driver"
    "encoding/xml"
    "time"
)
// --- operate.go：基础值操作与标识符校验 ---

func (s String) String() string          { return string(s) }
func (s String) TypedString() String     { return s }
func (s String) Bytes() []byte           { return []byte(s) }
func (s String) Len() int                { return len(s) }
func (s String) IsEmpty() bool           { return len(s) == 0 }
func (s String) IsBlank() bool           // 去除空白字符后是否为空
func (s String) Trim() String            // 返回去除首尾空白后的 String
func (s String) Upper() String           // 返回全大写变体
func (s String) Lower() String           // 返回全小写变体
func (s String) Equal(other String) bool { return s == other }
func (s String) IsValid() bool           // 校验 1~128 字节受控标识符语法

// --- convert.go：六通道双向编解码（值接收器编码，指针接收器解码即校验）---

func (s String) MarshalText() ([]byte, error)
func (s *String) UnmarshalText(text []byte) error
func (s String) MarshalBinary() ([]byte, error)
func (s *String) UnmarshalBinary(data []byte) error
func (s String) MarshalJSON() ([]byte, error)
func (s *String) UnmarshalJSON(raw []byte) error
func (s String) MarshalXML(e *xml.Encoder, start xml.StartElement) error
func (s *String) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error
func (s String) MarshalXMLAttr(name xml.Name) (xml.Attr, error)
func (s *String) UnmarshalXMLAttr(attr xml.Attr) error
func (s String) Value() (driver.Value, error)
func (s *String) Scan(src any) error
func (s *String) Set(raw string) error // 配合 String() 实现 flag.Value

// --- parse.go：原始文本解析为 Go 原生类型（解析前自动 TrimSpace）---

func (s String) Int() (int, error)
func (s String) Int8() (int8, error)
func (s String) Int16() (int16, error)
func (s String) Int32() (int32, error)
func (s String) Int64() (int64, error)
func (s String) Uint() (uint, error)
func (s String) Uint8() (uint8, error)
func (s String) Uint16() (uint16, error)
func (s String) Uint32() (uint32, error)
func (s String) Uint64() (uint64, error)
func (s String) Float32() (float32, error)
func (s String) Float64() (float64, error)
func (s String) Bool() (bool, error)
func (s String) Duration() (time.Duration, error)
```

### 3. 领域词汇派生范例（以 `ctxkey.Key` 为例）

```go
package ctxkey

import (
    "fmt"

    "github.com/bamboo-services/bamboo-base-go/foundation/v2/typed"
)

// Key 是基于 typed.String 派生的强类型上下文与插件标识键。
type Key typed.String

var _ typed.Operable = Key("")

func (k Key) String() string {
    return string(k)
}
func (k Key) TypedString() typed.String {
    return typed.String(k)
}
func (k Key) IsEmpty() bool {
    return len(k) == 0
}
func (k Key) IsValid() bool {
    return typed.String(k).IsValid()
}

// 以下节选 Text 委托；其它通道按下表实现，不是自动继承。
func (k Key) MarshalText() ([]byte, error) {
    return k.TypedString().MarshalText()
}
func (k *Key) UnmarshalText(text []byte) error {
    if k == nil {
        return fmt.Errorf("ctxkey.Key: nil receiver")
    }
    var candidate typed.String
    if err := candidate.UnmarshalText(text); err != nil {
        return err
    }
    *k = Key(candidate)
    return nil
}
```

其余方法遵守相同委托方式，`env.Key` 也一样，不在文档里再铺两份重复代码：

| 通道          | 值接收器委托               | 指针接收器委托                 |
| ------------- | -------------------------- | ------------------------------ |
| Text          | MarshalText                | UnmarshalText                  |
| JSON          | MarshalJSON                | UnmarshalJSON                  |
| XML 元素/属性 | MarshalXML、MarshalXMLAttr | UnmarshalXML、UnmarshalXMLAttr |
| Binary        | MarshalBinary              | UnmarshalBinary                |
| SQL           | Value                      | Scan                           |
| CLI Flag      | String                     | Set                            |

所有指针方法先查 nil，再解码到临时 `typed.String`，成功后才赋回领域类型。完整实现必须通过 `var _ typed.Convertible = (*Key)(nil)`，仅有上面的 Text 节选还不能通过该断言。

为可序列化的领域词汇补这些方法，不等于给所有原始文本套键名约束。环境变量的值仍通过 `Lookup(key)` 得到原始文本载体，再走数值/布尔解析；只有选择编解码键名时才执行 IsValid。

## 异常与兼容

1. **解码失败不污染接收器**：当 `*typed.String` 调用 `UnmarshalJSON`、`UnmarshalXML`、`UnmarshalXMLAttr`、`UnmarshalText`、`UnmarshalBinary`、`Scan` 或 `Set` 遇到非法输入、JSON `null`、XML 空标签、SQL `NULL` 或 nil 指针接收器时，直接返回明确错误，原指针指向的值保持调用前状态不变。
2. **位宽溢出与非有限浮点数拒绝**：
   - 例如 `typed.String("300").Uint8()` 会明确返回 `uint8` 溢出错误，绝不静默截断为 `44`；
   - `typed.String("NaN").Float64()` 与 `typed.String("Inf").Float64()` 直接返回解析错误，防止非有限浮点数混入业务计算或导致后续 JSON 序列化崩溃。
3. **字符串常量可以迁移，不等于旧 API 原样兼容**：
   `type Key typed.String` 的底层类型仍是 `string`，所以 `const DatabaseKey Key = "context_database"`、`const Debug Key = "APP_DEBUG"` 这样的字面量赋值仍然合法。但新的 `ctxkey.Key`、`env.Key` 与旧模块定义的类型并不相同；调用方需要迁移 import 和访问入口。整体 Release 与模块版本的区别统一见 [0003](./0003-foundation-module-consolidation.md#异常与兼容)。

## 备选与取舍

直接使用 string 加 strconv 最简单，也不会引入自定义编解码的隐性限制。本次保留 typed.String，是为了让受控键共享校验与接口委托，让环境变量读取后能直接解析；不是认为原生 string 本身不够好。

把“受控键”与“原始值”拆成两个类型会更严格，但会增加读取与转换入口。先保留一个类型的两条明确路径：解析原始文本不校验键名，键名编解码才校验。派生类型为此付出显式委托代码，但不使用反射去猜应该套哪种规则。

如果两个独立调用方因“原始环境值不能当键编码”反复踩同类问题，应重评拆成 Key 与 RawValue；如果没有真实场景需要完整通道，也应缩小未来新增领域类型的接口范围，不机械复制委托。已明确支持的 ctxkey.Key/env.Key 仍按本稿完整实现。

## 验证

本设计稿的文档级校验（已执行）：

- 遵循 `design-er` 六段结构，承接 `design/0003`，已在 `docs/README.md` 登记；
- 同步更新了 `design/0003` 中的 `foundation` 目录树与 `ctxkey.Key` / `env.Key` 派生关系。

编码实施后，`foundation/typed` 必须通过以下 5 组单元测试（`GOWORK=off go test -race ./typed/...`）：

1. **`operate_test.go`（值操作与语法边界）**：验证空串、128 字节边界、129 字节超长、数字开头、首尾下划线、连续分隔符 `a__b` / `a--b`、非 ASCII 字符的 `IsValid()` 结果，以及 `Upper()` / `Lower()` / `IsBlank()` 行为；
2. **`convert_test.go`（六通道编解码与原子性）**：验证合法键在 Text、JSON、XML（子元素 `<key>` 与属性 `key="..."`）、Binary（模拟 Redis/gob 字节流）、SQL `Value/Scan`、以及 `flag.Value` 下的往返一致性；验证向已有合法值的 `typed.String` 解码非法字符串、JSON `null`、SQL `NULL` 时返回错误且原值未被篡改；
3. **`parse_test.go`（全位宽数值、布尔与时长解析）**：覆盖带首尾空白的数字、各整数位宽上下界与溢出（`int8` `-129`/`128`、`uint8` `256` 等）、`NaN`/`Inf` 拒绝、10 种布尔词汇大小写组合、以及合法/非法 `Duration` 字符串；
4. **派生类型不是自动继承**：除基底接口断言外，确认 `ctxkey.Key`、`env.Key` 的值实现 Operable，指针实现 Convertible；分别对两种 Key 做六通道往返、非法输入不写回、nil 指针错误测试。Redis/gob 必须使用真实接口调用路径验证，不只模拟字节复制。
5. **零外部依赖验证**：`GOWORK=off go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./typed` 的非空输出只有 foundation/typed 自身，无其它非标准库包。

## 待确认

已确认：仅做 `typed.String`、使用五文件职责拆分，并作为领域词汇的派生基底。

派生类型显式委托完整编解码已经确认，本篇无剩余阻塞性方案选择。完整实现与接口断言仍需编码后验证，不能把本次文档修订写成测试已通过。
