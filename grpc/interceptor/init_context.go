package xGrpcInterface

import (
	"context"
	"log/slog"

	xCtx "github.com/bamboo-services/bamboo-base-go/context"
	xError "github.com/bamboo-services/bamboo-base-go/error"
	xGrpcError "github.com/bamboo-services/bamboo-base-go/grpc/error"
	xGrpcResult "github.com/bamboo-services/bamboo-base-go/grpc/result"
	xLog "github.com/bamboo-services/bamboo-base-go/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// InitContext 从主上下文中提取并传播注册节点列表，创建一个 gRPC 一元拦截器。
//
// 该函数从 mainCtx 中检索 RegNodeKey 对应的 ContextNodeList，
// 并将其注入到后续 RPC 请求的上下文中。
// 同时，拦截器负责将 RPC 处理器返回的错误转换为标准的 gRPC 状态错误。
//
// 参数 mainCtx 用于提取节点列表的根上下文。
//
// 返回的拦截器确保所有 RPC 调用共享相同的上下文节点链路。
func InitContext(mainCtx context.Context) grpc.UnaryServerInterceptor {
	var ctxNodeList xCtx.ContextNodeList
	if val := mainCtx.Value(xCtx.RegNodeKey); val != nil {
		if nodeList, ok := val.(xCtx.ContextNodeList); ok {
			ctxNodeList = nodeList
		}
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		newCtx := context.WithValue(ctx, xCtx.RegNodeKey, ctxNodeList)

		resp, err = handler(newCtx, req)
		if err != nil {
			resp = nil
			err = toStatusError(newCtx, xGrpcError.From(err))
		}
		return resp, err
	}
}

func toStatusError(ctx context.Context, grpcError *xGrpcError.Error) error {
	normalizedError := grpcError
	if normalizedError == nil {
		normalizedError = xGrpcError.New(xError.UnknownError, xGrpcError.ErrMessage(""), nil)
	}

	baseResponse := xGrpcResult.ErrorByGrpcError(ctx, normalizedError)
	grpcStatus := status.New(toGrpcStatusCode(normalizedError.GetErrorCode().Code), normalizedError.Error())
	grpcStatusWithDetails, detailError := grpcStatus.WithDetails(baseResponse)
	if detailError != nil {
		xLog.WithName(xLog.NamedGRPC).Warn(ctx, "附上 GRPC 错误详情失败",
			slog.Any("error", detailError),
		)
		return grpcStatus.Err()
	}
	return grpcStatusWithDetails.Err()
}

func toGrpcStatusCode(code uint) codes.Code {
	httpCode := code / 100
	switch httpCode {
	case 400:
		return codes.InvalidArgument
	case 401:
		return codes.Unauthenticated
	case 403:
		return codes.PermissionDenied
	case 404:
		return codes.NotFound
	case 405:
		return codes.Unimplemented
	case 406:
		return codes.FailedPrecondition
	case 408:
		return codes.DeadlineExceeded
	case 409:
		return codes.Aborted
	case 410:
		return codes.NotFound
	case 413:
		return codes.ResourceExhausted
	case 415:
		return codes.InvalidArgument
	case 422:
		return codes.FailedPrecondition
	case 429:
		return codes.ResourceExhausted
	case 500:
		return codes.Internal
	case 502:
		return codes.Unavailable
	case 503:
		return codes.Unavailable
	case 504:
		return codes.DeadlineExceeded
	default:
		return codes.Unknown
	}
}
