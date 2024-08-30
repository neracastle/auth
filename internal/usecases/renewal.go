package usecases

import (
	"context"
	"errors"
	"log/slog"

	syserr "github.com/neracastle/go-libs/pkg/sys/error"
	"github.com/neracastle/go-libs/pkg/sys/logger"

	"github.com/neracastle/auth/pkg/user_v1/auth"
)

// Renewal перевыпускает Access/Refresh токен
func (s *Service) Renewal(ctx context.Context, refreshToken string, isRenewAccess bool) (string, error) {
	log := logger.GetLogger(ctx).With(slog.String("method", "usecases.Renewal"))
	log.Debug("called")

	parsed, err := auth.ParseToken(refreshToken, []byte(s.Config.SecretKey))
	if err != nil {
		log.Error("failed to parse refresh token", err.Error())

		if errors.Is(err, auth.ErrTokenExpired) {
			err = syserr.New("Срок действия токена истек", syserr.PermissionDenied)
		}

		return "", err
	}

	duration := s.Config.AccessDuration
	if !isRenewAccess {
		duration = s.Config.RefreshDuration
	}

	token, err := auth.GenerateToken(parsed, []byte(s.Config.SecretKey), duration)
	if err != nil {
		log.Error("failed to generate token", err.Error())
		return "", syserr.New("Не удалось перевыпустить токен", syserr.Internal)
	}

	return token, nil
}
