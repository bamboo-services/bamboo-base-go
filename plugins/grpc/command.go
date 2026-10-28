package xGrpc

import (
	"google.golang.org/grpc/codes"
)

// ToGrpcStatusCode 将业务错误码转换为对应的标准 gRPC 状态码。
//
// 该函数接收 5 位业务错误码（例如 40001、50000 等），通过截取前 3 位（即 code / 100）
// 获取等价的 HTTP 状态码，并依循映射契约转换为对应的 [codes.Code]：
//   - 400、415：映射为 [codes.InvalidArgument]；
//   - 401：映射为 [codes.Unauthenticated]；
//   - 403：映射为 [codes.PermissionDenied]；
//   - 404、410：映射为 [codes.NotFound]；
//   - 405：映射为 [codes.Unimplemented]；
//   - 406、422：映射为 [codes.FailedPrecondition]；
//   - 408、504：映射为 [codes.DeadlineExceeded]；
//   - 409：映射为 [codes.Aborted]；
//   - 413、429：映射为 [codes.ResourceExhausted]；
//   - 500：映射为 [codes.Internal]；
//   - 502、503：映射为 [codes.Unavailable]；
//   - 其余未列出的状态码：一律回退为 [codes.Unknown]。
//
// 该函数为纯函数，不依赖外部状态，可被多 goroutine 并发调用。
func ToGrpcStatusCode(code uint) codes.Code {
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
