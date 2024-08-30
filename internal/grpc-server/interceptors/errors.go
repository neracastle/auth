package interceptors

import (
	"context"
	"errors"

	syserr "github.com/neracastle/go-libs/pkg/sys/error"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCStatusInterface interface {
	GRPCStatus() *status.Status
}

func ErrorCodesInterceptor(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (res interface{}, err error) {
	res, err = handler(ctx, req)
	if nil == err {
		return res, nil
	}

	switch {
	case syserr.IsCommonError(err):
		commEr := syserr.GetCommonError(err)
		code := toGRPCCode(commEr.Code())
		err = status.Error(code, commEr.Error())

	default:
		var se GRPCStatusInterface
		if errors.As(err, &se) {
			return nil, se.GRPCStatus().Err()
		} else {
			if errors.Is(err, context.DeadlineExceeded) {
				err = status.Error(codes.DeadlineExceeded, err.Error())
			} else if errors.Is(err, context.Canceled) {
				err = status.Error(codes.Canceled, err.Error())
			} else {
				err = status.Error(codes.Internal, "internal error")
			}
		}
	}

	return res, err
}

func toGRPCCode(code syserr.Code) codes.Code {
	var res codes.Code

	switch code {
	case syserr.OK:
		res = codes.OK
	case syserr.Canceled:
		res = codes.Canceled
	case syserr.InvalidArgument:
		res = codes.InvalidArgument
	case syserr.DeadlineExceeded:
		res = codes.DeadlineExceeded
	case syserr.NotFound:
		res = codes.NotFound
	case syserr.AlreadyExists:
		res = codes.AlreadyExists
	case syserr.PermissionDenied:
		res = codes.PermissionDenied
	case syserr.ResourceExhausted:
		res = codes.ResourceExhausted
	case syserr.Internal:
		res = codes.Internal
	case syserr.Unauthenticated:
		res = codes.Unauthenticated
	default:
		res = codes.Unknown
	}

	return res
}
