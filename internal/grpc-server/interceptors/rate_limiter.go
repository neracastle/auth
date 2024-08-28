package interceptors

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/neracastle/go-libs/pkg/sys/rate_limiter"
)

var rateLimiter *rate_limiter.RateLimiter

func NewRateLimitInterceptor(limiter *rate_limiter.RateLimiter) grpc.UnaryServerInterceptor {
	rateLimiter = limiter

	return interceptor
}

func interceptor(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if rateLimiter.Allow() {
		return handler(ctx, req)
	}

	return nil, status.Error(codes.ResourceExhausted, "Превышено кол-во запросов")
}
