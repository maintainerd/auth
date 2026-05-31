package setup

import (
	"context"

	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
)

type SeederHandler struct {
	authv1.UnimplementedSeederServiceServer
	registerService RegisterService
}

func NewSeederHandler(registerService RegisterService) *SeederHandler {
	return &SeederHandler{
		registerService: registerService,
	}
}

func (s *SeederHandler) TriggerSeeder(ctx context.Context, req *authv1.TriggerSeederRequest) (*authv1.TriggerSeederResponse, error) {
	return &authv1.TriggerSeederResponse{
		Success: true,
		Message: "Received",
	}, nil
}
