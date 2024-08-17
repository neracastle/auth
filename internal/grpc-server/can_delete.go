package grpc_server

import (
	"context"

	userdesc "github.com/neracastle/auth/pkg/user_v1"
)

// CanDelete обновляет данные клиента
func (s *Server) CanDelete(ctx context.Context, req *userdesc.RightsRequest) (*userdesc.RightsResponse, error) {
	can := s.srv.CanDelete(ctx, req.GetUserID())

	return &userdesc.RightsResponse{Can: can}, nil
}
