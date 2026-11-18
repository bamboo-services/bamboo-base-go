package main

import (
	"fmt"
	"sort"
)

const modulePrefix = "github.com/bamboo-services/bamboo-base-go/"

// EntryKind 标识白名单精确绑定对象的语法类别，禁止目录级或方法族通配。
type EntryKind string

const (
	KindMethod    EntryKind = "method"
	KindFunc      EntryKind = "func"
	KindFuncType  EntryKind = "func_type"
	KindFuncLit   EntryKind = "func_lit"
	KindField     EntryKind = "field"
	KindVar       EntryKind = "var"
	KindInterface EntryKind = "interface"
)

// WhitelistEntry 描述 spec/0003 中一条精确绑定的固定签名或结构例外。
type WhitelistEntry struct {
	ID       string
	Contract string
	Kind     EntryKind
	PkgPath  string // 相对仓库根的包路径，如 foundation/apperr
	Symbol   string // 精确符号或调用点描述，禁止通配符
	Reason   string
}

// Key 返回该白名单项的全局唯一索引键。
func (e WhitelistEntry) Key() string {
	return fmt.Sprintf("%s|%s|%s", e.Kind, e.PkgPath, e.Symbol)
}

func standardCodecEntries(id, pkgPath, valueType, ptrType string) []WhitelistEntry {
	return []WhitelistEntry{
		{ID: id, Contract: "encoding.TextMarshaler", Kind: KindMethod, PkgPath: pkgPath, Symbol: fmt.Sprintf("(%s).MarshalText", valueType), Reason: "标准 encoding.TextMarshaler 固定签名"},
		{ID: id, Contract: "encoding.TextUnmarshaler", Kind: KindMethod, PkgPath: pkgPath, Symbol: fmt.Sprintf("(%s).UnmarshalText", ptrType), Reason: "标准 encoding.TextUnmarshaler 固定签名"},
		{ID: id, Contract: "encoding.BinaryMarshaler", Kind: KindMethod, PkgPath: pkgPath, Symbol: fmt.Sprintf("(%s).MarshalBinary", valueType), Reason: "标准 encoding.BinaryMarshaler 固定签名"},
		{ID: id, Contract: "encoding.BinaryUnmarshaler", Kind: KindMethod, PkgPath: pkgPath, Symbol: fmt.Sprintf("(%s).UnmarshalBinary", ptrType), Reason: "标准 encoding.BinaryUnmarshaler 固定签名"},
		{ID: id, Contract: "json.Marshaler", Kind: KindMethod, PkgPath: pkgPath, Symbol: fmt.Sprintf("(%s).MarshalJSON", valueType), Reason: "标准 json.Marshaler 固定签名"},
		{ID: id, Contract: "json.Unmarshaler", Kind: KindMethod, PkgPath: pkgPath, Symbol: fmt.Sprintf("(%s).UnmarshalJSON", ptrType), Reason: "标准 json.Unmarshaler 固定签名"},
		{ID: id, Contract: "xml.Marshaler", Kind: KindMethod, PkgPath: pkgPath, Symbol: fmt.Sprintf("(%s).MarshalXML", valueType), Reason: "标准 xml.Marshaler 固定签名"},
		{ID: id, Contract: "xml.Unmarshaler", Kind: KindMethod, PkgPath: pkgPath, Symbol: fmt.Sprintf("(%s).UnmarshalXML", ptrType), Reason: "标准 xml.Unmarshaler 固定签名"},
		{ID: id, Contract: "xml.MarshalerAttr", Kind: KindMethod, PkgPath: pkgPath, Symbol: fmt.Sprintf("(%s).MarshalXMLAttr", valueType), Reason: "标准 xml.MarshalerAttr 固定签名"},
		{ID: id, Contract: "xml.UnmarshalerAttr", Kind: KindMethod, PkgPath: pkgPath, Symbol: fmt.Sprintf("(%s).UnmarshalXMLAttr", ptrType), Reason: "标准 xml.UnmarshalerAttr 固定签名"},
		{ID: id, Contract: "database/sql/driver.Valuer", Kind: KindMethod, PkgPath: pkgPath, Symbol: fmt.Sprintf("(%s).Value", valueType), Reason: "标准 driver.Valuer.Value 固定签名"},
		{ID: id, Contract: "database/sql.Scanner", Kind: KindMethod, PkgPath: pkgPath, Symbol: fmt.Sprintf("(%s).Scan", ptrType), Reason: "标准 sql.Scanner.Scan 固定签名"},
		{ID: id, Contract: "flag.Value.Set", Kind: KindMethod, PkgPath: pkgPath, Symbol: fmt.Sprintf("(%s).Set", ptrType), Reason: "标准 flag.Value.Set 固定签名"},
	}
}

// DefaultWhitelist 返回 spec/0003 登记的 W01-W12 及原因链载体字段精确白名单。
func DefaultWhitelist() []WhitelistEntry {
	entries := []WhitelistEntry{
		// W01: Unwrap() error / []error 标准错误链
		{
			ID:       "W01",
			Contract: "errors.Unwrap",
			Kind:     KindMethod,
			PkgPath:  "foundation/apperr",
			Symbol:   "(*Error).Unwrap",
			Reason:   "实现标准 errors.Unwrap() error 与 errors.Is/As 链解析",
		},
		{
			ID:       "W01",
			Contract: "errors.Unwrap",
			Kind:     KindMethod,
			PkgPath:  "application",
			Symbol:   "(*HookPanicError).Unwrap",
			Reason:   "保留 Hook panic 原始错误链的 Unwrap() error 接口契约",
		},
		{
			ID:       "W01",
			Contract: "errors.UnwrapMulti",
			Kind:     KindMethod,
			PkgPath:  "application",
			Symbol:   "(*StopIncompleteError).Unwrap",
			Reason:   "实现 Go 标准多错误 Unwrap() []error 接口契约",
		},
		// W02: context.Context.Err() error
		{
			ID:       "W02",
			Contract: "context.Context.Err",
			Kind:     KindMethod,
			PkgPath:  "application",
			Symbol:   "(*bridgedContext).Err",
			Reason:   "实现标准库 context.Context.Err() error 取消语义",
		},
	}

	// W03: foundation/typed.String, foundation/ctxkey.Key, foundation/env.Key 标准编解码与 SQL/Flag 接口
	entries = append(entries, standardCodecEntries("W03", "foundation/typed", "String", "*String")...)
	entries = append(entries, standardCodecEntries("W03", "foundation/ctxkey", "Key", "*Key")...)
	entries = append(entries, standardCodecEntries("W03", "foundation/env", "Key", "*Key")...)

	entries = append(entries,
		// W04: foundation/snowflake.SnowflakeID 标准编解码与 SQL 接口
		WhitelistEntry{ID: "W04", Contract: "json.Marshaler", Kind: KindMethod, PkgPath: "foundation/snowflake", Symbol: "(SnowflakeID).MarshalJSON", Reason: "标准 json.Marshaler 固定签名"},
		WhitelistEntry{ID: "W04", Contract: "json.Unmarshaler", Kind: KindMethod, PkgPath: "foundation/snowflake", Symbol: "(*SnowflakeID).UnmarshalJSON", Reason: "标准 json.Unmarshaler 固定签名"},
		WhitelistEntry{ID: "W04", Contract: "encoding.BinaryMarshaler", Kind: KindMethod, PkgPath: "foundation/snowflake", Symbol: "(SnowflakeID).MarshalBinary", Reason: "标准 encoding.BinaryMarshaler 固定签名"},
		WhitelistEntry{ID: "W04", Contract: "encoding.BinaryUnmarshaler", Kind: KindMethod, PkgPath: "foundation/snowflake", Symbol: "(*SnowflakeID).UnmarshalBinary", Reason: "标准 encoding.BinaryUnmarshaler 固定签名"},
		WhitelistEntry{ID: "W04", Contract: "database/sql/driver.Valuer", Kind: KindMethod, PkgPath: "foundation/snowflake", Symbol: "(SnowflakeID).Value", Reason: "标准 driver.Valuer.Value 固定签名"},
		WhitelistEntry{ID: "W04", Contract: "database/sql.Scanner", Kind: KindMethod, PkgPath: "foundation/snowflake", Symbol: "(*SnowflakeID).Scan", Reason: "标准 sql.Scanner.Scan 固定签名"},
		// W05: foundation/logging.LogHandler
		WhitelistEntry{ID: "W05", Contract: "slog.Handler.Handle", Kind: KindMethod, PkgPath: "foundation/logging", Symbol: "(*LogHandler).Handle", Reason: "标准 log/slog.Handler.Handle 固定签名"},
		// W06: foundation/logging.RotatingWriter
		WhitelistEntry{ID: "W06", Contract: "io.Writer.Write", Kind: KindMethod, PkgPath: "foundation/logging", Symbol: "(*RotatingWriter).Write", Reason: "标准 io.Writer.Write 固定签名"},
		WhitelistEntry{ID: "W06", Contract: "io.Closer.Close", Kind: KindMethod, PkgPath: "foundation/logging", Symbol: "(*RotatingWriter).Close", Reason: "标准 io.Closer.Close 固定签名"},
		// W07: plugins/email 网络连接与拨号器适配器
		WhitelistEntry{ID: "W07", Contract: "net.Conn.Close", Kind: KindMethod, PkgPath: "plugins/email", Symbol: "(*watchedConn).Close", Reason: "第三方 net.Conn.Close 固定签名"},
		WhitelistEntry{ID: "W07", Contract: "mail.WithDialContextFunc", Kind: KindFuncLit, PkgPath: "plugins/email", Symbol: "(*Client).Send/lit#1", Reason: "传给 go-mail/mail.WithDialContextFunc 的固定拨号适配器回调"},
		// W08: plugins/database/models GORM 实体钩子
		WhitelistEntry{ID: "W08", Contract: "gorm.BeforeCreateInterface", Kind: KindMethod, PkgPath: "plugins/database/models", Symbol: "(*BaseEntity).BeforeCreate", Reason: "GORM BeforeCreate(*gorm.DB) error 钩子固定签名"},
		WhitelistEntry{ID: "W08", Contract: "gorm.BeforeUpdateInterface", Kind: KindMethod, PkgPath: "plugins/database/models", Symbol: "(*BaseEntity).BeforeUpdate", Reason: "GORM BeforeUpdate(*gorm.DB) error 钩子固定签名"},
		WhitelistEntry{ID: "W08", Contract: "gorm.BeforeCreateInterface", Kind: KindMethod, PkgPath: "plugins/database/models", Symbol: "(*BaseEntityWithSoftDelete).BeforeCreate", Reason: "GORM BeforeCreate(*gorm.DB) error 钩子固定签名"},
		WhitelistEntry{ID: "W08", Contract: "gorm.BeforeUpdateInterface", Kind: KindMethod, PkgPath: "plugins/database/models", Symbol: "(*BaseEntityWithSoftDelete).BeforeUpdate", Reason: "GORM BeforeUpdate(*gorm.DB) error 钩子固定签名"},
		// W09: plugins/grpc 桥接器、中间件与链内 handler
		WhitelistEntry{ID: "W09", Contract: "grpc.normalizeRPCError", Kind: KindMethod, PkgPath: "plugins/grpc", Symbol: "(*Plugin).normalizeRPCError", Reason: "gRPC 协议边界 status 归一化适配器"},
		WhitelistEntry{ID: "W09", Contract: "grpc.UnaryServerInterceptor", Kind: KindFuncLit, PkgPath: "plugins/grpc", Symbol: "(*Plugin).unaryBridgeInterceptor/lit#1", Reason: "gRPC UnaryServerInterceptor 匿名桥接实现"},
		WhitelistEntry{ID: "W09", Contract: "grpc.UnaryHandler", Kind: KindFuncLit, PkgPath: "plugins/grpc", Symbol: "(*Plugin).unaryBridgeInterceptor/lit#2", Reason: "gRPC UnaryHandler 链式包装回调"},
		WhitelistEntry{ID: "W09", Contract: "grpc.StreamServerInterceptor", Kind: KindFuncLit, PkgPath: "plugins/grpc", Symbol: "(*Plugin).streamBridgeInterceptor/lit#1", Reason: "gRPC StreamServerInterceptor 匿名桥接实现"},
		WhitelistEntry{ID: "W09", Contract: "grpc.StreamHandler", Kind: KindFuncLit, PkgPath: "plugins/grpc", Symbol: "(*Plugin).streamBridgeInterceptor/lit#2", Reason: "gRPC StreamHandler 链式包装回调"},
		WhitelistEntry{ID: "W09", Contract: "grpc.UnaryServerInterceptor", Kind: KindFuncType, PkgPath: "plugins/grpc/middleware", Symbol: "UnaryMiddlewareFunc", Reason: "第三方 gRPC UnaryServerInterceptor 类型别名"},
		WhitelistEntry{ID: "W09", Contract: "grpc.StreamServerInterceptor", Kind: KindFuncType, PkgPath: "plugins/grpc/middleware", Symbol: "StreamMiddlewareFunc", Reason: "第三方 gRPC StreamServerInterceptor 类型别名"},
		// W10: plugins/grpc/grpcresult 对外协议状态编码适配器
		WhitelistEntry{ID: "W10", Contract: "grpc.StatusError", Kind: KindFunc, PkgPath: "plugins/grpc/grpcresult", Symbol: "ToStatusError", Reason: "对外 gRPC status.Error 编码适配器"},
		WhitelistEntry{ID: "W10", Contract: "grpc.StatusError", Kind: KindFunc, PkgPath: "plugins/grpc/grpcresult", Symbol: "NormalizeToStatusError", Reason: "R12 协议边界原生 status 补齐适配器"},
		WhitelistEntry{ID: "W10", Contract: "grpc.StatusError", Kind: KindFunc, PkgPath: "plugins/grpc/grpcresult", Symbol: "ensureNativeStatusEnvelope", Reason: "NormalizeToStatusError 私有原生信封辅助"},
		WhitelistEntry{ID: "W10", Contract: "anypb.New", Kind: KindVar, PkgPath: "plugins/grpc/grpcresult", Symbol: "marshalStatusDetail", Reason: "anypb.New 包内测试注入桩，签名固定为 func(proto.Message) (*anypb.Any, error)"},
		// W11: plugins/web-gin/validator 第三方翻译注册回调
		WhitelistEntry{ID: "W11", Contract: "validator.RegisterTranslation", Kind: KindFuncLit, PkgPath: "plugins/web-gin/validator", Symbol: "(*Engine).registerBuiltinTranslations/lit#1", Reason: "第三方 translator 注册回调 (oneof)"},
		WhitelistEntry{ID: "W11", Contract: "validator.RegisterTranslation", Kind: KindFuncLit, PkgPath: "plugins/web-gin/validator", Symbol: "(*Engine).registerBuiltinTranslations/lit#2", Reason: "第三方 translator 注册回调 (strict_url)"},
		WhitelistEntry{ID: "W11", Contract: "validator.RegisterTranslation", Kind: KindFuncLit, PkgPath: "plugins/web-gin/validator", Symbol: "(*Engine).registerBuiltinTranslations/lit#3", Reason: "第三方 translator 注册回调 (strict_uuid)"},
		WhitelistEntry{ID: "W11", Contract: "validator.RegisterTranslation", Kind: KindFuncLit, PkgPath: "plugins/web-gin/validator", Symbol: "(*Engine).registerBuiltinTranslations/lit#4", Reason: "第三方 translator 注册回调 (alphanum_underscore)"},
		WhitelistEntry{ID: "W11", Contract: "validator.RegisterTranslation", Kind: KindFuncLit, PkgPath: "plugins/web-gin/validator", Symbol: "(*Engine).registerBuiltinTranslations/lit#5", Reason: "第三方 translator 注册回调 (regexp)"},
		WhitelistEntry{ID: "W11", Contract: "validator.RegisterTranslation", Kind: KindFuncLit, PkgPath: "plugins/web-gin/validator", Symbol: "(*Engine).registerBuiltinTranslations/lit#6", Reason: "第三方 translator 注册回调 (enum_int)"},
		WhitelistEntry{ID: "W11", Contract: "validator.RegisterTranslation", Kind: KindFuncLit, PkgPath: "plugins/web-gin/validator", Symbol: "(*Engine).registerBuiltinTranslations/lit#7", Reason: "第三方 translator 注册回调 (enum_string)"},
		WhitelistEntry{ID: "W11", Contract: "validator.RegisterTranslation", Kind: KindFuncLit, PkgPath: "plugins/web-gin/validator", Symbol: "(*Engine).registerBuiltinTranslations/lit#8", Reason: "第三方 translator 注册回调 (enum_float)"},
		WhitelistEntry{ID: "W11", Contract: "validator.RegisterTranslation", Kind: KindFuncLit, PkgPath: "plugins/web-gin/validator", Symbol: "(*Engine).registerBuiltinTranslations/lit#9", Reason: "第三方 translator 注册回调 (snowflake)"},
		// 结构原因载体字段（非失败出口）
		WhitelistEntry{ID: "STRUCT-CAUSE", Contract: "error-cause-carrier", Kind: KindField, PkgPath: "foundation/apperr", Symbol: "Error.Err", Reason: "结构化错误底层原因字段"},
		WhitelistEntry{ID: "STRUCT-CAUSE", Contract: "error-cause-carrier", Kind: KindField, PkgPath: "foundation/apperr", Symbol: "chainIdentity.value", Reason: "原因链环检测节点键值载体"},
	)

	return entries
}

// ValidateWhitelist 校验白名单台账自身是否合法（无重复、无空字段、无通配符）。
func ValidateWhitelist(entries []WhitelistEntry) error {
	seen := make(map[string]struct{}, len(entries))
	var errs []string
	for i, e := range entries {
		if e.ID == "" || e.Contract == "" || e.Kind == "" || e.PkgPath == "" || e.Symbol == "" || e.Reason == "" {
			errs = append(errs, fmt.Sprintf("entry[%d] (%s) 存在空字段", i, e.Symbol))
			continue
		}
		if hasGlobPattern(e.PkgPath) || hasGlobPattern(e.Symbol) {
			errs = append(errs, fmt.Sprintf("entry %s:%s 含通配符，违反精确符号绑定规则", e.ID, e.Symbol))
		}
		k := e.Key()
		if _, dup := seen[k]; dup {
			errs = append(errs, fmt.Sprintf("重复白名单条目: %s", k))
		}
		seen[k] = struct{}{}
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("白名单台账校验失败:\n  - %s", joinLines(errs))
	}
	return nil
}

func hasGlobPattern(s string) bool {
	if contains(s, "?") || contains(s, "[") || contains(s, "]") || contains(s, "...") {
		return true
	}
	// 仅允许 "(*TypeName)." 作为指针接收者语法，其余位置的 '*' 一律视为通配符
	for i := 0; i < len(s); i++ {
		if s[i] == '*' {
			if i == 1 && s[0] == '(' {
				continue
			}
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func joinLines(lines []string) string {
	out := ""
	for i, l := range lines {
		if i > 0 {
			out += "\n  - "
		}
		out += l
	}
	return out
}
