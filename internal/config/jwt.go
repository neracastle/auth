package config

import "time"

// JWT настройки jwt-токенов
type JWT struct {
	SecretKey       string        `yaml:"secret_key" env:"JWT_SECRET_KEY" env-required:"true"`
	AccessDuration  time.Duration `yaml:"access_duration" env:"JWT_ACCESS_DURATION" env-default:"5m"`
	RefreshDuration time.Duration `yaml:"refresh_duration" env:"JWT_REFRESH_DURATION" env-default:"24h"`
}
