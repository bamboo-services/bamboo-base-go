package xGrpcIUnary

import (
	"context"
	"errors"
	"reflect"

	xError "github.com/bamboo-services/bamboo-base-go/error"
	xGrpc "github.com/bamboo-services/bamboo-base-go/grpc"
	xGrpcGenerate "github.com/bamboo-services/bamboo-base-go/grpc/generate"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/utility/ctxutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// ResponseBuilder 返回一个 gRPC 一元拦截器，负责统一构建最终响应。
//
// 行为逻辑：
//   - handler 返回错误：提取 *xError.Error，映射为 gRPC status error 直接返回。
//   - handler 返回成功响应：注入请求追踪 ID 和耗时后放行。
//   - handler 既无响应也无错误：视为开发者错误，返回 DeveloperError。
func ResponseBuilder() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)

		if err != nil {
			xErr := fromError(ctx, err)
			return nil, toStatusError(xErr)
		}

		if resp != nil {
			if br := extractBaseResponse(resp); br != nil {
				fillResponseMeta(ctx, br)
			}
			return resp, nil
		}

		xErr := xError.NewError(ctx, xError.DeveloperError, "没有正常输出信息或报错信息，请检查代码逻辑「开发者错误」", false)
		return nil, toStatusError(xErr)
	}
}

// fromError 从标准 error 中提取 *xError.Error。
//
// 优先使用 errors.As 尝试提取 *xError.Error，若失败则包装为 ServerInternalError。
func fromError(ctx context.Context, err error) *xError.Error {
	var xErr *xError.Error
	if errors.As(err, &xErr) {
		return xErr
	}
	return xError.NewError(ctx, xError.ServerInternalError, xError.ErrMessage(err.Error()), false, err)
}

// toStatusError 将 *xError.Error 映射为 gRPC status error。
func toStatusError(xErr *xError.Error) error {
	return status.Error(xGrpc.ToGrpcStatusCode(xErr.GetErrorCode().Code), xErr.Error())
}

// extractBaseResponse 从任意响应对象中提取 *BaseResponse。
//
// 支持两种结构：
//   - 直接 *BaseResponse（来自 Success()）
//   - 业务结构体中嵌入的 BaseResponse 字段（来自 SuccessWith[T]()）
func extractBaseResponse(resp interface{}) *xGrpcGenerate.BaseResponse {
	if br, ok := resp.(*xGrpcGenerate.BaseResponse); ok {
		return br
	}
	v := reflect.ValueOf(resp)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		elem := v.Elem()
		if elem.Kind() == reflect.Struct {
			field := elem.FieldByName("BaseResponse")
			if field.IsValid() && !field.IsNil() {
				if br, ok := field.Interface().(*xGrpcGenerate.BaseResponse); ok {
					return br
				}
			}
		}
	}
	return nil
}

// fillResponseMeta 向 BaseResponse 注入请求追踪 ID 和处理耗时。
//
// 该函数由 ResponseBuilder 拦截器在成功响应路径上调用，确保元数据统一注入。
func fillResponseMeta(ctx context.Context, baseResp *xGrpcGenerate.BaseResponse) {
	if baseResp == nil {
		return
	}
	baseResp.Context = xCtxUtil.GetRequestKey(ctx)
	overhead := xCtxUtil.CalcOverheadTime(ctx) / 1000
	if overhead > 0 {
		baseResp.Overhead = &overhead
	}
}
