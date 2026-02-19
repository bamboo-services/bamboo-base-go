package xGrpcResult

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	xError "github.com/bamboo-services/bamboo-base-go/error"
	xGrpcError "github.com/bamboo-services/bamboo-base-go/grpc/error"
	xGrpcGenerate "github.com/bamboo-services/bamboo-base-go/grpc/generate"
	xLog "github.com/bamboo-services/bamboo-base-go/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/utility/ctxutil"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

// Success 返回表示操作成功的标准响应结构。
//
// 该函数会记录 INFO 级别的日志，并构建包含状态码 200 的响应对象。
// 上下文中的请求唯一标识会被提取并写入响应，请求耗时也会自动计算并写入。
//
// 参数 ctx 必须包含请求元数据以便计算耗时和获取 Key。
// 参数 message 用于传递给调用者的成功提示信息。
//
// 返回填充好的 BaseResponse 指针，其中 Code 为 200，Output 为 "Success"。
func Success(ctx context.Context, message string) *xGrpcGenerate.BaseResponse {
	xLog.WithName(xLog.NamedRESU).Info(ctx, "[200]Success - "+message)
	return &xGrpcGenerate.BaseResponse{
		Context:  xCtxUtil.GetRequestKey(ctx),
		Output:   "Success",
		Code:     200,
		Message:  message,
		Overhead: toOverheadPointer(ctx),
	}
}

// SuccessHasData 生成包含业务数据的成功响应对象。
//
// 该函数用于封装操作成功且包含返回数据的响应，状态码固定为 200。
// 函数内部会自动计算请求耗时并将 payload 序列化为 Protobuf Any 类型。
// 同时会记录一条 Info 级别的结构化日志。
//
// 参数 ctx 用于传递请求上下文和获取链路追踪信息。
// 参数 message 为返回给客户端的成功提示信息。
// 参数 data 为具体的业务数据载荷，支持 proto.Message 或泛型结构。
//
// 返回填充完整字段的 BaseResponse 指针，其中 Data 字段已通过 Any 类型打包。
func SuccessHasData(ctx context.Context, message string, data interface{}) *xGrpcGenerate.BaseResponse {
	xLog.WithName(xLog.NamedRESU).Info(ctx, "[200]Success - "+message,
		slog.Any("data", data),
	)
	return &xGrpcGenerate.BaseResponse{
		Context:  xCtxUtil.GetRequestKey(ctx),
		Output:   "Success",
		Code:     200,
		Message:  message,
		Overhead: toOverheadPointer(ctx),
		Data:     toProtoAny(ctx, data),
	}
}

// Error 根据错误码和消息构建标准的错误响应结构。
//
// 该函数会规范化输入的错误码和错误信息，使用 Warn 级别记录包含详细上下文的日志，
// 并填充 BaseResponse 的所有必要字段（包括 Protobuf Any 类型的数据载荷和开销统计）。
// 如果 errorCode 为 nil，默认使用 UnknownError；如果 errorMessage 为空，默认使用 errorCode 中的 Message。
func Error(ctx context.Context, errorCode *xError.ErrorCode, errorMessage xGrpcError.ErrMessage, data interface{}) *xGrpcGenerate.BaseResponse {
	normalizedCode := normalizeErrorCode(errorCode)
	normalizedMessage := normalizeErrorMessage(normalizedCode, errorMessage)

	messageBuilder := strings.Builder{}
	messageBuilder.WriteString("[")
	messageBuilder.WriteString(strconv.Itoa(int(normalizedCode.Code)))
	messageBuilder.WriteString("]")
	messageBuilder.WriteString(normalizedCode.GetOutput())
	messageBuilder.WriteString(" | ")
	messageBuilder.WriteString(normalizedCode.Message)
	messageBuilder.WriteString(" - ")
	messageBuilder.WriteString(string(normalizedMessage))
	xLog.WithName(xLog.NamedRESU).Warn(ctx, messageBuilder.String(),
		slog.Uint64("code", uint64(normalizedCode.Code)),
		slog.String("output", normalizedCode.GetOutput()),
		slog.Any("data", data),
	)
	return &xGrpcGenerate.BaseResponse{
		Context:      xCtxUtil.GetRequestKey(ctx),
		Output:       normalizedCode.GetOutput(),
		Code:         uint64(normalizedCode.Code),
		Message:      normalizedCode.Message,
		Overhead:     toOverheadPointer(ctx),
		ErrorMessage: toErrorMessagePointer(normalizedMessage),
		Data:         toProtoAny(ctx, data),
		Error:        toProtoGrpcError(ctx, normalizedCode, normalizedMessage, data),
	}
}

// ErrorByGrpcError 将 gRPC 业务错误对象转换为标准的统一响应结构体。
//
// 如果传入的 gRPCError 为 nil，则使用 UnknownError 错误码进行响应。
// 否则，提取其中的错误码、错误消息和附加数据构建 BaseResponse。
func ErrorByGrpcError(ctx context.Context, grpcError *xGrpcError.Error) *xGrpcGenerate.BaseResponse {
	if grpcError == nil {
		return Error(ctx, xError.UnknownError, xGrpcError.ErrMessage(""), nil)
	}
	return Error(ctx, grpcError.GetErrorCode(), grpcError.GetErrorMessage(), grpcError.GetData())
}

func AbortError(ctx context.Context, errorCode *xError.ErrorCode, errorMessage xGrpcError.ErrMessage, data interface{}) *xGrpcGenerate.BaseResponse {
	return Error(ctx, errorCode, errorMessage, data)
}

func toOverheadPointer(ctx context.Context) *int64 {
	overhead := xCtxUtil.CalcOverheadTime(ctx) / 1000
	if overhead <= 0 {
		return nil
	}
	return &overhead
}

func toErrorMessagePointer(errorMessage xGrpcError.ErrMessage) *string {
	message := string(errorMessage)
	if message == "" {
		return nil
	}
	return &message
}

func toProtoGrpcError(ctx context.Context, errorCode *xError.ErrorCode, errorMessage xGrpcError.ErrMessage, data interface{}) *xGrpcGenerate.GrpcError {
	normalizedCode := normalizeErrorCode(errorCode)
	normalizedMessage := normalizeErrorMessage(normalizedCode, errorMessage)
	return &xGrpcGenerate.GrpcError{
		Code:         uint64(normalizedCode.Code),
		Output:       normalizedCode.GetOutput(),
		Message:      normalizedCode.Message,
		ErrorMessage: string(normalizedMessage),
		Data:         toProtoAny(ctx, data),
	}
}

func normalizeErrorCode(errorCode *xError.ErrorCode) *xError.ErrorCode {
	if errorCode == nil {
		return xError.UnknownError
	}
	return errorCode
}

func normalizeErrorMessage(errorCode *xError.ErrorCode, errorMessage xGrpcError.ErrMessage) xGrpcError.ErrMessage {
	if errorMessage == "" {
		return xGrpcError.ErrMessage(errorCode.Message)
	}
	return errorMessage
}

func toProtoAny(ctx context.Context, data interface{}) *anypb.Any {
	if data == nil {
		return nil
	}

	if anyData, ok := data.(*anypb.Any); ok {
		return anyData
	}

	if protoMessage, ok := data.(proto.Message); ok {
		anyData, err := anypb.New(protoMessage)
		if err == nil {
			return anyData
		}
		xLog.WithName(xLog.NamedRESU).Warn(ctx, "proto message pack to Any failed",
			slog.Any("error", err),
		)
		return nil
	}

	structValue, err := structpb.NewValue(data)
	if err != nil {
		xLog.WithName(xLog.NamedRESU).Warn(ctx, "generic data convert to structpb.Value failed",
			slog.Any("error", err),
		)
		return nil
	}

	anyData, err := anypb.New(structValue)
	if err != nil {
		xLog.WithName(xLog.NamedRESU).Warn(ctx, "structpb.Value pack to Any failed",
			slog.Any("error", err),
		)
		return nil
	}
	return anyData
}
