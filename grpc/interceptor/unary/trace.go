package xGrpcIUnary

import (
	"context"
	"log/slog"
	"time"

	xCtx "github.com/bamboo-services/bamboo-base-go/context"
	xGrpcConst "github.com/bamboo-services/bamboo-base-go/grpc/constant"
	xLog "github.com/bamboo-services/bamboo-base-go/log"
	xUtil "github.com/bamboo-services/bamboo-base-go/utility"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Trace 创建用于 gRPC 服务端的一元拦截器，实现请求链路追踪与上下文增强。
//
// 该拦截器会尝试从传入的 gRPC 元数据中提取 `x_request_uuid`。
// 若元数据中不存在该键值，则会自动生成一个新的 UUID 作为请求唯一标识。
// 此标识会被注入到上下文中，用于后续的日志关联和业务逻辑追踪。
//
// 同时，该拦截器会在上下文中记录请求的开始时间，便于计算请求总耗时。
// 在请求处理完成后，它会将 `x_request_uuid` 作为 Trailer 写回给客户端。
//
// 参数说明:
//   - 无参数。
//
// 返回值:
//   - `grpc.UnaryServerInterceptor`: 返回配置好的 gRPC 一元拦截器实例。
//
// 注意:
//   - 如果设置 gRPC Trailer 失败，会记录一条警告日志，但不会中断请求流程。
//   - 上下文中注入的 Key 分别为 `xCtx.RequestKey` 和 `xCtx.UserStartTimeKey`。
func Trace() grpc.UnaryServerInterceptor {
	log := xLog.WithName(xLog.NamedGRPC)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		requestUUID, err := xUtil.Grpc().ExtractMetadata(ctx, xGrpcConst.MetadataRequestUUID)
		if err != nil {
			requestUUID = uuid.NewString()
		}
		traceCtx := context.WithValue(ctx, xCtx.RequestKey, requestUUID)
		traceCtx = context.WithValue(traceCtx, xCtx.UserStartTimeKey, time.Now())

		header := metadata.Pairs(xGrpcConst.TrailerRequestUUID.String(), requestUUID)

		resp, err = handler(traceCtx, req)

		if trailerErr := grpc.SetTrailer(traceCtx, header); trailerErr != nil {
			log.Warn(traceCtx, "设置 gRPC 请求追踪尾失败", slog.Any("error", trailerErr))
		}

		return resp, err
	}
}
