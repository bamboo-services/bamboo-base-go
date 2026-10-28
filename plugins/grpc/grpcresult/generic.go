package grpcresult

import (
	"context"
	"fmt"
	"reflect"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const baseResponseFullName protoreflect.FullName = "xBase.BaseResponse"

// SuccessWith 创建并返回包含标准成功响应信封的 Protobuf 消息实例。
//
// 该函数在框架内部通过反射创建类型参数 T 的新结构体实例，并自动查找顶层字段中唯一匹配
// 框架 [generate.BaseResponse]（消息全称为 "xBase.BaseResponse"）的字段，调用 [Success]
// 构造成功信封注入该字段，最后将填充好的消息实例返回给业务调用方。
//
// # 泛型类型参数 T 的契约约束
//
//   - T 必须是指向具体结构体的指针类型（pointer to struct），禁止传入接口类型或非指针类型；
//   - T 必须实现 [proto.Message] 接口；
//   - T 所对应的 Protobuf Descriptor 顶层必须恰好有且仅有一个消息字段的类型为 [generate.BaseResponse]；
//   - 不支持嵌套深层查找，不对任意类型做无条件承诺；
//   - 任何违背上述约束的使用均属于编程错误，函数将立即触发 panic 快速失败。
//
// # 耗时与追踪注入
//
// 生成的 [generate.BaseResponse] 会自动从传入的 ctx 中提取请求追踪标识（[ctxkey.RequestKey]）；
// 仅当应用处于调试模式且上下文携带请求起始时间（[ctxkey.UserStartTimeKey]）时，才会计算并填充
// 微秒级整数耗时 Overhead，其余情况 Overhead 稳健保留为 nil。
func SuccessWith[T any](ctx context.Context, message string) T {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if t == nil {
		panic("grpcresult.SuccessWith: type parameter T must not be untyped nil")
	}
	if t.Kind() == reflect.Interface {
		panic(fmt.Sprintf("grpcresult.SuccessWith: T must be a concrete protobuf message pointer, got interface %s", t.String()))
	}
	if t.Kind() != reflect.Pointer {
		panic(fmt.Sprintf("grpcresult.SuccessWith: T must be a pointer type, got %s", t.Kind().String()))
	}
	elem := t.Elem()
	if elem.Kind() != reflect.Struct {
		panic(fmt.Sprintf("grpcresult.SuccessWith: T must be a pointer to struct, got pointer to %s", elem.Kind().String()))
	}

	protoMsgType := reflect.TypeOf((*proto.Message)(nil)).Elem()
	if !t.Implements(protoMsgType) {
		panic(fmt.Sprintf("grpcresult.SuccessWith: T (%s) does not implement proto.Message", t.String()))
	}

	instanceVal := reflect.New(elem)
	msg, ok := instanceVal.Interface().(proto.Message)
	if !ok {
		panic(fmt.Sprintf("grpcresult.SuccessWith: instantiated %s does not implement proto.Message", t.String()))
	}

	refl := msg.ProtoReflect()
	if refl == nil {
		panic(fmt.Sprintf("grpcresult.SuccessWith: ProtoReflect() returned nil for %s", t.String()))
	}
	desc := refl.Descriptor()
	if desc == nil {
		panic(fmt.Sprintf("grpcresult.SuccessWith: message descriptor is nil for %s", t.String()))
	}

	fields := desc.Fields()
	var matchedField protoreflect.FieldDescriptor
	matchCount := 0

	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if fd.Kind() == protoreflect.MessageKind {
			msgDesc := fd.Message()
			if msgDesc != nil && msgDesc.FullName() == baseResponseFullName {
				matchedField = fd
				matchCount++
			}
		}
	}

	if matchCount == 0 {
		panic(fmt.Sprintf("grpcresult.SuccessWith: %s has no field of type %s", desc.FullName(), baseResponseFullName))
	}
	if matchCount > 1 {
		panic(fmt.Sprintf("grpcresult.SuccessWith: %s has %d fields of type %s, expected exactly 1", desc.FullName(), matchCount, baseResponseFullName))
	}

	baseResp := Success(ctx, message)
	refl.Set(matchedField, protoreflect.ValueOfMessage(baseResp.ProtoReflect()))

	res, ok := instanceVal.Interface().(T)
	if !ok {
		panic(fmt.Sprintf("grpcresult.SuccessWith: failed to cast instance to %s", t.String()))
	}
	return res
}
