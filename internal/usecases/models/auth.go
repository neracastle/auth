package models

// AuthTokens токены выдаваемые при авторизации
type AuthTokens struct {
	AccessToken  string
	RefreshToken string
}
