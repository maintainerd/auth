package setup

import (
	"context"

	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/auth/internal/platform/runner"
	"gorm.io/gorm"
)

var setupRunSeeders = runner.RunSeeders

type SeederGRPCHandler struct {
	authv1.UnimplementedSeederServiceServer
	db *gorm.DB
}

func NewSeederGRPCHandler(db *gorm.DB) *SeederGRPCHandler {
	return &SeederGRPCHandler{db: db}
}

func (h *SeederGRPCHandler) TriggerSeeder(_ context.Context, _ *authv1.TriggerSeederRequest) (*authv1.TriggerSeederResponse, error) {
	if err := setupRunSeeders(h.db, "v0.1.0"); err != nil {
		return nil, err
	}
	return &authv1.TriggerSeederResponse{
		Success: true,
		Message: "seeders completed",
	}, nil
}
