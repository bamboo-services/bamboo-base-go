package xGrpcIStream

import (
	"context"
	"log/slog"
	"strings"
	"time"

	xCtx "github.com/bamboo-services/bamboo-base-go/context"
	xHttp "github.com/bamboo-services/bamboo-base-go/http"
	xLog "github.com/bamboo-services/bamboo-base-go/log"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Trace 返回一个 gRPC 流式拦截器，用于自动生成或复用请求追踪 UUID，记录请求开始时间，并设置响应元数据。
func Trace() grpc.StreamServerInterceptor {
	traceHeaderKey := strings.ToLower(xHttp.HeaderRequestUUID.String())
	log := xLog.WithName(xLog.NamedGRPC)

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		requestUUID := resolveRequestUUID(ss.Context(), traceHeaderKey)
		traceCtx := context.WithValue(ss.Context(), xCtx.RequestKey, requestUUID)
		traceCtx = context.WithValue(traceCtx, xCtx.UserStartTimeKey, time.Now())

		// 设置 header
		if headerErr := ss.SetHeader(metadata.Pairs(traceHeaderKey, requestUUID)); headerErr != nil {
			log.Warn(traceCtx, "设置 gRPC 流式请求追踪头失败", slog.Any("error", headerErr))
		}

		wrapped := &wrappedServerStream{ServerStream: ss, ctx: traceCtx}
		err := handler(srv, wrapped)

		// 设置 trailer
		ss.SetTrailer(metadata.Pairs(traceHeaderKey, requestUUID))

		return err
	}
}

func resolveRequestUUID(ctx context.Context, headerKey string) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get(headerKey)
		for _, value := range values {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return uuid.NewString()
}
