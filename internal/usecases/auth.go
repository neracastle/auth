package usecases

import (
	"context"
	"errors"

	"github.com/neracastle/go-libs/pkg/sys/logger"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/slog"

	"github.com/neracastle/auth/internal/repository/user"
	"github.com/neracastle/auth/internal/usecases/models"
	"github.com/neracastle/auth/pkg/user_v1/auth"
)

// ErrWrongLoginOrPwd неверный логин или пароль
var ErrWrongLoginOrPwd = errors.New("неверный логин или пароль")

// Auth возвращает пользователя по его логину и паролю
func (s *Service) Auth(ctx context.Context, login string, pwd string) (models.AuthTokens, error) {
	log := logger.GetLogger(ctx)
	log.Debug("called", slog.String("method", "usecases.Auth"))

	dbUser, err := s.usersRepo.Get(ctx, user.SearchFilter{Email: login})
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return models.AuthTokens{}, ErrWrongLoginOrPwd
		}

		return models.AuthTokens{}, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(pwd))
	if err != nil {
		return models.AuthTokens{}, ErrWrongLoginOrPwd
	}

	jwtUser := models.FromDomainToJWT(dbUser)

	accessToken, err := auth.GenerateToken(jwtUser, []byte(s.Config.SecretKey), s.Config.AccessDuration)
	if err != nil {
		return models.AuthTokens{}, err
	}

	refreshToken, err := auth.GenerateToken(jwtUser, []byte(s.Config.SecretKey), s.Config.RefreshDuration)
	if err != nil {
		return models.AuthTokens{}, err
	}

	return models.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
