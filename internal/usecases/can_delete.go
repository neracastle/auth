package usecases

import (
	"context"
	"log/slog"

	"github.com/neracastle/go-libs/pkg/sys/logger"
	"github.com/neracastle/go-libs/pkg/sys/tracer"
	"go.opentelemetry.io/otel/trace"

	"github.com/neracastle/auth/pkg/user_v1/auth"
)

// CanDelete проверяет, есть ли у пользователя права на удаление чего-либо в системе
func (s *Service) CanDelete(ctx context.Context, userID int64) bool {
	const method = "usecases.CanDelete"
	log := logger.GetLogger(ctx)
	log.Debug("called", slog.String("method", method), slog.Int64("user_id", userID))

	var span trace.Span
	ctx, span = tracer.Span(ctx, method)
	defer span.End()

	tokenUser := auth.UserFromContext(ctx)

	return tokenUser.IsAdmin
}
