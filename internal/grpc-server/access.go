package grpc_server

import (
	"context"

	userdesc "github.com/neracastle/auth/pkg/user_v1"
)

// GetAccessToken продлевает access-токен
func (s *Server) GetAccessToken(ctx context.Context, req *userdesc.AccessRequest) (*userdesc.AccessResponse, error) {
	token, err := s.srv.Renewal(ctx, req.GetRefreshToken(), true)
	if err != nil {
		return nil, err
	}

	return &userdesc.AccessResponse{AccessToken: token}, nil
}
