package usecases

import (
	"context"
	"errors"

	syserr "github.com/neracastle/go-libs/pkg/sys/error"
	"github.com/neracastle/go-libs/pkg/sys/logger"
	"github.com/neracastle/go-libs/pkg/sys/tracer"
	"go.opentelemetry.io/otel/trace"
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
	const method = "usecases.Auth"
	var span trace.Span
	ctx, span = tracer.Span(ctx, method)
	defer span.End()

	log := logger.GetLogger(ctx)
	log.Debug("called", slog.String("method", method))

	if login == "" || pwd == "" {
		return models.AuthTokens{}, syserr.NewFromError(ErrWrongLoginOrPwd, syserr.Unauthenticated)
	}

	dbUser, err := s.usersRepo.Get(ctx, user.SearchFilter{Email: login})
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return models.AuthTokens{}, syserr.NewFromError(ErrWrongLoginOrPwd, syserr.Unauthenticated)
		}

		return models.AuthTokens{}, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(pwd))
	if err != nil {
		return models.AuthTokens{}, syserr.NewFromError(ErrWrongLoginOrPwd, syserr.Unauthenticated)
	}

	span.AddEvent("generate tokens")
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
