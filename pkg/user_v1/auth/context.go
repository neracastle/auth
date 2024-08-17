package auth

import (
	"context"
)

// AuthorisedUserIDKey ключ для размещения данных авторизации в контексте
type AuthorisedUserIDKey struct{}

// AddUserToContext Добавляет данные авторизованного пользователя из access-токена в контекст
func AddUserToContext(ctx context.Context, user JWTUser) context.Context {
	ctx = context.WithValue(ctx, AuthorisedUserIDKey{}, user)

	return ctx
}

// UserFromContext Получает данные access-токена (авторизованного пользователя) из контекста
func UserFromContext(ctx context.Context) *JWTUser {
	if ctx == nil {
		return nil
	}

	userID := ctx.Value(AuthorisedUserIDKey{}).(JWTUser)

	return &userID
}
