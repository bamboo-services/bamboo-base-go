package grpcresult

import (
	"context"
	"fmt"
	"reflect"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const baseResponseFullName protoreflect.FullName = "xBase.BaseResponse"

// SuccessWith 创建包含业务响应数据的 Protobuf 消息实例，并通过 Protobuf 反射自动查找并注入 BaseResponse 字段。
//
// 约束规则：
// 1. T 必须为具体 Protobuf 消息指针（pointer to struct implementing proto.Message）；
// 2. T 顶层消息字段中必须恰好有且仅有一个 message 字段，其 message descriptor 匹配框架 BaseResponse（FullName 为 xBase.BaseResponse）；
// 3. 任何非 pointer、interface、非 proto.Message、缺失 BaseResponse 字段、存在多个 BaseResponse 字段等违背约定的情况均属于编程错误，直接 panic。
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
