package grpchandler

import (
	"context"

	authv1 "github.com/maintainerd/auth/internal/gen/go/auth/v1"
	"github.com/maintainerd/auth/internal/service"
)

type SeederHandler struct {
	authv1.UnimplementedSeederServiceServer
	authService service.AuthService
}

func NewSeederHandler(authService service.AuthService) *SeederHandler {
	return &SeederHandler{
		authService: authService,
	}
}

func (s *SeederHandler) TriggerSeeder(ctx context.Context, req *authv1.TriggerSeederRequest) (*authv1.TriggerSeederResponse, error) {
	// 🔧 TODO: Call real seeding/auth service logic here
	user, err := s.authService.GetUserByEmail(req.GetTriggeredBy(), 1)
	if err != nil {
		return nil, err
	}

	return &authv1.TriggerSeederResponse{
		Success: true,
		Message: "Seeder triggered by " + user.Email,
	}, nil
}
