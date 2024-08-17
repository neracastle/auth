package usecases

import (
	"context"
	"errors"

	"github.com/neracastle/go-libs/pkg/sys/logger"
	"golang.org/x/exp/slog"

	"github.com/neracastle/auth/internal/repository/user"
	"github.com/neracastle/auth/internal/usecases/models"
	"github.com/neracastle/auth/pkg/user_v1/auth"
)

// Get возвращает пользователя по его id
func (s *Service) Get(ctx context.Context, userID int64) (models.UserDTO, error) {
	log := logger.GetLogger(ctx)
	tokenUser := auth.UserFromContext(ctx)
	log.Debug("called", slog.String("method", "usecases.Get"), slog.Int64("user_id", userID), slog.Int64("auth_user_id", tokenUser.ID))

	//получить данные пользователь может только по себе, а админ по всем
	if tokenUser.ID != userID && !tokenUser.IsAdmin {
		return models.UserDTO{}, errors.New("нет доступа к данному id")
	}

	//проверяем сначало наличие в кэше
	dbUser, err := s.usersCache.GetByID(ctx, userID)

	if err == nil {
		log.Debug("user found in cache", slog.Int64("user_id", userID))
		return models.FromDomainToUsecase(dbUser), nil
	}

	if !errors.Is(err, user.ErrUserNotCached) {
		log.Error("failed to get user from redis cache", slog.Int64("user_id", userID), slog.String("error", err.Error()))
	} else {
		log.Debug("user not cached", slog.Int64("user_id", userID))
	}

	dbUser, err = s.usersRepo.Get(ctx, user.SearchFilter{ID: userID})
	if err != nil {
		return models.UserDTO{}, err
	}

	//ошибка сохранения в кэш не влияет на выдачу результата, просто залогируем
	err = s.usersCache.Save(ctx, dbUser, s.Config.CacheTTL)
	if err != nil {
		log.Error("failed to save user to redis cache", slog.Int64("user_id", userID), slog.String("error", err.Error()))
	} else {
		log.Debug("user saved to cache", slog.Int64("user_id", userID))
	}

	return models.FromDomainToUsecase(dbUser), nil
}
