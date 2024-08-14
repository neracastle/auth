package grpc_server

import (
	"context"

	userdesc "github.com/neracastle/auth/pkg/user_v1"
)

// GetRefreshToken перевыпускает refresh токен
func (s *Server) GetRefreshToken(ctx context.Context, req *userdesc.RefreshRequest) (*userdesc.RefreshResponse, error) {
	token, err := s.srv.Renewal(ctx, req.GetRefreshToken(), false)
	if err != nil {
		return nil, err
	}

	return &userdesc.RefreshResponse{RefreshToken: token}, nil
}
