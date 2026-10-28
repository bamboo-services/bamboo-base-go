// Package typed 提供 Bamboo Base 框架的强类型字符串原语 typed.String 与受控词汇基底。
//
// 核心设计原则：
//  1. 唯一强类型字符串原语 typed.String：集齐基础值操作、受控标识符语法校验、标准库六通道双向编解码
//     （Text / JSON / XML / Binary / SQL / CLI Flag），以及向 Go 原生数值/布尔/时长解析的完整能力。
//  2. 双重角色解耦：
//     - 受控机器键/词汇：走 operate.go 的 IsValid() 语法校验（1..128 字节、ASCII 字母开头、字母数字下划线连字符、
//     无末尾分隔符、无连续分隔符）与 convert.go 的六通道编解码（编码前校验，解码即校验，失败不污染接收器）。
//     - 原始文本值载体：走 parse.go 的原生类型解析族（自动 TrimSpace，按目标位宽严格解析，不强制键名语法）。
//  3. 严格遵循标准库与零外部依赖：纯 Go 标准库实现，不引入任何第三方依赖。
//  4. 按职责固定拆分为 5 个源文件：doc.go、index.go、operate.go、convert.go、parse.go。
package typed
