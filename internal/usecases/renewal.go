package usecases

import (
	"context"

	"github.com/neracastle/go-libs/pkg/sys/logger"
	"golang.org/x/exp/slog"

	"github.com/neracastle/auth/pkg/user_v1/auth"
)

// Renewal перевыпускает Access/Refresh токен
func (s *Service) Renewal(ctx context.Context, refreshToken string, isRenewAccess bool) (string, error) {
	log := logger.GetLogger(ctx)
	log.Debug("called", slog.String("method", "usecases.Renewal"))

	parsed, err := auth.ParseToken(refreshToken, []byte(s.Config.SecretKey))
	if err != nil {
		log.Error("failed to parse refresh token", err.Error())
		return "", err
	}

	duration := s.Config.AccessDuration
	if !isRenewAccess {
		duration = s.Config.RefreshDuration
	}

	token, err := auth.GenerateToken(parsed, []byte(s.Config.SecretKey), duration)
	if err != nil {
		log.Error("failed to generate token", err.Error())
		return "", err
	}

	return token, nil
}
