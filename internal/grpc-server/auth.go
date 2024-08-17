package grpc_server

import (
	"context"

	userdesc "github.com/neracastle/auth/pkg/user_v1"
)

// Auth авторизация пользователя
func (s *Server) Auth(ctx context.Context, req *userdesc.AuthRequest) (*userdesc.AuthResponse, error) {
	user, err := s.srv.Auth(ctx, req.GetLogin(), req.GetPassword())
	if err != nil {
		return nil, err
	}

	return &userdesc.AuthResponse{
		AccessToken:  user.AccessToken,
		RefreshToken: user.RefreshToken,
	}, nil
}
