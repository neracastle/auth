package usecases

import (
	"context"
	"errors"

	"github.com/neracastle/go-libs/pkg/sys/logger"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slog"

	"github.com/neracastle/auth/internal/repository/user"
	"github.com/neracastle/auth/internal/tracer"
	"github.com/neracastle/auth/internal/usecases/models"
	"github.com/neracastle/auth/pkg/user_v1/auth"
)

// Get возвращает пользователя по его id
func (s *Service) Get(ctx context.Context, userID int64) (models.UserDTO, error) {
	const method = "usecases.Get"
	var span trace.Span
	ctx, span = tracer.Span(ctx, method, trace.WithAttributes(attribute.Int64("user_id", userID)))
	defer span.End()

	log := logger.GetLogger(ctx)
	tokenUser := auth.UserFromContext(ctx)
	log.Debug("called", slog.String("method", method), slog.Int64("user_id", userID), slog.Int64("auth_user_id", tokenUser.ID))

	//получить данные пользователь может только по себе, а админ по всем
	if tokenUser.ID != userID && !tokenUser.IsAdmin {
		return models.UserDTO{}, errors.New("нет доступа к данному id")
	}

	//проверяем сначало наличие в кэше
	span.AddEvent("query cache")
	dbUser, err := s.usersCache.GetByID(ctx, userID)

	if err == nil {
		log.Debug("user found in cache", slog.Int64("user_id", userID))
		return models.FromDomainToUsecase(dbUser), nil
	}

	if !errors.Is(err, user.ErrUserNotCached) {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.Error("failed to get user from redis cache", slog.Int64("user_id", userID), slog.String("error", err.Error()))
	} else {
		log.Debug("user not cached", slog.Int64("user_id", userID))
	}

	span.AddEvent("query db")
	dbUser, err = s.usersRepo.Get(ctx, user.SearchFilter{ID: userID})
	if err != nil {
		return models.UserDTO{}, err
	}

	//ошибка сохранения в кэш не влияет на выдачу результата, просто залогируем
	span.AddEvent("save to cache")
	err = s.usersCache.Save(ctx, dbUser, s.Config.CacheTTL)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.Error("failed to save user to redis cache", slog.Int64("user_id", userID), slog.String("error", err.Error()))
	} else {
		log.Debug("user saved to cache", slog.Int64("user_id", userID))
	}

	return models.FromDomainToUsecase(dbUser), nil
}
