package auth

import "github.com/golang-jwt/jwt/v5"

// JWTUser данные для помещения в токены
type JWTUser struct {
	ID      int64    `json:"user_id"`
	IsAdmin bool     `json:"is_admin"`
	Scope   []string `json:"scope"`
}

// ClaimUser данные для помещения в токен
type ClaimUser struct {
	JWTUser
	jwt.RegisteredClaims
}
