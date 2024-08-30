package config

import "time"

type RateLimiter struct {
	Limit  uint          `yaml:"limit" env:"RL_LIMIT" env-default:"0"`
	Period time.Duration `yaml:"period" env:"RL_PERIOD" env-default:"1s"`
}
