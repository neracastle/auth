package usecases

import (
	"context"
	"math/rand/v2"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/neracastle/auth/internal/tracer"
)

// CanDelete проверяет, есть ли у пользователя права на удаление чего-либо в системе
func (s *Service) CanDelete(ctx context.Context, userID int64) bool {
	//dummy логика
	var span trace.Span
	ctx, span = tracer.Span(ctx, "can_delete")
	defer span.End()

	time.Sleep(time.Millisecond * time.Duration(rand.Uint32N(1000)))
	return userID > 0
}
