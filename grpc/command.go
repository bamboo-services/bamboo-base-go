package xGrpc

import "google.golang.org/grpc/codes"

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
