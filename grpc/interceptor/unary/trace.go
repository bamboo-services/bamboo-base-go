package xGrpcIUnary

import (
	"context"
	"log/slog"
	"strings"
	"time"

	xCtx "github.com/bamboo-services/bamboo-base-go/context"
	xGrpcConst "github.com/bamboo-services/bamboo-base-go/grpc/constant"
	xLog "github.com/bamboo-services/bamboo-base-go/log"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Trace 返回一个 gRPC 一元拦截器，用于自动生成或复用请求追踪 UUID，记录请求开始时间，并设置响应元数据。
func Trace() grpc.UnaryServerInterceptor {
	traceHeaderKey := xGrpcConst.TrailerRequestUUID.String()
	log := xLog.WithName(xLog.NamedGRPC)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		requestUUID := resolveRequestUUID(ctx, traceHeaderKey)
		traceCtx := context.WithValue(ctx, xCtx.RequestKey, requestUUID)
		traceCtx = context.WithValue(traceCtx, xCtx.UserStartTimeKey, time.Now())

		header := metadata.Pairs(traceHeaderKey, requestUUID)
		if headerErr := grpc.SetHeader(traceCtx, header); headerErr != nil {
			log.Warn(traceCtx, "设置 gRPC 请求追踪头失败", slog.Any("error", headerErr))
		}

		resp, err = handler(traceCtx, req)

		if trailerErr := grpc.SetTrailer(traceCtx, header); trailerErr != nil {
			log.Warn(traceCtx, "设置 gRPC 请求追踪尾失败", slog.Any("error", trailerErr))
		}

		return resp, err
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
