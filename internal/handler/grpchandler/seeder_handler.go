package grpchandler

import (
	"context"

	authv1 "github.com/maintainerd/auth/internal/gen/go/auth/v1"
	"github.com/maintainerd/auth/internal/service"
)

type SeederHandler struct {
	authv1.UnimplementedSeederServiceServer
	registerService service.RegisterService
}

func NewSeederHandler(registerService service.RegisterService) *SeederHandler {
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
