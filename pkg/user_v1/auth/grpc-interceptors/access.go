package interceptors

import (
	"context"
	"strings"

	"golang.org/x/exp/slices"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/neracastle/auth/pkg/user_v1/auth"
)

const authHeader = "Authorization"
const authPrefix = "Bearer "

var secureMethodsMap map[string]struct{}
var secretKey string

// NewAccessInterceptor для заданных методов проверяет наличие access-токена и наличие соответствующего scope в нем
// так же при успешной проверке записывает данные из токена в контекст
func NewAccessInterceptor(secureMethods []string, jwtSecretKey string) grpc.UnaryServerInterceptor {
	secretKey = jwtSecretKey

	if len(secureMethods) > 0 {
		secureMethodsMap = make(map[string]struct{}, len(secureMethods))
		for _, m := range secureMethods {
			secureMethodsMap[m] = struct{}{}
		}
	}

	return accessInterceptor
}

func accessInterceptor(ctx context.Context, req interface{}, i *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	//смотрим требует ли метод проверки доступа
	//если да, то смотрим наличие метода в scope разделе токена
	if _, needCheck := secureMethodsMap[i.FullMethod]; needCheck {
		meta, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
		}

		token := meta.Get(authHeader)
		if len(token) == 0 {
			return nil, status.Error(codes.Unauthenticated, "token is not provided")
		}

		if !strings.HasPrefix(token[0], authPrefix) {
			return nil, status.Error(codes.Unauthenticated, "invalid auth header format")
		}

		accessToken := strings.TrimPrefix(token[0], authPrefix)
		user, err := auth.ParseToken(accessToken, []byte(secretKey))
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}

		if !slices.Contains(user.Scope, i.FullMethod) {
			return nil, status.Error(codes.PermissionDenied, "нет доступа")
		}

		ctx = auth.AddUserToContext(ctx, user)
	}

	return handler(ctx, req)
}
