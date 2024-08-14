package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrTokenExpired возвращается если срок действия токена истек
	ErrTokenExpired = jwt.ErrTokenExpired
	// ErrTokenInvalid возвращается во всех случаях ошибки обработки токена, кроме истечения срока действия
	ErrTokenInvalid = errors.New("token is invalid")
)

// GenerateToken генерирует новый токен
func GenerateToken(user JWTUser, secretKey []byte, duration time.Duration) (string, error) {
	userClaims := ClaimUser{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
		},
		JWTUser: user,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, userClaims)

	return token.SignedString(secretKey)
}

// ParseToken парсит токен и проверяет его валидность
func ParseToken(tokenString string, secretKey []byte) (JWTUser, error) {
	token, err := jwt.ParseWithClaims(tokenString, &ClaimUser{}, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return JWTUser{}, ErrTokenExpired
		}

		return JWTUser{}, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*ClaimUser)
	if !ok {
		return JWTUser{}, ErrTokenInvalid
	}

	return JWTUser{
		ID:      claims.JWTUser.ID,
		IsAdmin: claims.JWTUser.IsAdmin,
		Scope:   claims.JWTUser.Scope,
	}, nil
}
