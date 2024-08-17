package usecases

import "context"

// CanDelete проверяет, есть ли у пользователя права на удаление чего-либо в системе
func (s *Service) CanDelete(ctx context.Context, userID int64) bool {
	//dummy логика
	return userID > 0
}
