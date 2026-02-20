package xGrpcResult

import (
	"context"
	"reflect"

	xGrpcGenerate "github.com/bamboo-services/bamboo-base-go/grpc/generate"
	xLog "github.com/bamboo-services/bamboo-base-go/log"
)

// Success 返回表示操作成功的标准响应结构。
//
// 该函数会记录 INFO 级别的日志，并构建包含状态码 200 的响应对象。
// 仅填充业务语义字段（Output、Code、Message），请求追踪 ID 和耗时由 ResponseBuilder 拦截器统一注入。
//
// 参数 message 用于传递给调用者的成功提示信息。
//
// 返回填充好的 BaseResponse 指针，其中 Code 为 200，Output 为 "Success"。
func Success(ctx context.Context, message string) *xGrpcGenerate.BaseResponse {
	xLog.WithName(xLog.NamedRESU).Info(ctx, "[200]Success - "+message)
	return &xGrpcGenerate.BaseResponse{
		Output:  "Success",
		Code:    200,
		Message: message,
	}
}

// SuccessWith 创建包含业务数据的成功响应，并通过反射自动注入 BaseResponse 字段。
//
// 泛型参数 T 必须是指向 Protobuf 生成结构体的指针类型（如 *pb.UploadResponse），
// 且该结构体中必须包含名为 BaseResponse、类型为 *xGrpcGenerate.BaseResponse 的嵌入字段。
//
// 函数内部会自动记录 INFO 级别日志、创建 BaseResponse 并通过反射写入 T 的对应字段。
// 仅填充业务语义字段（Output、Code、Message），请求追踪 ID 和耗时由 ResponseBuilder 拦截器统一注入。
// 若 T 不是指针类型或缺少可注入的 BaseResponse 字段，函数会 panic 以在开发阶段尽早暴露错误。
//
// 返回填充好 BaseResponse 的业务响应实例，调用者可继续填充其他业务字段后返回。
func SuccessWith[T any](ctx context.Context, message string) T {
	xLog.WithName(xLog.NamedRESU).Info(ctx, "[200]Success - "+message)

	t := reflect.TypeOf((*T)(nil)).Elem()
	if t.Kind() != reflect.Ptr {
		panic("xGrpcResult.SuccessWith: T must be a pointer type, got " + t.Kind().String())
	}

	instance := reflect.New(t.Elem())
	instance.Elem().FieldByName("BaseResponse").Set(reflect.ValueOf(&xGrpcGenerate.BaseResponse{
		Output:  "Success",
		Code:    200,
		Message: message,
	}))
	return instance.Interface().(T)
}
