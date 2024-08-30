package usecases

import (
	"context"
	"errors"
	"time"

	syserr "github.com/neracastle/go-libs/pkg/sys/error"
	"github.com/neracastle/go-libs/pkg/sys/logger"
	"golang.org/x/exp/slog"

	"github.com/neracastle/auth/internal/repository/action/postgres/model"
	userRepo "github.com/neracastle/auth/internal/repository/user"
	def "github.com/neracastle/auth/internal/usecases/models"
	"github.com/neracastle/auth/pkg/user_v1/auth"
)

// Update обновляет данные пользователя
func (s *Service) Update(ctx context.Context, user def.UpdateDTO) error {
	log := logger.GetLogger(ctx).With(slog.String("method", "usecases.Update"))
	log.Debug("called", slog.Int64("user_id", user.ID))

	tokenUser := auth.UserFromContext(ctx)
	//получить данные пользователь может только по себе, а админ по всем
	if tokenUser.ID != user.ID && !tokenUser.IsAdmin {
		return ErrUserPermissionDenied
	}

	dbUser, err := s.usersRepo.Get(ctx, userRepo.SearchFilter{ID: user.ID})

	if err != nil {
		if errors.Is(err, userRepo.ErrUserNotFound) {
			return ErrUserNotFound
		}

		return err
	}

	dbUser.Name = user.Name

	switch user.Role {
	case 1:
		dbUser.IsAdmin = false
	case 2:
		dbUser.IsAdmin = true
	}

	oldEmail := dbUser.Email
	err = dbUser.ChangeEmail(user.Email)
	if err != nil {
		return syserr.NewFromError(err, syserr.DomainLogic)
	}

	err = s.db.ReadCommitted(ctx, func(ctx context.Context) error {
		err = s.usersRepo.Update(ctx, dbUser)
		if err != nil {
			return err
		}

		err = s.actionsRepo.Save(ctx, model.ActionDTO{
			UserID:    dbUser.ID,
			Name:      "ChangeEmail",
			OldValue:  oldEmail,
			NewValue:  dbUser.Email,
			CreatedAt: time.Now(),
		})

		return err
	})

	if err != nil {
		return syserr.New("Не удалось обновить пользователя", syserr.Internal)
	}

	return nil
}
