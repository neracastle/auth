package models

import (
	"github.com/neracastle/chat-server/pkg/chat_v1"

	"github.com/neracastle/auth/internal/domain/user"
	"github.com/neracastle/auth/pkg/user_v1"
	"github.com/neracastle/auth/pkg/user_v1/auth"
)

// FromDomainToUsecase преобразует доменную сущность в дто из сервисного слоя
func FromDomainToUsecase(dbUser *user.User) UserDTO {
	return UserDTO{
		ID:        dbUser.ID,
		Email:     dbUser.Email,
		Password:  dbUser.Password,
		Name:      dbUser.Name,
		IsAdmin:   dbUser.IsAdmin,
		CreatedAt: dbUser.RegDate,
	}
}

// FromDomainToJWT преобразует доменную сущность в дто для генерации токенов
func FromDomainToJWT(dbUser *user.User) auth.JWTUser {
	return auth.JWTUser{
		ID:      dbUser.ID,
		IsAdmin: dbUser.IsAdmin,
		Scope: []string{
			user_v1.UserV1_Get_FullMethodName,
			user_v1.UserV1_Update_FullMethodName,
			user_v1.UserV1_Delete_FullMethodName,
			chat_v1.ChatV1_Create_FullMethodName,
			chat_v1.ChatV1_Delete_FullMethodName,
		},
	}
}
