package xGrpcIStream

import (
	"context"
	"fmt"
	"log/slog"

	xError "github.com/bamboo-services/bamboo-base-go/error"
	xGrpc "github.com/bamboo-services/bamboo-base-go/grpc"
	xLog "github.com/bamboo-services/bamboo-base-go/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// Recover 返回一个 gRPC 流式拦截器，用于捕获服务端处理过程中的 panic 并将其转换为标准的 gRPC 错误。
//
// 该拦截器通过 defer 和 recover() 机制保护 handler 调用，防止 panic 导致整个程序崩溃。
// 当检测到 panic 时，会使用 ServerInternalError 错误码构建错误响应，并记录包含 panic 值
// 和 RPC 方法的详细日志。
func Recover() grpc.StreamServerInterceptor {
	log := xLog.WithName(xLog.NamedGRPC)
	toPanicError := func(ctx context.Context, recovered interface{}) *xError.Error {
		if recoveredError, ok := recovered.(error); ok {
			return xError.NewError(
				ctx,
				xError.ServerInternalError,
				xError.ErrMessage(recoveredError.Error()),
				false,
				recoveredError,
			)
		}

		return xError.NewError(
			ctx,
			xError.ServerInternalError,
			xError.ErrMessage(fmt.Sprint(recovered)),
			false,
		)
	}

	recoverPanicStatusError := func(ctx context.Context, info *grpc.StreamServerInfo, recovered interface{}) error {
		method := ""
		if info != nil {
			method = info.FullMethod
		}

		log.Error(ctx, "gRPC 流式拦截器捕获到 panic",
			slog.Any("method", method),
			slog.Any("panic", recovered),
		)
		xErr := toPanicError(ctx, recovered)
		return toStatusError(xErr)
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				err = recoverPanicStatusError(ss.Context(), info, recovered)
			}
		}()
		return handler(srv, ss)
	}
}

// toStatusError 将 *xError.Error 映射为 gRPC status error。
func toStatusError(xErr *xError.Error) error {
	return status.Error(xGrpc.ToGrpcStatusCode(xErr.GetErrorCode().Code), xErr.Error())
}
