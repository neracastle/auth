package usecases

import (
	"context"
	"errors"

	"github.com/neracastle/go-libs/pkg/sys/logger"
	"golang.org/x/exp/slog"

	"github.com/neracastle/auth/pkg/user_v1/auth"
)

// Delete удаляет пользователя
func (s *Service) Delete(ctx context.Context, userID int64) error {
	log := logger.GetLogger(ctx)
	tokenUser := auth.UserFromContext(ctx)
	log.Debug("called", slog.String("method", "usecases.Delete"), slog.Int64("user_id", userID))
	//получить данные пользователь может только по себе, а админ по всем
	if tokenUser.ID != userID && !tokenUser.IsAdmin {
		return errors.New("нет доступа к данному id")
	}

	return s.usersRepo.Delete(ctx, userID)
}
