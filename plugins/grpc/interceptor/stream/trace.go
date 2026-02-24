package xGrpcIStream

import (
	"context"
	"log/slog"
	"time"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtx "github.com/bamboo-services/bamboo-base-go/defined/context"
	xGrpcConst "github.com/bamboo-services/bamboo-base-go/plugins/grpc/constant"
	xGrpcUtil "github.com/bamboo-services/bamboo-base-go/plugins/grpc/utility"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Trace 返回一个 gRPC 流式拦截器，用于自动生成或复用请求追踪 UUID，记录请求开始时间，并设置响应元数据。
func Trace() grpc.StreamServerInterceptor {
	log := xLog.WithName(xLog.NamedGRPC)

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		requestUUID, extractErr := xGrpcUtil.ExtractMetadata(ss.Context(), xGrpcConst.MetadataRequestUUID)
		if extractErr != nil {
			requestUUID = uuid.NewString()
		}
		traceCtx := context.WithValue(ss.Context(), xCtx.RequestKey, requestUUID)
		traceCtx = context.WithValue(traceCtx, xCtx.UserStartTimeKey, time.Now())

		// 设置 header 和 trailer
		md := metadata.Pairs(xGrpcConst.TrailerRequestUUID.String(), requestUUID)
		if headerErr := ss.SetHeader(md); headerErr != nil {
			log.Warn(traceCtx, "设置 gRPC 请求追踪头失败", slog.Any("error", headerErr))
		}

		wrapped := &wrappedServerStream{ServerStream: ss, ctx: traceCtx}
		err := handler(srv, wrapped)

		// 设置 trailer
		ss.SetTrailer(md)

		return err
	}
}
